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

	ProductKindCreditTopup = "credit_topup"
	ProductKindMembership  = "membership"

	RedeemKindCredits    = "credits"
	RedeemKindMembership = "membership"
	RedeemKindStorage    = "storage"

	MembershipGrantSourcePayment = "payment"
	MembershipGrantSourceRedeem  = "redeem"
	MembershipGrantSourceAdmin   = "admin"

	QuotaSourceOverride   = "admin_override"
	QuotaSourcePlan       = "plan_grant"
	QuotaSourceGlobal     = "global_default"
	MaxMembershipStorageB = int64(3) << 40
	CreditScale           = int64(1_000_000)
)

type MembershipProduct struct {
	ID                   string    `json:"id" gorm:"primaryKey;size:36"`
	SKU                  string    `json:"sku" gorm:"size:32;uniqueIndex"`
	Name                 string    `json:"name" gorm:"size:120"`
	Description          string    `json:"description" gorm:"size:500"`
	AmountFen            int64     `json:"amountFen"`
	CreditsMicrocredits  int64     `json:"creditsMicrocredits"`
	StorageQuotaBytes    int64     `json:"storageQuotaBytes"`
	DurationDays         int       `json:"durationDays"`
	Enabled              bool      `json:"enabled" gorm:"index"`
	SortOrder            int       `json:"sortOrder" gorm:"index"`
	CreatedBy            string    `json:"createdBy" gorm:"size:36"`
	UpdatedBy            string    `json:"updatedBy" gorm:"size:36"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type UserMembership struct {
	UserID                 string     `json:"userId" gorm:"primaryKey;size:36"`
	PermanentActive        bool       `json:"permanentActive"`
	PermanentGrantedAt     *time.Time `json:"permanentGrantedAt,omitempty"`
	AdvancedPlanSKU        string     `json:"advancedPlanSku" gorm:"size:32"`
	AdvancedExpiresAt      *time.Time `json:"advancedExpiresAt,omitempty" gorm:"index"`
	PlanStorageQuotaBytes  int64      `json:"planStorageQuotaBytes"`
	StorageOverrideBytes   *int64     `json:"storageOverrideBytes,omitempty"`
	StorageOverrideSetBy   string     `json:"storageOverrideSetBy,omitempty" gorm:"size:36"`
	StorageOverrideSetAt   *time.Time `json:"storageOverrideSetAt,omitempty"`
	StorageBonusBytes      int64      `json:"storageBonusBytes"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

type MembershipGrant struct {
	ID                   string     `json:"id" gorm:"primaryKey;size:36"`
	UserID               string     `json:"userId" gorm:"size:36;index;uniqueIndex:idx_membership_grant_admin_idempotency,priority:1"`
	Source               string     `json:"source" gorm:"size:24;index"`
	PaymentOrderID       *string    `json:"paymentOrderId,omitempty" gorm:"size:36;uniqueIndex"`
	RedeemCodeID         *string    `json:"redeemCodeId,omitempty" gorm:"size:36;uniqueIndex"`
	AdminIdempotencyKey  *string    `json:"adminIdempotencyKey,omitempty" gorm:"size:120;uniqueIndex:idx_membership_grant_admin_idempotency,priority:2"`
	ProductID            string     `json:"productId" gorm:"size:36"`
	PlanSKU              string     `json:"planSku" gorm:"size:32"`
	CreditsMicrocredits  int64      `json:"creditsMicrocredits"`
	StorageQuotaBytes    int64      `json:"storageQuotaBytes"`
	DurationDays         int        `json:"durationDays"`
	StartsAt             time.Time  `json:"startsAt"`
	EndsAt               *time.Time `json:"endsAt,omitempty"`
	Note                 string     `json:"note" gorm:"size:500"`
	CreatedAt            time.Time  `json:"createdAt" gorm:"index"`
}

func (MembershipGrant) TableName() string { return "membership_grants" }

// MembershipGrantSnapshot is persisted by repository inside the credit/redeem transaction.
type MembershipGrantSnapshot struct {
	ProductID            string
	PlanSKU              string
	CreditsMicrocredits  int64
	StorageQuotaBytes    int64
	DurationDays         int
	Source               string
	PaymentOrderID       string
	RedeemCodeID         string
	AdminIdempotencyKey  string
	Note                 string
	ActorUserID          string
}

func MembershipSKURank(sku string) int {
	switch strings.TrimSpace(sku) {
	case MembershipSKUAdvancedMonth:
		return 1
	case MembershipSKUAdvancedQuarter:
		return 2
	case MembershipSKUAdvancedYear:
		return 3
	default:
		return 0
	}
}

func IsMembershipSKU(sku string) bool {
	switch strings.TrimSpace(sku) {
	case MembershipSKUPermanent, MembershipSKUAdvancedMonth, MembershipSKUAdvancedQuarter, MembershipSKUAdvancedYear:
		return true
	default:
		return false
	}
}

func IsAdvancedMembershipSKU(sku string) bool {
	return MembershipSKURank(sku) > 0
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
		return 0, 0, 0, false
	}
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
	if !IsAdvancedMembershipSKU(snap.PlanSKU) {
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
	if kind == "" {
		return ProductKindCreditTopup
	}
	return kind
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
