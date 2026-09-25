package auth

import (
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
)

func TestRegisterHostLocksInviteAndSubdomainGate(t *testing.T) {
	svc, db := newPasswordResetTestService(t)
	if err := db.AutoMigrate(&model.Streamer{}); err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin", Username: "admin", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemSetting{Key: registrationSettingKey, ValueJSON: `{"enabled":false}`}).Error; err != nil {
		t.Fatal(err)
	}
	zhang := &model.Streamer{ID: "st-zhang", UserID: admin.ID, Slug: "zhangsan", InviteCode: "HB-ZHANGSAN-7K2M", Status: model.StreamerStatusActive, DisplayName: "张三"}
	li := &model.Streamer{ID: "st-li", UserID: "other", Slug: "lisi", InviteCode: "HB-LISI-AAAA", Status: model.StreamerStatusActive, DisplayName: "李四"}
	host := &streamerAuthHost{
		byHost: map[string]*model.Streamer{"zhangsan.huabutv.example": zhang, "lisi.huabutv.example": li},
		byCode: map[string]*model.Streamer{zhang.InviteCode: zhang, li.InviteCode: li},
		bound:  map[string]string{},
	}
	svc.host = host

	settings, err := svc.PublicAuthSettings("app.huabutv.example")
	if err != nil {
		t.Fatal(err)
	}
	if settings.RegistrationEnabled {
		t.Fatal("main domain must stay closed")
	}
	sub, err := svc.PublicAuthSettings("zhangsan.huabutv.example")
	if err != nil {
		t.Fatal(err)
	}
	if !sub.RegistrationEnabled || !sub.InviteLocked || sub.InviteDisplayName != "张三" {
		t.Fatalf("subdomain settings = %+v", sub)
	}

	var delivered string
	svc.SetMailSender(func(_ EmailSettingValue, _, _ string, body string) error {
		delivered = codeFromEmailBody(body)
		return nil
	})
	if err := svc.SendRegistrationEmailCode("fan@example.com", "", "zhangsan.huabutv.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(RegisterRequest{Username: "fan1", Email: "fan@example.com", Password: "strong-password", EmailCode: delivered, InviteCode: li.InviteCode, Host: "zhangsan.huabutv.example"}); err != nil {
		t.Fatal(err)
	}
	var user model.User
	if err := db.First(&user, "username = ?", "fan1").Error; err != nil {
		t.Fatal(err)
	}
	if host.bound[user.ID] != zhang.ID {
		t.Fatalf("bound %q, want zhang, invite from lisi must be ignored", host.bound[user.ID])
	}

	if err := svc.SendRegistrationEmailCode("nope@example.com", "", "app.huabutv.example"); err == nil || !strings.Contains(err.Error(), "未开放") {
		t.Fatalf("main domain email code should fail: %v", err)
	}
}

func TestInviteSubdomainSettingDefaultsOn(t *testing.T) {
	svc, _ := newPasswordResetTestService(t)
	enabled, err := svc.InviteSubdomainRegistrationEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("invite subdomain registration should default on")
	}
}

type streamerAuthHost struct {
	nopHost
	byHost map[string]*model.Streamer
	byCode map[string]*model.Streamer
	bound  map[string]string
}

func (h *streamerAuthHost) StreamerByHost(host string) (*model.Streamer, error) {
	if h.byHost == nil {
		return nil, nil
	}
	return h.byHost[host], nil
}

func (h *streamerAuthHost) ActiveStreamerByInvite(code string) (*model.Streamer, error) {
	if h.byCode == nil {
		return nil, nil
	}
	return h.byCode[code], nil
}

func (h *streamerAuthHost) BindUserStreamerInvite(userID, streamerID string) error {
	if h.bound == nil {
		h.bound = map[string]string{}
	}
	if _, exists := h.bound[userID]; exists {
		return nil
	}
	h.bound[userID] = streamerID
	return nil
}
