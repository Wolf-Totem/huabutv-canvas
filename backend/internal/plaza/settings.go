package plaza

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

func (s *Service) Settings() (Settings, error) {
	out := DefaultSettings()
	if s == nil || s.repo == nil {
		return out, nil
	}
	setting, err := s.repo.SystemSetting(model.PlazaSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if json.Unmarshal([]byte(setting.ValueJSON), &out) != nil {
		return DefaultSettings(), nil
	}
	if out.ApplyDailyLimit <= 0 {
		out.ApplyDailyLimit = 5
	}
	return out, nil
}

func (s *Service) UpdateSettings(actor *model.User, req Settings) (Settings, error) {
	if actor == nil {
		return Settings{}, kernel.Unauthorized("请先登录")
	}
	if req.ApplyDailyLimit <= 0 {
		req.ApplyDailyLimit = 5
	}
	if req.ApplyDailyLimit > 50 {
		req.ApplyDailyLimit = 50
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return Settings{}, err
	}
	now := time.Now()
	setting, err := s.repo.SystemSetting(model.PlazaSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = &model.SystemSetting{Key: model.PlazaSettingKey, CreatedAt: now}
	} else if err != nil {
		return Settings{}, err
	}
	setting.ValueJSON = string(encoded)
	setting.UpdatedBy = actor.ID
	setting.UpdatedAt = now
	if err := s.repo.SaveSystemSetting(setting); err != nil {
		return Settings{}, err
	}
	if err := s.host.RecordAdminAudit(actor, "plaza.settings.update", "system_setting", model.PlazaSettingKey, "更新作品广场开关", req); err != nil {
		return Settings{}, err
	}
	return req, nil
}

func (s *Service) requireEnabled() error {
	settings, err := s.Settings()
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return kernel.NotFound("作品广场未开放")
	}
	return nil
}

func (s *Service) requireApplyEnabled() error {
	settings, err := s.Settings()
	if err != nil {
		return err
	}
	if !settings.ApplyEnabled {
		return kernel.Forbidden("暂未开放申请上架")
	}
	return nil
}

func (s *Service) requireCopyEnabled() error {
	settings, err := s.Settings()
	if err != nil {
		return err
	}
	if !settings.CopyEnabled {
		return kernel.Forbidden("暂未开放复制项目")
	}
	return nil
}

func (s *Service) publicWatchAllowed(user *model.User) (bool, error) {
	settings, err := s.Settings()
	if err != nil {
		return false, err
	}
	if settings.PublicWatch {
		return true, nil
	}
	return user != nil, nil
}

func decodeStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var values []string
	if json.Unmarshal([]byte(raw), &values) != nil {
		return []string{}
	}
	return kernel.UniqueNonEmpty(values)
}

func encodeStringList(values []string) string {
	data, err := json.Marshal(kernel.UniqueNonEmpty(values))
	if err != nil {
		return "[]"
	}
	return string(data)
}
