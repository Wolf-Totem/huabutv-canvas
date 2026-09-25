package app

import (
	"encoding/json"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/database"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLandingHeroResourceIDsAndHolds(t *testing.T) {
	raw := json.RawMessage(`{
		"workflowTitle":"keep-me",
		"heroShowcase":{
			"create":{"title":"开始创作","href":"","imageResourceId":"img-create"},
			"banners":[{"id":"b1","title":"t","imageUrl":"https://old.example/a.jpg","imageResourceId":"img-banner","previewResourceId":"vid-banner","href":""}],
			"tiles":[{"id":"tile-model","title":"新模型","previewResourceId":"vid-tile"}]
		}
	}`)
	ids, err := landingHeroResourceIDs(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"img-create", "img-banner", "vid-banner", "vid-tile"} {
		if _, ok := ids[want]; !ok {
			t.Fatalf("missing %s in %#v", want, ids)
		}
	}
	normalized, err := normalizeAppearanceLanding(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(normalized), `"workflowTitle":"keep-me"`) {
		t.Fatalf("normalize dropped unrelated keys: %s", normalized)
	}
	if strings.Count(string(normalized), `"/create"`) < 2 {
		t.Fatalf("empty href should fall back: %s", normalized)
	}
	projected := projectPublicLanding(normalized)
	if !strings.Contains(string(projected), "/api/public/appearance/media/img-banner") {
		t.Fatalf("public projection missing media url: %s", projected)
	}
	if !strings.Contains(string(projected), `"imageUrl":"https://old.example/a.jpg"`) && !strings.Contains(string(projected), "img-banner") {
		t.Fatalf("expected resource id to win imageUrl: %s", projected)
	}
}

func TestAppearanceMediaUnknownIDIsNotFound(t *testing.T) {
	svc, admin := newAppearanceMediaTestService(t)
	_ = admin
	if _, err := svc.AppearanceMedia("missing", "display", 1400); err == nil {
		t.Fatal("expected missing media to fail")
	}
}

func TestAppearanceResourceReferencesIncludeLandingAndHolds(t *testing.T) {
	svc, admin := newAppearanceMediaTestService(t)
	pngBytes := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 32)...)
	uploaded, err := svc.UploadAppearanceMedia(admin, multipartFileHeader(t, "poster.png", "image/png", pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	refs := svc.appearanceResourceReferences([]string{uploaded.Resource.ID, "unrelated"})
	if len(refs[uploaded.Resource.ID]) == 0 {
		t.Fatalf("hold should block sweep: %#v", refs)
	}
	setting, err := svc.AdminAppearance(admin)
	if err != nil {
		t.Fatal(err)
	}
	setting.Landing = json.RawMessage(`{"heroShowcase":{"banners":[{"id":"b1","title":"t","imageResourceId":"` + uploaded.Resource.ID + `","href":"/create"}],"create":{"title":"开始创作","subtitle":"","href":"/create"},"tiles":[]}}`)
	if _, err := svc.UpdateAppearance(admin, setting.AppearanceSetting); err != nil {
		t.Fatal(err)
	}
	refs = svc.appearanceResourceReferences([]string{uploaded.Resource.ID})
	if len(refs[uploaded.Resource.ID]) == 0 {
		t.Fatalf("landing id should be referenced: %#v", refs)
	}
	public, err := svc.Appearance()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(public.Landing), uploaded.Resource.ID) {
		t.Fatalf("public landing should project media id: %s", public.Landing)
	}
	adminAfter, err := svc.AdminAppearance(admin)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(adminAfter.Landing), "/api/public/appearance/media/") {
		t.Fatal("admin landing must stay raw")
	}
}

func TestAppearanceMediaRejectsNonAdmin(t *testing.T) {
	svc, _ := newAppearanceMediaTestService(t)
	pngBytes := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 32)...)
	user := &model.User{ID: "plain", Username: "plain", Role: model.UserRoleUser, Status: model.UserStatusActive}
	if _, err := svc.UploadAppearanceMedia(user, multipartFileHeader(t, "poster.png", "image/png", pngBytes)); err == nil {
		t.Fatal("expected non-admin upload to fail")
	}
}

func newAppearanceMediaTestService(t *testing.T) (*Service, *model.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+newID()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := database.MigrateSchema(db); err != nil {
		t.Skipf("schema unavailable: %v", err)
	}
	admin := &model.User{ID: "appearance-media-admin", Username: "appearance-media-admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(admin).Error; err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	return New(repository.New(db), t.TempDir()), admin
}

func TestQuantizeAppearanceMediaWidth(t *testing.T) {
	if got := quantizeAppearanceMediaWidth(0); got != 1400 {
		t.Fatalf("default width = %d", got)
	}
	if got := quantizeAppearanceMediaWidth(900); got != 800 {
		t.Fatalf("900 -> %d", got)
	}
	if got := quantizeAppearanceMediaWidth(1600); got != 1400 {
		t.Fatalf("1600 -> %d", got)
	}
	if got := quantizeAppearanceMediaWidth(1900); got != 1920 {
		t.Fatalf("1900 -> %d", got)
	}
}
