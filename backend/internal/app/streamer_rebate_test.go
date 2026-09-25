package app

import (
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
)

func TestStreamerConsumptionRebateCreditsAgentOnce(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan", DisplayName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	fan := seedStreamerUser(t, svc, "fan-1", "fan")
	now := time.Now()
	if _, err := svc.repo.BindUserStreamerInviteOnce(fan.ID, created.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.BillingOrder{
		ID: "settled-rebate", UserID: fan.ID, IdempotencyKey: "rebate-1", Status: model.BillingStatusSettled,
		ActualAmountMicrocredits: 10 * CreditScale, RefundedAmountMicrocredits: 0, SettledAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.ApplyStreamerConsumptionRebate("settled-rebate"); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.ApplyStreamerConsumptionRebate("settled-rebate"); err != nil {
		t.Fatal(err)
	}
	account, err := svc.repo.CreditAccount(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if account.AvailableMicrocredits != 0 {
		t.Fatalf("rebate must not credit spendable balance, got %d", account.AvailableMicrocredits)
	}
	owned, err := svc.repo.StreamerByUserID(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if owned.RebateTotalMicrocredits != CreditScale {
		t.Fatalf("wallet total = %d, want 1000000", owned.RebateTotalMicrocredits)
	}
	summary, err := svc.StreamerConsoleSummary(owner, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RebateRateBps != model.DefaultStreamerRebateRateBps {
		t.Fatalf("rate = %d", summary.RebateRateBps)
	}
	if summary.RebateCredits != 1 || summary.ExpectedRebateCredits != 1 {
		t.Fatalf("summary rebate = %+v", summary)
	}
}

func TestStreamerOwnSpendDoesNotRebate(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := svc.repo.Create(&model.BillingOrder{
		ID: "self-spend", UserID: owner.ID, IdempotencyKey: "self", Status: model.BillingStatusSettled,
		ActualAmountMicrocredits: 8 * CreditScale, SettledAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.BindUserStreamerInviteOnce(owner.ID, created.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.ApplyStreamerConsumptionRebate("self-spend"); err != nil {
		t.Fatal(err)
	}
	account, err := svc.repo.CreditAccount(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if account.AvailableMicrocredits != 0 {
		t.Fatalf("self spend must not rebate, got %d", account.AvailableMicrocredits)
	}
}

func TestAdminUpdateStreamerRebateRate(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan"})
	if err != nil {
		t.Fatal(err)
	}
	rate := 2500
	updated, err := svc.AdminUpdateStreamer(admin, created.ID, UpdateStreamerRequest{RebateRateBps: &rate})
	if err != nil {
		t.Fatal(err)
	}
	if updated.RebateRateBps != 2500 {
		t.Fatalf("rate = %d", updated.RebateRateBps)
	}
	bad := 12000
	if _, err := svc.AdminUpdateStreamer(admin, created.ID, UpdateStreamerRequest{RebateRateBps: &bad}); err == nil {
		t.Fatal("over 100% should fail")
	}
}

func TestStreamerRebateUsesModelShareAndCapabilityRate(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan"})
	if err != nil {
		t.Fatal(err)
	}
	videoRate := 1000
	if _, err := svc.AdminUpdateStreamer(admin, created.ID, UpdateStreamerRequest{VideoRebateBps: &videoRate}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.LogicalModel{ID: "logical-video", Code: "video-a", Name: "Video A", Capability: "video", SourceChannelModelID: "cm-video", AgentShareEnabled: true, AgentShareBps: 5000}); err != nil {
		t.Fatal(err)
	}
	fan := seedStreamerUser(t, svc, "fan-1", "fan")
	now := time.Now()
	if _, err := svc.repo.BindUserStreamerInviteOnce(fan.ID, created.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.BillingOrder{
		ID: "video-rebate", UserID: fan.ID, IdempotencyKey: "video-1", Status: model.BillingStatusSettled,
		ChannelModelID: "cm-video", Capability: "video",
		ActualAmountMicrocredits: 10 * CreditScale, SettledAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.ApplyStreamerConsumptionRebate("video-rebate"); err != nil {
		t.Fatal(err)
	}
	owned, err := svc.repo.StreamerByUserID(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if owned.RebateTotalMicrocredits != CreditScale/2 {
		t.Fatalf("wallet total = %d, want 500000", owned.RebateTotalMicrocredits)
	}
}

func TestStreamerPayoutApproveAndReject(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	owner := seedStreamerUser(t, svc, "owner-1", "owner")
	created, err := svc.AdminCreateStreamer(admin, CreateStreamerRequest{UserID: owner.ID, Slug: "zhangsan"})
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := svc.repo.Streamer(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	wallet.RebateTotalMicrocredits = 5 * CreditScale
	if err := svc.repo.SaveStreamer(wallet); err != nil {
		t.Fatal(err)
	}
	owner, err = svc.repo.User(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	createdPayout, err := svc.CreateStreamerPayout(owner, CreateStreamerPayoutRequest{AmountCredits: 2, AlipayAccount: "13800138000", AlipayRealName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateStreamerPayout(owner, CreateStreamerPayoutRequest{AmountCredits: 1, AlipayAccount: "13800138000", AlipayRealName: "张三"}); err == nil {
		t.Fatal("second pending payout should fail")
	}
	if _, err := svc.AdminRejectStreamerPayout(admin, createdPayout.ID, RejectStreamerPayoutRequest{Reason: "账号不符"}); err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateStreamerPayout(owner, CreateStreamerPayoutRequest{AmountCredits: 3, AlipayAccount: "13800138000", AlipayRealName: "张三"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AdminApproveStreamerPayout(admin, second.ID); err != nil {
		t.Fatal(err)
	}
	row, err := svc.repo.Streamer(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.WithdrawableMicrocredits() != 2*CreditScale || row.RebateWithdrawnMicrocredits != 3*CreditScale || row.RebatePendingMicrocredits != 0 {
		t.Fatalf("wallet=%+v", row)
	}
}
