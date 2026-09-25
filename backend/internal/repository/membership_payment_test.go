package repository

import (
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
)

func TestCompletePaymentOrderGrantsPermanentMembershipWithoutLedger(t *testing.T) {
	db := openPaymentTestDB(t)
	repo := New(db)
	order := model.PaymentOrder{
		ID: "mem-order-1", UserID: "user-1", IdempotencyKey: "mem-1", MerchantOrderNo: "merchant-mem-1",
		ProductID: "membership-permanent", ProductName: "永久订阅", ProviderID: "wechat-native", PluginID: "plugin-1",
		ProviderConfigID: "config-1", ProviderConfigVersion: 1, AmountFen: 9900, Currency: "CNY",
		CreditsMicrocredits: 0, ProductKind: model.ProductKindMembership, PlanSKU: model.MembershipSKUPermanent,
		Status: model.PaymentOrderPending, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	completed, granted, err := repo.CompletePaymentOrder("wechat-native", "merchant-mem-1", PaymentEvidence{
		ProviderTradeNo: "wechat-mem-1", ProviderStatus: "SUCCESS", AmountFen: 9900, Currency: "CNY", PaidAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !granted || completed.Status != model.PaymentOrderCredited {
		t.Fatalf("completion = %#v granted=%v", completed, granted)
	}
	row, err := repo.UserMembership("user-1")
	if err != nil || !row.PermanentActive {
		t.Fatalf("membership = %#v err=%v", row, err)
	}
	var ledgerCount int64
	if err := db.Model(&model.CreditLedgerEntry{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 0 {
		t.Fatalf("ledger count = %d", ledgerCount)
	}
}

func TestCompletePaymentOrderYearThenMonthCoversQuota(t *testing.T) {
	db := openPaymentTestDB(t)
	repo := New(db)
	if err := repo.AdminGrantMembership("user-1", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUAdvancedYear, DurationDays: 20, StorageQuotaBytes: 3 << 40,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "year",
	}, nil); err != nil {
		t.Fatal(err)
	}
	order := model.PaymentOrder{
		ID: "mem-order-2", UserID: "user-1", IdempotencyKey: "mem-2", MerchantOrderNo: "merchant-mem-2",
		ProductID: "membership-month", ProductName: "月卡", ProviderID: "wechat-native", PluginID: "plugin-1",
		ProviderConfigID: "config-1", ProviderConfigVersion: 1, AmountFen: 3000, Currency: "CNY",
		CreditsMicrocredits: 300 * 1_000_000, ProductKind: model.ProductKindMembership, PlanSKU: model.MembershipSKUAdvancedMonth,
		MembershipDurationDays: 30, StorageQuotaBytes: 1 << 30, Status: model.PaymentOrderPending, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.CompletePaymentOrder("wechat-native", "merchant-mem-2", PaymentEvidence{
		ProviderTradeNo: "wechat-mem-2", ProviderStatus: "SUCCESS", AmountFen: 3000, Currency: "CNY", PaidAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	row, err := repo.UserMembership("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if row.AdvancedPlanSKU != model.MembershipSKUAdvancedMonth || row.PlanStorageQuotaBytes != 1<<30 {
		t.Fatalf("cover = %#v", row)
	}
}

func TestCreateMembershipPaymentOrderSameKeyReturnsExisting(t *testing.T) {
	db := openPaymentTestDB(t)
	repo := New(db)
	order := &model.PaymentOrder{
		ID: "mem-order-3", UserID: "user-1", IdempotencyKey: "same-key", MerchantOrderNo: "merchant-mem-3",
		ProductID: "membership-month", ProductName: "月卡", ProviderID: "wechat-native",
		AmountFen: 3000, Currency: "CNY", ProductKind: model.ProductKindMembership, PlanSKU: model.MembershipSKUAdvancedMonth,
		Status: model.PaymentOrderCreated, ExpiresAt: time.Now().Add(time.Hour),
	}
	first, created, err := repo.CreateMembershipPaymentOrder(order, nil)
	if err != nil || !created {
		t.Fatalf("first create = created=%v err=%v", created, err)
	}
	again := *order
	again.ID = "mem-order-4"
	again.MerchantOrderNo = "merchant-mem-4"
	second, created, err := repo.CreateMembershipPaymentOrder(&again, nil)
	if err != nil || created {
		t.Fatalf("second create = created=%v err=%v", created, err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same order %s got %s", first.ID, second.ID)
	}
}
