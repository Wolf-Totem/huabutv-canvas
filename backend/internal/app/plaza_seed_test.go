package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"infinite-canvas/backend/internal/auth"
	"infinite-canvas/backend/internal/database"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/gorm"
)

func TestAdminSeedPlazaExternalRejectsNonAdmin(t *testing.T) {
	svc := &Service{}
	_, err := svc.AdminSeedPlazaExternal(&model.User{ID: "user-1", Role: model.UserRoleUser}, nil)
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Status != 403 {
		t.Fatalf("err = %v", err)
	}
}

func TestSeedPlazaExternalRemapsUnknownAuthorAndSkipsUUID(t *testing.T) {
	svc, db := newPlazaSeedService(t)
	now := time.Now()
	admin := &model.User{ID: "admin-1", Username: "admin-1", Role: model.UserRoleAdmin, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: now, UpdatedAt: now}
	known := &model.User{ID: "known-author", Username: "known-author", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: now.Add(time.Second), UpdatedAt: now}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(known).Error; err != nil {
		t.Fatal(err)
	}
	const keep = "0251b9ae0e304f7fb96e353eecfe2204"
	keepWork := &model.PlazaWork{
		ID: "keep-work", Slug: keep, AuthorID: admin.ID, SourceProjectID: "ext:" + keep,
		Title: "《山海奇都之听月楼惊变》", Status: model.PlazaWorkListed, BadgesJSON: "[]",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(keepWork).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.PlazaSnapshot{ID: "keep-snap", WorkID: keepWork.ID, PayloadJSON: `{"nodes":[]}`, PayloadHash: "h", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	useSeedGraph(t, func(uuid string) (*auth.LibTVImportResult, error) {
		return usableSeedGraph(uuid), nil
	})
	externalAuthor := "4feba512bfa52a601ce593358b0fbd78"
	items := []PlazaExternalSeedItem{
		seedItem(keep, externalAuthor),
		seedItem("11111111111111111111111111111111", known.ID),
		seedItem("22222222222222222222222222222222", externalAuthor),
		seedItem("33333333333333333333333333333333", "plaza-demo"),
		seedItem("44444444444444444444444444444444", ""),
		seedItem("55555555555555555555555555555555", externalAuthor),
	}
	report, err := svc.SeedPlazaExternal(items, 4, keep, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if report.Imported != 4 || report.Skipped != 1 || report.Failed != 0 {
		t.Fatalf("report = %#v", report)
	}
	var stored model.PlazaWork
	if err := db.First(&stored, "id = ?", keepWork.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Title != "《山海奇都之听月楼惊变》" {
		t.Fatalf("keep title = %q", stored.Title)
	}
	var snapshots int64
	if err := db.Model(&model.PlazaSnapshot{}).Where("work_id = ?", keepWork.ID).Count(&snapshots).Error; err != nil {
		t.Fatal(err)
	}
	if snapshots != 1 {
		t.Fatalf("keep snapshots = %d", snapshots)
	}
	knownWork := mustPlazaWork(t, db, "11111111111111111111111111111111")
	if knownWork.AuthorID != known.ID {
		t.Fatalf("known author = %s", knownWork.AuthorID)
	}
	knownCanvas := mustCanvas(t, db, "plaza-11111111111111111111111111111111")
	if knownCanvas.UserID != known.ID {
		t.Fatalf("known canvas user = %s", knownCanvas.UserID)
	}
	externalWork := mustPlazaWork(t, db, "22222222222222222222222222222222")
	if externalWork.AuthorID != admin.ID {
		t.Fatalf("external author = %s", externalWork.AuthorID)
	}
	externalCanvas := mustCanvas(t, db, "plaza-22222222222222222222222222222222")
	if externalCanvas.UserID != admin.ID {
		t.Fatalf("external canvas user = %s", externalCanvas.UserID)
	}
	for _, slug := range []string{"33333333333333333333333333333333", "44444444444444444444444444444444"} {
		work := mustPlazaWork(t, db, slug)
		if work.AuthorID != admin.ID {
			t.Fatalf("remapped author for %s = %s", slug, work.AuthorID)
		}
	}
	if _, err := svc.repo.PlazaWorkBySlug("55555555555555555555555555555555"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("limit did not stop, err = %v", err)
	}
}

func TestSeedPlazaExternalSkipsUnusableGraph(t *testing.T) {
	svc, db := newPlazaSeedService(t)
	insertAdmin(t, db)
	uuid := "99999999999999999999999999999999"
	useSeedGraph(t, func(id string) (*auth.LibTVImportResult, error) {
		graph := usableSeedGraph(id)
		graph.ImportedNodeCount = 2
		graph.ImportedConnectionCount = 0
		graph.Nodes = graph.Nodes[:2]
		graph.Connections = nil
		return graph, nil
	})
	report, err := svc.SeedPlazaExternal([]PlazaExternalSeedItem{seedItem(uuid, "4feba512bfa52a601ce593358b0fbd78")}, 80, "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if report.Imported != 0 || report.Failed != 1 {
		t.Fatalf("report = %#v", report)
	}
	if _, err := svc.repo.PlazaWorkBySlug(uuid); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unusable graph imported, err = %v", err)
	}
	if _, err := svc.repo.CanvasProject("plaza-" + uuid); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unusable canvas created, err = %v", err)
	}
}

func TestSeedPlazaExternalLeavesForeignCanvas(t *testing.T) {
	svc, db := newPlazaSeedService(t)
	insertAdmin(t, db)
	now := time.Now()
	other := &model.User{ID: "other-user", Username: "other-user", Role: model.UserRoleUser, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(other).Error; err != nil {
		t.Fatal(err)
	}
	uuid := "cccccccccccccccccccccccccccccccc"
	if err := db.Create(&model.CanvasProject{ID: "plaza-" + uuid, UserID: other.ID, Title: "占用", PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	useSeedGraph(t, func(id string) (*auth.LibTVImportResult, error) {
		return usableSeedGraph(id), nil
	})
	report, err := svc.SeedPlazaExternal([]PlazaExternalSeedItem{seedItem(uuid, "admin-1")}, 80, "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if report.Imported != 0 || report.Failed != 1 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "画布主键仍被其他用户占用") {
		t.Fatalf("errors = %#v", report.Errors)
	}
	canvas, err := svc.repo.CanvasProject("plaza-" + uuid)
	if err != nil {
		t.Fatal(err)
	}
	if canvas.UserID != other.ID {
		t.Fatalf("canvas user = %s", canvas.UserID)
	}
	if _, err := svc.repo.PlazaWorkBySlug(uuid); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("work imported despite foreign canvas, err = %v", err)
	}
}

func TestSeedFailureTextStripsUpstreamNames(t *testing.T) {
	svc, db := newPlazaSeedService(t)
	insertAdmin(t, db)
	useSeedGraph(t, func(string) (*auth.LibTVImportResult, error) {
		return nil, errors.New("Get https://api.liblib.tv/canvas failed HTTP 404")
	})
	report, err := svc.SeedPlazaExternal([]PlazaExternalSeedItem{seedItem("abababababababababababababababab", "nobody")}, 1, "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 1 || len(report.Errors) != 1 {
		t.Fatalf("report = %#v", report)
	}
	lower := strings.ToLower(report.Errors[0])
	for _, needle := range []string{"libtv", "lumlum", "liblib"} {
		if strings.Contains(lower, needle) {
			t.Fatalf("error leaked upstream name: %s", report.Errors[0])
		}
	}
	if !strings.Contains(report.Errors[0], "HTTP 404") {
		t.Fatalf("error = %s", report.Errors[0])
	}
}

func TestAdminSeedPlazaExternalDoesNotDeleteAndStopsAt80(t *testing.T) {
	if plazaExternalSeedHTTPLimit != 80 {
		t.Fatalf("http limit = %d", plazaExternalSeedHTTPLimit)
	}
	svc, db := newPlazaSeedService(t)
	admin := insertAdmin(t, db)
	now := time.Now()
	stay := &model.PlazaWork{
		ID: "stay-id", Slug: "stay-slug", AuthorID: admin.ID, SourceProjectID: "ext:staystaystaystaystaystaystaystay",
		Title: "留下来的作品", Status: model.PlazaWorkListed, BadgesJSON: "[]", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(stay).Error; err != nil {
		t.Fatal(err)
	}
	var fetched int
	useSeedGraph(t, func(uuid string) (*auth.LibTVImportResult, error) {
		fetched++
		return usableSeedGraph(uuid), nil
	})
	items := make([]PlazaExternalSeedItem, 81)
	for i := range items {
		items[i] = seedItem(fmt.Sprintf("%032d", i+1), "4feba512bfa52a601ce593358b0fbd78")
	}
	report, err := svc.AdminSeedPlazaExternal(admin, items)
	if err != nil {
		t.Fatal(err)
	}
	if report.Imported != 80 || report.Failed != 0 || report.Deleted != 0 {
		t.Fatalf("report = %#v", report)
	}
	if fetched != 80 {
		t.Fatalf("fetched = %d, want 80", fetched)
	}
	var total int64
	if err := db.Model(&model.PlazaWork{}).Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != 81 {
		t.Fatalf("works = %d, want stay + 80", total)
	}
	stored := model.PlazaWork{}
	if err := db.First(&stored, "id = ?", stay.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Title != stay.Title {
		t.Fatalf("stay title = %q", stored.Title)
	}
	if _, err := svc.repo.PlazaWorkBySlug(fmt.Sprintf("%032d", 81)); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("81st item imported, err = %v", err)
	}
}

func newPlazaSeedService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := database.Open(database.Config{Driver: "sqlite", DSN: "file:" + name + "?mode=memory&cache=shared"})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(
		&model.User{},
		&model.PlazaWork{},
		&model.PlazaSnapshot{},
		&model.PlazaSnapshotAsset{},
		&model.PlazaWorkTag{},
		&model.CanvasProject{},
	); err != nil {
		t.Fatal(err)
	}
	return &Service{repo: repository.New(db)}, db
}

func insertAdmin(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	now := time.Now()
	admin := &model.User{ID: "admin-1", Username: "admin-1", Role: model.UserRoleAdmin, Status: model.UserStatusActive, PasswordHash: "x", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	return admin
}

func useSeedGraph(t *testing.T, fn func(uuid string) (*auth.LibTVImportResult, error)) {
	t.Helper()
	previous := seedPublicGraph
	seedPublicGraph = func(_ *Service, uuid string) (*auth.LibTVImportResult, error) {
		return fn(uuid)
	}
	t.Cleanup(func() { seedPublicGraph = previous })
}

func usableSeedGraph(uuid string) *auth.LibTVImportResult {
	return &auth.LibTVImportResult{
		ProjectUUID:             uuid,
		ProjectName:             "上游名称",
		ImportedNodeCount:       3,
		ImportedConnectionCount: 1,
		Nodes: []auth.LibTVCanvasNode{
			{ID: "n1", Type: "image", Title: "一"},
			{ID: "n2", Type: "text", Title: "二"},
			{ID: "n3", Type: "video", Title: "三"},
		},
		Connections: []auth.LibTVCanvasConnection{{ID: "c1", FromNodeID: "n1", ToNodeID: "n2"}},
	}
}

func seedItem(uuid, author string) PlazaExternalSeedItem {
	return PlazaExternalSeedItem{
		UUID: uuid, Slug: uuid, Title: "公开画布导入作品标题", AuthorID: author,
		CoverURL: "https://example.com/cover.jpg",
	}
}

func mustPlazaWork(t *testing.T, db *gorm.DB, slug string) model.PlazaWork {
	t.Helper()
	var work model.PlazaWork
	if err := db.First(&work, "slug = ?", slug).Error; err != nil {
		t.Fatal(err)
	}
	return work
}

func mustCanvas(t *testing.T, db *gorm.DB, id string) model.CanvasProject {
	t.Helper()
	var canvas model.CanvasProject
	if err := db.First(&canvas, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	return canvas
}
