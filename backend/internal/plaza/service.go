package plaza

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"infinite-canvas/backend/internal/assets"
	"infinite-canvas/backend/internal/canvas"
	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/gorm"
)

func (s *Service) Categories() ([]model.PlazaCategory, error) {
	return s.repo.PlazaCategories(true)
}

func (s *Service) Apply(user *model.User, projectID string, req ApplyRequest) (*ApplicationView, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	if err := s.requireApplyEnabled(); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, kernel.BadAuthRequest("缺少画布 ID")
	}
	project, err := s.repo.CanvasProjectForUser(user.ID, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, lookupErr := s.repo.CanvasProject(projectID); lookupErr == nil {
				return nil, kernel.Forbidden("只有画布所有者可以申请上架")
			}
			return nil, kernel.NotFound("画布不存在")
		}
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if n := utf8.RuneCountInString(title); n < 2 || n > 40 {
		return nil, kernel.BadAuthRequest("标题需要 2 到 40 个字")
	}
	subtitle := strings.TrimSpace(req.Subtitle)
	if utf8.RuneCountInString(subtitle) > 160 {
		return nil, kernel.BadAuthRequest("副标题不能超过 160 个字")
	}
	if !req.OriginalityAck {
		return nil, kernel.BadAuthRequest("请确认作品为原创或已获授权")
	}
	category, err := s.repo.PlazaCategory(strings.TrimSpace(req.CategoryID))
	if err != nil || category == nil || !category.Enabled || category.Slug == "all" {
		return nil, kernel.BadAuthRequest("请选择有效分类")
	}
	if _, err := s.repo.PendingPlazaApplication(user.ID, project.ID); err == nil {
		return nil, kernel.FailedPrecondition("该画布已有待审核申请")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	settings, err := s.Settings()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	count, err := s.repo.CountPlazaApplicationsSince(user.ID, dayStart)
	if err != nil {
		return nil, err
	}
	if int(count) >= settings.ApplyDailyLimit {
		return nil, kernel.RateLimited("今日申请次数已达上限")
	}
	doc, resourceSet, err := canvas.SanitizeCanvasDocument(project)
	if err != nil {
		return nil, kernel.BadAuthRequest(err.Error())
	}
	mediaCount := displayableMediaCount(doc)
	if mediaCount < 1 {
		return nil, kernel.BadAuthRequest("画布至少需要一个可展示的图片或视频节点")
	}
	coverNode := findNode(doc, req.CoverNodeID)
	if coverNode == nil {
		coverNode = firstDisplayableNode(doc)
	}
	if coverNode == nil {
		return nil, kernel.BadAuthRequest("请选择封面节点")
	}
	coverResourceID := nodeResourceID(coverNode)
	if coverResourceID == "" || !resourceSet[coverResourceID] {
		return nil, kernel.BadAuthRequest("封面必须来自当前画布的图片或视频节点")
	}
	watchResourceID := ""
	if strings.TrimSpace(req.WatchNodeID) != "" {
		watchNode := findNode(doc, req.WatchNodeID)
		if watchNode == nil || kernel.StringValue(watchNode["type"]) != "video" {
			return nil, kernel.BadAuthRequest("成片必须是画布中的视频节点")
		}
		watchResourceID = nodeResourceID(watchNode)
		if watchResourceID == "" || !resourceSet[watchResourceID] {
			return nil, kernel.BadAuthRequest("成片必须来自当前画布")
		}
	}
	campaignIDs, err := s.enabledCampaignIDs(req.CampaignIDs)
	if err != nil {
		return nil, err
	}
	workID := ""
	if existing, workErr := s.repo.PlazaWorkBySourceProject(project.ID); workErr == nil && existing.AuthorID == user.ID {
		workID = existing.ID
	} else if workErr != nil && !errors.Is(workErr, gorm.ErrRecordNotFound) {
		return nil, workErr
	}
	draft := plazaDraft{
		Document:        doc,
		ResourceIDs:     resourceIDList(resourceSet),
		CoverNodeID:     kernel.StringValue(coverNode["id"]),
		WatchNodeID:     strings.TrimSpace(req.WatchNodeID),
		CoverResourceID: coverResourceID,
		WatchResourceID: watchResourceID,
	}
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, err
	}
	application := &model.PlazaApplication{
		ID:               kernel.NewID(),
		UserID:           user.ID,
		ProjectID:        project.ID,
		WorkID:           workID,
		Title:            title,
		Subtitle:         subtitle,
		CategoryID:       category.ID,
		CampaignIDsJSON:  encodeStringList(campaignIDs),
		CoverResourceID:  coverResourceID,
		WatchResourceID:  watchResourceID,
		AllowWatch:       req.AllowWatch,
		AllowProcessView: req.AllowProcessView,
		AllowCopy:        req.AllowCopy,
		OriginalityAck:   true,
		Status:           model.PlazaApplicationPending,
		DraftPayloadJSON: string(draftJSON),
		NodeCount:        nodeCount(doc),
		MediaCount:       mediaCount,
		SubmittedAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.Create(application); err != nil {
		return nil, err
	}
	return s.applicationView(application), nil
}

func (s *Service) MyApplications(user *model.User) ([]ApplicationView, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	items, _, err := s.repo.ListPlazaApplications(user.ID, "", 100, 0)
	if err != nil {
		return nil, err
	}
	out := make([]ApplicationView, 0, len(items))
	for i := range items {
		view := s.applicationView(&items[i])
		view.Author = PublicAuthor{ID: user.ID, DisplayName: publicDisplayName(user)}
		out = append(out, *view)
	}
	return out, nil
}

func (s *Service) Withdraw(user *model.User, applicationID string) (*ApplicationView, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	application, err := s.repo.PlazaApplicationForUser(user.ID, applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.NotFound("申请不存在")
		}
		return nil, err
	}
	if application.Status != model.PlazaApplicationPending {
		return nil, kernel.FailedPrecondition("只能撤回待审核申请")
	}
	application.Status = model.PlazaApplicationWithdrawn
	now := time.Now()
	application.ReviewedAt = &now
	application.UpdatedAt = now
	if err := s.repo.Save(application); err != nil {
		return nil, err
	}
	return s.applicationView(application), nil
}

func (s *Service) MyWorks(user *model.User) ([]PublicWork, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	items, _, err := s.repo.ListPlazaWorks(user.ID, "", "", "new", 100, 0)
	if err != nil {
		return nil, err
	}
	return s.publicWorks(items, user.ID)
}

func (s *Service) Unpublish(user *model.User, workID string) (*PublicWork, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	work, err := s.repo.PlazaWork(workID)
	if err != nil {
		return nil, kernel.NotFound("作品不存在")
	}
	if work.AuthorID != user.ID {
		return nil, kernel.Forbidden("只能下架自己的作品")
	}
	if work.Status != model.PlazaWorkListed {
		return nil, kernel.FailedPrecondition("作品当前无法下架")
	}
	work.Status = model.PlazaWorkUnlisted
	work.UpdatedAt = time.Now()
	if err := s.repo.Save(work); err != nil {
		return nil, err
	}
	views, err := s.publicWorks([]model.PlazaWork{*work}, user.ID)
	if err != nil || len(views) == 0 {
		return nil, err
	}
	return &views[0], nil
}

func (s *Service) Republish(user *model.User, workID string) (*PublicWork, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	work, err := s.repo.PlazaWork(workID)
	if err != nil {
		return nil, kernel.NotFound("作品不存在")
	}
	if work.AuthorID != user.ID {
		return nil, kernel.Forbidden("只能重新上架自己的作品")
	}
	if work.Status != model.PlazaWorkUnlisted {
		return nil, kernel.FailedPrecondition("只能重新上架已下架的当前快照")
	}
	now := time.Now()
	work.Status = model.PlazaWorkListed
	work.ListedAt = &now
	work.UpdatedAt = now
	if err := s.repo.Save(work); err != nil {
		return nil, err
	}
	views, err := s.publicWorks([]model.PlazaWork{*work}, user.ID)
	if err != nil || len(views) == 0 {
		return nil, err
	}
	return &views[0], nil
}

func (s *Service) ListWorks(categorySlug, sort, viewerID string, page, pageSize int) (*WorkList, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 48 {
		pageSize = 24
	}
	categoryID := ""
	if slug := strings.TrimSpace(categorySlug); slug != "" && slug != "all" {
		category, err := s.repo.PlazaCategoryBySlug(slug)
		if err != nil {
			return nil, kernel.BadAuthRequest("分类不存在")
		}
		categoryID = category.ID
	}
	items, total, err := s.repo.ListPlazaWorks("", model.PlazaWorkListed, categoryID, sort, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	works, err := s.publicWorks(items, viewerID)
	if err != nil {
		return nil, err
	}
	return &WorkList{Works: works, HasMore: int64(page*pageSize) < total}, nil
}

func (s *Service) PublicWork(slugOrID, viewerID string) (*PublicWork, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	work, err := s.listedWork(slugOrID)
	if err != nil {
		return nil, err
	}
	views, err := s.publicWorks([]model.PlazaWork{*work}, viewerID)
	if err != nil || len(views) == 0 {
		return nil, err
	}
	return &views[0], nil
}

func (s *Service) PublicSnapshot(slugOrID string, viewer *model.User) (map[string]any, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	allowed, err := s.publicWatchAllowed(viewer)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, kernel.Unauthorized("请先登录后参观")
	}
	work, err := s.listedWork(slugOrID)
	if err != nil {
		return nil, err
	}
	if !work.AllowProcessView {
		return nil, kernel.Forbidden("该作品未开放制作过程")
	}
	snapshot, err := s.repo.PlazaSnapshot(work.SnapshotID)
	if err != nil {
		return nil, kernel.NotFound("作品快照不存在")
	}
	var doc map[string]any
	if json.Unmarshal([]byte(snapshot.PayloadJSON), &doc) != nil || doc == nil {
		return nil, kernel.NewAppError(500, "作品快照损坏")
	}
	return doc, nil
}

func (s *Service) OpenWorkAsset(slugOrID, assetID, rangeHeader string, viewer *model.User) (*assets.ResourceDelivery, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	work, err := s.listedWork(slugOrID)
	if err != nil {
		return nil, err
	}
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil, kernel.NotFound("资源不存在")
	}
	if work.WatchAssetID == assetID && (!work.AllowWatch || work.WatchAssetID == "") {
		return nil, kernel.NotFound("资源不存在")
	}
	if work.WatchAssetID == assetID {
		allowed, err := s.publicWatchAllowed(viewer)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, kernel.Unauthorized("请先登录后观看")
		}
	}
	snapshot, err := s.repo.PlazaSnapshot(work.SnapshotID)
	if err != nil {
		return nil, kernel.NotFound("资源不存在")
	}
	if _, err := s.repo.PlazaSnapshotAsset(snapshot.ID, assetID); err != nil {
		return nil, kernel.NotFound("资源不存在")
	}
	resource, err := s.repo.ResourceForUser(model.PlazaSystemUserID, assetID)
	if err != nil {
		return nil, kernel.NotFound("资源不存在")
	}
	return s.prepareOwnedDelivery(model.PlazaSystemUserID, resource, rangeHeader)
}

func (s *Service) RecordEvent(slugOrID, kind, userID string) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	switch kind {
	case model.PlazaEventView, model.PlazaEventWatch, model.PlazaEventTour:
	default:
		return kernel.BadAuthRequest("不支持的事件类型")
	}
	work, err := s.listedWork(slugOrID)
	if err != nil {
		return err
	}
	now := time.Now()
	event := &model.PlazaEvent{ID: kernel.NewID(), WorkID: work.ID, UserID: userID, Kind: kind, Day: now.Format("2006-01-02"), CreatedAt: now}
	if err := s.repo.Create(event); err != nil {
		return err
	}
	switch kind {
	case model.PlazaEventView:
		work.ViewCount++
	case model.PlazaEventWatch:
		work.WatchCount++
	case model.PlazaEventTour:
		work.TourCount++
	}
	work.Score = workScore(work)
	work.UpdatedAt = now
	return s.repo.Save(work)
}

func (s *Service) Like(user *model.User, workID string) (*PublicWork, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	work, err := s.listedWork(workID)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.CreatePlazaLike(&model.PlazaLike{WorkID: work.ID, UserID: user.ID, CreatedAt: time.Now()})
	if err != nil {
		return nil, err
	}
	if created {
		work.LikeCount++
		work.Score = workScore(work)
		work.UpdatedAt = time.Now()
		if err := s.repo.Save(work); err != nil {
			return nil, err
		}
		_ = s.repo.Create(&model.PlazaEvent{ID: kernel.NewID(), WorkID: work.ID, UserID: user.ID, Kind: model.PlazaEventLike, Day: time.Now().Format("2006-01-02"), CreatedAt: time.Now()})
	}
	views, err := s.publicWorks([]model.PlazaWork{*work}, user.ID)
	if err != nil || len(views) == 0 {
		return nil, err
	}
	views[0].Liked = true
	return &views[0], nil
}

func (s *Service) Unlike(user *model.User, workID string) (*PublicWork, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	work, err := s.listedWork(workID)
	if err != nil {
		return nil, err
	}
	like, err := s.repo.PlazaLike(work.ID, user.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		views, viewErr := s.publicWorks([]model.PlazaWork{*work}, user.ID)
		if viewErr != nil || len(views) == 0 {
			return nil, viewErr
		}
		views[0].Liked = false
		return &views[0], nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.repo.DeletePlazaLike(like.WorkID, like.UserID); err != nil {
		return nil, err
	}
	if work.LikeCount > 0 {
		work.LikeCount--
	}
	work.Score = workScore(work)
	work.UpdatedAt = time.Now()
	if err := s.repo.Save(work); err != nil {
		return nil, err
	}
	_ = s.repo.Create(&model.PlazaEvent{ID: kernel.NewID(), WorkID: work.ID, UserID: user.ID, Kind: model.PlazaEventUnlike, Day: time.Now().Format("2006-01-02"), CreatedAt: time.Now()})
	views, err := s.publicWorks([]model.PlazaWork{*work}, user.ID)
	if err != nil || len(views) == 0 {
		return nil, err
	}
	views[0].Liked = false
	return &views[0], nil
}

func (s *Service) AdminApplications(status string, page, pageSize int) ([]ApplicationView, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := s.repo.ListPlazaApplications("", status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	userIDs := make([]string, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.UserID)
	}
	users, err := s.repo.UsersByIDs(userIDs)
	if err != nil {
		return nil, 0, err
	}
	identities, err := s.repo.UserIdentitiesByUserIDs(userIDs)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ApplicationView, 0, len(items))
	for i := range items {
		view := s.applicationView(&items[i])
		if author, ok := users[items[i].UserID]; ok {
			view.Author = publicAuthor(author, identities[author.ID])
		}
		out = append(out, *view)
	}
	return out, total, nil
}

func (s *Service) AdminApplication(id string) (*ApplicationView, error) {
	application, err := s.repo.PlazaApplication(id)
	if err != nil {
		return nil, kernel.NotFound("申请不存在")
	}
	view := s.applicationView(application)
	if author, authorErr := s.repo.User(application.UserID); authorErr == nil {
		identities, _ := s.repo.UserIdentitiesByUserIDs([]string{author.ID})
		view.Author = publicAuthor(*author, identities[author.ID])
	}
	return view, nil
}

func (s *Service) AdminApplicationSnapshot(id string) (map[string]any, error) {
	application, err := s.repo.PlazaApplication(id)
	if err != nil {
		return nil, kernel.NotFound("申请不存在")
	}
	draft, err := parseDraft(application.DraftPayloadJSON)
	if err != nil {
		return nil, err
	}
	doc, err := cloneDocument(draft.Document)
	if err != nil {
		return nil, err
	}
	applyAdminDraftAssetURLs(doc, application.ID)
	return doc, nil
}

func (s *Service) OpenApplicationDraftAsset(applicationID, resourceID, rangeHeader string) (*assets.ResourceDelivery, error) {
	application, err := s.repo.PlazaApplication(applicationID)
	if err != nil {
		return nil, kernel.NotFound("申请不存在")
	}
	draft, err := parseDraft(application.DraftPayloadJSON)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, id := range draft.ResourceIDs {
		allowed[id] = true
	}
	if !allowed[resourceID] {
		return nil, kernel.NotFound("资源不存在")
	}
	resource, err := s.repo.ResourceForUser(application.UserID, resourceID)
	if err != nil {
		return nil, kernel.NotFound("资源不存在")
	}
	return s.prepareOwnedDelivery(application.UserID, resource, rangeHeader)
}

func (s *Service) prepareOwnedDelivery(userID string, resource *model.Resource, rangeHeader string) (*assets.ResourceDelivery, error) {
	delivery, err := s.host.PrepareResourceDelivery(userID, resource, assets.ResourceDeliveryOptions{})
	if err != nil || delivery == nil || delivery.RedirectURL != "" {
		return delivery, err
	}
	stream, err := s.host.OpenResourceRange(userID, resource, rangeHeader)
	if err != nil {
		return nil, err
	}
	delivery.Stream = stream
	return delivery, nil
}

func (s *Service) Approve(actor *model.User, applicationID string) (*PublicWork, error) {
	application, err := s.repo.PlazaApplication(applicationID)
	if err != nil {
		return nil, kernel.NotFound("申请不存在")
	}
	if application.Status != model.PlazaApplicationPending {
		return nil, kernel.FailedPrecondition("只能审核待处理申请")
	}
	draft, err := parseDraft(application.DraftPayloadJSON)
	if err != nil {
		return nil, err
	}
	// 硬约束：通过瞬间只使用申请草稿，禁止再读作者当前画布。
	copied, err := s.copyDraftResources(application.UserID, draft.ResourceIDs)
	if err != nil {
		return nil, err
	}
	doc, err := cloneDocument(draft.Document)
	if err != nil {
		return nil, err
	}
	rewritten, _ := rewriteDocumentResourceIDs(doc, copied).(map[string]any)
	if rewritten == nil {
		rewritten = doc
	}
	now := time.Now()
	work, err := s.repo.PlazaWorkBySourceProject(application.ProjectID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		work = &model.PlazaWork{
			ID:              kernel.NewID(),
			Slug:            plazaSlug(application.Title, kernel.NewID()),
			AuthorID:        application.UserID,
			SourceProjectID: application.ProjectID,
			CreatedAt:       now,
		}
		err = nil
	}
	if err != nil {
		return nil, err
	}
	applyPlazaAssetURLs(rewritten, work.ID)
	payload, err := json.Marshal(rewritten)
	if err != nil {
		return nil, err
	}
	coverAssetID := copied[draft.CoverResourceID]
	watchAssetID := copied[draft.WatchResourceID]
	snapshot := &model.PlazaSnapshot{
		ID:            kernel.NewID(),
		WorkID:        work.ID,
		ApplicationID: application.ID,
		PayloadJSON:   string(payload),
		PayloadHash:   hashPayload(payload),
		NodeCount:     application.NodeCount,
		MediaCount:    application.MediaCount,
		CoverAssetID:  coverAssetID,
		WatchAssetID:  watchAssetID,
		CreatedAt:     now,
	}
	assetRows := make([]model.PlazaSnapshotAsset, 0, len(copied))
	for sourceID, assetID := range copied {
		kind := model.PlazaAssetImage
		if sourceID == draft.WatchResourceID {
			kind = model.PlazaAssetWatch
		} else if sourceID == draft.CoverResourceID {
			kind = model.PlazaAssetCover
		}
		assetRows = append(assetRows, model.PlazaSnapshotAsset{SnapshotID: snapshot.ID, AssetID: assetID, SourceResourceID: sourceID, Kind: kind})
	}
	work.SnapshotID = snapshot.ID
	work.Title = application.Title
	work.Subtitle = application.Subtitle
	work.CategoryID = application.CategoryID
	work.CoverAssetID = coverAssetID
	work.WatchAssetID = watchAssetID
	work.Status = model.PlazaWorkListed
	work.AllowWatch = application.AllowWatch
	work.AllowProcessView = application.AllowProcessView
	work.AllowCopy = application.AllowCopy
	if work.BadgesJSON == "" {
		work.BadgesJSON = "[]"
	}
	work.ListedAt = &now
	work.UpdatedAt = now
	work.Score = workScore(work)
	application.Status = model.PlazaApplicationApproved
	application.ReviewerID = actor.ID
	application.ReviewedAt = &now
	application.WorkID = work.ID
	application.UpdatedAt = now
	tagIDs := append([]string{application.CategoryID}, decodeStringList(application.CampaignIDsJSON)...)
	err = s.repo.Transaction(func(tx *repository.Repository) error {
		if err := tx.Save(work); err != nil {
			return err
		}
		if err := tx.Create(snapshot); err != nil {
			return err
		}
		for i := range assetRows {
			if err := tx.Create(&assetRows[i]); err != nil {
				return err
			}
		}
		if err := tx.Save(application); err != nil {
			return err
		}
		return tx.ReplacePlazaWorkTags(work.ID, tagIDs)
	})
	if err != nil {
		return nil, err
	}
	if err := s.host.RecordAdminAudit(actor, "plaza.application.approve", "plaza_application", application.ID, "通过作品广场申请", map[string]any{"workId": work.ID, "snapshotId": snapshot.ID}); err != nil {
		return nil, err
	}
	views, err := s.publicWorks([]model.PlazaWork{*work}, "")
	if err != nil || len(views) == 0 {
		return nil, err
	}
	return &views[0], nil
}

func (s *Service) Reject(actor *model.User, applicationID, note string) (*ApplicationView, error) {
	application, err := s.repo.PlazaApplication(applicationID)
	if err != nil {
		return nil, kernel.NotFound("申请不存在")
	}
	if application.Status != model.PlazaApplicationPending {
		return nil, kernel.FailedPrecondition("只能审核待处理申请")
	}
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > 500 {
		return nil, kernel.BadAuthRequest("拒绝原因不能超过 500 字")
	}
	now := time.Now()
	application.Status = model.PlazaApplicationRejected
	application.ReviewerID = actor.ID
	application.ReviewNote = note
	application.ReviewedAt = &now
	application.UpdatedAt = now
	if err := s.repo.Save(application); err != nil {
		return nil, err
	}
	if err := s.host.RecordAdminAudit(actor, "plaza.application.reject", "plaza_application", application.ID, "拒绝作品广场申请", map[string]any{"note": note}); err != nil {
		return nil, err
	}
	return s.AdminApplication(application.ID)
}

func (s *Service) AdminWorks(status string, page, pageSize int) ([]PublicWork, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := s.repo.ListPlazaWorks("", status, "", "new", pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	works, err := s.publicWorks(items, "")
	return works, total, err
}

func (s *Service) TakeDown(actor *model.User, workID, note string) (*PublicWork, error) {
	work, err := s.repo.PlazaWork(workID)
	if err != nil {
		return nil, kernel.NotFound("作品不存在")
	}
	work.Status = model.PlazaWorkTakenDown
	work.UpdatedAt = time.Now()
	if err := s.repo.Save(work); err != nil {
		return nil, err
	}
	if err := s.host.RecordAdminAudit(actor, "plaza.work.take_down", "plaza_work", work.ID, "下架广场作品", map[string]any{"note": strings.TrimSpace(note)}); err != nil {
		return nil, err
	}
	views, err := s.publicWorks([]model.PlazaWork{*work}, "")
	if err != nil || len(views) == 0 {
		return nil, err
	}
	return &views[0], nil
}

func (s *Service) listedWork(slugOrID string) (*model.PlazaWork, error) {
	slugOrID = strings.TrimSpace(slugOrID)
	if slugOrID == "" {
		return nil, kernel.NotFound("作品不存在")
	}
	work, err := s.repo.PlazaWorkBySlug(slugOrID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		work, err = s.repo.PlazaWork(slugOrID)
	}
	if err != nil {
		return nil, kernel.NotFound("作品不存在")
	}
	if work.Status != model.PlazaWorkListed {
		return nil, kernel.NotFound("作品不存在")
	}
	return work, nil
}

func (s *Service) copyDraftResources(authorID string, resourceIDs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, sourceID := range resourceIDs {
		copied, err := s.duplicateResource(authorID, model.PlazaSystemUserID, sourceID, "", sourceID)
		if err != nil {
			return nil, kernel.WrapAppError(400, "复制广场媒体失败，请确认申请时的资源仍可读取", err)
		}
		out[sourceID] = copied.ID
	}
	return out, nil
}

func (s *Service) publicWorks(items []model.PlazaWork, viewerID string) ([]PublicWork, error) {
	userIDs := make([]string, 0, len(items))
	categoryIDs := make([]string, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.AuthorID)
		categoryIDs = append(categoryIDs, item.CategoryID)
	}
	users, err := s.repo.UsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}
	identities, err := s.repo.UserIdentitiesByUserIDs(userIDs)
	if err != nil {
		return nil, err
	}
	categories, err := s.repo.PlazaCategoriesByIDs(categoryIDs)
	if err != nil {
		return nil, err
	}
	out := make([]PublicWork, 0, len(items))
	for _, item := range items {
		view := PublicWork{
			ID: item.ID, Slug: item.Slug, Title: item.Title, Subtitle: item.Subtitle, Status: item.Status,
			AllowWatch: item.AllowWatch, AllowProcessView: item.AllowProcessView, AllowCopy: item.AllowCopy,
			Badges: decodeStringList(item.BadgesJSON), Score: item.Score,
			ViewCount: item.ViewCount, WatchCount: item.WatchCount, TourCount: item.TourCount, LikeCount: item.LikeCount, CopyCount: item.CopyCount,
			ListedAt: item.ListedAt, UpdatedAt: item.UpdatedAt,
		}
		if item.CoverAssetID != "" {
			view.CoverURL = plazaAssetURL(item.ID, item.CoverAssetID)
		} else if strings.TrimSpace(item.CoverExternalURL) != "" {
			view.CoverURL = strings.TrimSpace(item.CoverExternalURL)
		}
		if item.WatchAssetID != "" && item.AllowWatch {
			view.WatchURL = plazaAssetURL(item.ID, item.WatchAssetID)
		} else if item.AllowWatch && strings.TrimSpace(item.WatchExternalURL) != "" {
			view.WatchURL = strings.TrimSpace(item.WatchExternalURL)
		}
		if author, ok := users[item.AuthorID]; ok {
			view.Author = publicAuthor(author, identities[author.ID])
		} else {
			view.Author = PublicAuthor{ID: item.AuthorID, DisplayName: "创作者"}
		}
		if category, ok := categories[item.CategoryID]; ok {
			view.Category = PublicCategory{ID: category.ID, Slug: category.Slug, Name: category.Name, Kind: category.Kind}
		}
		if viewerID != "" {
			if _, likeErr := s.repo.PlazaLike(item.ID, viewerID); likeErr == nil {
				view.Liked = true
			}
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *Service) applicationView(application *model.PlazaApplication) *ApplicationView {
	draft, _ := parseDraft(application.DraftPayloadJSON)
	return &ApplicationView{
		ID: application.ID, UserID: application.UserID, ProjectID: application.ProjectID, WorkID: application.WorkID,
		Title: application.Title, Subtitle: application.Subtitle, CategoryID: application.CategoryID,
		CampaignIDs: decodeStringList(application.CampaignIDsJSON), CoverNodeID: draft.CoverNodeID, WatchNodeID: draft.WatchNodeID,
		AllowWatch: application.AllowWatch, AllowProcessView: application.AllowProcessView, AllowCopy: application.AllowCopy,
		Status: application.Status, ReviewNote: application.ReviewNote, NodeCount: application.NodeCount, MediaCount: application.MediaCount,
		SubmittedAt: application.SubmittedAt, ReviewedAt: application.ReviewedAt,
		Author: PublicAuthor{ID: application.UserID},
	}
}

func (s *Service) enabledCampaignIDs(ids []string) ([]string, error) {
	out := []string{}
	for _, id := range kernel.UniqueNonEmpty(ids) {
		category, err := s.repo.PlazaCategory(id)
		if err != nil || category == nil || !category.Enabled || category.Kind != model.PlazaCategoryCampaign {
			return nil, kernel.BadAuthRequest("活动标签无效")
		}
		out = append(out, category.ID)
	}
	return out, nil
}

func parseDraft(raw string) (plazaDraft, error) {
	var draft plazaDraft
	if json.Unmarshal([]byte(raw), &draft) != nil || draft.Document == nil {
		return plazaDraft{}, kernel.NewAppError(500, "申请草稿损坏")
	}
	return draft, nil
}

func firstDisplayableNode(doc map[string]any) map[string]any {
	rawNodes, _ := doc["nodes"].([]any)
	for _, raw := range rawNodes {
		node, _ := raw.(map[string]any)
		nodeType := kernel.StringValue(node["type"])
		if (nodeType == "image" || nodeType == "video") && nodeResourceID(node) != "" {
			return node
		}
	}
	return nil
}

func nodeCount(doc map[string]any) int {
	rawNodes, _ := doc["nodes"].([]any)
	return len(rawNodes)
}

func publicDisplayName(user *model.User) string {
	if user == nil {
		return "创作者"
	}
	return kernel.DefaultString(user.DisplayName, user.Username)
}

func publicAuthor(user model.User, identity model.UserIdentity) PublicAuthor {
	return PublicAuthor{ID: user.ID, DisplayName: publicDisplayName(&user), AvatarURL: strings.TrimSpace(identity.AvatarURL)}
}

func plazaSlug(title, id string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.TrimSpace(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			lastHyphen = false
			continue
		}
		if (unicode.IsSpace(r) || r == '-' || r == '_') && !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	base := strings.Trim(b.String(), "-")
	if base == "" {
		base = "work"
	}
	runes := []rune(base)
	if len(runes) > 48 {
		base = string(runes[:48])
	}
	short := strings.TrimSpace(id)
	if len(short) > 8 {
		short = short[:8]
	}
	return base + "-" + short
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
