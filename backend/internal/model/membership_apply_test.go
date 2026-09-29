package model

import (
	"testing"
	"time"
)

func TestApplyMembershipSnapshotPermanentKeepsAdvanced(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	expires := now.Add(20 * 24 * time.Hour)
	current := UserMembership{UserID: "u1", AdvancedPlanSKU: MembershipSKUAdvancedYear, AdvancedExpiresAt: &expires, PlanStorageQuotaBytes: 3 << 40}
	next := ApplyMembershipSnapshot(current, MembershipGrantSnapshot{PlanSKU: MembershipSKUPermanent, Source: MembershipGrantSourcePayment}, now)
	if !next.PermanentActive || next.PermanentGrantedAt == nil {
		t.Fatalf("permanent = %#v", next)
	}
	if next.AdvancedPlanSKU != MembershipSKUAdvancedYear || next.PlanStorageQuotaBytes != 3<<40 {
		t.Fatalf("advanced should remain: %#v", next)
	}
}

func TestApplyMembershipSnapshotPaymentCoversYearToMonth(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	expires := now.Add(20 * 24 * time.Hour)
	current := UserMembership{UserID: "u1", AdvancedPlanSKU: MembershipSKUAdvancedYear, AdvancedExpiresAt: &expires, PlanStorageQuotaBytes: 3 << 40}
	next := ApplyMembershipSnapshot(current, MembershipGrantSnapshot{
		PlanSKU: MembershipSKUAdvancedMonth, DurationDays: 30, StorageQuotaBytes: 1 << 30, Source: MembershipGrantSourcePayment,
	}, now)
	if next.AdvancedPlanSKU != MembershipSKUAdvancedMonth || next.PlanStorageQuotaBytes != 1<<30 {
		t.Fatalf("cover = %#v", next)
	}
	wantEnd := expires.Add(30 * 24 * time.Hour)
	if next.AdvancedExpiresAt == nil || !next.AdvancedExpiresAt.Equal(wantEnd) {
		t.Fatalf("end = %v want %v", next.AdvancedExpiresAt, wantEnd)
	}
}

func TestNormalizeStorageGrantDays(t *testing.T) {
	if got := NormalizeStorageGrantDays(0); got != DefaultStorageGrantDays {
		t.Fatalf("zero = %d", got)
	}
	if got := NormalizeStorageGrantDays(-3); got != DefaultStorageGrantDays {
		t.Fatalf("negative = %d", got)
	}
	if got := NormalizeStorageGrantDays(10); got != 10 {
		t.Fatalf("custom = %d", got)
	}
	if got := NormalizeStorageGrantDays(MaxStorageGrantDays + 1); got != MaxStorageGrantDays {
		t.Fatalf("clamp = %d", got)
	}
	if end := StorageGrantEndsAt(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 0); end != nil {
		t.Fatalf("permanent end = %v", end)
	}
}

func TestMembershipPurchaseBlockSvipCannotBuyVip(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	expires := now.Add(10 * 24 * time.Hour)
	row := UserMembership{UserID: "u1", AdvancedPlanSKU: MembershipSKUSvipMonth, AdvancedExpiresAt: &expires, PlanStorageQuotaBytes: 80 << 30}
	if got := MembershipPurchaseBlockReason(row, MembershipSKUVipMonth, 0, now); got != MembershipPurchaseBlockSvipToVip {
		t.Fatalf("svip to vip = %q", got)
	}
	if got := MembershipPurchaseBlockReason(row, MembershipSKUSvipYear, 0, now); got != "" {
		t.Fatalf("svip renew = %q", got)
	}
}

func TestMembershipPurchaseBlockRenewalWindow(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	expires := now.Add(40 * 24 * time.Hour)
	row := UserMembership{UserID: "u1", AdvancedPlanSKU: MembershipSKUVipMonth, AdvancedExpiresAt: &expires}
	if got := MembershipPurchaseBlockReason(row, MembershipSKUSvipMonth, 0, now); got != MembershipPurchaseBlockRenewalWindow {
		t.Fatalf("window = %q", got)
	}
	near := now.Add(10 * 24 * time.Hour)
	row.AdvancedExpiresAt = &near
	if got := MembershipPurchaseBlockReason(row, MembershipSKUSvipMonth, 0, now); got != "" {
		t.Fatalf("vip can buy svip in window = %q", got)
	}
}

func TestApplyMembershipSnapshotPaymentCoversVipToSvip(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	expires := now.Add(10 * 24 * time.Hour)
	current := UserMembership{UserID: "u1", AdvancedPlanSKU: MembershipSKUVipMonth, AdvancedExpiresAt: &expires, PlanStorageQuotaBytes: 30 << 30}
	next := ApplyMembershipSnapshot(current, MembershipGrantSnapshot{
		PlanSKU: MembershipSKUSvipYear, DurationDays: 365, StorageQuotaBytes: 80 << 30, Source: MembershipGrantSourcePayment,
	}, now)
	if next.AdvancedPlanSKU != MembershipSKUSvipYear || next.PlanStorageQuotaBytes != 80<<30 {
		t.Fatalf("cover = %#v", next)
	}
	wantEnd := expires.Add(365 * 24 * time.Hour)
	if next.AdvancedExpiresAt == nil || !next.AdvancedExpiresAt.Equal(wantEnd) {
		t.Fatalf("end = %v want %v", next.AdvancedExpiresAt, wantEnd)
	}
}

func TestApplyMembershipSnapshotAdminKeepsHigherRank(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	expires := now.Add(40 * 24 * time.Hour)
	current := UserMembership{UserID: "u1", AdvancedPlanSKU: MembershipSKUAdvancedYear, AdvancedExpiresAt: &expires, PlanStorageQuotaBytes: 3 << 40}
	next := ApplyMembershipSnapshot(current, MembershipGrantSnapshot{
		PlanSKU: MembershipSKUAdvancedMonth, DurationDays: 30, StorageQuotaBytes: 1 << 30, Source: MembershipGrantSourceAdmin,
	}, now)
	if next.AdvancedPlanSKU != MembershipSKUAdvancedYear || next.PlanStorageQuotaBytes != 3<<40 {
		t.Fatalf("admin keep high = %#v", next)
	}
}
