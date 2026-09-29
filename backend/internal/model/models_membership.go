package model

import (
	"strings"
	"time"

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

	QuotaSourceOverride   = "admin_override"
	QuotaSourcePlan       = "plan_grant"
	QuotaSourceGlobal     = "global_default"
	StorageDisplayPlatform = "platform"
	StorageDisplayPersonal = "personal"
	MaxMembershipStorageB  = int64(3) << 40
	CreditScale            = int64(1_000_000)

	MembershipRenewalWindow = 30 * 24 * time.Hour

	MembershipPurchaseBlockUnknownSKU     = "未知订阅套餐"
	MembershipPurchaseBlockOpenOrder      = "你有一笔未完成的订阅订单"
	MembershipPurchaseBlockRenewalWindow  = "当前订阅剩余超过 30 天，暂不可购买其他会员"
	MembershipPurchaseBlockSvipToVip      = "SVIP 有效期内不能改买 VIP，到期后再选"
	MembershipPurchaseBlockPermanentOwned = "已拥有永久订阅"
)

type MembershipProduct struct {
	ID                   string    `json:"id" gorm:"primaryKey;size:36"`
	SKU                  string    `json:"sku" gorm:"size:32;uniqueIndex"`
	Name                 string    `json:"name" gorm:"size:120"`
	Description          string    `json:"description" gorm:"size:500"`
	AmountFen            int64     `json:"amountFen"`
	OriginalAmountFen    int64     `json:"originalAmountFen"`
	CreditsMicrocredits  int64     `json:"creditsMicrocredits"`
	StorageQuotaBytes    int64     `json:"storageQuotaBytes"`
	DurationDays         int       `json:"durationDays"`
	Tier                 string    `json:"tier" gorm:"size:16;index"`
	Badge                string    `json:"badge" gorm:"size:40"`
	Highlighted          bool      `json:"highlighted"`
	Enabled              bool      `json:"enabled" gorm:"index"`
	SortOrder            int       `json:"sortOrder" gorm:"index"`
	CreatedBy            string    `json:"createdBy" gorm:"size:36"`
	UpdatedBy            string    `json:"updatedBy" gorm:"size:36"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
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
