package auth

import (
	"net/mail"
	"strings"
	"testing"
)

func TestVerificationEmailCopyUsesLocaleCatalog(t *testing.T) {
	zh := verificationEmailCopy(registrationEmailPurpose, "zh", "佳速无限画布", "user@example.com")
	if zh.Subject != "佳速无限画布邮箱验证邮件" || !strings.Contains(zh.Hello, "user@example.com") {
		t.Fatalf("zh copy = %#v", zh)
	}
	en := verificationEmailCopy(registrationEmailPurpose, "en", "Jiasu Canvas", "user@example.com")
	if en.Subject != "Jiasu Canvas email verification" || en.Title != "Verify your email" {
		t.Fatalf("en copy = %#v", en)
	}
	reset := verificationEmailCopy(passwordResetEmailPurpose, "vi", "Jiasu", "user@example.com")
	if !strings.Contains(reset.Subject, "Jiasu") || reset.Title == "" {
		t.Fatalf("vi reset copy = %#v", reset)
	}
}

func TestComposeEmailMessageIncludesHTMLAlternative(t *testing.T) {
	from := mail.Address{Name: "佳速无限画布", Address: "noreply@jiasuapi.com"}
	copy := verificationEmailCopy(registrationEmailPurpose, "zh", "佳速无限画布", "user@example.com")
	htmlBody := verificationEmailHTML(copy, "佳速无限画布", "260321")
	raw := string(composeEmailMessage(from, "user@example.com", copy.Subject, verificationEmailText(copy, "260321"), htmlBody))
	if !strings.Contains(raw, "multipart/alternative") || !strings.Contains(raw, "260321") || !strings.Contains(raw, "验证您的邮箱") {
		t.Fatalf("message missing html alternative: %s", raw)
	}
	if strings.Contains(htmlBody, "<script") {
		t.Fatal("html template must not include scripts")
	}
}
