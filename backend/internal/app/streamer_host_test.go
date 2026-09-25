package app

import (
	"testing"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStreamerSlugFromHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "legal subdomain", host: "zhangsan.huabutv.com", want: "zhangsan"},
		{name: "strips port", host: "zhangsan.huabutv.com:443", want: "zhangsan"},
		{name: "uppercase", host: "ZhangSan.HUABUTV.COM", want: "zhangsan"},
		{name: "reserved app", host: "app.huabutv.com", want: "app"},
		{name: "empty", host: "", want: ""},
		{name: "localhost", host: "localhost:3000", want: "localhost"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := StreamerSlugFromHost(test.host); got != test.want {
				t.Fatalf("StreamerSlugFromHost(%q) = %q, want %q", test.host, got, test.want)
			}
		})
	}
}

func TestReservedSlug(t *testing.T) {
	for _, slug := range ReservedStreamerSlugs() {
		if !IsReservedStreamerSlug(slug) {
			t.Fatalf("expected reserved slug %q", slug)
		}
		if err := ValidateStreamerSlug(slug); err == nil {
			t.Fatalf("ValidateStreamerSlug(%q) should fail", slug)
		}
	}
	if err := ValidateStreamerSlug("zhangsan"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStreamerSlug("-bad"); err == nil {
		t.Fatal("leading hyphen should fail")
	}
	if err := ValidateStreamerSlug("a"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStreamerSlug("canvas"); err == nil {
		t.Fatal("canvas is reserved for the main workspace host")
	}
	if err := ValidateStreamerSlug("agent"); err == nil {
		t.Fatal("agent is reserved for the streamer console host")
	}
}

func TestPublicHostsDefaultToJ11(t *testing.T) {
	t.Setenv("CANVAS_PUBLIC_PARENT_DOMAIN", "")
	t.Setenv("CANVAS_PUBLIC_AGENT_HOST", "")
	t.Setenv("CANVAS_PUBLIC_CANVAS_HOST", "")
	if PublicParentDomain() != "j11.net" {
		t.Fatalf("parent = %s", PublicParentDomain())
	}
	if PublicAgentHost() != "agent.j11.net" {
		t.Fatalf("agent = %s", PublicAgentHost())
	}
	if PublicCanvasHost() != "canvas.j11.net" {
		t.Fatalf("canvas = %s", PublicCanvasHost())
	}
	if StreamerLandingHost("a") != "a.j11.net" {
		t.Fatalf("landing = %s", StreamerLandingHost("a"))
	}
}

func TestResolveStreamerByHost(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	user := seedStreamerUser(t, svc, "user-1", "zhangsan-user")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "zhangsan", DisplayName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	if created.InviteCode == "" {
		t.Fatal("invite code should be generated")
	}

	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "legal host", host: "zhangsan.huabutv.com", want: created.ID},
		{name: "port stripped", host: "zhangsan.huabutv.com:443", want: created.ID},
		{name: "parent domain j11", host: "zhangsan.j11.net", want: created.ID},
		{name: "reserved app", host: "app.huabutv.com", want: ""},
		{name: "agent portal", host: "agent.j11.net", want: ""},
		{name: "canvas workspace", host: "canvas.j11.net", want: ""},
		{name: "unknown slug", host: "lisi.huabutv.com", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := svc.ResolveStreamerByHost(test.host)
			if err != nil {
				t.Fatal(err)
			}
			if test.want == "" {
				if got != nil {
					t.Fatalf("ResolveStreamerByHost(%q) = %+v, want nil", test.host, got)
				}
				return
			}
			if got == nil || got.ID != test.want {
				t.Fatalf("ResolveStreamerByHost(%q) = %+v, want id %s", test.host, got, test.want)
			}
		})
	}
}

func newStreamerTestService(t *testing.T) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Streamer{}, &model.SiteSkin{}, &model.StreamerRebate{}, &model.StreamerPayout{}, &model.LogicalModel{}, &model.CreditAccount{}, &model.CreditLedgerEntry{}, &model.BillingOrder{}, &model.SystemSetting{}, &model.AdminAuditEvent{}); err != nil {
		t.Fatal(err)
	}
	return &Service{repo: repository.New(db)}
}

func seedStreamerAdmin(t *testing.T, svc *Service) *model.User {
	t.Helper()
	admin := &model.User{ID: "admin-1", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive, DisplayName: "Admin"}
	if err := svc.repo.Create(admin); err != nil {
		t.Fatal(err)
	}
	return admin
}

func seedStreamerUser(t *testing.T, svc *Service, id, username string) *model.User {
	t.Helper()
	user := &model.User{ID: id, Username: username, Role: model.UserRoleUser, Status: model.UserStatusActive, DisplayName: username}
	if err := svc.repo.Create(user); err != nil {
		t.Fatal(err)
	}
	return user
}
