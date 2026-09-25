package app

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/gorm"
)

type CreateStreamerPayoutRequest struct {
	AmountCredits  float64 `json:"amountCredits"`
	AlipayAccount  string  `json:"alipayAccount"`
	AlipayRealName string  `json:"alipayRealName"`
}

type StreamerPayoutView struct {
	ID             string     `json:"id"`
	StreamerID     string     `json:"streamerId"`
	DisplayName    string     `json:"displayName,omitempty"`
	AmountCredits  float64    `json:"amountCredits"`
	AlipayAccount  string     `json:"alipayAccount"`
	AlipayRealName string     `json:"alipayRealName"`
	Status         string     `json:"status"`
	RejectReason   string     `json:"rejectReason,omitempty"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type StreamerPayoutPage struct {
	Items []StreamerPayoutView `json:"items"`
	Total int64                `json:"total"`
}

type AgentShareModelView struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Code              string `json:"code"`
	Capability        string `json:"capability"`
	Enabled           bool   `json:"enabled"`
	AgentShareEnabled bool   `json:"agentShareEnabled"`
	AgentShareBps     int    `json:"agentShareBps"`
}

type UpdateAgentShareRequest struct {
	Enabled  *bool `json:"enabled"`
	ShareBps *int  `json:"shareBps"`
}

type RejectStreamerPayoutRequest struct {
	Reason string `json:"reason"`
}

func (s *Service) CreateStreamerPayout(user *model.User, req CreateStreamerPayoutRequest) (*StreamerPayoutView, error) {
	streamer, err := s.RequireActiveStreamer(user)
	if err != nil {
		return nil, err
	}
	account := strings.TrimSpace(req.AlipayAccount)
	name := strings.TrimSpace(req.AlipayRealName)
	if account == "" || utf8.RuneCountInString(account) > 80 {
		return nil, BadAuthRequest("请填写有效的支付宝账号")
	}
	if name == "" || utf8.RuneCountInString(name) > 40 {
		return nil, BadAuthRequest("请填写支付宝实名")
	}
	amount := creditsToMicrocredits(req.AmountCredits)
	if amount <= 0 {
		return nil, BadAuthRequest("提现金额必须大于 0")
	}
	payout := &model.StreamerPayout{
		ID:                 kernel.NewID(),
		StreamerID:         streamer.ID,
		StreamerUserID:     streamer.UserID,
		AmountMicrocredits: amount,
		AlipayAccount:      account,
		AlipayRealName:     name,
		Status:             model.StreamerPayoutPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if err := s.repo.CreateStreamerPayout(payout); err != nil {
		return nil, mapPayoutError(err)
	}
	view := streamerPayoutView(*payout, streamer.DisplayName)
	return &view, nil
}

func (s *Service) StreamerConsolePayouts(user *model.User, page, size int) (*StreamerPayoutPage, error) {
	streamer, err := s.RequireActiveStreamer(user)
	if err != nil {
		return nil, err
	}
	return s.payoutPage(streamer.ID, streamer.DisplayName, page, size)
}

func (s *Service) AdminListStreamerPayouts(actor *model.User, page, size int) (*StreamerPayoutPage, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.payoutPage("", "", page, size)
}

func (s *Service) AdminApproveStreamerPayout(actor *model.User, id string) (*StreamerPayoutView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	row, err := s.repo.ApproveStreamerPayout(id, actor.ID)
	if err != nil {
		return nil, mapPayoutError(err)
	}
	name := s.streamerDisplayName(row.StreamerID)
	view := streamerPayoutView(*row, name)
	return &view, nil
}

func (s *Service) AdminRejectStreamerPayout(actor *model.User, id string, req RejectStreamerPayoutRequest) (*StreamerPayoutView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, BadAuthRequest("退回复核请填写原因")
	}
	if utf8.RuneCountInString(reason) > 500 {
		return nil, BadAuthRequest("原因过长")
	}
	row, err := s.repo.RejectStreamerPayout(id, actor.ID, reason)
	if err != nil {
		return nil, mapPayoutError(err)
	}
	name := s.streamerDisplayName(row.StreamerID)
	view := streamerPayoutView(*row, name)
	return &view, nil
}

func (s *Service) AdminListAgentShareModels(actor *model.User) ([]AgentShareModelView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	rows, err := s.repo.LogicalModels(true)
	if err != nil {
		return nil, err
	}
	out := make([]AgentShareModelView, 0, len(rows))
	for _, row := range rows {
		bps := row.AgentShareBps
		if bps <= 0 {
			bps = model.DefaultAgentShareBps
		}
		out = append(out, AgentShareModelView{
			ID: row.ID, Name: row.Name, Code: row.Code, Capability: row.Capability, Enabled: row.Enabled,
			AgentShareEnabled: row.AgentShareEnabled, AgentShareBps: bps,
		})
	}
	return out, nil
}

func (s *Service) AdminUpdateAgentShareModel(actor *model.User, id string, req UpdateAgentShareRequest) (*AgentShareModelView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	item, err := s.repo.LogicalModel(strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.NotFound("前台模型不存在")
		}
		return nil, err
	}
	if req.Enabled != nil {
		item.AgentShareEnabled = *req.Enabled
	}
	if req.ShareBps != nil {
		if *req.ShareBps < 100 || *req.ShareBps > 10000 {
			return nil, BadAuthRequest("模型代理定价比需在 1% 到 100% 之间")
		}
		item.AgentShareBps = *req.ShareBps
	}
	if item.AgentShareBps <= 0 {
		item.AgentShareBps = model.DefaultAgentShareBps
	}
	item.UpdatedAt = time.Now()
	if err := s.repo.SaveLogicalModel(item); err != nil {
		return nil, err
	}
	return &AgentShareModelView{
		ID: item.ID, Name: item.Name, Code: item.Code, Capability: item.Capability, Enabled: item.Enabled,
		AgentShareEnabled: item.AgentShareEnabled, AgentShareBps: item.AgentShareBps,
	}, nil
}

func (s *Service) payoutPage(streamerID, displayName string, page, size int) (*StreamerPayoutPage, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	total, err := s.repo.CountStreamerPayouts(streamerID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListStreamerPayouts(streamerID, size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	items := make([]StreamerPayoutView, 0, len(rows))
	names := map[string]string{}
	for _, row := range rows {
		name := displayName
		if name == "" {
			if cached, ok := names[row.StreamerID]; ok {
				name = cached
			} else {
				name = s.streamerDisplayName(row.StreamerID)
				names[row.StreamerID] = name
			}
		}
		items = append(items, streamerPayoutView(row, name))
	}
	return &StreamerPayoutPage{Items: items, Total: total}, nil
}

func (s *Service) streamerDisplayName(id string) string {
	row, err := s.repo.Streamer(id)
	if err != nil {
		return ""
	}
	return row.DisplayName
}

func streamerPayoutView(row model.StreamerPayout, displayName string) StreamerPayoutView {
	return StreamerPayoutView{
		ID: row.ID, StreamerID: row.StreamerID, DisplayName: displayName,
		AmountCredits: microcreditsToCredits(row.AmountMicrocredits),
		AlipayAccount: row.AlipayAccount, AlipayRealName: row.AlipayRealName,
		Status: row.Status, RejectReason: row.RejectReason, ReviewedAt: row.ReviewedAt, CreatedAt: row.CreatedAt,
	}
}

func mapPayoutError(err error) error {
	if errors.Is(err, repository.ErrPendingPayout) || errors.Is(err, repository.ErrPayoutAmount) || errors.Is(err, repository.ErrPayoutNotPending) {
		return BadAuthRequest(err.Error())
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return kernel.NotFound("提现单不存在")
	}
	return err
}

func creditsToMicrocredits(value float64) int64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return int64(math.Round(value * float64(CreditScale)))
}
