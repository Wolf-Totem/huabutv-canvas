package auth

import (
	"encoding/json"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
)

func TestAdminSendTestVerificationEmailUsesSavedSMTPAndSkipsWhitelist(t *testing.T) {
	svc, db := newPasswordResetTestService(t)
	admin := &model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	var delivered struct {
		to      string
		subject string
		body    string
	}
	svc.SetMailSender(func(_ EmailSettingValue, recipient, subject, body string) error {
		delivered.to, delivered.subject, delivered.body = recipient, subject, body
		return nil
	})
	if err := svc.AdminSendTestVerificationEmail(admin, "tester@qq.com", "zh"); err != nil {
		t.Fatal(err)
	}
	if delivered.to != "tester@qq.com" {
		t.Fatalf("to = %q", delivered.to)
	}
	if !strings.Contains(delivered.subject, "SMTP 测试") || !strings.Contains(delivered.body, "测试验证码") {
		t.Fatalf("unexpected test email: subject=%q body=%q", delivered.subject, delivered.body)
	}
	var stored model.EmailVerificationCode
	if err := db.Where("email = ? AND purpose = ?", "tester@qq.com", adminSMTPTestPurpose).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.VerifyRegistrationEmailCode("tester@qq.com", "000000"); err == nil {
		t.Fatal("smtp test codes must not verify registration")
	}
}

func TestAdminSendTestVerificationEmailRequiresEnabledSMTP(t *testing.T) {
	svc, db := newPasswordResetTestService(t)
	admin := &model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	disabled, err := json.Marshal(EmailSettingValue{Enabled: false, Host: "smtp.example.com", Port: 587, FromEmail: "noreply@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.SystemSetting{}).Where("key = ?", emailSettingKey).Update("value_json", string(disabled)).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminSendTestVerificationEmail(admin, "tester@qq.com", "zh"); err == nil || !strings.Contains(err.Error(), "启用") {
		t.Fatalf("disabled smtp should fail: %v", err)
	}
}
