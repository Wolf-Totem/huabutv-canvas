package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"mime"
	"net/mail"
	"strings"
	"time"

	"infinite-canvas/backend/internal/locale"
)

type emailCopy struct {
	Subject string
	Title   string
	Hello   string
	Lead    string
	Expiry  string
	Ignore  string
	Footer  string
}

func verificationEmailCopy(kind, localeCode, brand, recipient string) emailCopy {
	ns := loadEmailNamespace(localeCode)
	prefix := "register"
	if kind == passwordResetEmailPurpose {
		prefix = "reset"
	}
	if kind == adminSMTPTestPurpose {
		return emailCopy{
			Subject: brand + " SMTP 测试邮件",
			Title:   "SMTP 投递测试",
			Hello:   recipient + "，您好：",
			Lead:    "这是后台发出的测试验证码邮件，用来确认 SMTP 配置可用。",
			Expiry:  "此验证码仅用于测试，不能用于注册或找回密码。",
			Ignore:  "如果不是管理员操作，请忽略此邮件。",
			Footer:  "此邮件由" + brand + "发送。",
		}
	}
	replace := func(value string) string {
		value = strings.ReplaceAll(value, "{{brand}}", brand)
		return strings.ReplaceAll(value, "{{email}}", recipient)
	}
	return emailCopy{
		Subject: replace(emailCatalogString(ns, prefix+".subject", brand+"验证码")),
		Title:   replace(emailCatalogString(ns, prefix+".title", "验证您的邮箱")),
		Hello:   replace(emailCatalogString(ns, prefix+".hello", recipient+"，您好：")),
		Lead:    replace(emailCatalogString(ns, prefix+".lead", "请输入以下验证码以继续")),
		Expiry:  replace(emailCatalogString(ns, prefix+".expiry", "验证码将在 10 分钟后失效。")),
		Ignore:  replace(emailCatalogString(ns, prefix+".ignore", "如果不是您本人操作，请忽略此邮件。")),
		Footer:  replace(emailCatalogString(ns, prefix+".footer", "此邮件由"+brand+"发送，请勿直接回复。")),
	}
}

func loadEmailNamespace(localeCode string) map[string]string {
	data, err := locale.LoadNamespace(locale.Normalize(localeCode), "email")
	if err != nil {
		return map[string]string{}
	}
	var payload map[string]string
	if unmarshalErr := json.Unmarshal(data, &payload); unmarshalErr != nil {
		return map[string]string{}
	}
	return payload
}

func emailCatalogString(ns map[string]string, key, fallback string) string {
	if value := strings.TrimSpace(ns[key]); value != "" {
		return value
	}
	return fallback
}

func verificationEmailText(copy emailCopy, code string) string {
	return strings.Join([]string{
		copy.Title,
		"",
		copy.Hello,
		copy.Lead,
		"",
		code,
		"",
		copy.Expiry,
		copy.Ignore,
		copy.Footer,
	}, "\n")
}

func verificationEmailHTML(copy emailCopy, brand, code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="und">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="margin:0;padding:32px 16px;background:#f3f4f6;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#111827;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:560px;margin:0 auto;background:#ffffff;border-radius:16px;">
    <tr><td style="padding:40px 40px 32px;">
      <p style="margin:0 0 20px;font-size:14px;color:#6b7280;">%s</p>
      <h1 style="margin:0 0 20px;font-size:28px;line-height:1.25;font-weight:700;color:#111827;">%s</h1>
      <p style="margin:0 0 8px;font-size:15px;color:#374151;">%s</p>
      <p style="margin:0 0 24px;font-size:15px;color:#4b5563;">%s</p>
      <div style="margin:0 0 24px;padding:28px 16px;background:#f3f4f6;border-radius:12px;text-align:center;font-size:36px;letter-spacing:12px;font-weight:700;color:#111827;">%s</div>
      <p style="margin:0 0 28px;font-size:14px;color:#6b7280;">%s</p>
      <hr style="border:0;border-top:1px solid #e5e7eb;margin:0 0 20px;">
      <p style="margin:0 0 8px;font-size:13px;color:#9ca3af;">%s</p>
      <p style="margin:0;font-size:13px;color:#9ca3af;">%s</p>
    </td></tr>
  </table>
</body>
</html>`,
		html.EscapeString(brand),
		html.EscapeString(copy.Title),
		html.EscapeString(copy.Hello),
		html.EscapeString(copy.Lead),
		html.EscapeString(code),
		html.EscapeString(copy.Expiry),
		html.EscapeString(copy.Ignore),
		html.EscapeString(copy.Footer),
	)
}

func composeEmailMessage(from mail.Address, recipient, subject, textBody, htmlBody string) []byte {
	var buf bytes.Buffer
	boundary := "canvas-email-alt"
	buf.WriteString("From: " + from.String() + "\r\n")
	buf.WriteString("To: " + recipient + "\r\n")
	buf.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n")
	buf.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	if strings.TrimSpace(htmlBody) == "" {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(textBody)
		return buf.Bytes()
	}
	buf.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	buf.WriteString(textBody + "\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	buf.WriteString(htmlBody + "\r\n")
	buf.WriteString("--" + boundary + "--\r\n")
	return buf.Bytes()
}
