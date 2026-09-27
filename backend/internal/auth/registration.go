package auth

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const registrationSettingKey = "registration"
const inviteSubdomainRegistrationSettingKey = "registration_invite_subdomain"

type RegistrationSettingRequest struct {
	Enabled                bool    `json:"enabled"`
	InviteSubdomainEnabled *bool   `json:"inviteSubdomainEnabled"`
	AgreementTitle         *string `json:"agreementTitle"`
	AgreementContent       *string `json:"agreementContent"`
}

type PublicRegistrationSetting struct {
	Enabled                bool      `json:"enabled"`
	InviteSubdomainEnabled bool      `json:"inviteSubdomainEnabled"`
	AgreementTitle         string    `json:"agreementTitle"`
	AgreementContent       string    `json:"agreementContent"`
	UpdatedBy              string    `json:"updatedBy"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type registrationSettingValue struct {
	Enabled          bool   `json:"enabled"`
	AgreementTitle   string `json:"agreementTitle"`
	AgreementContent string `json:"agreementContent"`
}

func (s *Service) AdminRegistrationSetting(actor *model.User) (*PublicRegistrationSetting, error) {
	if err := s.host.RequireAdmin(actor); err != nil {
		return nil, err
	}
	setting, value, err := s.readRegistrationSetting()
	if err != nil {
		return nil, err
	}
	out := publicRegistrationSetting(setting, value)
	out.InviteSubdomainEnabled, err = s.InviteSubdomainRegistrationEnabled()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) UpdateRegistrationSetting(actor *model.User, req RegistrationSettingRequest) (*PublicRegistrationSetting, error) {
	if err := s.host.RequireAdmin(actor); err != nil {
		return nil, err
	}
	current, currentValue, err := s.readRegistrationSetting()
	if err != nil {
		return nil, err
	}
	nextValue := registrationSettingValue{
		Enabled:          req.Enabled,
		AgreementTitle:   currentValue.AgreementTitle,
		AgreementContent: currentValue.AgreementContent,
	}
	if req.AgreementTitle != nil {
		nextValue.AgreementTitle = strings.TrimSpace(*req.AgreementTitle)
	}
	if req.AgreementContent != nil {
		nextValue.AgreementContent = strings.TrimSpace(*req.AgreementContent)
	}
	encoded, err := json.Marshal(nextValue)
	if err != nil {
		return nil, err
	}
	setting := model.SystemSetting{Key: registrationSettingKey, ValueJSON: string(encoded), UpdatedBy: actor.ID}
	if current != nil {
		setting.CreatedAt = current.CreatedAt
	}
	if err := s.repo.SaveSystemSetting(&setting); err != nil {
		return nil, err
	}
	inviteEnabled := true
	if req.InviteSubdomainEnabled != nil {
		inviteEnabled = *req.InviteSubdomainEnabled
		if err := s.saveInviteSubdomainRegistrationSetting(actor.ID, inviteEnabled); err != nil {
			return nil, err
		}
	} else {
		inviteEnabled, err = s.InviteSubdomainRegistrationEnabled()
		if err != nil {
			return nil, err
		}
	}
	out := publicRegistrationSetting(&setting, nextValue)
	out.InviteSubdomainEnabled = inviteEnabled
	return out, nil
}

func (s *Service) RegistrationEnabled() (bool, error) {
	_, value, err := s.readRegistrationSetting()
	return value.Enabled, err
}

func (s *Service) AgreementTitleForMessage() string {
	title, _ := s.RegistrationAgreement()
	if strings.TrimSpace(title) == "" {
		return "服务协议"
	}
	return title
}

func (s *Service) RegistrationAgreement() (string, string) {
	_, value, err := s.readRegistrationSetting()
	if err != nil {
		return "", ""
	}
	title := strings.Trim(strings.TrimSpace(value.AgreementTitle), "《》")
	if title == "" {
		title = s.defaultAgreementTitle()
	}
	return title, value.AgreementContent
}

func (s *Service) defaultAgreementTitle() string {
	brand := strings.TrimSpace(s.host.BrandName())
	if brand == "" {
		brand = DefaultBrandName
	}
	return brand + "服务协议"
}

func (s *Service) InviteSubdomainRegistrationEnabled() (bool, error) {
	setting, err := s.repo.SystemSetting(inviteSubdomainRegistrationSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	value := registrationSettingValue{Enabled: true}
	if strings.TrimSpace(setting.ValueJSON) == "" || json.Unmarshal([]byte(setting.ValueJSON), &value) != nil {
		return true, nil
	}
	return value.Enabled, nil
}

func (s *Service) saveInviteSubdomainRegistrationSetting(actorID string, enabled bool) error {
	encoded, err := json.Marshal(registrationSettingValue{Enabled: enabled})
	if err != nil {
		return err
	}
	current, err := s.repo.SystemSetting(inviteSubdomainRegistrationSettingKey)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	setting := model.SystemSetting{Key: inviteSubdomainRegistrationSettingKey, ValueJSON: string(encoded), UpdatedBy: actorID}
	if current != nil && err == nil {
		setting.CreatedAt = current.CreatedAt
	}
	return s.repo.SaveSystemSetting(&setting)
}

func (s *Service) resolveRegistrationStreamer(host, inviteCode string) (*model.Streamer, error) {
	if s == nil || s.host == nil {
		return nil, nil
	}
	streamer, err := s.host.StreamerByHost(host)
	if err != nil {
		return nil, err
	}
	if streamer != nil && streamer.Status == model.StreamerStatusActive {
		return streamer, nil
	}
	return s.host.ActiveStreamerByInvite(inviteCode)
}

func (s *Service) registrationAllowed(host, inviteCode string) (bool, *model.Streamer, error) {
	streamer, err := s.resolveRegistrationStreamer(host, inviteCode)
	if err != nil {
		return false, nil, err
	}
	mainEnabled, err := s.RegistrationEnabled()
	if err != nil {
		return false, nil, err
	}
	if mainEnabled {
		return true, streamer, nil
	}
	if streamer == nil || streamer.Status != model.StreamerStatusActive {
		return false, streamer, nil
	}
	subEnabled, err := s.InviteSubdomainRegistrationEnabled()
	if err != nil {
		return false, nil, err
	}
	if !subEnabled {
		return false, streamer, nil
	}
	hostStreamer, err := s.host.StreamerByHost(host)
	if err != nil {
		return false, nil, err
	}
	if hostStreamer == nil || hostStreamer.Status != model.StreamerStatusActive {
		return false, streamer, nil
	}
	return true, hostStreamer, nil
}

func (s *Service) readRegistrationSetting() (*model.SystemSetting, registrationSettingValue, error) {
	setting, err := s.repo.SystemSetting(registrationSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, registrationSettingValue{Enabled: registrationEnabledFromEnvironment()}, nil
	}
	if err != nil {
		return nil, registrationSettingValue{}, err
	}
	value := registrationSettingValue{}
	if strings.TrimSpace(setting.ValueJSON) == "" || json.Unmarshal([]byte(setting.ValueJSON), &value) != nil {
		return nil, registrationSettingValue{}, errors.New("用户注册配置格式无效")
	}
	return setting, value, nil
}

func publicRegistrationSetting(setting *model.SystemSetting, value registrationSettingValue) *PublicRegistrationSetting {
	result := &PublicRegistrationSetting{
		Enabled:          value.Enabled,
		AgreementTitle:   value.AgreementTitle,
		AgreementContent: value.AgreementContent,
	}
	if setting != nil {
		result.UpdatedBy = setting.UpdatedBy
		result.CreatedAt = setting.CreatedAt
		result.UpdatedAt = setting.UpdatedAt
	}
	return result
}

func registrationEnabledFromEnvironment() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("CANVAS_REGISTRATION_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}
