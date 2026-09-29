package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
)

func TestResolveEffectiveStoredFileBytesPriority(t *testing.T) {
	svc := newResourceTestService(t)
	override := int64(10 << 30)
	if err := svc.repo.UpdateUserStorageOverride("user-1", "admin", &override); err != nil {
		t.Fatal(err)
	}
	resolved, err := svc.ResolveEffectiveStoredFileBytes("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Bytes != override || resolved.Source != model.QuotaSourceOverride {
		t.Fatalf("override = %#v", resolved)
	}
	if err := svc.repo.UpdateUserStorageOverride("user-1", "admin", nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.AdminGrantMembership("user-1", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUAdvancedYear, DurationDays: 365, StorageQuotaBytes: 3 << 40,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "year-1",
	}, nil); err != nil {
		t.Fatal(err)
	}
	resolved, err = svc.ResolveEffectiveStoredFileBytes("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Bytes != 3<<40 || resolved.Source != model.QuotaSourcePlan {
		t.Fatalf("plan = %#v", resolved)
	}
}

func TestStorageBonusAddsOnTopOfPlanAndClamps(t *testing.T) {
	svc := newResourceTestService(t)
	if err := svc.repo.AdminGrantMembership("user-bonus", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUAdvancedMonth, DurationDays: 30, StorageQuotaBytes: 1 << 30,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "month-bonus",
	}, nil); err != nil {
		t.Fatal(err)
	}
	row, err := svc.repo.UserMembership("user-bonus")
	if err != nil {
		t.Fatal(err)
	}
	row.StorageBonusBytes = 2 << 30
	if err := svc.repo.Save(row); err != nil {
		t.Fatal(err)
	}
	resolved, err := svc.ResolveEffectiveStoredFileBytes("user-bonus")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Bytes != 3<<30 || resolved.Source != model.QuotaSourcePlan {
		t.Fatalf("bonus on plan = %#v", resolved)
	}
	row.StorageBonusBytes = 4 << 40
	if err := svc.repo.Save(row); err != nil {
		t.Fatal(err)
	}
	resolved, err = svc.ResolveEffectiveStoredFileBytes("user-bonus")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Bytes != model.MaxMembershipStorageB {
		t.Fatalf("clamped bonus = %#v", resolved)
	}
}

func TestYearCardQuotaIsNotClampedTo999GB(t *testing.T) {
	svc := newResourceTestService(t)
	if err := svc.repo.AdminGrantMembership("user-year", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUAdvancedYear, DurationDays: 365, StorageQuotaBytes: 3 << 40,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "year",
	}, nil); err != nil {
		t.Fatal(err)
	}
	resolved, err := svc.ResolveEffectiveStoredFileBytes("user-year")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Bytes != 3<<40 {
		t.Fatalf("year quota = %d", resolved.Bytes)
	}
}

func TestPersonalDestinationSkipsPlatformQuota(t *testing.T) {
	t.Setenv("CANVAS_ALLOW_PRIVATE_UPSTREAMS", "true")
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	svc := newResourceTestService(t)
	enablePersonalStorageForTest(t, svc, "user-1")
	systemJSON, _ := json.Marshal(ossSettingValue{Enabled: true, Provider: "aliyun", Endpoint: server.URL, Bucket: "platform", AccessKeyID: "id", AccessKeySecret: "secret", AllowUserS3: true})
	if err := svc.repo.SaveSystemSetting(&model.SystemSetting{Key: ossSettingKey, ValueJSON: string(systemJSON)}); err != nil {
		t.Fatal(err)
	}
	userJSON, _ := json.Marshal(ossSettingValue{Enabled: true, Provider: "aliyun", Endpoint: "http://127.0.0.1:9", Bucket: "personal", AccessKeyID: "id", AccessKeySecret: "secret"})
	if err := svc.repo.CreateUserOSSSetting(&model.UserOSSSetting{ID: "user-oss-1", UserID: "user-1", Enabled: true, ValueJSON: string(userJSON)}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.Resource{ID: "full", UserID: "user-1", Status: model.ResourceStatusReady, Size: gigabytes(defaultRuntimePolicy().Resource.StoredFileGB)}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.reserveChunkedUploadQuota("user-1", 1<<30); err != nil {
		t.Fatalf("personal destination should skip platform quota: %v", err)
	}
}

func TestNonPermanentIgnoresEnabledPersonalOSS(t *testing.T) {
	t.Setenv("CANVAS_ALLOW_PRIVATE_UPSTREAMS", "true")
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	svc := newResourceTestService(t)
	systemJSON, _ := json.Marshal(ossSettingValue{Enabled: true, Provider: "aliyun", Endpoint: server.URL, Bucket: "platform", AccessKeyID: "id", AccessKeySecret: "secret", AllowUserS3: true})
	if err := svc.repo.SaveSystemSetting(&model.SystemSetting{Key: ossSettingKey, ValueJSON: string(systemJSON)}); err != nil {
		t.Fatal(err)
	}
	userJSON, _ := json.Marshal(ossSettingValue{Enabled: true, Provider: "aliyun", Endpoint: server.URL, Bucket: "personal", AccessKeyID: "id", AccessKeySecret: "secret"})
	if err := svc.repo.CreateUserOSSSetting(&model.UserOSSSetting{ID: "user-oss-2", UserID: "user-2", Enabled: true, ValueJSON: string(userJSON)}); err != nil {
		t.Fatal(err)
	}
	setting, _, useOSS, err := svc.activeResourceOSSSetting("user-2")
	if err != nil {
		t.Fatal(err)
	}
	if !useOSS || setting.Bucket != "platform" {
		t.Fatalf("expected platform fallback, got %#v useOSS=%v", setting, useOSS)
	}
}

func TestAssertCanPurchaseAdvancedRemainingWindow(t *testing.T) {
	svc := newResourceTestService(t)
	if err := svc.repo.AdminGrantMembership("locked", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUAdvancedYear, DurationDays: 31, StorageQuotaBytes: 3 << 40,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "lock",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.AssertCanPurchaseMembership("locked", model.MembershipSKUAdvancedMonth, 0); err == nil || !strings.Contains(err.Error(), "30 天") {
		t.Fatalf("expected 30 day lock, got %v", err)
	}
	if err := svc.repo.AdminGrantMembership("window", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUAdvancedYear, DurationDays: 20, StorageQuotaBytes: 3 << 40,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "window",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.AssertCanPurchaseMembership("window", model.MembershipSKUAdvancedMonth, 0); err != nil {
		t.Fatalf("window should allow month: %v", err)
	}
}

func TestPlatformStoredBytesExcludePersonalOSS(t *testing.T) {
	svc := newResourceTestService(t)
	if err := svc.repo.CreateUserOSSSetting(&model.UserOSSSetting{ID: "oss-user", UserID: "user-1", Enabled: true, ValueJSON: "{}"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.Resource{ID: "platform", UserID: "user-1", Status: model.ResourceStatusReady, Provider: "local", ObjectKey: "a.png", Size: 11}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.Resource{ID: "personal", UserID: "user-1", Status: model.ResourceStatusReady, Provider: "s3", ObjectKey: "b.png", Size: 99, StorageSettingID: "oss-user"}); err != nil {
		t.Fatal(err)
	}
	used, err := svc.repo.UserPlatformStoredFileBytes("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if used != 11 {
		t.Fatalf("platform bytes = %d", used)
	}
}

func TestAssertCanPurchaseSvipBlocksVipEvenInWindow(t *testing.T) {
	svc := newResourceTestService(t)
	if err := svc.repo.AdminGrantMembership("svip-user", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUSvipMonth, DurationDays: 20, StorageQuotaBytes: 80 << 30,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "svip",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.AssertCanPurchaseMembership("svip-user", model.MembershipSKUVipMonth, 0); err == nil || !strings.Contains(err.Error(), "SVIP") {
		t.Fatalf("expected svip lock, got %v", err)
	}
	if err := svc.AssertCanPurchaseMembership("svip-user", model.MembershipSKUSvipYear, 0); err != nil {
		t.Fatalf("svip renew should be allowed: %v", err)
	}
}

func TestPersonalStorageAllowedForVip(t *testing.T) {
	svc := newResourceTestService(t)
	systemJSON, _ := json.Marshal(ossSettingValue{AllowUserS3: true})
	if err := svc.repo.SaveSystemSetting(&model.SystemSetting{Key: ossSettingKey, ValueJSON: string(systemJSON)}); err != nil {
		t.Fatal(err)
	}
	allowed, err := svc.PersonalStorageAllowed("user-none", false)
	if err != nil || allowed {
		t.Fatalf("unsubscribed allowed=%v err=%v", allowed, err)
	}
	if err := svc.repo.AdminGrantMembership("user-vip", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUVipMonth, DurationDays: 30, StorageQuotaBytes: 30 << 30,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "vip-oss",
	}, nil); err != nil {
		t.Fatal(err)
	}
	allowed, err = svc.PersonalStorageAllowed("user-vip", false)
	if err != nil || !allowed {
		t.Fatalf("vip allowed=%v err=%v", allowed, err)
	}
}

func TestAccountFileStorageUsagePersonalDisplay(t *testing.T) {
	t.Setenv("CANVAS_ALLOW_PRIVATE_UPSTREAMS", "true")
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	svc := newResourceTestService(t)
	if err := svc.repo.AdminGrantMembership("user-1", model.MembershipGrantSnapshot{
		PlanSKU: model.MembershipSKUVipMonth, DurationDays: 30, StorageQuotaBytes: 30 << 30,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: "vip-display",
	}, nil); err != nil {
		t.Fatal(err)
	}
	systemJSON, _ := json.Marshal(ossSettingValue{Enabled: true, Provider: "aliyun", Endpoint: server.URL, Bucket: "platform", AccessKeyID: "id", AccessKeySecret: "secret", AllowUserS3: true})
	if err := svc.repo.SaveSystemSetting(&model.SystemSetting{Key: ossSettingKey, ValueJSON: string(systemJSON)}); err != nil {
		t.Fatal(err)
	}
	userJSON, _ := json.Marshal(ossSettingValue{Enabled: true, Provider: "aliyun", Endpoint: server.URL, Bucket: "personal", AccessKeyID: "id", AccessKeySecret: "secret"})
	if err := svc.repo.CreateUserOSSSetting(&model.UserOSSSetting{ID: "user-oss-display", UserID: "user-1", Enabled: true, ValueJSON: string(userJSON)}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.Resource{ID: "platform", UserID: "user-1", Status: model.ResourceStatusReady, Provider: "local", ObjectKey: "a.png", Size: 11}); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.Create(&model.Resource{ID: "personal", UserID: "user-1", Status: model.ResourceStatusReady, Provider: "s3", ObjectKey: "b.png", Size: 99, StorageSettingID: "user-oss-display"}); err != nil {
		t.Fatal(err)
	}
	usage, err := svc.AccountFileStorageUsage("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if usage.StorageDisplay != model.StorageDisplayPersonal || usage.UsedBytes != 99 || usage.TotalBytes != 0 || !usage.PersonalBucketEnabled {
		t.Fatalf("personal display = %#v", usage)
	}
}

func TestPublicMembershipProductsOnlyCatalogSKUs(t *testing.T) {
	svc := newResourceTestService(t)
	for _, product := range []model.MembershipProduct{
		{ID: "membership-permanent", SKU: model.MembershipSKUPermanent, Name: "永久订阅", Enabled: true, SortOrder: 10},
		{ID: "membership-advanced-month", SKU: model.MembershipSKUAdvancedMonth, Name: "旧月卡", Enabled: true, SortOrder: 20},
		{ID: "membership-vip-month", SKU: model.MembershipSKUVipMonth, Name: "VIP 月卡", Enabled: true, SortOrder: 100, Tier: model.MembershipTierVip},
		{ID: "membership-svip-month", SKU: model.MembershipSKUSvipMonth, Name: "SVIP 月卡", Enabled: true, SortOrder: 200, Tier: model.MembershipTierSvip},
	} {
		item := product
		if err := svc.repo.Create(&item); err != nil {
			t.Fatal(err)
		}
	}
	views, err := svc.PublicMembershipProducts()
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("catalog size = %d", len(views))
	}
	for _, view := range views {
		if !model.IsCatalogMembershipSKU(view.SKU) {
			t.Fatalf("legacy sku leaked: %#v", view)
		}
	}
}

func TestUpdateMembershipProductCreditsAndStorage(t *testing.T) {
	svc := newResourceTestService(t)
	admin := &model.User{ID: "admin-1", Role: model.UserRoleAdmin}
	if err := svc.repo.Create(admin); err != nil {
		t.Fatal(err)
	}
	product := &model.MembershipProduct{ID: "membership-vip-month", SKU: model.MembershipSKUVipMonth, Name: "VIP 月卡", DurationDays: 30, Tier: model.MembershipTierVip, AmountFen: 3000, CreditsMicrocredits: 100, StorageQuotaBytes: 1 << 30, Enabled: true}
	if err := svc.repo.Create(product); err != nil {
		t.Fatal(err)
	}
	credits := int64(2888 * model.CreditScale)
	storage := int64(30 << 30)
	updated, err := svc.UpdateMembershipProduct(admin, product.ID, UpdateMembershipProductRequest{
		Name: "VIP 月卡", AmountFen: 3000, Enabled: true, SortOrder: 10, CreditsMicrocredits: &credits, StorageQuotaBytes: &storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CreditsMicrocredits != credits || updated.StorageQuotaBytes != storage {
		t.Fatalf("updated = %#v", updated)
	}
}
