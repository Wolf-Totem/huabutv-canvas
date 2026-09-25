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
