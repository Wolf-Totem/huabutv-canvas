package app

import (
	"testing"
	"time"
)

func TestBindUserStreamerInviteWritesOnceAndSkipsSelf(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	fan := seedStreamerUser(t, svc, "fan-1", "fan")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: seedStreamerUser(t, svc, "owner-2", "lisi").ID, Slug: "lisi"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.BindUserStreamerInvite(owner.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	reloaded, err := svc.repo.User(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.InvitedByStreamerID != nil {
		t.Fatal("streamer must not bind themselves")
	}
	if err := svc.BindUserStreamerInvite(fan.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.BindUserStreamerInvite(fan.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	fanReloaded, err := svc.repo.User(fan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fanReloaded.InvitedByStreamerID == nil || *fanReloaded.InvitedByStreamerID != created.ID {
		t.Fatalf("first invite must stick, got %+v", fanReloaded.InvitedByStreamerID)
	}
	if fanReloaded.InvitedAt == nil || fanReloaded.InvitedAt.After(time.Now().Add(time.Minute)) {
		t.Fatal("invited_at missing")
	}
}
