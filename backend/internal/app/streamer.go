package app

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const (
	streamerInvitePrefix       = "HB-"
	streamerInviteRandLen      = 4
	streamerInviteRandAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	defaultStreamerSiteTitle   = model.DefaultStreamerSiteTitle
)

var streamerSlugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$`)
var streamerInviteCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{2,30}[A-Z0-9]$`)

var reservedStreamerSlugs = map[string]struct{}{
	"www": {}, "app": {}, "api": {}, "agent": {}, "admin": {}, "static": {}, "cdn": {}, "mail": {}, "canvas": {},
}

const defaultPublicParentDomain = "j11.net"

// ReservedStreamerSlugs 子域最左 label 不可用作主播 slug。
func ReservedStreamerSlugs() []string {
	return []string{"www", "app", "api", "agent", "admin", "static", "cdn", "mail", "canvas"}
}

// PublicParentDomain 主播专属页和代理后台共用的父域，默认 j11.net。
func PublicParentDomain() string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("CANVAS_PUBLIC_PARENT_DOMAIN")))
	value = strings.TrimPrefix(value, ".")
	if value == "" {
		return defaultPublicParentDomain
	}
	return value
}

func PublicAgentHost() string {
	if host := strings.ToLower(strings.TrimSpace(os.Getenv("CANVAS_PUBLIC_AGENT_HOST"))); host != "" {
		return host
	}
	return "agent." + PublicParentDomain()
}

func PublicCanvasHost() string {
	if host := strings.ToLower(strings.TrimSpace(os.Getenv("CANVAS_PUBLIC_CANVAS_HOST"))); host != "" {
		return host
	}
	return "canvas." + PublicParentDomain()
}

func StreamerLandingHost(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return ""
	}
	return slug + "." + PublicParentDomain()
}

func publicHTTPS(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	return "https://" + host
}

type CreateStreamerRequest struct {
	UserID                string `json:"userId"`
	Slug                  string `json:"slug"`
	DisplayName           string `json:"displayName"`
	InviteCode            string `json:"inviteCode"`
	SerialNo              *int   `json:"serialNo"`
	Note                  string `json:"note"`
	RebateRateBps         *int   `json:"rebateRateBps"`
	TextRebateBps         *int   `json:"textRebateBps"`
	ImageRebateBps        *int   `json:"imageRebateBps"`
	VideoRebateBps        *int   `json:"videoRebateBps"`
	CustomChannelsEnabled *bool  `json:"customChannelsEnabled"`
}

type UpdateStreamerRequest struct {
	DisplayName           *string `json:"displayName"`
	Slug                  *string `json:"slug"`
	InviteCode            *string `json:"inviteCode"`
	SerialNo              *int    `json:"serialNo"`
	Note                  *string `json:"note"`
	RebateRateBps         *int    `json:"rebateRateBps"`
	TextRebateBps         *int    `json:"textRebateBps"`
	ImageRebateBps        *int    `json:"imageRebateBps"`
	VideoRebateBps        *int    `json:"videoRebateBps"`
	CustomChannelsEnabled *bool   `json:"customChannelsEnabled"`
}

type UpdateStreamerSkinRequest struct {
	LogoURL           string          `json:"logoUrl"`
	Title             string          `json:"title"`
	Tagline           string          `json:"tagline"`
	Theme             json.RawMessage `json:"theme"`
	HomeTemplateID    string          `json:"homeTemplateId"`
	Home              json.RawMessage `json:"home"`
	CustomHomeEnabled *bool           `json:"customHomeEnabled"`
	HeroVideoURL      *string         `json:"heroVideoUrl"`
	HeroPosterURL     *string         `json:"heroPosterUrl"`
	ClearHeroVideo    bool            `json:"clearHeroVideo"`
	ClearHeroPoster   bool            `json:"clearHeroPoster"`
}

type StreamerAdminView struct {
	ID                        string    `json:"id"`
	UserID                    string    `json:"userId"`
	Slug                      string    `json:"slug"`
	InviteCode                string    `json:"inviteCode"`
	Status                    string    `json:"status"`
	DisplayName               string    `json:"displayName"`
	CustomHomeEnabled         bool      `json:"customHomeEnabled"`
	SerialNo                  int       `json:"serialNo"`
	Note                      string    `json:"note"`
	RebateRateBps             int       `json:"rebateRateBps"`
	TextRebateBps             int       `json:"textRebateBps"`
	ImageRebateBps            int       `json:"imageRebateBps"`
	VideoRebateBps            int       `json:"videoRebateBps"`
	CustomChannelsEnabled     bool      `json:"customChannelsEnabled"`
	RebateTotalCredits        float64   `json:"rebateTotalCredits"`
	RebatePendingCredits      float64   `json:"rebatePendingCredits"`
	RebateWithdrawnCredits    float64   `json:"rebateWithdrawnCredits"`
	RebateWithdrawableCredits float64   `json:"rebateWithdrawableCredits"`
	AlipayAccount             string    `json:"alipayAccount"`
	AlipayRealName            string    `json:"alipayRealName"`
	CreatedBy                 string    `json:"createdBy"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

type PublicSiteSkin struct {
	Slug                string          `json:"slug"`
	DisplayName         string          `json:"displayName"`
	Title               string          `json:"title"`
	LogoURL             string          `json:"logoUrl"`
	Tagline             string          `json:"tagline"`
	Theme               map[string]any  `json:"theme"`
	InviteLocked        bool            `json:"inviteLocked"`
	RegistrationEnabled bool            `json:"registrationEnabled"`
	CustomHomeEnabled   bool            `json:"customHomeEnabled"`
	Home                json.RawMessage `json:"home"`
	HeroVideoURL        string          `json:"heroVideoUrl,omitempty"`
	HeroPosterURL       string          `json:"heroPosterUrl,omitempty"`
	StreamerActive      bool            `json:"streamerActive"`
	ParentDomain        string          `json:"parentDomain"`
	AgentHost           string          `json:"agentHost"`
	CanvasHost          string          `json:"canvasHost"`
	LandingHost         string          `json:"landingHost,omitempty"`
}

type StreamerConsoleMe struct {
	Slug                  string `json:"slug"`
	InviteCode            string `json:"inviteCode"`
	Host                  string `json:"host"`
	LandingURL            string `json:"landingUrl"`
	AgentURL              string `json:"agentUrl"`
	CanvasURL             string `json:"canvasUrl"`
	Status                string `json:"status"`
	SerialNo              int    `json:"serialNo"`
	DisplayName           string `json:"displayName"`
	RebateRateBps         int    `json:"rebateRateBps"`
	TextRebateBps         int    `json:"textRebateBps"`
	ImageRebateBps        int    `json:"imageRebateBps"`
	VideoRebateBps        int    `json:"videoRebateBps"`
	CustomChannelsEnabled bool   `json:"customChannelsEnabled"`
	AlipayAccount         string `json:"alipayAccount"`
	AlipayRealName        string `json:"alipayRealName"`
}

type StreamerConsoleSummary struct {
	ReferredUserCount         int64   `json:"referredUserCount"`
	ConsumedCredits           float64 `json:"consumedCredits"`
	RemainingCreditsSum       float64 `json:"remainingCreditsSum"`
	RebateRateBps             int     `json:"rebateRateBps"`
	TextRebateBps             int     `json:"textRebateBps"`
	ImageRebateBps            int     `json:"imageRebateBps"`
	VideoRebateBps            int     `json:"videoRebateBps"`
	ExpectedRebateCredits     float64 `json:"expectedRebateCredits"`
	RebateCredits             float64 `json:"rebateCredits"`
	RebateTotalCredits        float64 `json:"rebateTotalCredits"`
	RebatePendingCredits      float64 `json:"rebatePendingCredits"`
	RebateWithdrawnCredits    float64 `json:"rebateWithdrawnCredits"`
	RebateWithdrawableCredits float64 `json:"rebateWithdrawableCredits"`
	RebateRequestedCredits    float64 `json:"rebateRequestedCredits"`
	Period                    string  `json:"period"`
}

type StreamerConsoleUser struct {
	UserIDMasked      string     `json:"userIdMasked"`
	DisplayNameMasked string     `json:"displayNameMasked"`
	RegisteredAt      time.Time  `json:"registeredAt"`
	ConsumedCredits   float64    `json:"consumedCredits"`
	RebateCredits     float64    `json:"rebateCredits"`
	RemainingCredits  float64    `json:"remainingCredits"`
	LastActiveAt      *time.Time `json:"lastActiveAt"`
}

type StreamerConsoleUserPage struct {
	Items []StreamerConsoleUser `json:"items"`
	Total int64                 `json:"total"`
}

type StreamerConsoleRebate struct {
	ID               string    `json:"id"`
	SourceUserMasked string    `json:"sourceUserMasked"`
	Capability       string    `json:"capability"`
	ConsumedCredits  float64   `json:"consumedCredits"`
	ModelShareBps    int       `json:"modelShareBps"`
	AgentRebateBps   int       `json:"agentRebateBps"`
	RebateCredits    float64   `json:"rebateCredits"`
	CreatedAt        time.Time `json:"createdAt"`
}

type StreamerConsoleRebatePage struct {
	Items []StreamerConsoleRebate `json:"items"`
	Total int64                   `json:"total"`
}

func StreamerSlugFromHost(host string) string {
	value := strings.ToLower(strings.TrimSpace(host))
	if value == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(value); err == nil {
		value = strings.ToLower(h)
	} else {
		value = strings.TrimSuffix(value, ".")
	}
	if strings.HasPrefix(value, "[") && strings.Contains(value, "]") {
		return ""
	}
	label, _, _ := strings.Cut(value, ".")
	return strings.TrimSpace(label)
}

func ValidateStreamerSlug(slug string) error {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" || !streamerSlugPattern.MatchString(slug) {
		return BadAuthRequest("子域 slug 格式无效")
	}
	if _, reserved := reservedStreamerSlugs[slug]; reserved {
		return BadAuthRequest("该子域为系统保留字，不能用作主播域名")
	}
	return nil
}

func IsReservedStreamerSlug(slug string) bool {
	_, ok := reservedStreamerSlugs[strings.ToLower(strings.TrimSpace(slug))]
	return ok
}

// ResolveStreamerByHost 取 Host 最左 label。保留字或未命中返回 (nil, nil)。
func (s *Service) ResolveStreamerByHost(host string) (*model.Streamer, error) {
	slug := StreamerSlugFromHost(host)
	if slug == "" || IsReservedStreamerSlug(slug) || streamerSlugPattern.MatchString(slug) == false {
		return nil, nil
	}
	if s == nil || s.repo == nil {
		return nil, nil
	}
	row, err := s.repo.StreamerBySlug(slug)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) ActiveStreamerByInvite(code string) (*model.Streamer, error) {
	code = strings.TrimSpace(code)
	if code == "" || s == nil || s.repo == nil {
		return nil, nil
	}
	row, err := s.repo.StreamerByInviteCode(code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Status != model.StreamerStatusActive {
		return nil, nil
	}
	return row, nil
}

func (s *Service) BindUserStreamerInvite(userID, streamerID string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	userID = strings.TrimSpace(userID)
	streamerID = strings.TrimSpace(streamerID)
	if userID == "" || streamerID == "" {
		return nil
	}
	streamer, err := s.repo.Streamer(streamerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if streamer.UserID == userID {
		return nil
	}
	_, err = s.repo.BindUserStreamerInviteOnce(userID, streamer.ID, time.Now())
	return err
}

func (s *Service) AdminListStreamers(actor *model.User) ([]StreamerAdminView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListStreamers()
	if err != nil {
		return nil, err
	}
	out := make([]StreamerAdminView, 0, len(rows))
	for _, row := range rows {
		out = append(out, streamerAdminView(row))
	}
	return out, nil
}

func (s *Service) AdminCreateStreamer(actor *model.User, req CreateStreamerRequest) (*StreamerAdminView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	userID := strings.TrimSpace(req.UserID)
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if userID == "" {
		return nil, BadAuthRequest("请选择要设为主播的用户")
	}
	if err := ValidateStreamerSlug(slug); err != nil {
		return nil, err
	}
	user, err := s.repo.User(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, BadAuthRequest("用户不存在")
		}
		return nil, err
	}
	if _, err := s.repo.StreamerByUserID(user.ID); err == nil {
		return nil, BadAuthRequest("该用户已经是主播")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if _, err := s.repo.StreamerBySlug(slug); err == nil {
		return nil, BadAuthRequest("子域 slug 已被占用")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	code, err := s.resolveStreamerInviteCode(req.InviteCode, slug, "")
	if err != nil {
		return nil, err
	}
	serial := 0
	if req.SerialNo != nil {
		serial = *req.SerialNo
	}
	if serial <= 0 {
		serial, err = s.repo.NextStreamerSerial()
		if err != nil {
			return nil, err
		}
	} else {
		taken, serialErr := s.repo.SerialTaken(serial, "")
		if serialErr != nil {
			return nil, serialErr
		}
		if taken {
			return nil, BadAuthRequest("代理序号已被占用")
		}
	}
	now := time.Now()
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = user.DisplayName
	}
	baseRate := normalizeRebateRateBps(req.RebateRateBps)
	customChannels := true
	if req.CustomChannelsEnabled != nil {
		customChannels = *req.CustomChannelsEnabled
	}
	streamer := &model.Streamer{
		ID:                    kernel.NewID(),
		UserID:                user.ID,
		Slug:                  slug,
		InviteCode:            code,
		Status:                model.StreamerStatusActive,
		DisplayName:           displayName,
		CustomHomeEnabled:     false,
		SerialNo:              serial,
		Note:                  strings.TrimSpace(req.Note),
		RebateRateBps:         baseRate,
		TextRebateBps:         normalizeRebateRateBps(firstInt(req.TextRebateBps, &baseRate)),
		ImageRebateBps:        normalizeRebateRateBps(firstInt(req.ImageRebateBps, &baseRate)),
		VideoRebateBps:        normalizeRebateRateBps(firstInt(req.VideoRebateBps, &baseRate)),
		CustomChannelsEnabled: customChannels,
		CreatedBy:             actor.ID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	skin := &model.SiteSkin{StreamerID: streamer.ID, UpdatedAt: now}
	if err := s.repo.CreateStreamerWithSkin(streamer, skin); err != nil {
		return nil, err
	}
	view := streamerAdminView(*streamer)
	return &view, nil
}

func (s *Service) AdminUpdateStreamer(actor *model.User, id string, req UpdateStreamerRequest) (*StreamerAdminView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	row, err := s.repo.Streamer(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.NotFound("主播不存在")
		}
		return nil, err
	}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" {
			return nil, BadAuthRequest("展示名不能为空")
		}
		if utf8.RuneCountInString(name) > 80 {
			return nil, BadAuthRequest("展示名过长")
		}
		row.DisplayName = name
	}
	if req.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*req.Slug))
		if err := ValidateStreamerSlug(slug); err != nil {
			return nil, err
		}
		taken, slugErr := s.repo.SlugTaken(slug, row.ID)
		if slugErr != nil {
			return nil, slugErr
		}
		if taken {
			return nil, BadAuthRequest("子域 slug 已被占用")
		}
		row.Slug = slug
	}
	if req.InviteCode != nil {
		code, codeErr := s.resolveStreamerInviteCode(*req.InviteCode, row.Slug, row.ID)
		if codeErr != nil {
			return nil, codeErr
		}
		row.InviteCode = code
	}
	if req.SerialNo != nil {
		if *req.SerialNo <= 0 {
			return nil, BadAuthRequest("代理序号必须为正整数")
		}
		taken, serialErr := s.repo.SerialTaken(*req.SerialNo, row.ID)
		if serialErr != nil {
			return nil, serialErr
		}
		if taken {
			return nil, BadAuthRequest("代理序号已被占用")
		}
		row.SerialNo = *req.SerialNo
	}
	if req.Note != nil {
		note := strings.TrimSpace(*req.Note)
		if utf8.RuneCountInString(note) > 500 {
			return nil, BadAuthRequest("备注过长")
		}
		row.Note = note
	}
	if req.RebateRateBps != nil {
		if *req.RebateRateBps < 0 || *req.RebateRateBps > 10000 {
			return nil, BadAuthRequest("返利比例需在 0% 到 100% 之间")
		}
		row.RebateRateBps = *req.RebateRateBps
		row.TextRebateBps, row.ImageRebateBps, row.VideoRebateBps = *req.RebateRateBps, *req.RebateRateBps, *req.RebateRateBps
	}
	if req.TextRebateBps != nil {
		row.TextRebateBps = normalizeRebateRateBps(req.TextRebateBps)
		row.RebateRateBps = row.TextRebateBps
	}
	if req.ImageRebateBps != nil {
		row.ImageRebateBps = normalizeRebateRateBps(req.ImageRebateBps)
	}
	if req.VideoRebateBps != nil {
		row.VideoRebateBps = normalizeRebateRateBps(req.VideoRebateBps)
	}
	if req.CustomChannelsEnabled != nil {
		row.CustomChannelsEnabled = *req.CustomChannelsEnabled
	}
	row.UpdatedAt = time.Now()
	if err := s.repo.SaveStreamer(row); err != nil {
		return nil, err
	}
	view := streamerAdminView(*row)
	return &view, nil
}

func (s *Service) AdminSetStreamerStatus(actor *model.User, id, status string) (*StreamerAdminView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	if status != model.StreamerStatusActive && status != model.StreamerStatusDisabled {
		return nil, BadAuthRequest("无效的主播状态")
	}
	row, err := s.repo.Streamer(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.NotFound("主播不存在")
		}
		return nil, err
	}
	row.Status = status
	row.UpdatedAt = time.Now()
	if err := s.repo.SaveStreamer(row); err != nil {
		return nil, err
	}
	view := streamerAdminView(*row)
	return &view, nil
}

func (s *Service) AdminRotateStreamerInviteCode(actor *model.User, id string) (*StreamerAdminView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	row, err := s.repo.Streamer(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.NotFound("主播不存在")
		}
		return nil, err
	}
	code, err := s.allocateStreamerInviteCode(row.Slug)
	if err != nil {
		return nil, err
	}
	row.InviteCode = code
	row.UpdatedAt = time.Now()
	if err := s.repo.SaveStreamer(row); err != nil {
		return nil, err
	}
	view := streamerAdminView(*row)
	return &view, nil
}

func (s *Service) AdminGetStreamerSkin(actor *model.User, id string) (*model.SiteSkin, *model.Streamer, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, nil, err
	}
	streamer, err := s.repo.Streamer(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, kernel.NotFound("主播不存在")
		}
		return nil, nil, err
	}
	skin, err := s.repo.SiteSkin(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.SiteSkin{StreamerID: id}, streamer, nil
		}
		return nil, nil, err
	}
	return skin, streamer, nil
}

func (s *Service) AdminUpdateStreamerSkin(actor *model.User, id string, req UpdateStreamerSkinRequest) (*model.SiteSkin, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	streamer, err := s.repo.Streamer(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.NotFound("主播不存在")
		}
		return nil, err
	}
	skin, err := s.repo.SiteSkin(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		skin = &model.SiteSkin{StreamerID: id}
		err = nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(req.Home)) != "" && string(req.Home) != "null" {
		normalized, _, err := NormalizeStreamerHomePayload(req.HomeTemplateID, req.Home)
		if err != nil {
			return nil, err
		}
		skin.HomePayloadJSON = string(normalized)
		skin.HomeTemplateID = strings.TrimSpace(req.HomeTemplateID)
	}
	if req.HomeTemplateID != "" {
		if err := validateHomeTemplateID(req.HomeTemplateID); err != nil {
			return nil, err
		}
		skin.HomeTemplateID = strings.TrimSpace(req.HomeTemplateID)
	}
	if req.Theme != nil {
		if err := validateThemeJSON(req.Theme); err != nil {
			return nil, err
		}
		skin.ThemeJSON = strings.TrimSpace(string(req.Theme))
	}
	if strings.TrimSpace(req.LogoURL) != "" || strings.TrimSpace(req.Title) != "" || strings.TrimSpace(req.Tagline) != "" {
		skin.LogoURL = strings.TrimSpace(req.LogoURL)
		skin.Title = strings.TrimSpace(req.Title)
		skin.Tagline = strings.TrimSpace(req.Tagline)
	}
	if req.ClearHeroVideo {
		skin.HeroVideoURL = ""
		skin.HeroVideoResourceID = ""
	} else if req.HeroVideoURL != nil {
		skin.HeroVideoURL = sanitizeHomeURL(*req.HeroVideoURL)
	}
	if req.ClearHeroPoster {
		skin.HeroPosterURL = ""
		skin.HeroPosterResourceID = ""
	} else if req.HeroPosterURL != nil {
		skin.HeroPosterURL = sanitizeHomeURL(*req.HeroPosterURL)
	}
	skin.UpdatedAt = time.Now()
	if err := s.repo.SaveSiteSkin(skin); err != nil {
		return nil, err
	}
	if req.CustomHomeEnabled != nil {
		streamer.CustomHomeEnabled = *req.CustomHomeEnabled
		streamer.UpdatedAt = time.Now()
		if err := s.repo.SaveStreamer(streamer); err != nil {
			return nil, err
		}
	}
	return skin, nil
}

func (s *Service) PublicSiteSkin(host string) (*PublicSiteSkin, error) {
	fallback, err := s.officialPublicSiteSkin(host)
	if err != nil {
		return nil, err
	}
	streamer, err := s.ResolveStreamerByHost(host)
	if err != nil {
		return nil, err
	}
	if streamer == nil || streamer.Status != model.StreamerStatusActive {
		return fallback, nil
	}
	settings, err := s.PublicAuthSettings(host)
	if err != nil {
		return nil, err
	}
	skin, err := s.repo.SiteSkin(streamer.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		skin = &model.SiteSkin{}
		err = nil
	}
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(skin.Title)
	if title == "" {
		title = defaultStreamerSiteTitle
	}
	theme := map[string]any{}
	if strings.TrimSpace(skin.ThemeJSON) != "" {
		_ = json.Unmarshal([]byte(skin.ThemeJSON), &theme)
	}
	out := &PublicSiteSkin{
		Slug:                streamer.Slug,
		DisplayName:         streamer.DisplayName,
		Title:               title,
		LogoURL:             strings.TrimSpace(skin.LogoURL),
		Tagline:             strings.TrimSpace(skin.Tagline),
		Theme:               theme,
		InviteLocked:        true,
		RegistrationEnabled: settings.RegistrationEnabled,
		CustomHomeEnabled:   false,
		Home:                nil,
		StreamerActive:      true,
		ParentDomain:        PublicParentDomain(),
		AgentHost:           PublicAgentHost(),
		CanvasHost:          PublicCanvasHost(),
		LandingHost:         StreamerLandingHost(streamer.Slug),
		HeroVideoURL:        resolvedStreamerHeroURL(skin, "video"),
		HeroPosterURL:       resolvedStreamerHeroURL(skin, "poster"),
	}
	if streamer.CustomHomeEnabled {
		rendered, err := RenderStreamerHome(skin.HomeTemplateID, []byte(skin.HomePayloadJSON))
		if err == nil && rendered != nil {
			out.CustomHomeEnabled = true
			out.Home = rendered
		}
	}
	return out, nil
}

func (s *Service) officialPublicSiteSkin(host string) (*PublicSiteSkin, error) {
	settings, err := s.PublicAuthSettings(host)
	if err != nil {
		return nil, err
	}
	title := defaultStreamerSiteTitle
	if s != nil {
		if name := strings.TrimSpace(s.appearanceBrandName()); name != "" {
			title = name
		}
	}
	return &PublicSiteSkin{
		Title:               title,
		Theme:               map[string]any{},
		InviteLocked:        false,
		RegistrationEnabled: settings.RegistrationEnabled,
		CustomHomeEnabled:   false,
		Home:                nil,
		StreamerActive:      false,
		ParentDomain:        PublicParentDomain(),
		AgentHost:           PublicAgentHost(),
		CanvasHost:          PublicCanvasHost(),
	}, nil
}

func (s *Service) RequireActiveStreamer(user *model.User) (*model.Streamer, error) {
	if user == nil {
		return nil, Unauthorized("请先登录")
	}
	if latest, err := s.repo.User(user.ID); err == nil && latest != nil {
		user = latest
	}
	if user.Role != model.UserRoleAdmin && !s.UserHasPermission(user, model.PermAgentConsole) {
		return nil, Forbidden("需要代理身份")
	}
	row, err := s.repo.StreamerByUserID(user.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, Forbidden("需要主播身份")
	}
	if err != nil {
		return nil, err
	}
	if row.Status != model.StreamerStatusActive {
		return nil, Forbidden("主播账号已禁用，无法打开代理后台")
	}
	return row, nil
}

func (s *Service) StreamerConsoleMe(user *model.User, requestHost string) (*StreamerConsoleMe, error) {
	streamer, err := s.RequireActiveStreamer(user)
	if err != nil {
		return nil, err
	}
	host := StreamerLandingHost(streamer.Slug)
	if host == "" {
		parent := cookieParentHost(requestHost)
		if parent != "" {
			host = streamer.Slug + "." + parent
		} else {
			host = streamer.Slug
		}
	}
	return &StreamerConsoleMe{
		Slug:                  streamer.Slug,
		InviteCode:            streamer.InviteCode,
		Host:                  host,
		LandingURL:            publicHTTPS(host),
		AgentURL:              publicHTTPS(PublicAgentHost()),
		CanvasURL:             publicHTTPS(PublicCanvasHost()),
		Status:                streamer.Status,
		SerialNo:              streamer.SerialNo,
		DisplayName:           streamer.DisplayName,
		RebateRateBps:         streamer.RebateRateBps,
		TextRebateBps:         streamer.TextRebateBps,
		ImageRebateBps:        streamer.ImageRebateBps,
		VideoRebateBps:        streamer.VideoRebateBps,
		CustomChannelsEnabled: streamer.CustomChannelsEnabled,
		AlipayAccount:         streamer.AlipayAccount,
		AlipayRealName:        streamer.AlipayRealName,
	}, nil
}

func (s *Service) StreamerConsoleSummary(user *model.User, from, to *time.Time) (*StreamerConsoleSummary, error) {
	streamer, err := s.RequireActiveStreamer(user)
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountStreamerReferredUsers(streamer.ID, streamer.UserID)
	if err != nil {
		return nil, err
	}
	consumed, err := s.repo.SumStreamerConsumedMicrocredits(streamer.ID, streamer.UserID, from, to)
	if err != nil {
		return nil, err
	}
	remaining, err := s.repo.SumStreamerRemainingMicrocredits(streamer.ID, streamer.UserID)
	if err != nil {
		return nil, err
	}
	rebated, err := s.repo.SumStreamerRebateMicrocredits(streamer.ID)
	if err != nil {
		return nil, err
	}
	period := "all"
	if from != nil || to != nil {
		period = "custom"
	}
	rate := streamer.RebateRateBps
	if rate < 0 {
		rate = 0
	}
	return &StreamerConsoleSummary{
		ReferredUserCount:         count,
		ConsumedCredits:           microcreditsToCredits(consumed),
		RemainingCreditsSum:       microcreditsToCredits(remaining),
		RebateRateBps:             rate,
		TextRebateBps:             streamer.TextRebateBps,
		ImageRebateBps:            streamer.ImageRebateBps,
		VideoRebateBps:            streamer.VideoRebateBps,
		ExpectedRebateCredits:     microcreditsToCredits(consumed * int64(rate) / 10000),
		RebateCredits:             microcreditsToCredits(rebated),
		RebateTotalCredits:        microcreditsToCredits(streamer.RebateTotalMicrocredits),
		RebatePendingCredits:      microcreditsToCredits(streamer.RebatePendingMicrocredits),
		RebateWithdrawnCredits:    microcreditsToCredits(streamer.RebateWithdrawnMicrocredits),
		RebateWithdrawableCredits: microcreditsToCredits(streamer.WithdrawableMicrocredits()),
		RebateRequestedCredits:    microcreditsToCredits(streamer.RebateWithdrawnMicrocredits + streamer.RebatePendingMicrocredits),
		Period:                    period,
	}, nil
}

func (s *Service) StreamerConsoleUsers(user *model.User, page, size int) (*StreamerConsoleUserPage, error) {
	streamer, err := s.RequireActiveStreamer(user)
	if err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	total, err := s.repo.CountStreamerReferredUsers(streamer.ID, streamer.UserID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListStreamerReferredUsers(streamer.ID, streamer.UserID, size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	sourceIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		sourceIDs = append(sourceIDs, row.ID)
	}
	rebatesByUser, err := s.repo.SumStreamerRebatesBySourceUsers(streamer.ID, sourceIDs)
	if err != nil {
		return nil, err
	}
	items := make([]StreamerConsoleUser, 0, len(rows))
	for _, row := range rows {
		consumed, err := s.repo.SumUserSettledConsumedMicrocredits(row.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, StreamerConsoleUser{
			UserIDMasked:      maskUserID(row.ID),
			DisplayNameMasked: maskDisplayName(row.DisplayName),
			RegisteredAt:      row.CreatedAt,
			ConsumedCredits:   microcreditsToCredits(consumed),
			RebateCredits:     microcreditsToCredits(rebatesByUser[row.ID]),
			RemainingCredits:  microcreditsToCredits(row.AvailableMicrocredits),
			LastActiveAt:      row.LastLoginAt,
		})
	}
	return &StreamerConsoleUserPage{Items: items, Total: total}, nil
}

func (s *Service) StreamerConsoleRebates(user *model.User, capability string, page, size int) (*StreamerConsoleRebatePage, error) {
	streamer, err := s.RequireActiveStreamer(user)
	if err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	total, err := s.repo.CountStreamerRebates(streamer.ID, capability)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListStreamerRebates(streamer.ID, capability, size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	items := make([]StreamerConsoleRebate, 0, len(rows))
	for _, row := range rows {
		capabilityLabel := strings.TrimSpace(row.Capability)
		if capabilityLabel == "" {
			capabilityLabel = "text"
		}
		items = append(items, StreamerConsoleRebate{
			ID:               row.ID,
			SourceUserMasked: maskUserID(row.SourceUserID),
			Capability:       capabilityLabel,
			ConsumedCredits:  microcreditsToCredits(row.ConsumedMicrocredits),
			ModelShareBps:    row.ModelShareBps,
			AgentRebateBps:   firstPositive(row.AgentRebateBps, row.RateBps),
			RebateCredits:    microcreditsToCredits(row.RebateMicrocredits),
			CreatedAt:        row.CreatedAt,
		})
	}
	return &StreamerConsoleRebatePage{Items: items, Total: total}, nil
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func (s *Service) allocateStreamerInviteCode(slug string) (string, error) {
	base := strings.ToUpper(strings.ReplaceAll(slug, "-", ""))
	for i := 0; i < 8; i++ {
		code := streamerInvitePrefix + base + "-" + randomInviteSuffix()
		exists, err := s.repo.InviteCodeExists(code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("无法生成唯一邀请码")
}

func streamerAdminView(row model.Streamer) StreamerAdminView {
	return StreamerAdminView{
		ID:                        row.ID,
		UserID:                    row.UserID,
		Slug:                      row.Slug,
		InviteCode:                row.InviteCode,
		Status:                    row.Status,
		DisplayName:               row.DisplayName,
		CustomHomeEnabled:         row.CustomHomeEnabled,
		SerialNo:                  row.SerialNo,
		Note:                      row.Note,
		RebateRateBps:             row.RebateRateBps,
		TextRebateBps:             row.TextRebateBps,
		ImageRebateBps:            row.ImageRebateBps,
		VideoRebateBps:            row.VideoRebateBps,
		CustomChannelsEnabled:     row.CustomChannelsEnabled,
		RebateTotalCredits:        microcreditsToCredits(row.RebateTotalMicrocredits),
		RebatePendingCredits:      microcreditsToCredits(row.RebatePendingMicrocredits),
		RebateWithdrawnCredits:    microcreditsToCredits(row.RebateWithdrawnMicrocredits),
		RebateWithdrawableCredits: microcreditsToCredits(row.WithdrawableMicrocredits()),
		AlipayAccount:             row.AlipayAccount,
		AlipayRealName:            row.AlipayRealName,
		CreatedBy:                 row.CreatedBy,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}
}

func firstInt(value *int, fallback *int) *int {
	if value != nil {
		return value
	}
	return fallback
}

func (s *Service) resolveStreamerInviteCode(raw, slug, exceptID string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" {
		return s.allocateStreamerInviteCode(slug)
	}
	if !streamerInviteCodePattern.MatchString(code) {
		return "", BadAuthRequest("邀请码需为 4-32 位大写字母、数字或短横线")
	}
	taken, err := s.repo.InviteCodeTaken(code, exceptID)
	if err != nil {
		return "", err
	}
	if taken {
		return "", BadAuthRequest("邀请码已被占用")
	}
	return code, nil
}

func normalizeRebateRateBps(value *int) int {
	if value == nil {
		return model.DefaultStreamerRebateRateBps
	}
	if *value < 0 {
		return 0
	}
	if *value > 10000 {
		return 10000
	}
	return *value
}

func randomInviteSuffix() string {
	buf := make([]byte, streamerInviteRandLen)
	if _, err := rand.Read(buf); err != nil {
		return "7K2M"
	}
	out := make([]byte, streamerInviteRandLen)
	for i := range buf {
		out[i] = streamerInviteRandAlphabet[int(buf[i])%len(streamerInviteRandAlphabet)]
	}
	return string(out)
}

func microcreditsToCredits(value int64) float64 {
	return float64(value) / float64(CreditScale)
}

func maskUserID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) >= 4 {
		return "u_" + id[:4]
	}
	if id == "" {
		return "u_****"
	}
	return "u_" + id
}

func maskDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "*"
	}
	r, size := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError && size == 0 {
		return "*"
	}
	return string(r) + "*"
}

func cookieParentHost(host string) string {
	slug := StreamerSlugFromHost(host)
	value := strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(value); err == nil {
		value = strings.ToLower(h)
	}
	if slug != "" && strings.HasPrefix(value, slug+".") {
		return strings.TrimPrefix(value, slug+".")
	}
	parts := strings.Split(value, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[1:], ".")
	}
	return value
}

func validateThemeJSON(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return BadAuthRequest("主题 JSON 无效")
	}
	for key, item := range value {
		if key != "primary" && key != "background" && key != "accent" && key != "text" {
			return BadAuthRequest("主题只允许 primary/background/accent/text")
		}
		text, ok := item.(string)
		if !ok || strings.ContainsAny(text, "<>") {
			return BadAuthRequest("主题色值无效")
		}
	}
	return nil
}
