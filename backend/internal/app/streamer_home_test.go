package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeStreamerHomePayloadRejectsHTMLAndUnknownBlocks(t *testing.T) {
	_, _, err := NormalizeStreamerHomePayload("landing-simple", json.RawMessage(`{"script":"<script>alert(1)</script>"}`))
	if err == nil {
		t.Fatal("unknown block should fail")
	}
	encoded, payload, err := NormalizeStreamerHomePayload("landing-simple", json.RawMessage(`{"hero":{"title":"<b>Hello</b>","subtitle":"ok"},"features":[{"title":"x"}],"cta":{"label":"Go","href":"javascript:alert(1)"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if payload.Hero.Title != "Hello" {
		t.Fatalf("title = %q", payload.Hero.Title)
	}
	if payload.CTA.Href != "" {
		t.Fatalf("javascript href leaked: %q", payload.CTA.Href)
	}
	if payload.Features != nil {
		t.Fatal("simple template should drop features")
	}
	if strings.Contains(string(encoded), "<") {
		t.Fatalf("html leaked: %s", encoded)
	}
}

func TestRenderStreamerHomeDefaultOffMeansNilWhenEmpty(t *testing.T) {
	rendered, err := RenderStreamerHome("landing-feature", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if rendered != nil {
		t.Fatalf("empty payload should render nil, got %s", rendered)
	}
}

func TestPublicSiteSkinHidesHomeWhenDisabled(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	user := seedStreamerUser(t, svc, "user-1", "one")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "zhangsan", DisplayName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	home, _ := json.Marshal(map[string]any{"hero": map[string]string{"title": "自定义"}})
	enabled := true
	if _, err := svc.AdminUpdateStreamerSkin(admin, created.ID, UpdateStreamerSkinRequest{
		Title: "张三画布", HomeTemplateID: "landing-simple", Home: home, CustomHomeEnabled: &enabled,
	}); err != nil {
		t.Fatal(err)
	}
	open, err := svc.PublicSiteSkin("zhangsan.huabutv.com")
	if err != nil {
		t.Fatal(err)
	}
	if !open.CustomHomeEnabled || open.Home == nil {
		t.Fatalf("enabled home missing: %+v", open)
	}
	off := false
	if _, err := svc.AdminUpdateStreamerSkin(admin, created.ID, UpdateStreamerSkinRequest{CustomHomeEnabled: &off}); err != nil {
		t.Fatal(err)
	}
	closed, err := svc.PublicSiteSkin("zhangsan.huabutv.com")
	if err != nil {
		t.Fatal(err)
	}
	if closed.CustomHomeEnabled || closed.Home != nil {
		t.Fatalf("disabled home leaked: %+v", closed)
	}
	official, err := svc.PublicSiteSkin("app.huabutv.com")
	if err != nil {
		t.Fatal(err)
	}
	if official.InviteLocked || official.StreamerActive {
		t.Fatalf("reserved host should be official skin: %+v", official)
	}
}

func TestPublicSiteSkinReturnsHeroVideoOverride(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	user := seedStreamerUser(t, svc, "user-1", "one")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "zhangsan", DisplayName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	video := "https://cdn.example.com/aa-hero.mp4"
	poster := "https://cdn.example.com/aa-hero.jpg"
	if _, err := svc.AdminUpdateStreamerSkin(admin, created.ID, UpdateStreamerSkinRequest{
		HeroVideoURL:  &video,
		HeroPosterURL: &poster,
	}); err != nil {
		t.Fatal(err)
	}
	open, err := svc.PublicSiteSkin("zhangsan.j11.net")
	if err != nil {
		t.Fatal(err)
	}
	if !open.StreamerActive {
		t.Fatal("expected active streamer skin")
	}
	if open.HeroVideoURL != video || open.HeroPosterURL != poster {
		t.Fatalf("hero media = %q %q", open.HeroVideoURL, open.HeroPosterURL)
	}
	bad := "javascript:alert(1)"
	if _, err := svc.AdminUpdateStreamerSkin(admin, created.ID, UpdateStreamerSkinRequest{HeroVideoURL: &bad}); err != nil {
		t.Fatal(err)
	}
	cleared, err := svc.PublicSiteSkin("zhangsan.j11.net")
	if err != nil {
		t.Fatal(err)
	}
	if cleared.HeroVideoURL != "" {
		t.Fatalf("javascript url leaked: %q", cleared.HeroVideoURL)
	}
	if _, err := svc.AdminUpdateStreamerSkin(admin, created.ID, UpdateStreamerSkinRequest{ClearHeroPoster: true}); err != nil {
		t.Fatal(err)
	}
	emptyPoster, err := svc.PublicSiteSkin("zhangsan.j11.net")
	if err != nil {
		t.Fatal(err)
	}
	if emptyPoster.HeroPosterURL != "" {
		t.Fatalf("poster should clear, got %q", emptyPoster.HeroPosterURL)
	}
}
