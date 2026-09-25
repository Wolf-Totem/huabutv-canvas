package repository

import (
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) CreateStreamerWithSkin(streamer *model.Streamer, skin *model.SiteSkin) error {
	if r == nil || r.db == nil || streamer == nil || skin == nil {
		return errors.New("streamer create requires streamer and skin")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(streamer).Error; err != nil {
			return err
		}
		if err := tx.Create(skin).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).
			Where("id = ? AND role = ?", streamer.UserID, model.UserRoleUser).
			Updates(map[string]any{"role": model.UserRoleAgent, "updated_at": time.Now()}).Error
	})
}

func (r *Repository) SaveStreamer(streamer *model.Streamer) error {
	return r.db.Save(streamer).Error
}

func (r *Repository) SaveSiteSkin(skin *model.SiteSkin) error {
	return r.db.Save(skin).Error
}

func (r *Repository) Streamer(id string) (*model.Streamer, error) {
	var row model.Streamer
	if err := r.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) StreamerByUserID(userID string) (*model.Streamer, error) {
	var row model.Streamer
	if err := r.db.First(&row, "user_id = ?", strings.TrimSpace(userID)).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) StreamerBySlug(slug string) (*model.Streamer, error) {
	var row model.Streamer
	if err := r.db.First(&row, "slug = ?", strings.ToLower(strings.TrimSpace(slug))).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) StreamerByInviteCode(code string) (*model.Streamer, error) {
	var row model.Streamer
	if err := r.db.Where("upper(invite_code) = ?", strings.ToUpper(strings.TrimSpace(code))).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) ListStreamers() ([]model.Streamer, error) {
	var rows []model.Streamer
	if err := r.db.Order("serial_no asc, created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) SiteSkin(streamerID string) (*model.SiteSkin, error) {
	var row model.SiteSkin
	if err := r.db.First(&row, "streamer_id = ?", strings.TrimSpace(streamerID)).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) BindUserStreamerInviteOnce(userID, streamerID string, at time.Time) (bool, error) {
	result := r.db.Model(&model.User{}).
		Where("id = ? AND invited_by_streamer_id IS NULL", strings.TrimSpace(userID)).
		Updates(map[string]any{"invited_by_streamer_id": streamerID, "invited_at": at, "updated_at": at})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

type StreamerReferredUserRow struct {
	ID                    string
	DisplayName           string
	CreatedAt             time.Time
	LastLoginAt           *time.Time
	AvailableMicrocredits int64
}

func (r *Repository) CountStreamerReferredUsers(streamerID, streamerUserID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).
		Where("invited_by_streamer_id = ? AND id <> ?", streamerID, streamerUserID).
		Count(&count).Error
	return count, err
}

func (r *Repository) ListStreamerReferredUsers(streamerID, streamerUserID string, limit, offset int) ([]StreamerReferredUserRow, error) {
	var rows []StreamerReferredUserRow
	err := r.db.Table("users u").
		Select("u.id, u.display_name, u.created_at, u.last_login_at, COALESCE(ca.available_microcredits, 0) as available_microcredits").
		Joins("LEFT JOIN credit_accounts ca ON ca.user_id = u.id").
		Where("u.invited_by_streamer_id = ? AND u.id <> ?", streamerID, streamerUserID).
		Order("u.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) SumStreamerRemainingMicrocredits(streamerID, streamerUserID string) (int64, error) {
	var total int64
	err := r.db.Table("users u").
		Select("COALESCE(SUM(ca.available_microcredits), 0)").
		Joins("JOIN credit_accounts ca ON ca.user_id = u.id").
		Where("u.invited_by_streamer_id = ? AND u.id <> ?", streamerID, streamerUserID).
		Scan(&total).Error
	return total, err
}

func (r *Repository) SumStreamerConsumedMicrocredits(streamerID, streamerUserID string, from, to *time.Time) (int64, error) {
	query := r.db.Table("users u").
		Select("COALESCE(SUM(bo.actual_amount_microcredits - bo.refunded_amount_microcredits), 0)").
		Joins("JOIN billing_orders bo ON bo.user_id = u.id").
		Where("u.invited_by_streamer_id = ? AND u.id <> ? AND bo.status = ?", streamerID, streamerUserID, model.BillingStatusSettled)
	if from != nil {
		query = query.Where("bo.settled_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("bo.settled_at <= ?", *to)
	}
	var total int64
	err := query.Scan(&total).Error
	return total, err
}

func (r *Repository) SumUserSettledConsumedMicrocredits(userID string) (int64, error) {
	var total int64
	err := r.db.Model(&model.BillingOrder{}).
		Select("COALESCE(SUM(actual_amount_microcredits - refunded_amount_microcredits), 0)").
		Where("user_id = ? AND status = ?", userID, model.BillingStatusSettled).
		Scan(&total).Error
	return total, err
}

func (r *Repository) InviteCodeExists(code string) (bool, error) {
	return r.InviteCodeTaken(code, "")
}

func (r *Repository) InviteCodeTaken(code, exceptID string) (bool, error) {
	query := r.db.Model(&model.Streamer{}).Where("upper(invite_code) = ?", strings.ToUpper(strings.TrimSpace(code)))
	if strings.TrimSpace(exceptID) != "" {
		query = query.Where("id <> ?", strings.TrimSpace(exceptID))
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *Repository) SlugTaken(slug, exceptID string) (bool, error) {
	query := r.db.Model(&model.Streamer{}).Where("slug = ?", strings.ToLower(strings.TrimSpace(slug)))
	if strings.TrimSpace(exceptID) != "" {
		query = query.Where("id <> ?", strings.TrimSpace(exceptID))
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *Repository) SerialTaken(serial int, exceptID string) (bool, error) {
	if serial <= 0 {
		return false, nil
	}
	query := r.db.Model(&model.Streamer{}).Where("serial_no = ?", serial)
	if strings.TrimSpace(exceptID) != "" {
		query = query.Where("id <> ?", strings.TrimSpace(exceptID))
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *Repository) NextStreamerSerial() (int, error) {
	var max int
	err := r.db.Model(&model.Streamer{}).Select("COALESCE(MAX(serial_no), 0)").Scan(&max).Error
	return max + 1, err
}

func (r *Repository) ListStreamerPayouts(streamerID string, limit, offset int) ([]model.StreamerPayout, error) {
	var rows []model.StreamerPayout
	query := r.db.Model(&model.StreamerPayout{})
	if strings.TrimSpace(streamerID) != "" {
		query = query.Where("streamer_id = ?", strings.TrimSpace(streamerID))
	}
	err := query.Order("created_at desc").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, err
}

func (r *Repository) CountStreamerPayouts(streamerID string) (int64, error) {
	query := r.db.Model(&model.StreamerPayout{})
	if strings.TrimSpace(streamerID) != "" {
		query = query.Where("streamer_id = ?", strings.TrimSpace(streamerID))
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r *Repository) StreamerPayout(id string) (*model.StreamerPayout, error) {
	var row model.StreamerPayout
	if err := r.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CountPendingStreamerPayouts(streamerID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.StreamerPayout{}).Where("streamer_id = ? AND status = ?", strings.TrimSpace(streamerID), model.StreamerPayoutPending).Count(&count).Error
	return count, err
}

func (r *Repository) ListStreamerRebates(streamerID, capability string, limit, offset int) ([]model.StreamerRebate, error) {
	var rows []model.StreamerRebate
	query := r.db.Model(&model.StreamerRebate{}).Where("streamer_id = ?", strings.TrimSpace(streamerID))
	if cap := normalizeRebateCapability(capability); cap != "" {
		query = query.Where("capability = ?", cap)
	}
	err := query.Order("created_at desc").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, err
}

func (r *Repository) CountStreamerRebates(streamerID, capability string) (int64, error) {
	query := r.db.Model(&model.StreamerRebate{}).Where("streamer_id = ?", strings.TrimSpace(streamerID))
	if cap := normalizeRebateCapability(capability); cap != "" {
		query = query.Where("capability = ?", cap)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r *Repository) SumStreamerRebatesBySourceUsers(streamerID string, sourceUserIDs []string) (map[string]int64, error) {
	out := map[string]int64{}
	if len(sourceUserIDs) == 0 {
		return out, nil
	}
	type row struct {
		SourceUserID string
		Total        int64
	}
	var rows []row
	err := r.db.Model(&model.StreamerRebate{}).
		Select("source_user_id, COALESCE(SUM(rebate_microcredits), 0) AS total").
		Where("streamer_id = ? AND source_user_id IN ?", strings.TrimSpace(streamerID), sourceUserIDs).
		Group("source_user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.SourceUserID] = item.Total
	}
	return out, nil
}

func normalizeRebateCapability(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "text", "image", "video":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func (r *Repository) SumStreamerRebateMicrocredits(streamerID string) (int64, error) {
	var total int64
	err := r.db.Model(&model.StreamerRebate{}).
		Select("COALESCE(SUM(rebate_microcredits), 0)").
		Where("streamer_id = ?", strings.TrimSpace(streamerID)).
		Scan(&total).Error
	return total, err
}

// ApplyStreamerConsumptionRebate 名下用户成功结算后，按主播返利比例给代理入账。同一订单只发一次。
func (r *Repository) ApplyStreamerConsumptionRebate(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || r == nil || r.db == nil {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var order model.BillingOrder
		if err := tx.First(&order, "id = ?", orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if order.Status != model.BillingStatusSettled {
			return nil
		}
		consumed := order.ActualAmountMicrocredits - order.RefundedAmountMicrocredits
		if consumed <= 0 {
			return nil
		}
		var consumer model.User
		if err := tx.First(&consumer, "id = ?", order.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if consumer.InvitedByStreamerID == nil || strings.TrimSpace(*consumer.InvitedByStreamerID) == "" {
			return nil
		}
		var streamer model.Streamer
		if err := tx.First(&streamer, "id = ?", *consumer.InvitedByStreamerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if streamer.Status != model.StreamerStatusActive || streamer.UserID == order.UserID {
			return nil
		}
		capability := strings.TrimSpace(order.Capability)
		agentBps := clampRebateBps(streamer.RebateBpsForCapability(capability))
		if agentBps <= 0 {
			return nil
		}
		shareBps := modelShareBpsForOrder(tx, order)
		rebate := consumed * int64(shareBps) / 10000
		rebate = rebate * int64(agentBps) / 10000
		if rebate <= 0 {
			return nil
		}
		rec := model.StreamerRebate{
			ID:                   newRepositoryID(),
			StreamerID:           streamer.ID,
			StreamerUserID:       streamer.UserID,
			SourceUserID:         order.UserID,
			BillingOrderID:       order.ID,
			ConsumedMicrocredits: consumed,
			RateBps:              agentBps,
			ModelShareBps:        shareBps,
			AgentRebateBps:       agentBps,
			Capability:           capability,
			RebateMicrocredits:   rebate,
			CreatedAt:            time.Now(),
		}
		created := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "billing_order_id"}}, DoNothing: true}).Create(&rec)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 0 {
			return nil
		}
		return tx.Model(&model.Streamer{}).Where("id = ?", streamer.ID).Updates(map[string]any{
			"rebate_total_microcredits": gorm.Expr("rebate_total_microcredits + ?", rebate),
			"updated_at":                time.Now(),
		}).Error
	})
}

func clampRebateBps(value int) int {
	if value < 0 {
		return 0
	}
	if value > 10000 {
		return 10000
	}
	return value
}

func modelShareBpsForOrder(tx *gorm.DB, order model.BillingOrder) int {
	logicalID := ""
	if strings.TrimSpace(order.TaskID) != "" {
		var task model.Task
		if err := tx.Select("logical_model_id").First(&task, "id = ?", order.TaskID).Error; err == nil {
			logicalID = strings.TrimSpace(task.LogicalModelID)
		}
	}
	var item model.LogicalModel
	if logicalID != "" {
		if err := tx.First(&item, "id = ?", logicalID).Error; err == nil {
			return logicalModelShareBps(item)
		}
	}
	if strings.TrimSpace(order.ChannelModelID) != "" {
		if err := tx.Where("source_channel_model_id = ? AND archived_at IS NULL", order.ChannelModelID).Order("updated_at desc").First(&item).Error; err == nil {
			return logicalModelShareBps(item)
		}
	}
	return model.DefaultAgentShareBps
}

func logicalModelShareBps(item model.LogicalModel) int {
	if !item.AgentShareEnabled {
		return model.DefaultAgentShareBps
	}
	return clampRebateBps(item.AgentShareBps)
}

func (r *Repository) CreateStreamerPayout(payout *model.StreamerPayout) error {
	if r == nil || r.db == nil || payout == nil {
		return errors.New("payout required")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var streamer model.Streamer
		if err := lockRow(tx).First(&streamer, "id = ?", payout.StreamerID).Error; err != nil {
			return err
		}
		var pending int64
		if err := tx.Model(&model.StreamerPayout{}).Where("streamer_id = ? AND status = ?", streamer.ID, model.StreamerPayoutPending).Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return ErrPendingPayout
		}
		if payout.AmountMicrocredits <= 0 || payout.AmountMicrocredits > streamer.WithdrawableMicrocredits() {
			return ErrPayoutAmount
		}
		if err := tx.Create(payout).Error; err != nil {
			return err
		}
		return tx.Model(&model.Streamer{}).Where("id = ?", streamer.ID).Updates(map[string]any{
			"rebate_pending_microcredits": gorm.Expr("rebate_pending_microcredits + ?", payout.AmountMicrocredits),
			"alipay_account":              payout.AlipayAccount,
			"alipay_real_name":            payout.AlipayRealName,
			"updated_at":                  time.Now(),
		}).Error
	})
}

func (r *Repository) ApproveStreamerPayout(id, reviewerID string) (*model.StreamerPayout, error) {
	var row model.StreamerPayout
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
			return err
		}
		if row.Status != model.StreamerPayoutPending {
			return ErrPayoutNotPending
		}
		now := time.Now()
		if err := tx.Model(&row).Updates(map[string]any{
			"status":      model.StreamerPayoutApproved,
			"reviewed_by": strings.TrimSpace(reviewerID),
			"reviewed_at": now,
			"updated_at":  now,
		}).Error; err != nil {
			return err
		}
		row.Status = model.StreamerPayoutApproved
		row.ReviewedBy = strings.TrimSpace(reviewerID)
		row.ReviewedAt = &now
		return tx.Model(&model.Streamer{}).Where("id = ? AND rebate_pending_microcredits >= ?", row.StreamerID, row.AmountMicrocredits).Updates(map[string]any{
			"rebate_pending_microcredits":   gorm.Expr("rebate_pending_microcredits - ?", row.AmountMicrocredits),
			"rebate_withdrawn_microcredits": gorm.Expr("rebate_withdrawn_microcredits + ?", row.AmountMicrocredits),
			"updated_at":                    now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) RejectStreamerPayout(id, reviewerID, reason string) (*model.StreamerPayout, error) {
	var row model.StreamerPayout
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
			return err
		}
		if row.Status != model.StreamerPayoutPending {
			return ErrPayoutNotPending
		}
		now := time.Now()
		if err := tx.Model(&row).Updates(map[string]any{
			"status":        model.StreamerPayoutRejected,
			"reject_reason": strings.TrimSpace(reason),
			"reviewed_by":   strings.TrimSpace(reviewerID),
			"reviewed_at":   now,
			"updated_at":    now,
		}).Error; err != nil {
			return err
		}
		row.Status = model.StreamerPayoutRejected
		row.RejectReason = strings.TrimSpace(reason)
		row.ReviewedBy = strings.TrimSpace(reviewerID)
		row.ReviewedAt = &now
		return tx.Model(&model.Streamer{}).Where("id = ? AND rebate_pending_microcredits >= ?", row.StreamerID, row.AmountMicrocredits).Updates(map[string]any{
			"rebate_pending_microcredits": gorm.Expr("rebate_pending_microcredits - ?", row.AmountMicrocredits),
			"updated_at":                  now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

var (
	ErrPendingPayout    = errors.New("已有提现申请正在审核")
	ErrPayoutAmount     = errors.New("提现金额超过可提现余额")
	ErrPayoutNotPending = errors.New("该提现单不在待审状态")
)

func lockRow(tx *gorm.DB) *gorm.DB {
	if tx != nil && tx.Dialector.Name() == "postgres" {
		return tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return tx
}
