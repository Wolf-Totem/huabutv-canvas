package auth

import (
	"encoding/json"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
)

func TestSMSRegistrationAndPhoneLogin(t *testing.T) {
	svc, db := newPasswordResetTestService(t)
	admin := &model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemSetting{Key: registrationSettingKey, ValueJSON: `{"enabled":true}`}).Error; err != nil {
		t.Fatal(err)
	}
	smsJSON, err := json.Marshal(SMSSettingValue{Enabled: true, AccessKeyID: "id", AccessKeySecret: "secret", SignName: "画布TV", TemplateCode: "SMS_TEST"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemSetting{Key: smsSettingKey, ValueJSON: string(smsJSON)}).Error; err != nil {
		t.Fatal(err)
	}
	var delivered string
	svc.SetSMSSender(func(_ SMSSettingValue, phone, code string) error {
		if phone != "13800138000" {
			t.Fatalf("phone = %s", phone)
		}
		delivered = code
		return nil
	})
	if err := svc.SendRegistrationSMSCode("13800138000"); err != nil {
		t.Fatal(err)
	}
	if delivered == "" {
		t.Fatal("code not sent")
	}
	result, err := svc.Register(RegisterRequest{Username: "phoneuser", Password: "strong-password", Phone: "13800138000", SmsCode: delivered, Channel: "sms"})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Phone != "13800138000" || result.User.Email != "" {
		t.Fatalf("user contact = %+v", result.User)
	}
	login, err := svc.Login(LoginRequest{Username: "13800138000", Password: "strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	if login.User.ID != result.User.ID {
		t.Fatal("phone login missed the account")
	}
	if err := svc.SendRegistrationSMSCode("13800138000"); err == nil || !strings.Contains(err.Error(), "已被注册") {
		t.Fatalf("duplicate phone: %v", err)
	}
}

func TestAdminSMSTestCodeCannotRegister(t *testing.T) {
	svc, db := newPasswordResetTestService(t)
	admin := &model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemSetting{Key: registrationSettingKey, ValueJSON: `{"enabled":true}`}).Error; err != nil {
		t.Fatal(err)
	}
	smsJSON, err := json.Marshal(SMSSettingValue{Enabled: true, AccessKeyID: "id", AccessKeySecret: "secret", SignName: "画布TV", TemplateCode: "SMS_TEST"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemSetting{Key: smsSettingKey, ValueJSON: string(smsJSON)}).Error; err != nil {
		t.Fatal(err)
	}
	var delivered string
	svc.SetSMSSender(func(_ SMSSettingValue, _, code string) error {
		delivered = code
		return nil
	})
	if err := svc.AdminSendTestSMS(admin, "13900139000"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(RegisterRequest{Username: "tester", Password: "strong-password", Phone: "13900139000", SmsCode: delivered, Channel: "sms"}); err == nil {
		t.Fatal("admin test code must not register")
	}
}

func TestNormalizeChinaMobile(t *testing.T) {
	if got := kernel.NormalizeChinaMobile("+86 138-0013-8000"); got != "13800138000" {
		t.Fatalf("got %s", got)
	}
	if !kernel.IsChinaMobile("13800138000") || kernel.IsChinaMobile("12345") {
		t.Fatal("mobile validation")
	}
}
