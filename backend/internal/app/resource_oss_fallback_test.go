package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newResourceFallbackTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+newID()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.Resource{},
		&model.UserOSSSetting{},
		&model.SystemSetting{},
		&model.StorageLocation{},
		&model.UserDailyActivity{},
	); err != nil {
		t.Fatal(err)
	}
	return &Service{repo: repository.New(db), dataDir: t.TempDir()}, db
}

func TestStoreResourceFailsWhenOSSUnavailable(t *testing.T) {
	service, db := newResourceFallbackTestService(t)
	if err := db.Create(&model.SystemSetting{Key: "oss", ValueJSON: `{"enabled":true,"provider":"aliyun","endpoint":"http://127.0.0.1:1","bucket":"test-bucket","accessKeyId":"ak","accessKeySecret":"sk","region":"cn-shenzhen"}`}).Error; err != nil {
		t.Fatal(err)
	}

	_, _, err := service.storeResource(
		"user-1", "video", "intro.mp4", "video/mp4", 1024,
		1920, 1080, 0, bytes.NewReader([]byte("fake-mp4-bytes")), nil, false,
	)
	if err == nil {
		t.Fatal("storeResource() error = nil, want object storage failure")
	}
}

// TestStoreResourceLocalPathUnaffectedByOSS 未启用 OSS 时上传仍走本地存储，不受其他用户 OSS 设置影响。
func TestStoreResourceLocalPathUnaffectedByOSS(t *testing.T) {
	service, db := newResourceFallbackTestService(t)
	seedOSSEnabled(t, db, "user-other", "http://127.0.0.1:1")

	resource, created, err := service.storeResource(
		"user-2", "video", "local.mp4", "video/mp4", 512,
		1280, 720, 0, bytes.NewReader([]byte("local-bytes")), nil, false,
	)
	if err != nil {
		t.Fatalf("storeResource: %v", err)
	}
	if !created {
		t.Fatal("storeResource returned created=false, want true")
	}
	if resource.Provider != "local" {
		t.Fatalf("resource.Provider = %q, want local", resource.Provider)
	}
	payload, err := os.ReadFile(filepath.Join(service.dataDir, "resources", filepath.FromSlash(resource.ObjectKey)))
	if err != nil {
		t.Fatalf("local object not written: %v", err)
	}
	if string(payload) != "local-bytes" {
		t.Fatalf("local object content = %q, want local-bytes", payload)
	}
}

func seedOSSEnabled(t *testing.T, db *gorm.DB, userID string, endpoint string) {
	t.Helper()
	setting := model.UserOSSSetting{
		UserID:    userID,
		Enabled:   true,
		ValueJSON: `{"provider":"aliyun","endpoint":"` + endpoint + `","bucket":"test-bucket","accessKeyId":"ak-test","accessKeySecret":"sk-plaintext","region":"cn-shenzhen"}`,
	}
	if err := db.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
}
