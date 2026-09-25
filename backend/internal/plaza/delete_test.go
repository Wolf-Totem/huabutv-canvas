package plaza

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/gorm"
)

func TestHardDeleteAuditMetadataCapsAssetIDs(t *testing.T) {
	ids := make([]string, 21)
	for i := range ids {
		ids[i] = fmt.Sprintf("asset-%02d", i)
	}
	meta := hardDeleteAuditMetadata(&model.PlazaWork{Slug: "s", Title: "t", SourceProjectID: "ext:s"}, ids)
	shown, _ := meta["systemAssetIds"].([]string)
	if len(shown) != 20 || meta["systemAssetCount"] != 21 {
		t.Fatalf("metadata = %#v", meta)
	}
	if meta["slug"] != "s" || meta["title"] != "t" || meta["sourceProjectId"] != "ext:s" {
		t.Fatalf("metadata = %#v", meta)
	}
}

func TestHardDeletePlazaWorkNotFound(t *testing.T) {
	svc, _, _ := plazaTestService(t)
	_, err := svc.HardDelete(&model.User{ID: "admin", Role: model.UserRoleAdmin}, "missing-work")
	if appErr, ok := err.(*kernel.AppError); !ok || appErr.Status != 404 {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestDeleteExceptKeepsWorkAndRejectsEmptyID(t *testing.T) {
	svc, repo, _ := plazaTestService(t)
	now := time.Now()
	keep := listedWork("keep-id", "keep-slug", "author-a", "ext:keepkeepkeepkeepkeepkeepkeepkeep", "保留", now)
	other := listedWork("other-id", "other-slug", "author-a", "ext:otherotherotherotherotherotherot", "其他", now)
	if err := repo.Create(keep); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(other); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteExcept(""); err == nil {
		t.Fatal("expected empty keep id to fail")
	}
	if _, err := repo.PlazaWork(other.ID); err != nil {
		t.Fatal(err)
	}
	deleted, err := svc.DeleteExcept(keep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := repo.PlazaWork(keep.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.PlazaWork(other.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("other work err = %v", err)
	}
}

func TestListWorksPageSize80DoesNotCollapse(t *testing.T) {
	svc, repo, _ := plazaTestService(t)
	seedListedWorks(t, repo, 90)
	page, err := svc.ListWorks("", "new", "", 1, 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Works) != 80 || !page.HasMore {
		t.Fatalf("pageSize 80 -> len %d hasMore %v, want 80 true", len(page.Works), page.HasMore)
	}
	over, err := svc.ListWorks("", "new", "", 1, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(over.Works) != 80 {
		t.Fatalf("pageSize 200 collapsed to %d, want 80", len(over.Works))
	}
	fallback, err := svc.ListWorks("", "new", "", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(fallback.Works) != 24 {
		t.Fatalf("pageSize 0 -> len %d, want 24", len(fallback.Works))
	}
}

func TestAdminWorksPageSize100DoesNotCollapse(t *testing.T) {
	svc, repo, _ := plazaTestService(t)
	seedListedWorks(t, repo, 120)
	page, total, err := svc.AdminWorks("", 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != 120 || len(page) != 100 {
		t.Fatalf("pageSize 100 -> len %d total %d, want 100/120", len(page), total)
	}
	over, _, err := svc.AdminWorks("", 1, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(over) != 100 {
		t.Fatalf("pageSize 500 collapsed to %d, want 100", len(over))
	}
	fallback, _, err := svc.AdminWorks("", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(fallback) != 20 {
		t.Fatalf("pageSize 0 -> len %d, want 20", len(fallback))
	}
}

func TestHardDeletePlazaWorkRemovesPublicPageAndKeepsOtherCanvases(t *testing.T) {
	svc, repo, host := plazaTestService(t)
	now := time.Now()
	authorA := insertUser(t, repo, "author-a")
	authorB := insertUser(t, repo, "author-b")
	other := insertUser(t, repo, "other-user")
	const (
		uuidA    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		uuidB    = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		canvasA  = "plaza-" + uuidA
		canvasB  = "plaza-" + uuidB
		sameName = "同名作品"
	)
	if err := repo.CreateResource(&model.Resource{ID: "sys-cover-a", UserID: model.PlazaSystemUserID, Kind: "image", Status: model.ResourceStatusReady, MimeType: "image/png", Size: 4, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateResource(&model.Resource{ID: "author-watch-a", UserID: authorA.ID, Kind: "image", Status: model.ResourceStatusReady, MimeType: "image/png", Size: 4, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	workA := listedWork("work-a", "slug-a", authorA.ID, "ext:"+uuidA, sameName, now)
	workA.CoverAssetID = "sys-cover-a"
	workA.WatchAssetID = "author-watch-a"
	workA.SnapshotID = "snap-a"
	workB := listedWork("work-b", "slug-b", authorB.ID, "ext:"+uuidB, "另一件", now)
	workLocal := listedWork("work-local", "slug-local", authorA.ID, "local-canvas-1", sameName, now)
	workDown := listedWork("work-down", "slug-down", authorA.ID, "ext:dddddddddddddddddddddddddddddddd", "已下架", now)
	workDown.Status = model.PlazaWorkTakenDown
	for _, work := range []*model.PlazaWork{workA, workB, workLocal, workDown} {
		if err := repo.Create(work); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Create(&model.PlazaSnapshot{ID: "snap-a", WorkID: workA.ID, PayloadJSON: `{"secret":"SECRET_SNAPSHOT"}`, PayloadHash: "h", NodeCount: 3, CoverAssetID: "sys-cover-a", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.PlazaSnapshotAsset{SnapshotID: "snap-a", AssetID: "sys-cover-a", Kind: "image"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.PlazaSnapshotAsset{SnapshotID: "snap-a", AssetID: "author-watch-a", Kind: "image"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.PlazaWorkTag{WorkID: workA.ID, CategoryID: "cat-short"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.PlazaLike{WorkID: workA.ID, UserID: "viewer-1", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.PlazaLike{WorkID: workB.ID, UserID: "viewer-1", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.PlazaEvent{ID: "event-a", WorkID: workA.ID, UserID: authorA.ID, Kind: model.PlazaEventView, Day: "2026-09-25", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	application := &model.PlazaApplication{ID: "app-a", UserID: authorA.ID, ProjectID: "proj-a", WorkID: workA.ID, Title: "申请", Status: model.PlazaApplicationApproved, SubmittedAt: now, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(application); err != nil {
		t.Fatal(err)
	}
	canvases := []*model.CanvasProject{
		{ID: canvasA, UserID: authorA.ID, Title: sameName, PayloadJSON: `{"id":"` + canvasA + `"}`, CreatedAt: now, UpdatedAt: now},
		{ID: canvasB, UserID: other.ID, Title: "别人的广场主键", PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now},
		{ID: "other-same-title", UserID: other.ID, Title: sameName, PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now},
		{ID: "other-unrelated", UserID: other.ID, Title: "别的画布", PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now},
		{ID: "local-canvas-1", UserID: authorA.ID, Title: sameName, PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now},
	}
	for _, canvas := range canvases {
		if err := repo.Create(canvas); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Create(&model.CanvasShare{ID: "share-a", UserID: authorA.ID, ProjectID: canvasA, TokenHash: "hash-author-a", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.CanvasShare{ID: "share-other", UserID: other.ID, ProjectID: canvasA, TokenHash: "hash-other", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.CanvasUnitLink{ID: "link-a", ProjectID: "proj-a", CanvasID: canvasA, UnitID: "unit-a", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&model.CanvasUnitLink{ID: "link-other", ProjectID: "proj-other", CanvasID: "other-unrelated", UnitID: "unit-other", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	authorTask := &model.Task{ID: "task-author", UserID: authorA.ID, ProjectID: canvasA, Status: model.TaskStatusSucceeded, CreatedAt: now, UpdatedAt: now}
	otherTask := &model.Task{ID: "task-other", UserID: other.ID, ProjectID: canvasA, Status: model.TaskStatusSucceeded, CreatedAt: now, UpdatedAt: now}
	foreignTask := &model.Task{ID: "task-foreign", UserID: other.ID, ProjectID: canvasB, Status: model.TaskStatusSucceeded, CreatedAt: now, UpdatedAt: now}
	for _, task := range []*model.Task{authorTask, otherTask, foreignTask} {
		if err := repo.Create(task); err != nil {
			t.Fatal(err)
		}
	}

	admin := &model.User{ID: "admin-del", Username: "admin-del", Role: model.UserRoleAdmin}
	result, err := svc.HardDelete(admin, workA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || !result.Deleted || result.ID != workA.ID || result.Slug != workA.Slug {
		t.Fatalf("result = %#v", result)
	}
	if _, err := svc.PublicWork(workA.Slug, ""); err == nil {
		t.Fatal("expected public 404 after hard delete")
	} else if appErr, ok := err.(*kernel.AppError); !ok || appErr.Status != 404 {
		t.Fatalf("public work err = %v", err)
	}
	if _, err := repo.PlazaWork(workA.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("work err = %v", err)
	}
	if _, err := repo.PlazaLike(workA.ID, "viewer-1"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("like err = %v", err)
	}
	if _, err := repo.PlazaLike(workB.ID, "viewer-1"); err != nil {
		t.Fatalf("other like deleted: %v", err)
	}
	if count := countWhere(t, host, &model.PlazaEvent{}, "work_id = ?", workA.ID); count != 0 {
		t.Fatalf("events = %d", count)
	}
	if count := countWhere(t, host, &model.PlazaWorkTag{}, "work_id = ?", workA.ID); count != 0 {
		t.Fatalf("tags = %d", count)
	}
	if _, err := repo.PlazaSnapshot("snap-a"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("snapshot err = %v", err)
	}
	if count := countWhere(t, host, &model.PlazaSnapshotAsset{}, "snapshot_id = ?", "snap-a"); count != 0 {
		t.Fatalf("snapshot assets = %d", count)
	}
	storedApp, err := repo.PlazaApplication(application.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedApp.WorkID != "" {
		t.Fatalf("application work id = %q", storedApp.WorkID)
	}
	if _, err := repo.CanvasProject(canvasA); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("author plaza canvas err = %v", err)
	}
	if _, err := repo.CanvasShareForProject(authorA.ID, canvasA); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("author share err = %v", err)
	}
	if _, err := repo.CanvasShareForProject(other.ID, canvasA); err != nil {
		t.Fatalf("other user share deleted: %v", err)
	}
	if _, err := repo.CanvasUnitLink("proj-a", canvasA, "unit-a"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unit link err = %v", err)
	}
	if _, err := repo.CanvasUnitLink("proj-other", "other-unrelated", "unit-other"); err != nil {
		t.Fatalf("unrelated unit link deleted: %v", err)
	}
	storedAuthorTask, err := repo.Task(authorTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedAuthorTask.ProjectID != "" {
		t.Fatalf("author task project = %q", storedAuthorTask.ProjectID)
	}
	storedOtherTask, err := repo.Task(otherTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedOtherTask.ProjectID != canvasA {
		t.Fatalf("other task project = %q", storedOtherTask.ProjectID)
	}
	for _, id := range []string{canvasB, "other-same-title", "other-unrelated", "local-canvas-1"} {
		if _, err := repo.CanvasProject(id); err != nil {
			t.Fatalf("canvas %s deleted: %v", id, err)
		}
	}
	if _, err := repo.Resource("sys-cover-a"); err != nil {
		t.Fatalf("system resource deleted: %v", err)
	}
	if _, err := repo.Resource("author-watch-a"); err != nil {
		t.Fatalf("author resource deleted: %v", err)
	}
	if len(host.audits) != 1 {
		t.Fatalf("audits = %d", len(host.audits))
	}
	audit := host.audits[0]
	if audit.action != "plaza.work.hard_delete" || audit.targetType != "plaza_work" || audit.targetID != workA.ID {
		t.Fatalf("audit = %#v", audit)
	}
	meta, _ := audit.metadata.(map[string]any)
	if meta["slug"] != workA.Slug || meta["title"] != sameName || meta["sourceProjectId"] != workA.SourceProjectID {
		t.Fatalf("audit metadata = %#v", meta)
	}
	assetIDs, _ := meta["systemAssetIds"].([]string)
	if len(assetIDs) != 1 || assetIDs[0] != "sys-cover-a" || meta["systemAssetCount"] != 1 {
		t.Fatalf("audit assets = %#v", meta)
	}
	encoded := fmt.Sprint(meta)
	if strings.Contains(encoded, "SECRET_SNAPSHOT") || strings.Contains(encoded, "author-watch-a") {
		t.Fatalf("audit leaked %s", encoded)
	}

	if _, err := svc.HardDelete(admin, workB.ID); err != nil {
		t.Fatal(err)
	}
	foreign, err := repo.CanvasProject(canvasB)
	if err != nil {
		t.Fatal(err)
	}
	if foreign.UserID != other.ID {
		t.Fatalf("foreign canvas user = %s", foreign.UserID)
	}
	storedForeignTask, err := repo.Task(foreignTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedForeignTask.ProjectID != canvasB {
		t.Fatalf("foreign task project = %q", storedForeignTask.ProjectID)
	}
	if _, err := svc.HardDelete(admin, workLocal.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CanvasProject("local-canvas-1"); err != nil {
		t.Fatalf("non-ext canvas deleted: %v", err)
	}
	sameTitle, err := repo.CanvasProject("other-same-title")
	if err != nil {
		t.Fatal(err)
	}
	if sameTitle.Title != sameName || sameTitle.UserID != other.ID {
		t.Fatalf("same-title canvas = %#v", sameTitle)
	}
	if _, err := svc.HardDelete(admin, workDown.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.PlazaWork(workDown.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("taken down work err = %v", err)
	}
	if _, err := svc.PublicWork(workDown.Slug, ""); err == nil {
		t.Fatal("expected public 404 for deleted taken-down work")
	}
}

func seedListedWorks(t *testing.T, repo *repository.Repository, count int) {
	t.Helper()
	now := time.Now()
	for i := 0; i < count; i++ {
		work := listedWork(fmt.Sprintf("page-work-%03d", i), fmt.Sprintf("page-slug-%03d", i), model.PlazaSystemUserID, fmt.Sprintf("page-src-%03d", i), fmt.Sprintf("作品%03d", i), now)
		if err := repo.Create(work); err != nil {
			t.Fatal(err)
		}
	}
}

func listedWork(id, slug, authorID, source, title string, now time.Time) *model.PlazaWork {
	listed := now
	return &model.PlazaWork{
		ID: id, Slug: slug, AuthorID: authorID, SourceProjectID: source, Title: title,
		CategoryID: "cat-short", Status: model.PlazaWorkListed, BadgesJSON: "[]",
		ListedAt: &listed, CreatedAt: now, UpdatedAt: now,
	}
}

func insertUser(t *testing.T, repo *repository.Repository, id string) *model.User {
	t.Helper()
	now := time.Now()
	user := &model.User{ID: id, Username: id, DisplayName: id, Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(user); err != nil {
		t.Fatal(err)
	}
	return user
}

func countWhere(t *testing.T, host *memoryHost, modelValue any, query string, args ...any) int64 {
	t.Helper()
	var count int64
	if err := host.db.Model(modelValue).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}
