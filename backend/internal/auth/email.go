package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"infinite-canvas/backend/internal/kernel"
	"log"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const emailSettingKey = "email"
const registrationEmailPurpose = "registration"
const adminSMTPTestPurpose = "admin_smtp_test"
const registrationCodeTTL = 10 * time.Minute

var defaultRegistrationEmailDomains = []string{
	"gmail.com",
	"163.com",
	"126.com",
	"qq.com",
	"outlook.com",
	"hotmail.com",
	"icloud.com",
	"yahoo.com",
	"foxmail.com",
}

type EmailSettingRequest struct {
	Enabled                    bool     `json:"enabled"`
	Host                       string   `json:"host"`
	Port                       int      `json:"port"`
	Username                   string   `json:"username"`
	Password                   string   `json:"password"`
	Encryption                 string   `json:"encryption"`
	FromEmail                  string   `json:"fromEmail"`
	FromName                   string   `json:"fromName"`
	RegistrationAllowedDomains []string `json:"registrationAllowedDomains"`
}

type PublicEmailSetting struct {
	Enabled                    bool      `json:"enabled"`
	Host                       string    `json:"host"`
	Port                       int       `json:"port"`
	Username                   string    `json:"username"`
	Encryption                 string    `json:"encryption"`
	FromEmail                  string    `json:"fromEmail"`
	FromName                   string    `json:"fromName"`
	FromNameInherited          bool      `json:"fromNameInherited"`
	HasPassword                bool      `json:"hasPassword"`
	RegistrationAllowedDomains []string  `json:"registrationAllowedDomains"`
	UpdatedBy                  string    `json:"updatedBy"`
	CreatedAt                  time.Time `json:"createdAt"`
	UpdatedAt                  time.Time `json:"updatedAt"`
}

type EmailSettingValue struct {
	Enabled                    bool     `json:"enabled"`
	Host                       string   `json:"host"`
	Port                       int      `json:"port"`
	Username                   string   `json:"username"`
	Password                   string   `json:"password"`
	Encryption                 string   `json:"encryption"`
	FromEmail                  string   `json:"fromEmail"`
	FromName                   string   `json:"fromName"`
	RegistrationAllowedDomains []string `json:"registrationAllowedDomains"`
}

func (s *Service) AdminEmailSetting(actor *model.User) (*PublicEmailSetting, error) {
	if err := s.host.RequireAdmin(actor); err != nil {
		return nil, err
	}
	setting, value, err := s.readEmailSetting()
	if err != nil {
		return nil, err
	}
	return s.publicEmailSetting(setting, value), nil
}

func (s *Service) UpdateEmailSetting(actor *model.User, req EmailSettingRequest) (*PublicEmailSetting, error) {
	if err := s.host.RequireAdmin(actor); err != nil {
		return nil, err
	}
	currentSetting, current, err := s.readEmailSetting()
	if err != nil {
		return nil, err
	}
	allowedDomains := req.RegistrationAllowedDomains
	if allowedDomains == nil {
		allowedDomains = current.RegistrationAllowedDomains
	}
	next := normalizeEmailSetting(EmailSettingValue{
		Enabled: req.Enabled, Host: req.Host, Port: req.Port, Username: req.Username, Password: req.Password,
		Encryption: req.Encryption, FromEmail: req.FromEmail, FromName: req.FromName,
		RegistrationAllowedDomains: allowedDomains,
	})
	if next.Password == "" {
		next.Password = current.Password
	}
	if err := validateEmailSetting(next); err != nil {
		return nil, err
	}
	stored := next
	stored.Password, err = s.host.EncryptSecret(next.Password)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	setting := model.SystemSetting{Key: emailSettingKey, ValueJSON: string(encoded), UpdatedBy: actor.ID}
	if currentSetting != nil {
		setting.CreatedAt = currentSetting.CreatedAt
	}
	if err := s.repo.SaveSystemSetting(&setting); err != nil {
		return nil, err
	}
	return s.publicEmailSetting(&setting, next), nil
}

func (s *Service) EmailEnabled() (bool, error) {
	_, value, err := s.readEmailSetting()
	if err != nil {
		return false, err
	}
	return value.Enabled && value.Host != "" && value.Port > 0 && value.FromEmail != "", nil
}

func (s *Service) AdminSendTestVerificationEmail(actor *model.User, rawEmail string, localeCode string) error {
	if err := s.host.RequireAdmin(actor); err != nil {
		return err
	}
	email := NormalizeEmail(rawEmail)
	if err := ValidateEmail(email); err != nil {
		return err
	}
	_, setting, err := s.readEmailSetting()
	if err != nil {
		return err
	}
	if !setting.Enabled {
		return kernel.Forbidden("请先保存并启用邮件服务")
	}
	if err := validateEmailSetting(setting); err != nil {
		return err
	}
	s.emailCodeMu.Lock()
	defer s.emailCodeMu.Unlock()
	if latest, err := s.repo.LatestEmailVerificationCode(email, adminSMTPTestPurpose); err == nil && time.Since(latest.CreatedAt) < time.Minute {
		seconds := max(1, int((time.Until(latest.CreatedAt.Add(time.Minute))+time.Second-1)/time.Second))
		return &EmailCodeCooldownError{Seconds: seconds}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	code, err := randomNumericCode(6)
	if err != nil {
		return err
	}
	codeHash, err := s.emailVerificationCodeHash(adminSMTPTestPurpose, email, code)
	if err != nil {
		return err
	}
	now := time.Now()
	record := model.EmailVerificationCode{ID: kernel.NewID(), Email: email, CodeHash: codeHash, Purpose: adminSMTPTestPurpose, ExpiresAt: now.Add(registrationCodeTTL), CreatedAt: now}
	if err := s.repo.Create(&record); err != nil {
		return err
	}
	setting = resolveEmailSender(setting, s.host.BrandName())
	if err := s.deliverVerificationEmail(setting, email, adminSMTPTestPurpose, localeCode, code); err != nil {
		cleanupErr := s.repo.DeleteEmailVerificationCode(record.ID)
		if cleanupErr != nil {
			return errors.Join(mailSendFailed(err), fmt.Errorf("清理失效测试邮件失败：%w", cleanupErr))
		}
		return mailSendFailed(err)
	}
	return nil
}

func (s *Service) SendRegistrationEmailCode(rawEmail string, localeCode string, host string) error {
	email := NormalizeEmail(rawEmail)
	if err := ValidateEmail(email); err != nil {
		return err
	}
	count, err := s.repo.UserCount()
	if err != nil {
		return err
	}
	if count == 0 {
		return kernel.BadAuthRequest("首个管理员账号不需要邮箱验证码")
	}
	allowed, _, err := s.registrationAllowed(host, "")
	if err != nil {
		return err
	}
	if !allowed {
		return kernel.Forbidden("管理员未开放新用户注册")
	}
	if _, err := s.repo.UserByEmail(email); err == nil {
		return kernel.BadAuthRequest("邮箱已被注册")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	_, setting, err := s.readEmailSetting()
	if err != nil {
		return err
	}
	if err := validateRegistrationEmailDomain(email, setting.RegistrationAllowedDomains); err != nil {
		return err
	}
	if !setting.Enabled {
		return kernel.Forbidden("平台尚未启用注册邮件，请联系管理员")
	}
	s.emailCodeMu.Lock()
	defer s.emailCodeMu.Unlock()
	if latest, err := s.repo.LatestEmailVerificationCode(email, registrationEmailPurpose); err == nil && time.Since(latest.CreatedAt) < time.Minute {
		seconds := max(1, int((time.Until(latest.CreatedAt.Add(time.Minute))+time.Second-1)/time.Second))
		return &EmailCodeCooldownError{Seconds: seconds}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	code, err := randomNumericCode(6)
	if err != nil {
		return err
	}
	codeHash, err := s.emailVerificationCodeHash(registrationEmailPurpose, email, code)
	if err != nil {
		return err
	}
	now := time.Now()
	record := model.EmailVerificationCode{ID: kernel.NewID(), Email: email, CodeHash: codeHash, Purpose: registrationEmailPurpose, ExpiresAt: now.Add(registrationCodeTTL), CreatedAt: now}
	if err := s.repo.Create(&record); err != nil {
		return err
	}
	setting = resolveEmailSender(setting, s.host.BrandName())
	if err := s.deliverVerificationEmail(setting, email, registrationEmailPurpose, localeCode, code); err != nil {
		cleanupErr := s.repo.DeleteEmailVerificationCode(record.ID)
		if cleanupErr != nil {
			return errors.Join(mailSendFailed(err), fmt.Errorf("清理失效验证码失败：%w", cleanupErr))
		}
		return mailSendFailed(err)
	}
	if cleanupErr := s.repo.DeleteExpiredEmailVerificationCodes(now.Add(-24 * time.Hour)); cleanupErr != nil {
		log.Printf("expired registration code cleanup failed: error=%v", cleanupErr)
	}
	return nil
}

func (s *Service) VerifyRegistrationEmailCode(email string, rawCode string) (*model.EmailVerificationCode, error) {
	emailEnabled, err := s.EmailEnabled()
	if err != nil {
		return nil, err
	}
	if !emailEnabled {
		return nil, kernel.Forbidden("平台尚未启用注册邮件，请联系管理员")
	}
	code := strings.TrimSpace(rawCode)
	if len(code) != 6 {
		return nil, kernel.BadAuthRequest("请输入 6 位邮箱验证码")
	}
	record, err := s.repo.LatestEmailVerificationCode(email, registrationEmailPurpose)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, kernel.BadAuthRequest("请先获取邮箱验证码")
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(record.ExpiresAt) {
		return nil, kernel.BadAuthRequest("邮箱验证码已过期，请重新获取")
	}
	hash, err := s.emailVerificationCodeHash(registrationEmailPurpose, email, code)
	if err != nil {
		return nil, err
	}
	if !hmac.Equal([]byte(hash), []byte(record.CodeHash)) {
		return nil, kernel.BadAuthRequest("邮箱验证码不正确")
	}
	return record, nil
}

func (s *Service) emailVerificationCodeHash(purpose string, email string, code string) (string, error) {
	key, err := s.host.SettingsEncryptionKey()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(strings.TrimSpace(purpose) + ":" + NormalizeEmail(email) + ":" + strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) readEmailSetting() (*model.SystemSetting, EmailSettingValue, error) {
	setting, err := s.repo.SystemSetting(emailSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, normalizeEmailSetting(EmailSettingValue{}), nil
	}
	if err != nil {
		return nil, EmailSettingValue{}, err
	}
	value := EmailSettingValue{}
	if err := json.Unmarshal([]byte(setting.ValueJSON), &value); err != nil {
		return nil, EmailSettingValue{}, errors.New("邮件配置格式无效")
	}
	value.Password, err = s.host.DecryptSecret(value.Password)
	if err != nil {
		return nil, EmailSettingValue{}, err
	}
	return setting, normalizeEmailSetting(value), nil
}

func validateEmailSetting(value EmailSettingValue) error {
	if err := validateRegistrationEmailDomains(value.RegistrationAllowedDomains); err != nil {
		return err
	}
	if !value.Enabled {
		return nil
	}
	if value.Host == "" || value.Port < 1 || value.Port > 65535 || value.FromEmail == "" {
		return kernel.BadAuthRequest("启用邮件前请完整填写 SMTP 主机、端口和发件邮箱")
	}
	if err := ValidateEmail(value.FromEmail); err != nil {
		return kernel.BadAuthRequest("发件邮箱格式不正确")
	}
	if value.Username != "" && value.Password == "" {
		return kernel.BadAuthRequest("SMTP 用户名已填写，请同时填写密码")
	}
	if strings.ContainsAny(value.FromName, "\r\n") {
		return kernel.BadAuthRequest("发件人名称不能包含换行")
	}
	return nil
}

func normalizeEmailSetting(value EmailSettingValue) EmailSettingValue {
	value.Host = strings.TrimSpace(value.Host)
	value.Username = strings.TrimSpace(value.Username)
	value.Password = strings.TrimSpace(value.Password)
	value.FromEmail = NormalizeEmail(value.FromEmail)
	value.FromName = strings.TrimSpace(value.FromName)
	if value.RegistrationAllowedDomains == nil {
		value.RegistrationAllowedDomains = append([]string(nil), defaultRegistrationEmailDomains...)
	}
	value.RegistrationAllowedDomains = normalizeEmailDomains(value.RegistrationAllowedDomains)
	if value.Port == 0 {
		value.Port = 587
	}
	switch value.Encryption {
	case "tls", "none":
	default:
		value.Encryption = "starttls"
	}
	return value
}

func (s *Service) publicEmailSetting(setting *model.SystemSetting, value EmailSettingValue) *PublicEmailSetting {
	inherited := value.FromName == "" || value.FromName == DefaultBrandName
	value = resolveEmailSender(value, s.host.BrandName())
	result := &PublicEmailSetting{
		Enabled: value.Enabled, Host: value.Host, Port: value.Port, Username: value.Username, Encryption: value.Encryption,
		FromEmail: value.FromEmail, FromName: value.FromName, FromNameInherited: inherited, HasPassword: value.Password != "",
		RegistrationAllowedDomains: value.RegistrationAllowedDomains,
	}
	if setting != nil {
		result.UpdatedBy = setting.UpdatedBy
		result.CreatedAt = setting.CreatedAt
		result.UpdatedAt = setting.UpdatedAt
	}
	return result
}

func normalizeEmailDomains(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "@")))
		value = strings.TrimSuffix(value, ".")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func validateRegistrationEmailDomains(allowed []string) error {
	if len(allowed) > 100 {
		return kernel.BadAuthRequest("电子邮件域名白名单最多填写 100 项")
	}
	for _, domain := range allowed {
		if !validEmailDomain(domain) {
			return kernel.BadAuthRequest("电子邮件域名格式不正确：" + domain)
		}
	}
	return nil
}

func validEmailDomain(value string) bool {
	if len(value) > 253 || !strings.Contains(value, ".") || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

func (s *Service) validateRegistrationEmailDomain(email string) error {
	_, setting, err := s.readEmailSetting()
	if err != nil {
		return err
	}
	return validateRegistrationEmailDomain(email, setting.RegistrationAllowedDomains)
}

func validateRegistrationEmailDomain(email string, allowedDomains []string) error {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return kernel.BadAuthRequest("邮箱格式不正确")
	}
	domain := strings.ToLower(parts[1])
	if len(allowedDomains) == 0 {
		return nil
	}
	for _, allowed := range allowedDomains {
		if domain == allowed {
			return nil
		}
	}
	return kernel.BadAuthRequest("该邮箱域名不在管理员设置的白名单内")
}

func resolveEmailSender(value EmailSettingValue, brandName string) EmailSettingValue {
	if value.FromName == "" || value.FromName == DefaultBrandName {
		value.FromName = brandName
	}
	return value
}

func mailSendFailed(err error) error {
	log.Printf("smtp send failed: %v", err)
	message := "邮件发送失败，请稍后重试。"
	if err != nil && strings.Contains(err.Error(), "535") {
		message = "邮件发送失败：Cloudflare 拒绝了 SMTP 认证。请到后台「邮件服务」把用户名设为 api_token，并把 Email Sending API Token 重新填入 SMTP 密码后保存。"
	}
	return kernel.WrapAppError(400, message, err)
}

// implicitTLSPlainAuth is AUTH PLAIN for SMTPS (port 465).
// net/smtp.PlainAuth refuses these connections because Client.tls stays false
// when the socket is already wrapped by tls.Dial.
type implicitTLSPlainAuth struct {
	identity, username, password, host string
}

func (a implicitTLSPlainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if server == nil {
		return "", nil, errors.New("missing smtp server info")
	}
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a implicitTLSPlainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected smtp challenge")
	}
	return nil, nil
}

func smtpAuth(setting EmailSettingValue) smtp.Auth {
	if setting.Encryption == "tls" {
		return implicitTLSPlainAuth{username: setting.Username, password: setting.Password, host: setting.Host}
	}
	return smtp.PlainAuth("", setting.Username, setting.Password, setting.Host)
}

func sendSMTPMail(setting EmailSettingValue, recipient string, subject string, body string, htmlBody string) error {
	address := net.JoinHostPort(setting.Host, strconv.Itoa(setting.Port))
	tlsConfig := &tls.Config{ServerName: setting.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 12 * time.Second}
	network := "tcp4"
	var client *smtp.Client
	var err error
	if setting.Encryption == "tls" {
		connection, dialErr := tls.DialWithDialer(dialer, network, address, tlsConfig)
		if dialErr != nil {
			return dialErr
		}
		client, err = smtp.NewClient(connection, setting.Host)
	} else {
		connection, dialErr := dialer.Dial(network, address)
		if dialErr != nil {
			return dialErr
		}
		client, err = smtp.NewClient(connection, setting.Host)
		if err == nil && setting.Encryption == "starttls" {
			err = client.StartTLS(tlsConfig)
		}
	}
	if err != nil {
		return err
	}
	defer client.Close()
	if setting.Username != "" {
		if err := client.Auth(smtpAuth(setting)); err != nil {
			return err
		}
	}
	if err := client.Mail(setting.FromEmail); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return err
	}
	wc, err := client.Data()
	if err != nil {
		return err
	}
	from := mail.Address{Name: setting.FromName, Address: setting.FromEmail}
	message := composeEmailMessage(from, recipient, subject, body, htmlBody)
	if _, err := wc.Write(message); err != nil {
		_ = wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (s *Service) deliverEmail(setting EmailSettingValue, recipient string, subject string, body string) error {
	return s.deliverHTMLEmail(setting, recipient, subject, body, "")
}

func (s *Service) deliverHTMLEmail(setting EmailSettingValue, recipient string, subject string, body string, htmlBody string) error {
	if s.mailSender != nil {
		return s.mailSender(setting, recipient, subject, body)
	}
	return sendSMTPMail(setting, recipient, subject, body, htmlBody)
}

func (s *Service) deliverVerificationEmail(setting EmailSettingValue, recipient, purpose, localeCode, code string) error {
	copy := verificationEmailCopy(purpose, localeCode, setting.FromName, recipient)
	return s.deliverHTMLEmail(setting, recipient, copy.Subject, verificationEmailText(copy, code), verificationEmailHTML(copy, setting.FromName, code))
}

func randomNumericCode(length int) (string, error) {
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", length, value.Int64()), nil
}
