package repository

import (
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RedeemHooks struct {
	BeforeApply func(tx *gorm.DB, code *model.RedeemCode) error
}

type RedeemResult struct {
	Account *model.CreditAccount
	Code    model.RedeemCode
}

func (r *Repository) MembershipProduct(id string) (*model.MembershipProduct, error) {
	var product model.MembershipProduct
	return &product, r.db.First(&product, "id = ?", strings.TrimSpace(id)).Error
}

func (r *Repository) MembershipProductBySKU(sku string) (*model.MembershipProduct, error) {
	var product model.MembershipProduct
	return &product, r.db.First(&product, "sku = ?", strings.TrimSpace(sku)).Error
}

func (r *Repository) MembershipProducts(includeDisabled bool) ([]model.MembershipProduct, error) {
	var products []model.MembershipProduct
	query := r.db.Model(&model.MembershipProduct{})
	if !includeDisabled {
		query = query.Where("enabled = ?", true)
	}
	err := query.Order("sort_order asc, sku asc").Find(&products).Error
	return products, err
}

func (r *Repository) SaveMembershipProduct(product *model.MembershipProduct) error {
	return r.db.Save(product).Error
}

func (r *Repository) UserMembership(userID string) (*model.UserMembership, error) {
	var row model.UserMembership
	err := r.db.First(&row, "user_id = ?", strings.TrimSpace(userID)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &row, err
}

func (r *Repository) UserMemberships(userIDs []string) ([]model.UserMembership, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var rows []model.UserMembership
	err := r.db.Where("user_id IN ?", userIDs).Find(&rows).Error
	return rows, err
}

func (r *Repository) OccupiedMembershipPaymentOrderCount(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.PaymentOrder{}).Where("user_id = ? AND product_kind = ? AND status IN ?", userID, model.ProductKindMembership, []model.PaymentOrderStatus{
		model.PaymentOrderCreated, model.PaymentOrderPending, model.PaymentOrderClosing, model.PaymentOrderCreateFailed,
	}).Count(&count).Error
	return count, err
}

func (r *Repository) OccupiedMembershipPaymentOrder(userID string) (*model.PaymentOrder, error) {
	var order model.PaymentOrder
	err := r.db.Where("user_id = ? AND product_kind = ? AND status IN ?", userID, model.ProductKindMembership, []model.PaymentOrderStatus{
		model.PaymentOrderCreated, model.PaymentOrderPending, model.PaymentOrderClosing, model.PaymentOrderCreateFailed,
	}).Order("created_at desc").First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &order, err
}

func (r *Repository) CreateMembershipPaymentOrder(order *model.PaymentOrder, assert func(tx *gorm.DB) error) (*model.PaymentOrder, bool, error) {
	var created bool
	var result model.PaymentOrder
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if _, err := lockUserMembership(tx, order.UserID); err != nil {
			return err
		}
		var existing model.PaymentOrder
		lookup := tx.Where("user_id = ? AND idempotency_key = ?", order.UserID, order.IdempotencyKey).First(&existing)
		if lookup.Error == nil {
			if existing.ProductID != order.ProductID || existing.ProviderID != order.ProviderID {
				return ErrPaymentOrderStateConflict
			}
			result = existing
			created = false
			return nil
		}
		if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
			return lookup.Error
		}
		if assert != nil {
			if err := assert(tx); err != nil {
				return err
			}
		}
		insert := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "idempotency_key"}},
			DoNothing: true,
		}).Create(order)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 1 {
			result = *order
			created = true
			return nil
		}
		if err := tx.Where("user_id = ? AND idempotency_key = ?", order.UserID, order.IdempotencyKey).First(&existing).Error; err != nil {
			return err
		}
		result = existing
		created = false
		return nil
	})
	return &result, created, err
}

func (r *Repository) UpdateUserStorageOverride(userID, actorID string, override *int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		row, err := lockUserMembership(tx, userID)
		if err != nil {
			return err
		}
		now := time.Now()
		row.StorageOverrideBytes = override
		row.StorageOverrideSetBy = ""
		row.StorageOverrideSetAt = nil
		if override != nil {
			row.StorageOverrideSetBy = actorID
			row.StorageOverrideSetAt = &now
		}
		row.UpdatedAt = now
		return tx.Save(row).Error
	})
}

func (r *Repository) ApplyMembershipGrant(tx *gorm.DB, userID string, snap model.MembershipGrantSnapshot) error {
	if tx == nil {
		return errors.New("membership grant transaction is required")
	}
	row, err := lockUserMembership(tx, userID)
	if err != nil {
		return err
	}
	now := time.Now()
	starts := now
	applied := model.ApplyMembershipSnapshot(*row, snap, now)
	applied.UpdatedAt = now
	if err := tx.Save(&applied).Error; err != nil {
		return err
	}
	var grantEnds *time.Time
	if model.IsAdvancedMembershipSKU(snap.PlanSKU) {
		grantEnds = applied.AdvancedExpiresAt
	}
	grant := model.MembershipGrant{
		ID:                  newRepositoryID(),
		UserID:              userID,
		Source:              snap.Source,
		ProductID:           snap.ProductID,
		PlanSKU:             snap.PlanSKU,
		CreditsMicrocredits: snap.CreditsMicrocredits,
		StorageQuotaBytes:   snap.StorageQuotaBytes,
		DurationDays:        snap.DurationDays,
		StartsAt:            starts,
		EndsAt:              grantEnds,
		Note:                snap.Note,
		CreatedAt:           now,
	}
	if snap.PaymentOrderID != "" {
		id := snap.PaymentOrderID
		grant.PaymentOrderID = &id
	}
	if snap.RedeemCodeID != "" {
		id := snap.RedeemCodeID
		grant.RedeemCodeID = &id
	}
	if strings.TrimSpace(snap.AdminIdempotencyKey) != "" {
		key := strings.TrimSpace(snap.AdminIdempotencyKey)
		grant.AdminIdempotencyKey = &key
	}
	return tx.Create(&grant).Error
}

func (r *Repository) AdminGrantMembership(userID string, snap model.MembershipGrantSnapshot, assert func(tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if _, err := lockUserMembership(tx, userID); err != nil {
			return err
		}
		if assert != nil {
			if err := assert(tx); err != nil {
				return err
			}
		}
		if snap.CreditsMicrocredits > 0 {
			if err := grantCreditsInTx(tx, userID, snap.CreditsMicrocredits, model.CreditLedgerAdminGrant, "", "", "", snap.Note, "membership-admin:"+userID+":"+strings.TrimSpace(snap.AdminIdempotencyKey)); err != nil {
				return err
			}
		}
		return r.ApplyMembershipGrant(tx, userID, snap)
	})
}

func lockUserMembership(tx *gorm.DB, userID string) (*model.UserMembership, error) {
	now := time.Now()
	row := model.UserMembership{UserID: userID, CreatedAt: now, UpdatedAt: now}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return nil, err
	}
	var locked model.UserMembership
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &locked, nil
}

func grantCreditsInTx(tx *gorm.DB, userID string, amount int64, ledgerType model.CreditLedgerType, paymentOrderID, redeemCodeID, actorUserID, note, referenceKey string) error {
	if amount <= 0 {
		return nil
	}
	now := time.Now()
	account := model.CreditAccount{UserID: userID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&account).Error; err != nil {
		return err
	}
	entry := model.CreditLedgerEntry{
		ID: newRepositoryID(), UserID: userID, Type: ledgerType,
		AmountMicrocredits: amount, PaymentOrderID: paymentOrderID, RedeemCodeID: redeemCodeID, ActorUserID: actorUserID, Note: note,
	}
	if strings.TrimSpace(referenceKey) != "" {
		key := referenceKey
		entry.ReferenceKey = &key
		created := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "reference_key"}}, DoNothing: true}).Create(&entry)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 0 {
			return nil
		}
	} else if err := tx.Create(&entry).Error; err != nil {
		return err
	}
	updated := tx.Model(&model.CreditAccount{}).Where("user_id = ?", userID).Updates(map[string]any{
		"available_microcredits": gorm.Expr("available_microcredits + ?", amount),
		"version":                gorm.Expr("version + 1"),
		"updated_at":             now,
	})
	if updated.Error != nil {
		return updated.Error
	}
	if err := tx.First(&account, "user_id = ?", userID).Error; err != nil {
		return err
	}
	return tx.Model(&entry).Updates(map[string]any{
		"available_delta_microcredits": amount,
		"available_after_microcredits": account.AvailableMicrocredits,
		"reserved_after_microcredits":  account.ReservedMicrocredits,
	}).Error
}

func (r *Repository) UserPlatformStoredFileBytes(userID string) (int64, error) {
	var total int64
	err := r.db.Raw(`
		SELECT COALESCE((
			SELECT SUM(physical_resources.size)
			FROM (
				SELECT MAX(resources.size) AS size
				FROM resources
				WHERE resources.user_id = ? AND resources.status = ?
				  AND (
					resources.storage_setting_id IS NULL
					OR resources.storage_setting_id = ''
					OR resources.storage_setting_id NOT IN (
						SELECT id FROM user_oss_settings WHERE user_id = ?
						UNION
						SELECT id FROM storage_locations WHERE scope = 'user' AND owner_id = ?
					)
				  )
				GROUP BY COALESCE(NULLIF(resources.provider, ''), 'local'), resources.endpoint, resources.bucket, resources.object_key
			) AS physical_resources
		), 0)
	`, userID, model.ResourceStatusReady, userID, userID).Scan(&total).Error
	return total, err
}

func OccupiedMembershipCountInTx(tx *gorm.DB, userID string) (int64, error) {
	var count int64
	err := tx.Model(&model.PaymentOrder{}).Where("user_id = ? AND product_kind = ? AND status IN ?", userID, model.ProductKindMembership, []model.PaymentOrderStatus{
		model.PaymentOrderCreated, model.PaymentOrderPending, model.PaymentOrderClosing, model.PaymentOrderCreateFailed,
	}).Count(&count).Error
	return count, err
}

func UserMembershipInTx(tx *gorm.DB, userID string) (*model.UserMembership, error) {
	var row model.UserMembership
	err := tx.First(&row, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.UserMembership{UserID: userID}, nil
	}
	return &row, err
}
