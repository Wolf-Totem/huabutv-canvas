package plaza

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"infinite-canvas/backend/internal/assets"
	"infinite-canvas/backend/internal/database"
	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
)

type memoryHost struct {
	repo  *repository.Repository
	blobs map[string][]byte
}

func (h *memoryHost) OpenResource(userID, resourceID string) (*model.Resource, io.ReadCloser, error) {
	resource, err := h.repo.ResourceForUser(userID, resourceID)
	if err != nil {
		return nil, nil, err
	}
	payload := h.blobs[userID+"/"+resourceID]
	return resource, io.NopCloser(bytes.NewReader(payload)), nil
}

func (h *memoryHost) OpenResourceRange(userID string, resource *model.Resource, _ string) (*assets.ResourceStream, error) {
	payload := h.blobs[userID+"/"+resource.ID]
	return &assets.ResourceStream{Resource: resource, Body: io.NopCloser(bytes.NewReader(payload)), StatusCode: 200, ContentLength: int64(len(payload))}, nil
}

func (h *memoryHost) StoreResource(userID, kind, fileName, mimeType string, size int64, width, height int, durationMs int64, body io.Reader) (*model.Resource, error) {
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	resource := &model.Resource{
		ID: kernel.NewID(), UserID: userID, Kind: kind, Status: model.ResourceStatusReady,
		MimeType: mimeType, Size: int64(len(payload)), Width: width, Height: height, DurationMs: durationMs,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := h.repo.CreateResource(resource); err != nil {
		return nil, err
	}
	if h.blobs == nil {
		h.blobs = map[string][]byte{}
	}
	h.blobs[userID+"/"+resource.ID] = payload
	return resource, nil
}

func (h *memoryHost) PrepareResourceDelivery(userID string, resource *model.Resource, _ assets.ResourceDeliveryOptions) (*assets.ResourceDelivery, error) {
	return &assets.ResourceDelivery{Resource: resource}, nil
}

func (h *memoryHost) RecordAdminAudit(*model.User, string, string, string, string, any) error {
	return nil
}

func plazaTestService(t *testing.T) (*Service, *repository.Repository, *memoryHost) {
	t.Helper()
	db, err := database.Open(database.Config{Driver: "sqlite", DSN: "file:" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(
		&model.User{}, &model.UserIdentity{}, &model.SystemSetting{}, &model.CanvasProject{}, &model.CanvasShare{},
		&model.CanvasUnitLink{}, &model.Task{}, &model.Resource{},
		&model.PlazaCategory{}, &model.PlazaApplication{}, &model.PlazaWork{}, &model.PlazaWorkTag{},
		&model.PlazaSnapshot{}, &model.PlazaSnapshotAsset{}, &model.PlazaEvent{}, &model.PlazaLike{},
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_plaza_applications_pending_project ON plaza_applications (user_id, project_id) WHERE status = 'pending'").Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Create(&model.User{ID: model.PlazaSystemUserID, Username: "plaza-system", DisplayName: "作品广场", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "!", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.PlazaCategory{ID: "cat-short", Slug: "short-drama", Name: "短剧漫剧", Kind: model.PlazaCategoryGenre, Sort: 1, Enabled: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemSetting{Key: model.PlazaSettingKey, ValueJSON: `{"enabled":true,"applyEnabled":true,"copyEnabled":true,"publicWatch":true,"applyDailyLimit":5}`, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.New(db)
	host := &memoryHost{repo: repo, blobs: map[string][]byte{}}
	return New(repo, host), repo, host
}

func createAuthor(t *testing.T, repo *repository.Repository, host *memoryHost, userID string) (*model.User, *model.CanvasProject, string) {
	t.Helper()
	now := time.Now()
	user := &model.User{ID: userID, Username: userID, DisplayName: "作者", Email: userID + "@hidden.example", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(user); err != nil {
		t.Fatal(err)
	}
	resourceID := "res_" + userID
	resource := &model.Resource{ID: resourceID, UserID: userID, Kind: "image", Status: model.ResourceStatusReady, MimeType: "image/png", Size: 4, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateResource(resource); err != nil {
		t.Fatal(err)
	}
	host.blobs[userID+"/"+resourceID] = []byte("PNG!")
	payload := map[string]any{
		"id": "proj-" + userID, "title": "原画布",
		"nodes": []any{map[string]any{
			"id": "node-1", "type": "image", "title": "镜头", "position": map[string]any{"x": 0, "y": 0}, "width": 100, "height": 80,
			"metadata": map[string]any{"storageKey": "resource:" + resourceID, "prompt": "公开提示", "apiKey": "secret-key", "taskId": "task-hidden", "content": "data:image/png;base64,secret"},
		}},
		"connections": []any{},
	}
	raw, _ := json.Marshal(payload)
	project := &model.CanvasProject{ID: "proj-" + userID, UserID: userID, Title: "原画布", PayloadJSON: string(raw), CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(project); err != nil {
		t.Fatal(err)
	}
	return user, project, resourceID
}

func applyReq() ApplyRequest {
	return ApplyRequest{Title: "测试作品", Subtitle: "副标题", CategoryID: "cat-short", CoverNodeID: "node-1", AllowWatch: true, AllowProcessView: true, AllowCopy: true, OriginalityAck: true}
}

func TestPlazaApplyRejectsNonOwner(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	owner, project, _ := createAuthor(t, repo, host, "owner-1")
	other := &model.User{ID: "other-1", Username: "other-1", DisplayName: "路人", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Create(other); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Apply(other, project.ID, applyReq())
	if err == nil {
		t.Fatal("expected non-owner apply to fail")
	}
	if appErr, ok := err.(*kernel.AppError); !ok || appErr.Status != 403 {
		t.Fatalf("expected 403, got %v", err)
	}
	if _, err := svc.Apply(owner, project.ID, applyReq()); err != nil {
		t.Fatal(err)
	}
}

func TestPlazaApplyRejectsEmptyCanvas(t *testing.T) {
	svc, repo, _ := plazaTestService(t)
	user := &model.User{ID: "empty-user", Username: "empty-user", DisplayName: "空", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Create(user); err != nil {
		t.Fatal(err)
	}
	project := &model.CanvasProject{ID: "empty-proj", UserID: user.ID, Title: "空画布", PayloadJSON: `{"id":"empty-proj","nodes":[],"connections":[]}`, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Create(project); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Apply(user, project.ID, applyReq())
	if err == nil {
		t.Fatal("expected empty canvas to fail")
	}
	if appErr, ok := err.(*kernel.AppError); !ok || appErr.Status != 400 {
		t.Fatalf("expected 400, got %v", err)
	}
}

func TestPlazaApplyRejectsDuplicatePending(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, _ := createAuthor(t, repo, host, "dup-user")
	if _, err := svc.Apply(user, project.ID, applyReq()); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Apply(user, project.ID, applyReq())
	if err == nil {
		t.Fatal("expected duplicate pending to fail")
	}
	if appErr, ok := err.(*kernel.AppError); !ok || appErr.Status != 409 {
		t.Fatalf("expected 409, got %v", err)
	}
}

func TestPlazaSanitizeDropsSecrets(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, _ := createAuthor(t, repo, host, "secret-user")
	application, err := svc.Apply(user, project.ID, applyReq())
	if err != nil {
		t.Fatal(err)
	}
	row, err := repo.PlazaApplication(application.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(row.DraftPayloadJSON, "secret-key") || strings.Contains(row.DraftPayloadJSON, "task-hidden") || strings.Contains(row.DraftPayloadJSON, "apiKey") || strings.Contains(row.DraftPayloadJSON, "storageKey") {
		t.Fatalf("draft leaked secrets: %s", row.DraftPayloadJSON)
	}
}

func TestPlazaApproveUsesDraftNotLiveCanvas(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, resourceID := createAuthor(t, repo, host, "draft-user")
	application, err := svc.Apply(user, project.ID, applyReq())
	if err != nil {
		t.Fatal(err)
	}
	project.PayloadJSON = `{"id":"` + project.ID + `","title":"事后改过","nodes":[{"id":"node-1","type":"image","title":"新镜头","position":{"x":1,"y":1},"width":10,"height":10,"metadata":{"storageKey":"resource:` + resourceID + `","prompt":"不该出现"}}],"connections":[]}`
	if err := repo.Save(project); err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin-1", Username: "admin-1", DisplayName: "管理员", Role: model.UserRoleAdmin, Status: model.UserStatusActive, PasswordHash: "x"}
	work, err := svc.Approve(admin, application.ID)
	if err != nil {
		t.Fatal(err)
	}
	row := mustWork(t, repo, work.ID)
	snapshot, err := repo.PlazaSnapshot(row.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(snapshot.PayloadJSON, "不该出现") || strings.Contains(snapshot.PayloadJSON, "事后改过") {
		t.Fatalf("approve read live canvas: %s", snapshot.PayloadJSON)
	}
	if !strings.Contains(snapshot.PayloadJSON, "公开提示") {
		t.Fatalf("draft prompt missing: %s", snapshot.PayloadJSON)
	}
}

func TestPlazaTourListedAndUnlisted(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, _ := createAuthor(t, repo, host, "tour-user")
	application, err := svc.Apply(user, project.ID, applyReq())
	if err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin-2", Username: "admin-2", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	work, err := svc.Approve(admin, application.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublicSnapshot(work.Slug, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Unpublish(user, work.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublicSnapshot(work.Slug, nil); err == nil {
		t.Fatal("unlisted work should 404")
	} else if appErr, ok := err.(*kernel.AppError); !ok || appErr.Status != 404 {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestPlazaCopyDeepCopiesResources(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, authorRes := createAuthor(t, repo, host, "copy-author")
	application, err := svc.Apply(user, project.ID, applyReq())
	if err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin-3", Username: "admin-3", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	work, err := svc.Approve(admin, application.ID)
	if err != nil {
		t.Fatal(err)
	}
	copier := &model.User{ID: "copier-1", Username: "copier-1", DisplayName: "复制者", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Create(copier); err != nil {
		t.Fatal(err)
	}
	result, err := svc.Copy(copier, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProjectID == project.ID {
		t.Fatal("copy reused source project id")
	}
	copied, err := repo.CanvasProjectForUser(copier.ID, result.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(copied.PayloadJSON, authorRes) || strings.Contains(copied.PayloadJSON, "plaza-works") {
		t.Fatalf("copy still references source or plaza assets: %s", copied.PayloadJSON)
	}
	plazaAssets, err := repo.PlazaSnapshotAssets((mustWork(t, repo, work.ID)).SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range plazaAssets {
		if strings.Contains(copied.PayloadJSON, asset.AssetID) {
			t.Fatalf("copy still references plaza asset %s: %s", asset.AssetID, copied.PayloadJSON)
		}
	}
}

func TestPlazaTourSurvivesAuthorCanvasDelete(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, _ := createAuthor(t, repo, host, "del-author")
	application, err := svc.Apply(user, project.ID, applyReq())
	if err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin-4", Username: "admin-4", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	work, err := svc.Approve(admin, application.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteCanvasProject(user.ID, project.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublicSnapshot(work.Slug, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPlazaCopyDisabled(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, _ := createAuthor(t, repo, host, "flag-author")
	application, err := svc.Apply(user, project.ID, applyReq())
	if err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin-5", Username: "admin-5", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	work, err := svc.Approve(admin, application.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveSystemSetting(&model.SystemSetting{Key: model.PlazaSettingKey, ValueJSON: `{"enabled":true,"applyEnabled":true,"copyEnabled":false,"publicWatch":true,"applyDailyLimit":5}`}); err != nil {
		t.Fatal(err)
	}
	copier := &model.User{ID: "copier-2", Username: "copier-2", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Create(copier); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Copy(copier, work.ID)
	if err == nil {
		t.Fatal("expected copy disabled")
	}
	if appErr, ok := err.(*kernel.AppError); !ok || appErr.Status != 403 {
		t.Fatalf("expected 403, got %v", err)
	}
}

func TestPlazaPublicWorkOmitsPrivateAuthorFields(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	user, project, _ := createAuthor(t, repo, host, "privacy-author")
	application, err := svc.Apply(user, project.ID, applyReq())
	if err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin-6", Username: "admin-6", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	work, err := svc.Approve(admin, application.ID)
	if err != nil {
		t.Fatal(err)
	}
	public, err := svc.PublicWork(work.Slug, "")
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(public)
	text := string(encoded)
	if strings.Contains(text, "@hidden.example") || strings.Contains(text, "endpoint") || strings.Contains(text, "accessKey") {
		t.Fatalf("public work leaked private fields: %s", text)
	}
}

func mustWork(t *testing.T, repo *repository.Repository, id string) *model.PlazaWork {
	t.Helper()
	work, err := repo.PlazaWork(id)
	if err != nil {
		t.Fatal(err)
	}
	return work
}
