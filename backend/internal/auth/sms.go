package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const smsSettingKey = "sms"
const registrationSMSPurpose = "registration"
const adminSMSTestPurpose = "admin_sms_test"

type SMSSettingRequest struct {
	Enabled         bool   `json:"enabled"`
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SignName        string `json:"signName"`
	TemplateCode    string `json:"templateCode"`
	Region          string `json:"region"`
}

type PublicSMSSetting struct {
	Enabled            bool      `json:"enabled"`
	AccessKeyID        string    `json:"accessKeyId"`
	HasAccessKeySecret bool      `json:"hasAccessKeySecret"`
	SignName           string    `json:"signName"`
	TemplateCode       string    `json:"templateCode"`
	Region             string    `json:"region"`
	UpdatedBy          string    `json:"updatedBy"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type SMSSettingValue struct {
	Enabled         bool   `json:"enabled"`
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SignName        string `json:"signName"`
	TemplateCode    string `json:"templateCode"`
	Region          string `json:"region"`
}

func (s *Service) AdminSMSSetting(actor *model.User) (*PublicSMSSetting, error) {
	if err := s.host.RequireAdmin(actor); err != nil {
		return nil, err
	}
	setting, value, err := s.readSMSSetting()
	if err != nil {
		return nil, err
	}
	return s.publicSMSSetting(setting, value), nil
}

func (s *Service) UpdateSMSSetting(actor *model.User, req SMSSettingRequest) (*PublicSMSSetting, error) {
	if err := s.host.RequireAdmin(actor); err != nil {
		return nil, err
	}
	currentSetting, current, err := s.readSMSSetting()
	if err != nil {
		return nil, err
	}
	next := normalizeSMSSetting(SMSSettingValue{
		Enabled: req.Enabled, AccessKeyID: req.AccessKeyID, AccessKeySecret: req.AccessKeySecret,
		SignName: req.SignName, TemplateCode: req.TemplateCode, Region: req.Region,
	})
	if next.AccessKeySecret == "" {
		next.AccessKeySecret = current.AccessKeySecret
	}
	if err := validateSMSSetting(next); err != nil {
		return nil, err
	}
	stored := next
	stored.AccessKeySecret, err = s.host.EncryptSecret(next.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	setting := model.SystemSetting{Key: smsSettingKey, ValueJSON: string(encoded), UpdatedBy: actor.ID}
	if currentSetting != nil {
		setting.CreatedAt = currentSetting.CreatedAt
	}
	if err := s.repo.SaveSystemSetting(&setting); err != nil {
		return nil, err
	}
	return s.publicSMSSetting(&setting, next), nil
}

func (s *Service) SMSEnabled() (bool, error) {
	_, value, err := s.readSMSSetting()
	if err != nil {
		return false, err
	}
	return smsReady(value), nil
}

func (s *Service) SendRegistrationSMSCode(rawPhone string) error {
	phone, err := requireChinaMobile(rawPhone)
	if err != nil {
		return err
	}
	count, err := s.repo.UserCount()
	if err != nil {
		return err
	}
	if count == 0 {
		return kernel.BadAuthRequest("首个管理员账号不需要短信验证码")
	}
	if _, err := s.repo.UserByPhone(phone); err == nil {
		return kernel.BadAuthRequest("手机号已被注册")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	_, setting, err := s.readSMSSetting()
	if err != nil {
		return err
	}
	if !smsReady(setting) {
		return kernel.Forbidden("平台尚未启用注册短信，请联系管理员")
	}
	return s.issueAndSendSMS(phone, registrationSMSPurpose, setting)
}

func (s *Service) AdminSendTestSMS(actor *model.User, rawPhone string) error {
	if err := s.host.RequireAdmin(actor); err != nil {
		return err
	}
	phone, err := requireChinaMobile(rawPhone)
	if err != nil {
		return err
	}
	_, setting, err := s.readSMSSetting()
	if err != nil {
		return err
	}
	if !smsReady(setting) {
		return kernel.Forbidden("请先保存并启用短信服务")
	}
	return s.issueAndSendSMS(phone, adminSMSTestPurpose, setting)
}

func (s *Service) VerifyRegistrationSMSCode(rawPhone string, rawCode string) (*model.SmsVerificationCode, error) {
	phone, err := requireChinaMobile(rawPhone)
	if err != nil {
		return nil, err
	}
	smsEnabled, err := s.SMSEnabled()
	if err != nil {
		return nil, err
	}
	if !smsEnabled {
		return nil, kernel.Forbidden("平台尚未启用注册短信，请联系管理员")
	}
	code := strings.TrimSpace(rawCode)
	if len(code) != 6 {
		return nil, kernel.BadAuthRequest("请输入 6 位短信验证码")
	}
	record, err := s.repo.LatestSmsVerificationCode(phone, registrationSMSPurpose)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, kernel.BadAuthRequest("请先获取短信验证码")
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(record.ExpiresAt) {
		return nil, kernel.BadAuthRequest("短信验证码已过期，请重新获取")
	}
	hash, err := s.smsVerificationCodeHash(registrationSMSPurpose, phone, code)
	if err != nil {
		return nil, err
	}
	if !hmac.Equal([]byte(hash), []byte(record.CodeHash)) {
		return nil, kernel.BadAuthRequest("短信验证码不正确")
	}
	return record, nil
}

func (s *Service) issueAndSendSMS(phone, purpose string, setting SMSSettingValue) error {
	s.smsCodeMu.Lock()
	defer s.smsCodeMu.Unlock()
	if latest, err := s.repo.LatestSmsVerificationCode(phone, purpose); err == nil && time.Since(latest.CreatedAt) < time.Minute {
		seconds := max(1, int((time.Until(latest.CreatedAt.Add(time.Minute))+time.Second-1)/time.Second))
		return &EmailCodeCooldownError{Seconds: seconds}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	code, err := randomNumericCode(6)
	if err != nil {
		return err
	}
	codeHash, err := s.smsVerificationCodeHash(purpose, phone, code)
	if err != nil {
		return err
	}
	now := time.Now()
	record := model.SmsVerificationCode{ID: kernel.NewID(), Phone: phone, CodeHash: codeHash, Purpose: purpose, ExpiresAt: now.Add(registrationCodeTTL), CreatedAt: now}
	if err := s.repo.Create(&record); err != nil {
		return err
	}
	if err := s.deliverSMS(setting, phone, code); err != nil {
		cleanupErr := s.repo.DeleteSmsVerificationCode(record.ID)
		if cleanupErr != nil {
			return errors.Join(smsSendFailed(err), fmt.Errorf("清理失效短信验证码失败：%w", cleanupErr))
		}
		return smsSendFailed(err)
	}
	return nil
}

func (s *Service) deliverSMS(setting SMSSettingValue, phone, code string) error {
	if s.smsSender != nil {
		return s.smsSender(setting, phone, code)
	}
	return sendAliyunSMS(setting, phone, code)
}

func (s *Service) smsVerificationCodeHash(purpose, phone, code string) (string, error) {
	key, err := s.host.SettingsEncryptionKey()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(strings.TrimSpace(purpose) + ":sms:" + kernel.NormalizeChinaMobile(phone) + ":" + strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) readSMSSetting() (*model.SystemSetting, SMSSettingValue, error) {
	setting, err := s.repo.SystemSetting(smsSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, normalizeSMSSetting(SMSSettingValue{}), nil
	}
	if err != nil {
		return nil, SMSSettingValue{}, err
	}
	var value SMSSettingValue
	if strings.TrimSpace(setting.ValueJSON) != "" {
		if err := json.Unmarshal([]byte(setting.ValueJSON), &value); err != nil {
			return nil, SMSSettingValue{}, err
		}
	}
	value.AccessKeySecret, err = s.host.DecryptSecret(value.AccessKeySecret)
	if err != nil {
		return nil, SMSSettingValue{}, err
	}
	return setting, normalizeSMSSetting(value), nil
}

func (s *Service) publicSMSSetting(setting *model.SystemSetting, value SMSSettingValue) *PublicSMSSetting {
	result := &PublicSMSSetting{
		Enabled: value.Enabled, AccessKeyID: value.AccessKeyID, HasAccessKeySecret: value.AccessKeySecret != "",
		SignName: value.SignName, TemplateCode: value.TemplateCode, Region: value.Region,
	}
	if setting != nil {
		result.UpdatedBy = setting.UpdatedBy
		result.CreatedAt = setting.CreatedAt
		result.UpdatedAt = setting.UpdatedAt
	}
	return result
}

func normalizeSMSSetting(value SMSSettingValue) SMSSettingValue {
	value.AccessKeyID = strings.TrimSpace(value.AccessKeyID)
	value.AccessKeySecret = strings.TrimSpace(value.AccessKeySecret)
	value.SignName = strings.TrimSpace(value.SignName)
	value.TemplateCode = strings.TrimSpace(value.TemplateCode)
	value.Region = strings.TrimSpace(value.Region)
	if value.Region == "" {
		value.Region = "cn-hangzhou"
	}
	return value
}

func validateSMSSetting(value SMSSettingValue) error {
	if !value.Enabled {
		return nil
	}
	if value.AccessKeyID == "" || value.AccessKeySecret == "" || value.SignName == "" || value.TemplateCode == "" {
		return kernel.BadAuthRequest("启用短信前请完整填写 AccessKey、签名和模板 CODE")
	}
	return nil
}

func smsReady(value SMSSettingValue) bool {
	return value.Enabled && value.AccessKeyID != "" && value.AccessKeySecret != "" && value.SignName != "" && value.TemplateCode != ""
}

func requireChinaMobile(raw string) (string, error) {
	phone := kernel.NormalizeChinaMobile(raw)
	if !kernel.IsChinaMobile(phone) {
		return "", kernel.BadAuthRequest("请输入中国大陆 11 位手机号")
	}
	return phone, nil
}

func smsSendFailed(err error) error {
	message := "短信发送失败，请稍后重试。"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = "短信发送失败：" + err.Error()
	}
	return kernel.WrapAppError(400, message, err)
}
