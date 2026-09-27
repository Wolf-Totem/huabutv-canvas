package auth

import (
	"testing"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRegistrationAgreementFollowsBrandNameWhenUnset(t *testing.T) {
	svc := New(repository.New(newAgreementTestDB(t)), brandHost{name: "画布TV"}, nil)

	title, content := svc.RegistrationAgreement()
	if title != "画布TV服务协议" {
		t.Fatalf("default agreement title = %q", title)
	}
	if content != "" {
		t.Fatalf("default agreement content = %q", content)
	}

	settings, err := svc.PublicAuthSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if settings.AgreementTitle != "画布TV服务协议" || settings.AgreementContent != "" {
		t.Fatalf("public settings agreement = %q / %q", settings.AgreementTitle, settings.AgreementContent)
	}
}

func TestUpdateRegistrationSettingStoresAgreement(t *testing.T) {
	db := newAgreementTestDB(t)
	svc := New(repository.New(db), brandHost{name: "画布TV"}, nil)
	if err := db.Create(&model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}

	saved, err := svc.UpdateRegistrationSetting(admin, RegistrationSettingRequest{
		Enabled:          true,
		AgreementTitle:   boolString("《画布TV服务条款》"),
		AgreementContent: boolString("第一条 服务说明\n\n第二条 内容合规"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Enabled || saved.AgreementTitle != "《画布TV服务条款》" {
		t.Fatalf("saved setting = %#v", saved)
	}

	reloaded, err := svc.AdminRegistrationSetting(admin)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.AgreementContent != "第一条 服务说明\n\n第二条 内容合规" {
		t.Fatalf("reloaded content = %q", reloaded.AgreementContent)
	}

	title, content := svc.RegistrationAgreement()
	if title != "画布TV服务条款" || content != "第一条 服务说明\n\n第二条 内容合规" {
		t.Fatalf("agreement = %q / %q", title, content)
	}

	settings, err := svc.PublicAuthSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if settings.AgreementTitle != "画布TV服务条款" || settings.AgreementContent != "第一条 服务说明\n\n第二条 内容合规" {
		t.Fatalf("public settings agreement = %q / %q", settings.AgreementTitle, settings.AgreementContent)
	}
}

func TestRegistrationToggleKeepsSavedAgreement(t *testing.T) {
	db := newAgreementTestDB(t)
	svc := New(repository.New(db), brandHost{name: "画布TV"}, nil)
	admin := &model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}

	if _, err := svc.UpdateRegistrationSetting(admin, RegistrationSettingRequest{Enabled: true, AgreementContent: boolString("条款正文")}); err != nil {
		t.Fatal(err)
	}
	enabled, err := svc.RegistrationEnabled()
	if err != nil || !enabled {
		t.Fatalf("RegistrationEnabled() = %v, %v", enabled, err)
	}

	switched, err := svc.UpdateRegistrationSetting(admin, RegistrationSettingRequest{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if switched.Enabled || switched.AgreementContent != "条款正文" {
		t.Fatalf("toggle wiped agreement: %#v", switched)
	}

	cleared, err := svc.UpdateRegistrationSetting(admin, RegistrationSettingRequest{Enabled: true, AgreementContent: boolString("")})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.AgreementContent != "" {
		t.Fatalf("explicit clear ignored: %#v", cleared)
	}
}

func TestRegistrationAgreementDegradesWhenSettingUnreadable(t *testing.T) {
	db := newAgreementTestDB(t)
	svc := New(repository.New(db), brandHost{name: "画布TV"}, nil)

	if err := db.Create(&model.SystemSetting{Key: registrationSettingKey, ValueJSON: "{not-json"}).Error; err != nil {
		t.Fatal(err)
	}

	title, content := svc.RegistrationAgreement()
	if title != "" || content != "" {
		t.Fatalf("broken setting should degrade to empty, got %q / %q", title, content)
	}
	if message := svc.AgreementTitleForMessage(); message != "服务协议" {
		t.Fatalf("fallback message title = %q", message)
	}
}

func boolString(value string) *string { return &value }

func boolPtr(value bool) *bool { return &value }

func newAgreementTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+kernel.NewID()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	return db
}
