package app

import (
	"strings"
	"testing"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
)

func TestStreamerConsoleExcludesSelfAndFailedOrders(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan", DisplayName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	referred := seedStreamerUser(t, svc, "fan-1", "fan")
	other := seedStreamerUser(t, svc, "other-1", "other")
	now := time.Now()
	if _, err := svc.repo.BindUserStreamerInviteOnce(referred.ID, created.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.CreditAccount{UserID: referred.ID, AvailableMicrocredits: 2 * CreditScale}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.CreditAccount{UserID: owner.ID, AvailableMicrocredits: 99 * CreditScale}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.BillingOrder{
		ID: "settled-1", UserID: referred.ID, IdempotencyKey: "s1", Status: model.BillingStatusSettled,
		ActualAmountMicrocredits: 5 * CreditScale, RefundedAmountMicrocredits: CreditScale, SettledAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.BillingOrder{
		ID: "failed-1", UserID: referred.ID, IdempotencyKey: "f1", Status: model.BillingStatusUncertain,
		ActualAmountMicrocredits: 40 * CreditScale, SettledAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.BillingOrder{
		ID: "self-1", UserID: owner.ID, IdempotencyKey: "self", Status: model.BillingStatusSettled,
		ActualAmountMicrocredits: 70 * CreditScale, SettledAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.BillingOrder{
		ID: "other-1", UserID: other.ID, IdempotencyKey: "o1", Status: model.BillingStatusSettled,
		ActualAmountMicrocredits: 80 * CreditScale, SettledAt: &now,
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := svc.StreamerConsoleSummary(owner, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ReferredUserCount != 1 {
		t.Fatalf("count = %d", summary.ReferredUserCount)
	}
	if summary.ConsumedCredits != 4 {
		t.Fatalf("consumed = %v, want 4 (settled minus refund, exclude self/failed/other)", summary.ConsumedCredits)
	}
	if summary.RemainingCreditsSum != 2 {
		t.Fatalf("remaining = %v", summary.RemainingCreditsSum)
	}

	page, err := svc.StreamerConsoleUsers(owner, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page = %+v", page)
	}
	if page.Items[0].DisplayNameMasked != "f*" {
		t.Fatalf("masked name = %s", page.Items[0].DisplayNameMasked)
	}
	if strings.Contains(strings.ToLower(page.Items[0].UserIDMasked), "fan") {
		t.Fatalf("id leak: %s", page.Items[0].UserIDMasked)
	}

	if _, err := svc.StreamerConsoleSummary(other, nil, nil); err == nil {
		t.Fatal("other user should not open console")
	}
	disabled, err := svc.AdminSetStreamerStatus(admin, created.ID, model.StreamerStatusDisabled)
	if err != nil || disabled.Status != model.StreamerStatusDisabled {
		t.Fatalf("disable: %v %+v", err, disabled)
	}
	if _, err := svc.StreamerConsoleMe(owner, "zhangsan.huabutv.com"); err == nil {
		t.Fatal("disabled streamer console should 403")
	}
}

func TestStreamerConsoleHasNoCreditWriteAPI(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	if _, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.AdminAdjustCredits(owner, owner.ID, AdminCreditAdjustmentRequest{AmountMicrocredits: CreditScale, Note: "nope"})
	if err == nil {
		t.Fatal("streamer must not adjust credits through admin API")
	}
	if kernelErr, ok := err.(*kernel.AppError); !ok || kernelErr.Status != 403 {
		t.Fatalf("want 403, got %v", err)
	}
}
