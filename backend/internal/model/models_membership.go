package model

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

const (
	MembershipSKUPermanent       = "permanent"
	MembershipSKUAdvancedMonth   = "advanced_month"
	MembershipSKUAdvancedQuarter = "advanced_quarter"
	MembershipSKUAdvancedYear    = "advanced_year"
	MembershipSKUVipMonth        = "vip_month"
	MembershipSKUVipQuarter      = "vip_quarter"
	MembershipSKUVipYear         = "vip_year"
	MembershipSKUSvipMonth       = "svip_month"
	MembershipSKUSvipQuarter     = "svip_quarter"
	MembershipSKUSvipYear        = "svip_year"

	MembershipTierVip  = "vip"
	MembershipTierSvip = "svip"

	ProductKindCreditTopup   = "credit_topup"
	ProductKindMembership    = "membership"
	ProductKindStorageTopup  = "storage_topup"

	RedeemKindCredits    = "credits"
	RedeemKindMembership = "membership"
	RedeemKindStorage    = "storage"

	MembershipGrantSourcePayment = "payment"
	MembershipGrantSourceRedeem  = "redeem"
	MembershipGrantSourceAdmin   = "admin"

	StorageGrantSourcePayment = "payment"
	StorageGrantSourceRedeem  = "redeem"
	StorageGrantSourceAdmin   = "admin"
	StorageGrantSourceMigrate = "migrate"

	QuotaSourceOverride    = "admin_override"
	QuotaSourcePlan        = "plan_grant"
	QuotaSourceGlobal      = "global_default"
	StorageDisplayPlatform = "platform"
	StorageDisplayPersonal = "personal"
	MaxMembershipStorageB  = int64(3) << 40
	CreditScale            = int64(1_000_000)
	DefaultStorageGrantDays = 365
	MaxStorageGrantDays     = 3650
	MaxMembershipFeatureLines = 12
	MaxMembershipShowcaseRunes = 40
	MembershipFreeShowcaseSettingKey = "membership_free_showcase"

	MembershipRenewalWindow = 30 * 24 * time.Hour

	MembershipPurchaseBlockUnknownSKU     = "未知订阅套餐"
	MembershipPurchaseBlockOpenOrder      = "你有一笔未完成的订阅订单"
	MembershipPurchaseBlockRenewalWindow  = "当前订阅剩余超过 30 天，暂不可购买其他会员"
	MembershipPurchaseBlockSvipToVip      = "SVIP 有效期内不能改买 VIP，到期后再选"
	MembershipPurchaseBlockPermanentOwned = "已拥有永久订阅"
)

type MembershipFeatureLine struct {
	Text     string `json:"text"`
	Included bool   `json:"included"`
}

type MembershipShowcase struct {
	Title        string                  `json:"title,omitempty"`
	Description  string                  `json:"description,omitempty"`
	EntryLabel   string                  `json:"entryLabel,omitempty"`
	Audience     string                  `json:"audience,omitempty"`
	AddOnLabel   string                  `json:"addOnLabel,omitempty"`
	FeatureLines []MembershipFeatureLine `json:"featureLines,omitempty"`
}

type MembershipProduct struct {
	ID                   string                  `json:"id" gorm:"primaryKey;size:36"`
	SKU                  string                  `json:"sku" gorm:"size:32;uniqueIndex"`
	Name                 string                  `json:"name" gorm:"size:120"`
	Description          string                  `json:"description" gorm:"size:500"`
	AmountFen            int64                   `json:"amountFen"`
	OriginalAmountFen    int64                   `json:"originalAmountFen"`
	CreditsMicrocredits  int64                   `json:"creditsMicrocredits"`
	StorageQuotaBytes    int64                   `json:"storageQuotaBytes"`
	DurationDays         int                     `json:"durationDays"`
	Tier                 string                  `json:"tier" gorm:"size:16;index"`
	Badge                string                  `json:"badge" gorm:"size:40"`
	Highlighted          bool                    `json:"highlighted"`
	Enabled              bool                    `json:"enabled" gorm:"index"`
	SortOrder            int                     `json:"sortOrder" gorm:"index"`
	EntryLabel           string                  `json:"entryLabel" gorm:"size:40"`
	Audience             string                  `json:"audience" gorm:"size:40"`
	AddOnLabel           string                  `json:"addOnLabel" gorm:"size:40"`
	FeatureLines         []MembershipFeatureLine `json:"featureLines" gorm:"-"`
	FeatureLinesJSON     string                  `json:"-" gorm:"column:feature_lines;type:text"`
	CreatedBy            string                  `json:"createdBy" gorm:"size:36"`
	UpdatedBy            string                  `json:"updatedBy" gorm:"size:36"`
	CreatedAt            time.Time               `json:"createdAt"`
	UpdatedAt            time.Time               `json:"updatedAt"`
}

func (p *MembershipProduct) BeforeSave(_ *gorm.DB) error {
	if p == nil {
		return nil
	}
	p.FeatureLines = NormalizeMembershipFeatureLines(p.FeatureLines)
	p.EntryLabel = TruncateRunes(strings.TrimSpace(p.EntryLabel), MaxMembershipShowcaseRunes)
	p.Audience = TruncateRunes(strings.TrimSpace(p.Audience), MaxMembershipShowcaseRunes)
	p.AddOnLabel = TruncateRunes(strings.TrimSpace(p.AddOnLabel), MaxMembershipShowcaseRunes)
	p.FeatureLinesJSON = EncodeMembershipFeatureLines(p.FeatureLines)
	return nil
}

func (p *MembershipProduct) AfterFind(_ *gorm.DB) error {
	if p == nil {
		return nil
	}
	if len(p.FeatureLines) == 0 {
		p.FeatureLines = DecodeMembershipFeatureLines(p.FeatureLinesJSON)
	}
	return nil
}

type UserMembership struct {
	UserID                string     `json:"userId" gorm:"primaryKey;size:36"`
	PermanentActive       bool       `json:"permanentActive"`
	PermanentGrantedAt    *time.Time `json:"permanentGrantedAt,omitempty"`
	AdvancedPlanSKU       string     `json:"advancedPlanSku" gorm:"size:32"`
	AdvancedExpiresAt     *time.Time `json:"advancedExpiresAt,omitempty" gorm:"index"`
	PlanStorageQuotaBytes int64      `json:"planStorageQuotaBytes"`
	StorageOverrideBytes  *int64     `json:"storageOverrideBytes,omitempty"`
	StorageOverrideSetBy  string     `json:"storageOverrideSetBy,omitempty" gorm:"size:36"`
	StorageOverrideSetAt  *time.Time `json:"storageOverrideSetAt,omitempty"`
	StorageBonusBytes     int64      `json:"storageBonusBytes"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type MembershipGrant struct {
	ID                  string     `json:"id" gorm:"primaryKey;size:36"`
	UserID              string     `json:"userId" gorm:"size:36;index;uniqueIndex:idx_membership_grant_admin_idempotency,priority:1"`
	Source              string     `json:"source" gorm:"size:24;index"`
	PaymentOrderID      *string    `json:"paymentOrderId,omitempty" gorm:"size:36;uniqueIndex"`
	RedeemCodeID        *string    `json:"redeemCodeId,omitempty" gorm:"size:36;uniqueIndex"`
	AdminIdempotencyKey *string    `json:"adminIdempotencyKey,omitempty" gorm:"size:120;uniqueIndex:idx_membership_grant_admin_idempotency,priority:2"`
	ProductID           string     `json:"productId" gorm:"size:36"`
	PlanSKU             string     `json:"planSku" gorm:"size:32"`
	CreditsMicrocredits int64      `json:"creditsMicrocredits"`
	StorageQuotaBytes   int64      `json:"storageQuotaBytes"`
	DurationDays        int        `json:"durationDays"`
	StartsAt            time.Time  `json:"startsAt"`
	EndsAt              *time.Time `json:"endsAt,omitempty"`
	Note                string     `json:"note" gorm:"size:500"`
	CreatedAt           time.Time  `json:"createdAt" gorm:"index"`
}

func (MembershipGrant) TableName() string { return "membership_grants" }

type StorageGrant struct {
	ID             string     `json:"id" gorm:"primaryKey;size:36"`
	UserID         string     `json:"userId" gorm:"size:36;index"`
	Bytes          int64      `json:"bytes"`
	Source         string     `json:"source" gorm:"size:24;index"`
	PaymentOrderID *string    `json:"paymentOrderId,omitempty" gorm:"size:36;uniqueIndex"`
	RedeemCodeID   *string    `json:"redeemCodeId,omitempty" gorm:"size:36;uniqueIndex"`
	ProductID      string     `json:"productId" gorm:"size:36"`
	DurationDays   int        `json:"durationDays"`
	StartsAt       time.Time  `json:"startsAt"`
	EndsAt         *time.Time `json:"endsAt,omitempty" gorm:"index"`
	Note           string     `json:"note" gorm:"size:500"`
	CreatedAt      time.Time  `json:"createdAt" gorm:"index"`
}

func (StorageGrant) TableName() string { return "storage_grants" }

// MembershipGrantSnapshot is persisted by repository inside the credit/redeem transaction.
type MembershipGrantSnapshot struct {
	ProductID           string
	PlanSKU             string
	CreditsMicrocredits int64
	StorageQuotaBytes   int64
	DurationDays        int
	Source              string
	PaymentOrderID      string
	RedeemCodeID        string
	AdminIdempotencyKey string
	Note                string
	ActorUserID         string
}

func MembershipSKURank(sku string) int {
	switch strings.TrimSpace(sku) {
	case MembershipSKUVipMonth, MembershipSKUAdvancedMonth:
		return 1
	case MembershipSKUVipQuarter, MembershipSKUAdvancedQuarter:
		return 2
	case MembershipSKUVipYear, MembershipSKUAdvancedYear:
		return 3
	case MembershipSKUSvipMonth:
		return 4
	case MembershipSKUSvipQuarter:
		return 5
	case MembershipSKUSvipYear:
		return 6
	default:
		return 0
	}
}

func MembershipSKUTier(sku string) string {
	switch strings.TrimSpace(sku) {
	case MembershipSKUSvipMonth, MembershipSKUSvipQuarter, MembershipSKUSvipYear:
		return MembershipTierSvip
	case MembershipSKUVipMonth, MembershipSKUVipQuarter, MembershipSKUVipYear,
		MembershipSKUAdvancedMonth, MembershipSKUAdvancedQuarter, MembershipSKUAdvancedYear:
		return MembershipTierVip
	default:
		return ""
	}
}

func MembershipSKUDurationDays(sku string) int {
	switch strings.TrimSpace(sku) {
	case MembershipSKUVipMonth, MembershipSKUSvipMonth, MembershipSKUAdvancedMonth:
		return 30
	case MembershipSKUVipQuarter, MembershipSKUSvipQuarter, MembershipSKUAdvancedQuarter:
		return 90
	case MembershipSKUVipYear, MembershipSKUSvipYear, MembershipSKUAdvancedYear:
		return 365
	default:
		return 0
	}
}

func IsCatalogMembershipSKU(sku string) bool {
	switch strings.TrimSpace(sku) {
	case MembershipSKUVipMonth, MembershipSKUVipQuarter, MembershipSKUVipYear,
		MembershipSKUSvipMonth, MembershipSKUSvipQuarter, MembershipSKUSvipYear:
		return true
	default:
		return false
	}
}

func IsMembershipSKU(sku string) bool {
	switch strings.TrimSpace(sku) {
	case MembershipSKUPermanent, MembershipSKUAdvancedMonth, MembershipSKUAdvancedQuarter, MembershipSKUAdvancedYear,
		MembershipSKUVipMonth, MembershipSKUVipQuarter, MembershipSKUVipYear,
		MembershipSKUSvipMonth, MembershipSKUSvipQuarter, MembershipSKUSvipYear:
		return true
	default:
		return false
	}
}

func IsTimedMembershipSKU(sku string) bool {
	return MembershipSKURank(sku) > 0
}

func IsAdvancedMembershipSKU(sku string) bool {
	return IsTimedMembershipSKU(sku)
}

func (p MembershipProduct) EffectiveTier() string {
	if tier := strings.TrimSpace(p.Tier); tier != "" {
		return tier
	}
	return MembershipSKUTier(p.SKU)
}

func MembershipSKUSpec(sku string) (credits int64, storage int64, days int, ok bool) {
	switch strings.TrimSpace(sku) {
	case MembershipSKUPermanent:
		return 0, 0, 0, true
	case MembershipSKUAdvancedMonth:
		return 300 * CreditScale, 1 << 30, 30, true
	case MembershipSKUAdvancedQuarter:
		return 1200 * CreditScale, 5 << 30, 90, true
	case MembershipSKUAdvancedYear:
		return 3600 * CreditScale, 3 << 40, 365, true
	default:
		return 0, 0, MembershipSKUDurationDays(sku), false
	}
}

func ActiveMembershipTier(row UserMembership, now time.Time) string {
	if row.AdvancedExpiresAt == nil || !row.AdvancedExpiresAt.After(now) {
		return ""
	}
	return MembershipSKUTier(row.AdvancedPlanSKU)
}

func MembershipRemaining(row UserMembership, now time.Time) time.Duration {
	if row.AdvancedExpiresAt == nil || !row.AdvancedExpiresAt.After(now) {
		return 0
	}
	return row.AdvancedExpiresAt.Sub(now)
}

func TimedMembershipActive(row UserMembership, now time.Time) bool {
	return ActiveMembershipTier(row, now) != ""
}

func PersonalMembershipEligible(row UserMembership, now time.Time) bool {
	return row.PermanentActive || TimedMembershipActive(row, now)
}

func MembershipPurchaseBlockReason(row UserMembership, sku string, occupied int64, now time.Time) string {
	if !IsMembershipSKU(sku) {
		return MembershipPurchaseBlockUnknownSKU
	}
	if occupied > 0 {
		return MembershipPurchaseBlockOpenOrder
	}
	if sku == MembershipSKUPermanent && row.PermanentActive {
		return MembershipPurchaseBlockPermanentOwned
	}
	remaining := MembershipRemaining(row, now)
	if remaining > MembershipRenewalWindow {
		return MembershipPurchaseBlockRenewalWindow
	}
	if remaining > 0 && ActiveMembershipTier(row, now) == MembershipTierSvip && MembershipSKUTier(sku) == MembershipTierVip {
		return MembershipPurchaseBlockSvipToVip
	}
	return ""
}

// ApplyMembershipSnapshot is the lock-held grant math. Callers must pass the
// current row after FOR UPDATE. payment/redeem overwrite sku+quota; admin keeps
// the higher active rank and max storage.
func ApplyMembershipSnapshot(current UserMembership, snap MembershipGrantSnapshot, now time.Time) UserMembership {
	next := current
	if snap.PlanSKU == MembershipSKUPermanent {
		next.PermanentActive = true
		if next.PermanentGrantedAt == nil {
			granted := now
			next.PermanentGrantedAt = &granted
		}
		return next
	}
	if !IsTimedMembershipSKU(snap.PlanSKU) {
		return next
	}
	end := now.Add(time.Duration(snap.DurationDays) * 24 * time.Hour)
	if current.AdvancedExpiresAt != nil && current.AdvancedExpiresAt.After(now) {
		end = current.AdvancedExpiresAt.Add(time.Duration(snap.DurationDays) * 24 * time.Hour)
	}
	next.AdvancedExpiresAt = &end
	if snap.Source == MembershipGrantSourceAdmin {
		activeStorage := int64(0)
		if current.AdvancedExpiresAt != nil && current.AdvancedExpiresAt.After(now) && current.PlanStorageQuotaBytes > 0 {
			activeStorage = current.PlanStorageQuotaBytes
		}
		if snap.StorageQuotaBytes > activeStorage {
			next.PlanStorageQuotaBytes = snap.StorageQuotaBytes
		} else {
			next.PlanStorageQuotaBytes = activeStorage
		}
		if current.AdvancedExpiresAt != nil && current.AdvancedExpiresAt.After(now) && MembershipSKURank(current.AdvancedPlanSKU) > MembershipSKURank(snap.PlanSKU) {
			next.AdvancedPlanSKU = current.AdvancedPlanSKU
		} else {
			next.AdvancedPlanSKU = snap.PlanSKU
		}
		return next
	}
	next.PlanStorageQuotaBytes = snap.StorageQuotaBytes
	next.AdvancedPlanSKU = snap.PlanSKU
	return next
}

func NormalizeProductKind(value string) string {
	kind := strings.TrimSpace(value)
	switch kind {
	case ProductKindMembership, ProductKindStorageTopup:
		return kind
	case "":
		return ProductKindCreditTopup
	default:
		return kind
	}
}

func NormalizeTopupKind(value string) string {
	if strings.TrimSpace(value) == ProductKindStorageTopup {
		return ProductKindStorageTopup
	}
	return ProductKindCreditTopup
}

func NormalizeRedeemKind(value string) string {
	kind := strings.TrimSpace(value)
	switch kind {
	case RedeemKindMembership, RedeemKindStorage:
		return kind
	default:
		return RedeemKindCredits
	}
}

func IsRedeemKind(value string) bool {
	kind := strings.TrimSpace(value)
	if kind == "" {
		return true
	}
	switch kind {
	case RedeemKindCredits, RedeemKindMembership, RedeemKindStorage:
		return true
	default:
		return false
	}
}

func (order *PaymentOrder) BeforeCreate(_ *gorm.DB) error {
	if order == nil {
		return nil
	}
	order.ProductKind = NormalizeProductKind(order.ProductKind)
	return nil
}

func (batch *RedeemBatch) BeforeCreate(_ *gorm.DB) error {
	if batch == nil {
		return nil
	}
	batch.Kind = NormalizeRedeemKind(batch.Kind)
	return nil
}

func (code *RedeemCode) BeforeCreate(_ *gorm.DB) error {
	if code == nil {
		return nil
	}
	code.Kind = NormalizeRedeemKind(code.Kind)
	return nil
}

func TruncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func NormalizeMembershipFeatureLines(lines []MembershipFeatureLine) []MembershipFeatureLine {
	out := make([]MembershipFeatureLine, 0, len(lines))
	for _, line := range lines {
		text := TruncateRunes(strings.TrimSpace(line.Text), MaxMembershipShowcaseRunes)
		if text == "" {
			continue
		}
		out = append(out, MembershipFeatureLine{Text: text, Included: line.Included})
		if len(out) >= MaxMembershipFeatureLines {
			break
		}
	}
	return out
}

func EncodeMembershipFeatureLines(lines []MembershipFeatureLine) string {
	lines = NormalizeMembershipFeatureLines(lines)
	if len(lines) == 0 {
		return ""
	}
	raw, err := json.Marshal(lines)
	if err != nil {
		return ""
	}
	return string(raw)
}

func DecodeMembershipFeatureLines(raw string) []MembershipFeatureLine {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var lines []MembershipFeatureLine
	if err := json.Unmarshal([]byte(raw), &lines); err != nil {
		return nil
	}
	return NormalizeMembershipFeatureLines(lines)
}

func NormalizeShowcaseLabel(value string) string {
	return TruncateRunes(strings.TrimSpace(value), MaxMembershipShowcaseRunes)
}

func NormalizeStorageGrantDays(days int) int {
	if days <= 0 {
		return DefaultStorageGrantDays
	}
	if days > MaxStorageGrantDays {
		return MaxStorageGrantDays
	}
	return days
}

func StorageGrantEndsAt(starts time.Time, days int) *time.Time {
	if days <= 0 {
		return nil
	}
	end := starts.Add(time.Duration(days) * 24 * time.Hour)
	return &end
}

func DefaultMembershipShowcaseCopy(tier string) (entry, audience, addOn string, lines []MembershipFeatureLine) {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case MembershipTierSvip:
		return "含短剧与投稿展示", "短剧 / 多项目", "可叠加", []MembershipFeatureLine{
			{Text: "包含 VIP 全部能力", Included: true},
			{Text: "短剧工作台", Included: true},
			{Text: "广场投稿", Included: true},
			{Text: "个人对象存储", Included: true},
			{Text: "更大开通礼包与套餐容量", Included: true},
		}
	case MembershipTierVip:
		return "创作工作台", "个人稳定产出", "可叠加", []MembershipFeatureLine{
			{Text: "基础创作、画布、素材、任务", Included: true},
			{Text: "剧本 / 分镜 / 批量表", Included: true},
			{Text: "导演台 / 时间线", Included: true},
			{Text: "云端 Agent、技能库、插件、个人渠道", Included: true},
			{Text: "个人对象存储", Included: true},
			{Text: "SVIP 短剧工作台与广场投稿", Included: false},
		}
	default:
		return "开放（基础）", "试用与轻量创作", "可买，不加会员", []MembershipFeatureLine{
			{Text: "基础创作与预览", Included: true},
			{Text: "素材库、任务、广场浏览", Included: true},
			{Text: "剧本 / 分镜 / 导演台 / 时间线", Included: false},
			{Text: "云端 Agent、技能、插件、个人渠道", Included: false},
			{Text: "短剧工作台与广场投稿", Included: false},
			{Text: "个人对象存储", Included: false},
		}
	}
}

func ApplyDefaultMembershipShowcase(product *MembershipProduct, overwrite bool) {
	if product == nil {
		return
	}
	tier := product.EffectiveTier()
	entry, audience, addOn, lines := DefaultMembershipShowcaseCopy(tier)
	if overwrite || strings.TrimSpace(product.EntryLabel) == "" {
		product.EntryLabel = entry
	}
	if overwrite || strings.TrimSpace(product.Audience) == "" {
		product.Audience = audience
	}
	if overwrite || strings.TrimSpace(product.AddOnLabel) == "" {
		product.AddOnLabel = addOn
	}
	if overwrite || len(NormalizeMembershipFeatureLines(product.FeatureLines)) == 0 {
		if len(product.FeatureLines) == 0 {
			product.FeatureLines = DecodeMembershipFeatureLines(product.FeatureLinesJSON)
		}
		if overwrite || len(product.FeatureLines) == 0 {
			product.FeatureLines = lines
		}
	}
}

func DefaultFreeMembershipShowcase() MembershipShowcase {
	entry, audience, addOn, lines := DefaultMembershipShowcaseCopy("")
	return MembershipShowcase{
		Title:        "免费使用",
		Description:  "平台基础功能开放；VIP / SVIP 主要提升云存储、开通礼包与创作额度。",
		EntryLabel:   entry,
		Audience:     audience,
		AddOnLabel:   addOn,
		FeatureLines: lines,
	}
}

func NormalizeMembershipShowcase(value MembershipShowcase) MembershipShowcase {
	value.Title = TruncateRunes(strings.TrimSpace(value.Title), MaxMembershipShowcaseRunes)
	value.Description = TruncateRunes(strings.TrimSpace(value.Description), 120)
	value.EntryLabel = NormalizeShowcaseLabel(value.EntryLabel)
	value.Audience = NormalizeShowcaseLabel(value.Audience)
	value.AddOnLabel = NormalizeShowcaseLabel(value.AddOnLabel)
	value.FeatureLines = NormalizeMembershipFeatureLines(value.FeatureLines)
	return value
}
