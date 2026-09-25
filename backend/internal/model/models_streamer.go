package model

import "time"

const (
	StreamerStatusActive   = "active"
	StreamerStatusDisabled = "disabled"

	DefaultStreamerSiteTitle = "画布 TV"

	HomeTemplateSimple  = "landing-simple"
	HomeTemplateFeature = "landing-feature"
)

const DefaultStreamerRebateRateBps = 1000
const DefaultAgentShareBps = 10000

const (
	StreamerPayoutPending  = "pending"
	StreamerPayoutApproved = "approved"
	StreamerPayoutRejected = "rejected"
)

// Streamer 是主播档案。身份仍以 user_id 为准；创建时同步 users.role=agent。
// 子域只做营销皮 + 归因码，登录后同一套画布。
type Streamer struct {
	ID                          string    `json:"id" gorm:"primaryKey;size:36"`
	UserID                      string    `json:"userId" gorm:"uniqueIndex;size:36;not null"`
	Slug                        string    `json:"slug" gorm:"uniqueIndex;size:48;not null"`
	InviteCode                  string    `json:"inviteCode" gorm:"uniqueIndex;size:32;not null"`
	Status                      string    `json:"status" gorm:"index;size:16;not null;default:active"`
	DisplayName                 string    `json:"displayName" gorm:"size:80;not null;default:''"`
	CustomHomeEnabled           bool      `json:"customHomeEnabled" gorm:"not null;default:false"`
	SerialNo                    int       `json:"serialNo" gorm:"index;not null;default:0"`
	Note                        string    `json:"note" gorm:"size:500;not null;default:''"`
	RebateRateBps               int       `json:"rebateRateBps" gorm:"not null;default:1000"`
	TextRebateBps               int       `json:"textRebateBps" gorm:"not null;default:1000"`
	ImageRebateBps              int       `json:"imageRebateBps" gorm:"not null;default:1000"`
	VideoRebateBps              int       `json:"videoRebateBps" gorm:"not null;default:1000"`
	RebateTotalMicrocredits     int64     `json:"rebateTotalMicrocredits"`
	RebatePendingMicrocredits   int64     `json:"rebatePendingMicrocredits"`
	RebateWithdrawnMicrocredits int64     `json:"rebateWithdrawnMicrocredits"`
	AlipayAccount               string    `json:"alipayAccount" gorm:"size:80;not null;default:''"`
	AlipayRealName              string    `json:"alipayRealName" gorm:"size:40;not null;default:''"`
	CustomChannelsEnabled       bool      `json:"customChannelsEnabled" gorm:"not null;default:true"`
	CreatedBy                   string    `json:"createdBy" gorm:"size:36;not null"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

func (s Streamer) WithdrawableMicrocredits() int64 {
	value := s.RebateTotalMicrocredits - s.RebatePendingMicrocredits - s.RebateWithdrawnMicrocredits
	if value < 0 {
		return 0
	}
	return value
}

func (s Streamer) RebateBpsForCapability(capability string) int {
	switch capability {
	case "image":
		return s.ImageRebateBps
	case "video":
		return s.VideoRebateBps
	default:
		return s.TextRebateBps
	}
}

func (Streamer) TableName() string { return "streamers" }

// SiteSkin 一主播一行营销皮。专属首页与官网同一套，只允许换背景视频。
type SiteSkin struct {
	StreamerID           string    `json:"streamerId" gorm:"primaryKey;size:36"`
	LogoURL              string    `json:"logoUrl" gorm:"size:500;not null;default:''"`
	Title                string    `json:"title" gorm:"size:80;not null;default:''"`
	Tagline              string    `json:"tagline" gorm:"size:200;not null;default:''"`
	ThemeJSON            string    `json:"themeJson" gorm:"type:text;not null"`
	HomeTemplateID       string    `json:"homeTemplateId" gorm:"size:64;not null;default:''"`
	HomePayloadJSON      string    `json:"homePayloadJson" gorm:"type:text;not null"`
	HeroVideoURL         string    `json:"heroVideoUrl" gorm:"size:500;not null;default:''"`
	HeroPosterURL        string    `json:"heroPosterUrl" gorm:"size:500;not null;default:''"`
	HeroVideoResourceID  string    `json:"heroVideoResourceId" gorm:"size:36;not null;default:''"`
	HeroPosterResourceID string    `json:"heroPosterResourceId" gorm:"size:36;not null;default:''"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

func (SiteSkin) TableName() string { return "site_skins" }

// StreamerRebate 名下账号成功扣费后的消费返利入账记录。一单一行，禁止重复发放。
type StreamerRebate struct {
	ID                   string    `json:"id" gorm:"primaryKey;size:36"`
	StreamerID           string    `json:"streamerId" gorm:"index;size:36;not null"`
	StreamerUserID       string    `json:"streamerUserId" gorm:"index;size:36;not null"`
	SourceUserID         string    `json:"sourceUserId" gorm:"index;size:36;not null"`
	BillingOrderID       string    `json:"billingOrderId" gorm:"uniqueIndex;size:36;not null"`
	ConsumedMicrocredits int64     `json:"consumedMicrocredits"`
	RateBps              int       `json:"rateBps"`
	ModelShareBps        int       `json:"modelShareBps"`
	AgentRebateBps       int       `json:"agentRebateBps"`
	Capability           string    `json:"capability" gorm:"size:32;not null;default:''"`
	RebateMicrocredits   int64     `json:"rebateMicrocredits"`
	CreatedAt            time.Time `json:"createdAt"`
}

func (StreamerRebate) TableName() string { return "streamer_rebates" }

// StreamerPayout 代理提现申请。同意只扣钱包，打款在线下完成。
type StreamerPayout struct {
	ID                 string     `json:"id" gorm:"primaryKey;size:36"`
	StreamerID         string     `json:"streamerId" gorm:"index;size:36;not null"`
	StreamerUserID     string     `json:"streamerUserId" gorm:"index;size:36;not null"`
	AmountMicrocredits int64      `json:"amountMicrocredits"`
	AlipayAccount      string     `json:"alipayAccount" gorm:"size:80;not null"`
	AlipayRealName     string     `json:"alipayRealName" gorm:"size:40;not null"`
	Status             string     `json:"status" gorm:"index;size:16;not null"`
	RejectReason       string     `json:"rejectReason" gorm:"size:500;not null;default:''"`
	ReviewedBy         string     `json:"reviewedBy" gorm:"size:36;not null;default:''"`
	ReviewedAt         *time.Time `json:"reviewedAt"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (StreamerPayout) TableName() string { return "streamer_payouts" }
