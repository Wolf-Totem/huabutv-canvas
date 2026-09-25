package app

import (
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
)

func TestCreateStreamerDuplicateSlug(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	first := seedStreamerUser(t, svc, "user-1", "one")
	second := seedStreamerUser(t, svc, "user-2", "two")
	if _, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: first.ID, Slug: "zhangsan", DisplayName: "张三"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: second.ID, Slug: "zhangsan", DisplayName: "李四"})
	if err == nil || !strings.Contains(err.Error(), "占用") {
		t.Fatalf("duplicate slug error = %v", err)
	}
}

func TestCreateStreamerRejectsReservedSlugAndMissingUser(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	user := seedStreamerUser(t, svc, "user-1", "one")
	if _, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "admin"}); err == nil {
		t.Fatal("reserved slug should fail")
	}
	if _, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: "missing", Slug: "okslug"}); err == nil {
		t.Fatal("missing user should fail")
	}
	if _, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "okslug"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "other"}); err == nil {
		t.Fatal("same user cannot be two streamers")
	}
}

func TestDisableStreamerStatus(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	user := seedStreamerUser(t, svc, "user-1", "one")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "zhangsan"})
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := svc.AdminSetStreamerStatus(admin, created.ID, model.StreamerStatusDisabled)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != model.StreamerStatusDisabled {
		t.Fatalf("status = %s", disabled.Status)
	}
	enabled, err := svc.AdminSetStreamerStatus(admin, created.ID, model.StreamerStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Status != model.StreamerStatusActive {
		t.Fatalf("status = %s", enabled.Status)
	}
}

func TestCreateStreamerInsertsEmptySkinAndPromotesAgentRole(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	user := seedStreamerUser(t, svc, "user-1", "one")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: user.ID, Slug: "zhangsan", DisplayName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	skin, err := svc.repo.SiteSkin(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if skin.HomePayloadJSON != "" || skin.LogoURL != "" {
		t.Fatalf("skin should start empty: %+v", skin)
	}
	reloaded, err := svc.repo.User(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Role != model.UserRoleAgent {
		t.Fatalf("creating a streamer should promote user to agent, got %s", reloaded.Role)
	}
	if created.RebateRateBps != model.DefaultStreamerRebateRateBps {
		t.Fatalf("default rebate = %d", created.RebateRateBps)
	}
	if !strings.HasPrefix(created.InviteCode, "HB-ZHANGSAN-") {
		t.Fatalf("invite code = %s", created.InviteCode)
	}
}

func TestNonAdminCannotCreateStreamer(t *testing.T) {
	svc := newStreamerTestService(t)
	user := seedStreamerUser(t, svc, "user-1", "one")
	if _, err := svc.AdminCreateStreamer(user, CreateStreamerRequest{UserID: user.ID, Slug: "zhangsan"}); err == nil {
		t.Fatal("non-admin create should fail")
	}
}
