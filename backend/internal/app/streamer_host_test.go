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
	for _, slug := range []string{"oss", "smtp", "track", "mx", "ns", "email"} {
		if !IsReservedStreamerSlug(slug) {
			t.Fatalf("expected reserved slug %q", slug)
		}
		if err := ValidateStreamerSlug(slug); err == nil {
			t.Fatalf("ValidateStreamerSlug(%q) should fail", slug)
		}
	}
}

func TestPublicHostsDefaultToHuabutv(t *testing.T) {
	t.Setenv("CANVAS_PUBLIC_PARENT_DOMAIN", "")
	t.Setenv("CANVAS_PUBLIC_AGENT_HOST", "")
	t.Setenv("CANVAS_PUBLIC_CANVAS_HOST", "")
	t.Setenv("CANVAS_PUBLIC_BACKUP_PARENT_DOMAIN", "")
	if PublicParentDomain() != "huabutv.com" {
		t.Fatalf("parent = %s", PublicParentDomain())
	}
	if PublicAgentHost() != "agent.huabutv.com" {
		t.Fatalf("agent = %s", PublicAgentHost())
	}
	if PublicCanvasHost() != "www.huabutv.com" {
		t.Fatalf("canvas = %s", PublicCanvasHost())
	}
	if StreamerLandingHost("a") != "a.huabutv.com" {
		t.Fatalf("landing = %s", StreamerLandingHost("a"))
	}
	if CanvasHostForParent("huabutv.com") != "www.huabutv.com" {
		t.Fatalf("canvas parent = %s", CanvasHostForParent("huabutv.com"))
	}
	if CanvasHostForParent("j11.net") != "canvas.j11.net" {
		t.Fatalf("j11 canvas = %s", CanvasHostForParent("j11.net"))
	}
	if AgentHostForParent("huabutv.com") != "agent.huabutv.com" {
		t.Fatalf("agent parent = %s", AgentHostForParent("huabutv.com"))
	}
	if AgentHostForParent("j11.net") != "agent.j11.net" {
		t.Fatalf("j11 agent = %s", AgentHostForParent("j11.net"))
	}
}

func TestPublicHostsFollowEnvOverrideToJ11(t *testing.T) {
	t.Setenv("CANVAS_PUBLIC_PARENT_DOMAIN", "j11.net")
	t.Setenv("CANVAS_PUBLIC_AGENT_HOST", "agent.j11.net")
	t.Setenv("CANVAS_PUBLIC_CANVAS_HOST", "canvas.j11.net")
	t.Setenv("CANVAS_PUBLIC_BACKUP_PARENT_DOMAIN", "")
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
	if matchingKnownParent("www.huabutv.com") != "huabutv.com" {
		t.Fatalf("www still maps to brand parent, got %q", matchingKnownParent("www.huabutv.com"))
	}
	if CanvasHostForParent("huabutv.com") != "www.huabutv.com" {
		t.Fatalf("huabutv canvas under j11 env = %s", CanvasHostForParent("huabutv.com"))
	}
	if CanvasHostForParent("j11.net") != "canvas.j11.net" {
		t.Fatalf("j11 canvas = %s", CanvasHostForParent("j11.net"))
	}
}

func TestKnownParentDomainsKeepJ11WhenBackupEmpty(t *testing.T) {
	t.Setenv("CANVAS_PUBLIC_PARENT_DOMAIN", "huabutv.com")
	t.Setenv("CANVAS_PUBLIC_BACKUP_PARENT_DOMAIN", "")
	parents := knownParentDomains()
	if !containsParent(parents, "huabutv.com") || !containsParent(parents, "j11.net") {
		t.Fatalf("known parents = %v", parents)
	}
	if matchingKnownParent("canvas.j11.net") != "j11.net" {
		t.Fatalf("j11 host = %q", matchingKnownParent("canvas.j11.net"))
	}
	if matchingKnownParent("www.huabutv.com:443") != "huabutv.com" {
		t.Fatalf("huabutv host = %q", matchingKnownParent("www.huabutv.com:443"))
	}
	if matchingKnownParent("localhost") != "" {
		t.Fatalf("localhost should not match a brand parent")
	}
}

func TestPublicSiteSkinFollowsRequestParent(t *testing.T) {
	t.Setenv("CANVAS_PUBLIC_PARENT_DOMAIN", "j11.net")
	t.Setenv("CANVAS_PUBLIC_AGENT_HOST", "agent.j11.net")
	t.Setenv("CANVAS_PUBLIC_CANVAS_HOST", "canvas.j11.net")
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	user := seedStreamerUser(t, svc, "user-1", "zhangsan-user")
	if _, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "zhangsan", DisplayName: "张三"}); err != nil {
		t.Fatal(err)
	}

	huabu, err := svc.PublicSiteSkin("zhangsan.huabutv.com")
	if err != nil {
		t.Fatal(err)
	}
	if huabu.ParentDomain != "huabutv.com" || huabu.CanvasHost != "www.huabutv.com" || huabu.AgentHost != "agent.huabutv.com" || huabu.LandingHost != "zhangsan.huabutv.com" {
		t.Fatalf("huabutv skin hosts = %+v", huabu)
	}

	backup, err := svc.PublicSiteSkin("zhangsan.j11.net")
	if err != nil {
		t.Fatal(err)
	}
	if backup.ParentDomain != "j11.net" || backup.CanvasHost != "canvas.j11.net" || backup.AgentHost != "agent.j11.net" || backup.LandingHost != "zhangsan.j11.net" {
		t.Fatalf("j11 skin hosts = %+v", backup)
	}

	me, err := svc.StreamerConsoleMe(user, "agent.huabutv.com")
	if err != nil {
		t.Fatal(err)
	}
	if me.Host != "zhangsan.huabutv.com" || me.CanvasURL != "https://www.huabutv.com" || me.AgentURL != "https://agent.huabutv.com" {
		t.Fatalf("console me = %+v", me)
	}
}

func containsParent(parents []string, want string) bool {
	for _, parent := range parents {
		if parent == want {
			return true
		}
	}
	return false
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
