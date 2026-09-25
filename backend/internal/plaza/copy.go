package plaza

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

func (s *Service) Copy(user *model.User, workID string) (*CopyResult, error) {
	if user == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	if err := s.requireCopyEnabled(); err != nil {
		return nil, err
	}
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	work, err := s.listedWork(workID)
	if err != nil {
		return nil, err
	}
	if !work.AllowCopy {
		return nil, kernel.Forbidden("该作品未开放复制")
	}
	if last, lastErr := s.repo.LastPlazaEvent(work.ID, user.ID, model.PlazaEventCopy, time.Now().Add(-time.Minute)); lastErr == nil && last != nil {
		return nil, kernel.RateLimited("复制过于频繁，请稍后再试")
	} else if lastErr != nil && !errors.Is(lastErr, gorm.ErrRecordNotFound) {
		return nil, lastErr
	}
	snapshot, err := s.repo.PlazaSnapshot(work.SnapshotID)
	if err != nil {
		return nil, kernel.NotFound("作品快照不存在")
	}
	var doc map[string]any
	if json.Unmarshal([]byte(snapshot.PayloadJSON), &doc) != nil || doc == nil {
		return nil, kernel.NewAppError(500, "作品快照损坏")
	}
	assets, err := s.repo.PlazaSnapshotAssets(snapshot.ID)
	if err != nil {
		return nil, err
	}
	mapping := map[string]string{}
	for _, asset := range assets {
		copied, copyErr := s.duplicateResource(model.PlazaSystemUserID, user.ID, asset.AssetID, asset.Kind, asset.AssetID)
		if copyErr != nil {
			return nil, kernel.WrapAppError(400, "复制媒体失败", copyErr)
		}
		mapping[asset.AssetID] = copied.ID
	}
	rewritten, _ := rewriteDocumentResourceIDs(doc, mapping).(map[string]any)
	if rewritten == nil {
		rewritten = doc
	}
	projectID := kernel.NewID()
	title := "「" + strings.TrimSpace(work.Title) + "」副本"
	now := time.Now()
	rewritten["id"] = projectID
	rewritten["title"] = title
	rewritten["createdAt"] = now.UTC().Format(time.RFC3339Nano)
	rewritten["updatedAt"] = now.UTC().Format(time.RFC3339Nano)
	rewritten["sourcePlazaWorkId"] = work.ID
	rewritten["sourceAuthorId"] = work.AuthorID
	rewritten["sourceSnapshotId"] = snapshot.ID
	rewritten["chatSessions"] = []any{}
	rewritten["activeChatId"] = nil
	payload, err := json.Marshal(rewritten)
	if err != nil {
		return nil, err
	}
	project := &model.CanvasProject{
		ID:          projectID,
		UserID:      user.ID,
		Title:       title,
		PayloadJSON: string(payload),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(project); err != nil {
		return nil, err
	}
	work.CopyCount++
	work.Score = workScore(work)
	work.UpdatedAt = now
	if err := s.repo.Save(work); err != nil {
		return nil, err
	}
	_ = s.repo.Create(&model.PlazaEvent{ID: kernel.NewID(), WorkID: work.ID, UserID: user.ID, Kind: model.PlazaEventCopy, Day: now.Format("2006-01-02"), CreatedAt: now})
	return &CopyResult{ProjectID: projectID, Title: title}, nil
}
