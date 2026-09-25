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
