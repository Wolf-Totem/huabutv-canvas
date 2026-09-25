package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const CurrentSchemaVersion int64 = 42

const baselineSchemaChecksum = "sha256:open-ai-canvas-schema-v1-20260830"
const schemaMigrationAppliedAtIndexChecksum = "sha256:schema-migrations-applied-at-index-v2-20260830"
const assetTaxonomyCandidateIdentityChecksum = "sha256:asset-taxonomy-candidate-identity-v3-20260831-r1"
const resourceUploadKeyChecksum = "sha256:resource-upload-key-v4-20260901"
const paymentTopupChecksum = "sha256:payment-topup-v5-20260902"
const resourcePlaybackChecksum = "sha256:resource-playback-v6-20260902"
const assetLibraryFoldersChecksum = "sha256:asset-library-folders-v6-20260902"
const logicalModelActiveCodeChecksum = "sha256:logical-model-active-code-v8-20260905"
const creationRuntimeChecksum = "sha256:creation-runtime-v10-20260909"

const postgresSchemaMigrationLockID int64 = 73123910420260830

type SchemaStatus struct {
	Current  int64 `json:"current"`
	Expected int64 `json:"expected"`
	Ready    bool  `json:"ready"`
}

type schemaMigration struct {
	Version   int64     `gorm:"primaryKey"`
	Name      string    `gorm:"size:160;not null"`
	Checksum  string    `gorm:"size:96;not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

type migration struct {
	version  int64
	name     string
	checksum string
	apply    func(*gorm.DB) error
}

var schemaMigrations = []migration{
	{version: 1, name: "baseline_gorm_schema", checksum: baselineSchemaChecksum, apply: migrateSchemaV1},
	{version: 2, name: "schema_migrations_applied_at_index", checksum: schemaMigrationAppliedAtIndexChecksum, apply: migrateSchemaV2},
	{version: 3, name: "asset_taxonomy_candidate_identity", checksum: assetTaxonomyCandidateIdentityChecksum, apply: migrateSchemaV3},
	{version: 4, name: "resource_upload_key", checksum: resourceUploadKeyChecksum, apply: migrateSchemaV4},
	{version: 5, name: "payment_topup", checksum: paymentTopupChecksum, apply: migrateSchemaV5},
	{version: 6, name: "resource_playback_variant", checksum: resourcePlaybackChecksum, apply: migrateSchemaV6},
	{version: 7, name: "asset_library_folders", checksum: assetLibraryFoldersChecksum, apply: migrateSchemaV7},
	{version: 8, name: "logical_model_active_code", checksum: logicalModelActiveCodeChecksum, apply: migrateSchemaV8},
	{version: 9, name: "channel_presentation", checksum: "sha256:channel-presentation-v9-20260908", apply: migrateChannelPresentation},
	{version: 10, name: "creation_runtime", checksum: creationRuntimeChecksum, apply: migrateSchemaV10},
	{version: 11, name: "cloud_agent_runtime", checksum: "sha256:cloud-agent-runtime-v11-20260912", apply: func(tx *gorm.DB) error { return tx.AutoMigrate(&model.CloudAgentExecution{}) }},
	{version: 12, name: "agent_token_charge_limit", checksum: "sha256:agent-token-charge-limit-v12-20260913", apply: migrateSchemaV12},
	{version: 13, name: "cloud_agent_canvas_mutation", checksum: "sha256:cloud-agent-canvas-mutation-v13-20260913", apply: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.CloudAgentCanvasMutation{})
	}},
	{version: 14, name: "cloud_agent_recovery_control", checksum: "sha256:cloud-agent-recovery-control-v14", apply: migrateSchemaV14},
	{version: 15, name: "agent_profiles", checksum: "sha256:agent-profiles-v15-20260914", apply: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.AgentProfile{})
	}},
	{version: 16, name: "agent_lessons", checksum: "sha256:agent-lessons-v16-20260917", apply: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.AgentLesson{})
	}},
	{version: 17, name: "agent_lessons_owner_index", checksum: "sha256:agent-lessons-owner-index-v17-20260917", apply: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.AgentLesson{})
	}},
	{version: 18, name: "agent_memory_settings", checksum: "sha256:agent-memory-settings-v18-20260917", apply: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.AgentMemorySetting{})
	}},
	{version: 19, name: "payment_plugin_version", checksum: "sha256:payment-plugin-version-v19-20260917", apply: migrateSchemaV19},
	{version: 20, name: "membership_storage_commerce", checksum: "sha256:membership-storage-commerce-v20-20260918", apply: migrateSchemaV20},
	{version: 21, name: "storage_bonus_redeem", checksum: "sha256:storage-bonus-redeem-v21-20260918", apply: migrateSchemaV21},
	{version: 22, name: "generation_ticket", checksum: "sha256:generation-ticket-v22-20260919", apply: migrateSchemaV22},
	{version: 23, name: "user_locale", checksum: "sha256:user-locale-v23-20260919", apply: migrateSchemaV23},
	// 24–35 已在现网 v1.5.7 落地。这里只登记相同名称和校验和，避免本机后端启动时 schema 对不上。
	{version: 24, name: "banner_announcements", checksum: "sha256:banner-announcements-v20-20260917", apply: acknowledgeExistingSchema},
	{version: 25, name: "banner_announcement_title_runs", checksum: "sha256:banner-announcement-title-runs-v21-20260917", apply: acknowledgeExistingSchema},
	{version: 26, name: "banner_announcement_notice_type", checksum: "sha256:banner-announcement-notice-type-v22-20260917", apply: acknowledgeExistingSchema},
	{version: 27, name: "canvas_revision_history", checksum: "sha256:canvas-revision-history-v23-20260918", apply: acknowledgeExistingSchema},
	{version: 28, name: "channel_model_label", checksum: "sha256:channel-model-label-v24", apply: acknowledgeExistingSchema},
	{version: 29, name: "video_token_formula_snapshot", checksum: "sha256:video-token-formula-snapshot-v25", apply: acknowledgeExistingSchema},
	{version: 30, name: "channel_model_description", checksum: "sha256:channel-model-description-v26", apply: acknowledgeExistingSchema},
	{version: 31, name: "channel_credit_cost", checksum: "sha256:channel-credit-cost-v27", apply: acknowledgeExistingSchema},
	{version: 32, name: "agent_execution_journal", checksum: "sha256:agent-execution-journal-v28", apply: acknowledgeExistingSchema},
	{version: 33, name: "agent_resource_leases", checksum: "sha256:agent-resource-leases-v29-20260919", apply: acknowledgeExistingSchema},
	{version: 34, name: "builtin_tools", checksum: "sha256:builtin-tools-v30", apply: acknowledgeExistingSchema},
	{version: 35, name: "tool_favorites", checksum: "sha256:tool-favorites-v31", apply: acknowledgeExistingSchema},
	{version: 36, name: "streamer_subdomain", checksum: "sha256:streamer-subdomain-v36-20260923", apply: migrateSchemaV36},
	{version: 37, name: "streamer_rebate_and_agent_role", checksum: "sha256:streamer-rebate-agent-role-v37-20260923", apply: migrateSchemaV37},
	{version: 38, name: "streamer_hero_video", checksum: "sha256:streamer-hero-video-v38-20260923", apply: migrateSchemaV38},
	{version: 39, name: "user_phone_sms", checksum: "sha256:user-phone-sms-v39-20260924", apply: migrateSchemaV39},
	{version: 40, name: "streamer_wallet_payout_share", checksum: "sha256:streamer-wallet-payout-share-v40-20260924", apply: migrateSchemaV40},
	{version: 41, name: "plaza_community", checksum: "sha256:plaza-community-v41-20260924", apply: migrateSchemaV41},
	{version: 42, name: "plaza_external_cover", checksum: "sha256:plaza-external-cover-v42-20260925", apply: migrateSchemaV42},
}

func acknowledgeExistingSchema(_ *gorm.DB) error {
	return nil
}

func migrateSchemaV42(tx *gorm.DB) error {
	return tx.AutoMigrate(&model.PlazaWork{})
}

func migrateSchemaV14(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.CloudAgentExecution{}); err != nil {
		return err
	}
	// Keep cancellation recoverable for executions admitted before this schema.
	// Bound memory while retaining the migration transaction's all-or-nothing semantics.
	after := ""
	for {
		var runs []model.CloudAgentExecution
		if err := tx.Where("id > ? AND status <> ?", after, "completed").Order("id ASC").Limit(100).Find(&runs).Error; err != nil {
			return err
		}
		if len(runs) == 0 {
			return nil
		}
		for _, run := range runs {
			var state struct {
				Request struct {
					CanvasID string `json:"canvasId"`
				} `json:"request"`
				ActiveTaskID string `json:"activeTaskId"`
				MediaTaskID  string `json:"mediaTaskId"`
			}
			// The root task ID is always a safe cancellation anchor. If an old
			// transcript is damaged, retain a durable warning and cancel that
			// root task during recovery instead of blocking the whole deployment.
			updates := map[string]any{"active_task_id": run.ID}
			if err := json.Unmarshal([]byte(run.StateJSON), &state); err != nil {
				updates["failure_message"] = "旧 Agent 运行记录损坏，已保留根任务并进入安全收尾；请核对任务中心"
			} else {
				updates["canvas_id"] = state.Request.CanvasID
				if state.ActiveTaskID != "" {
					updates["active_task_id"] = state.ActiveTaskID
				}
				updates["media_task_id"] = state.MediaTaskID
			}
			if run.Status == "cancelled" || run.Status == "failed" {
				updates["cleanup_pending"] = true
			}
			if err := tx.Model(&model.CloudAgentExecution{}).Where("id = ?", run.ID).Updates(updates).Error; err != nil {
				return err
			}
			after = run.ID
		}
	}
}

func migrateChannelPresentation(tx *gorm.DB) error {
	for _, column := range []struct {
		model any
		field string
	}{{&model.ModelChannel{}, "PublicAlias"}, {&model.ModelChannel{}, "SortOrder"}, {&model.ChannelModel{}, "SortOrder"}} {
		if !tx.Migrator().HasColumn(column.model, column.field) {
			if err := tx.Migrator().AddColumn(column.model, column.field); err != nil {
				return err
			}
		}
	}
	return nil
}

func migrationsForDatabase(db *gorm.DB) ([]migration, error) {
	var applied schemaMigration
	err := db.First(&applied, "version = ?", 6).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return schemaMigrations, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取数据库迁移 6：%w", err)
	}
	if applied.Name != "asset_library_folders" {
		return schemaMigrations, nil
	}
	legacy := migration{version: 6, name: "asset_library_folders", checksum: assetLibraryFoldersChecksum, apply: migrateSchemaV7}
	if err := validateMigrationRecord(applied, legacy); err != nil {
		return nil, err
	}
	plan := append([]migration(nil), schemaMigrations...)
	for index, item := range plan {
		switch item.version {
		case 6:
			plan[index] = legacy
		case 7:
			plan[index] = migration{version: 7, name: "resource_playback_variant", checksum: resourcePlaybackChecksum, apply: migrateSchemaV6}
		}
	}
	return plan, nil
}

func migrateSchemaV2(tx *gorm.DB) error {
	return tx.Exec("CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations (applied_at)").Error
}

func migrateSchemaV3(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.ProjectAssetCandidate{}); err != nil {
		return fmt.Errorf("扩展资产候选身份字段：%w", err)
	}
	if err := tx.Exec("UPDATE assets SET category = 'prop' WHERE category IN ('wardrobe', 'weapon', 'accessory')").Error; err != nil {
		return fmt.Errorf("合并资产道具分类：%w", err)
	}
	if err := tx.Exec("UPDATE assets SET category = 'material' WHERE category = 'style' OR (category = 'other' AND kind IN ('image', 'video', 'audio', 'model'))").Error; err != nil {
		return fmt.Errorf("迁移资产素材分类：%w", err)
	}
	if err := tx.Exec("UPDATE project_asset_candidates SET category = 'prop' WHERE category IN ('wardrobe', 'weapon', 'accessory')").Error; err != nil {
		return fmt.Errorf("合并候选道具分类：%w", err)
	}
	if err := tx.Exec("UPDATE project_asset_candidates SET category = 'material' WHERE category = 'style'").Error; err != nil {
		return fmt.Errorf("迁移候选素材分类：%w", err)
	}
	var candidates []model.ProjectAssetCandidate
	if err := tx.Order("created_at asc, id asc").Find(&candidates).Error; err != nil {
		return fmt.Errorf("读取资产候选身份：%w", err)
	}
	seenPending := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		nameKey := model.AssetCandidateNameKey(candidate.Name)
		updates := map[string]any{"name_key": nameKey}
		identity := candidate.ProjectID + ":" + string(candidate.Category) + ":" + nameKey
		if candidate.Status == "pending_confirmation" && nameKey != "" {
			if _, exists := seenPending[identity]; exists {
				updates["status"] = "ignored"
			} else {
				seenPending[identity] = candidate.ID
			}
		}
		if err := tx.Model(&model.ProjectAssetCandidate{}).Where("id = ?", candidate.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("回填资产候选身份 %s：%w", candidate.ID, err)
		}
	}
	return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_project_asset_candidates_pending_identity ON project_asset_candidates(project_id, category, name_key) WHERE status = 'pending_confirmation' AND name_key <> ''").Error
}

func migrateSchemaV4(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.Resource{}) {
		return fmt.Errorf("资源表不存在")
	}
	if !tx.Migrator().HasColumn(&model.Resource{}, "upload_key") {
		if err := tx.Migrator().AddColumn(&model.Resource{}, "UploadKey"); err != nil {
			return fmt.Errorf("增加资源上传幂等列：%w", err)
		}
	}
	if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_resources_user_upload_key ON resources (user_id, upload_key)").Error; err != nil {
		return fmt.Errorf("创建资源上传幂等索引：%w", err)
	}
	return nil
}
func migrateSchemaV5(tx *gorm.DB) error {
	if err := tx.AutoMigrate(
		&model.CreditLedgerEntry{},
		&model.TopupProduct{},
		&model.PaymentProviderConfig{},
		&model.PaymentOrder{},
		&model.PaymentNotification{},
		&model.PaymentReconciliationRun{},
		&model.PaymentReconciliationItem{},
	); err != nil {
		return fmt.Errorf("创建积分支付与对账结构：%w", err)
	}
	return nil
}

func migrateSchemaV19(tx *gorm.DB) error {
	for _, value := range []any{&model.PaymentProviderConfig{}, &model.PaymentOrder{}} {
		if !tx.Migrator().HasTable(value) {
			continue
		}
		if err := addPaymentPluginVersionColumn(tx, value); err != nil {
			return err
		}
	}
	return nil
}

func migrateSchemaV20(tx *gorm.DB) error {
	if err := tx.AutoMigrate(
		&model.MembershipProduct{},
		&model.UserMembership{},
		&model.MembershipGrant{},
		&model.PaymentOrder{},
		&model.RedeemBatch{},
		&model.RedeemCode{},
	); err != nil {
		return err
	}
	if err := tx.Exec("UPDATE payment_orders SET product_kind = ? WHERE product_kind IS NULL OR product_kind = ''", model.ProductKindCreditTopup).Error; err != nil {
		return err
	}
	if err := tx.Exec("UPDATE redeem_batches SET kind = ? WHERE kind IS NULL OR kind = ''", model.RedeemKindCredits).Error; err != nil {
		return err
	}
	if err := tx.Exec("UPDATE redeem_codes SET kind = ? WHERE kind IS NULL OR kind = ''", model.RedeemKindCredits).Error; err != nil {
		return err
	}
	now := time.Now().UTC()
	const creditScale int64 = 1_000_000
	seeds := []model.MembershipProduct{
		{ID: "membership-permanent", SKU: model.MembershipSKUPermanent, Name: "永久订阅", Description: "画布基础功能永久使用，可配置我的存储空间，不发放积分。", AmountFen: 0, CreditsMicrocredits: 0, StorageQuotaBytes: 0, DurationDays: 0, Enabled: false, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: "membership-advanced-month", SKU: model.MembershipSKUAdvancedMonth, Name: "月卡", Description: "高级订阅 30 天，发放 300 积分，平台存储 1GiB。", AmountFen: 0, CreditsMicrocredits: 300 * creditScale, StorageQuotaBytes: 1 << 30, DurationDays: 30, Enabled: false, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: "membership-advanced-quarter", SKU: model.MembershipSKUAdvancedQuarter, Name: "季卡", Description: "高级订阅 90 天，发放 1200 积分，平台存储 5GiB。", AmountFen: 0, CreditsMicrocredits: 1200 * creditScale, StorageQuotaBytes: 5 << 30, DurationDays: 90, Enabled: false, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: "membership-advanced-year", SKU: model.MembershipSKUAdvancedYear, Name: "年卡", Description: "高级订阅 365 天，发放 3600 积分，平台存储 3TiB。", AmountFen: 0, CreditsMicrocredits: 3600 * creditScale, StorageQuotaBytes: 3 << 40, DurationDays: 365, Enabled: false, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
	}
	for _, seed := range seeds {
		item := seed
		var existing model.MembershipProduct
		err := tx.Where("sku = ?", item.SKU).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateSchemaV21(tx *gorm.DB) error {
	return tx.AutoMigrate(&model.UserMembership{}, &model.RedeemBatch{}, &model.RedeemCode{})
}

func migrateSchemaV22(tx *gorm.DB) error {
	return tx.AutoMigrate(&model.Task{})
}

func migrateSchemaV23(tx *gorm.DB) error {
	return tx.AutoMigrate(&model.User{})
}

func migrateSchemaV36(tx *gorm.DB) error {
	// 主播子域：streamers + site_skins，users 只加邀请归属两列。SQLite 不加 FK。
	if err := tx.AutoMigrate(&model.Streamer{}, &model.SiteSkin{}, &model.User{}); err != nil {
		return err
	}
	return nil
}

func migrateSchemaV37(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.Streamer{}, &model.StreamerRebate{}, &model.User{}, &model.CreditLedgerEntry{}); err != nil {
		return err
	}
	if err := tx.Exec("UPDATE streamers SET rebate_rate_bps = ? WHERE rebate_rate_bps = 0 OR rebate_rate_bps IS NULL", model.DefaultStreamerRebateRateBps).Error; err != nil {
		return err
	}
	return tx.Exec("UPDATE users SET role = ? WHERE role = ? AND EXISTS (SELECT 1 FROM streamers WHERE streamers.user_id = users.id)", model.UserRoleAgent, model.UserRoleUser).Error
}

func migrateSchemaV38(tx *gorm.DB) error {
	return tx.AutoMigrate(&model.SiteSkin{})
}

func migrateSchemaV39(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.User{}, &model.SmsVerificationCode{}); err != nil {
		return err
	}
	if tx.Dialector.Name() == "postgres" {
		return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users (phone) WHERE phone <> ''").Error
	}
	return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users (phone) WHERE phone != ''").Error
}

func migrateSchemaV40(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.Streamer{}, &model.StreamerRebate{}, &model.StreamerPayout{}, &model.LogicalModel{}); err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE streamers SET
		text_rebate_bps = CASE WHEN text_rebate_bps = 0 THEN rebate_rate_bps ELSE text_rebate_bps END,
		image_rebate_bps = CASE WHEN image_rebate_bps = 0 THEN rebate_rate_bps ELSE image_rebate_bps END,
		video_rebate_bps = CASE WHEN video_rebate_bps = 0 THEN rebate_rate_bps ELSE video_rebate_bps END
		WHERE rebate_rate_bps > 0`).Error; err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE logical_models SET agent_share_bps = ? WHERE agent_share_bps = 0 OR agent_share_bps IS NULL`, model.DefaultAgentShareBps).Error; err != nil {
		return err
	}
	var streamers []model.Streamer
	if err := tx.Order("created_at asc").Find(&streamers).Error; err != nil {
		return err
	}
	for index, row := range streamers {
		if row.SerialNo > 0 {
			continue
		}
		if err := tx.Model(&model.Streamer{}).Where("id = ?", row.ID).Update("serial_no", index+1).Error; err != nil {
			return err
		}
	}
	if tx.Dialector.Name() == "postgres" {
		return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_streamers_serial_no ON streamers (serial_no)").Error
	}
	return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_streamers_serial_no ON streamers (serial_no)").Error
}

func migrateSchemaV41(tx *gorm.DB) error {
	if err := tx.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.PlazaCategory{},
		&model.PlazaApplication{},
		&model.PlazaWork{},
		&model.PlazaWorkTag{},
		&model.PlazaSnapshot{},
		&model.PlazaSnapshotAsset{},
		&model.PlazaEvent{},
		&model.PlazaLike{},
	); err != nil {
		return err
	}
	if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_plaza_applications_pending_project ON plaza_applications (user_id, project_id) WHERE status = 'pending'").Error; err != nil {
		return err
	}
	now := time.Now()
	var system model.User
	if err := tx.First(&system, "id = ?", model.PlazaSystemUserID).Error; err != nil {
		if err := tx.Create(&model.User{
			ID:           model.PlazaSystemUserID,
			Username:     "plaza-system",
			DisplayName:  "作品广场",
			Role:         model.UserRoleUser,
			Status:       model.UserStatusActive,
			PasswordHash: "!",
			CreatedAt:    now,
			UpdatedAt:    now,
		}).Error; err != nil {
			return err
		}
	}
	seeds := []model.PlazaCategory{
		{ID: "plaza-cat-all", Slug: "all", Name: "全部", Kind: model.PlazaCategoryGenre, Sort: 0, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-featured", Slug: "featured", Name: "精选画布", Kind: model.PlazaCategoryFeatured, Sort: 10, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-short-drama", Slug: "short-drama", Name: "短剧漫剧", Kind: model.PlazaCategoryGenre, Sort: 20, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-film", Slug: "film", Name: "专业影视", Kind: model.PlazaCategoryGenre, Sort: 30, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-ad", Slug: "ad", Name: "商业广告", Kind: model.PlazaCategoryGenre, Sort: 40, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-game", Slug: "game", Name: "动漫游戏", Kind: model.PlazaCategoryGenre, Sort: 50, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-edu", Slug: "edu", Name: "教育生活", Kind: model.PlazaCategoryGenre, Sort: 60, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-campaign-manhua", Slug: "campaign-manhua", Name: "AI漫剧崛起计划", Kind: model.PlazaCategoryCampaign, Sort: 110, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plaza-cat-campaign-director", Slug: "campaign-director", Name: "全民导演请开机", Kind: model.PlazaCategoryCampaign, Sort: 120, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, seed := range seeds {
		var existing model.PlazaCategory
		if err := tx.First(&existing, "id = ? OR slug = ?", seed.ID, seed.Slug).Error; err == nil {
			continue
		}
		if err := tx.Create(&seed).Error; err != nil {
			return err
		}
	}
	var setting model.SystemSetting
	if err := tx.First(&setting, "key = ?", model.PlazaSettingKey).Error; err != nil {
		value := `{"enabled":false,"applyEnabled":false,"copyEnabled":false,"publicWatch":true,"applyDailyLimit":5}`
		return tx.Create(&model.SystemSetting{Key: model.PlazaSettingKey, ValueJSON: value, CreatedAt: now, UpdatedAt: now}).Error
	}
	return nil
}

func addPaymentPluginVersionColumn(tx *gorm.DB, value any) error {
	if tx.Migrator().HasColumn(value, "plugin_version") {
		return nil
	}
	if err := tx.Migrator().AddColumn(value, "PluginVersion"); err != nil {
		return fmt.Errorf("增加支付插件版本列：%w", err)
	}
	return nil
}

func migrateSchemaV6(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.Resource{}) {
		return fmt.Errorf("资源表不存在")
	}
	if !tx.Migrator().HasColumn(&model.Resource{}, "playback_status") {
		if err := tx.Migrator().AddColumn(&model.Resource{}, "PlaybackStatus"); err != nil {
			return fmt.Errorf("增加播放副本状态列：%w", err)
		}
	}
	if !tx.Migrator().HasColumn(&model.Resource{}, "playback_object_key") {
		if err := tx.Migrator().AddColumn(&model.Resource{}, "PlaybackObjectKey"); err != nil {
			return fmt.Errorf("增加播放副本对象键列：%w", err)
		}
	}
	if !tx.Migrator().HasColumn(&model.Resource{}, "playback_error") {
		if err := tx.Migrator().AddColumn(&model.Resource{}, "PlaybackError"); err != nil {
			return fmt.Errorf("增加播放副本错误列：%w", err)
		}
	}
	return nil
}

func migrateSchemaV7(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.Asset{}, &model.AssetFolder{}); err != nil {
		return fmt.Errorf("创建个人素材分类并扩展素材目录字段：%w", err)
	}
	return nil
}

func migrateSchemaV8(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.LogicalModel{}) {
		return nil
	}
	if err := tx.Exec("DROP INDEX IF EXISTS idx_logical_models_code").Error; err != nil {
		return fmt.Errorf("移除前台模型旧 code 唯一索引：%w", err)
	}
	if err := tx.Exec("CREATE UNIQUE INDEX idx_logical_models_code ON logical_models(code) WHERE archived_at IS NULL").Error; err != nil {
		return fmt.Errorf("创建前台模型活动 code 唯一索引：%w", err)
	}
	return nil
}

// migrateSchemaV10 只增加创作运行时表和任务幂等关联；旧任务的空 submission ID 必须继续合法。
func migrateSchemaV10(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.CreationRun{}, &model.CreationSubmission{}, &model.Task{}); err != nil {
		return fmt.Errorf("创建创作运行时结构：%w", err)
	}
	return nil
}

// migrateSchemaV12 为 Agent 的 Token 计费增加最终扣费上限；旧账单保持 0，继续沿用既有按 usage 结算语义。
func migrateSchemaV12(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.BillingOrder{}) {
		return nil
	}
	if tx.Migrator().HasColumn(&model.BillingOrder{}, "ChargeLimitMicrocredits") {
		return nil
	}
	if err := tx.Migrator().AddColumn(&model.BillingOrder{}, "ChargeLimitMicrocredits"); err != nil {
		return fmt.Errorf("增加 Agent Token 扣费上限列：%w", err)
	}
	return nil
}

func MigrateSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", postgresSchemaMigrationLockID).Error; err != nil {
				return fmt.Errorf("获取数据库迁移锁：%w", err)
			}
		}
		if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
			return fmt.Errorf("初始化数据库迁移记录：%w", err)
		}
		plan, err := migrationsForDatabase(tx)
		if err != nil {
			return err
		}
		for _, item := range plan {
			var applied schemaMigration
			err := tx.First(&applied, "version = ?", item.version).Error
			if err == nil {
				if err := validateMigrationRecord(applied, item); err != nil {
					return err
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("读取数据库迁移 %d：%w", item.version, err)
			}
			if err := item.apply(tx); err != nil {
				return fmt.Errorf("执行数据库迁移 %d（%s）：%w", item.version, item.name, err)
			}
			record := schemaMigration{Version: item.version, Name: item.name, Checksum: item.checksum, AppliedAt: time.Now().UTC()}
			if err := tx.Create(&record).Error; err != nil {
				return fmt.Errorf("记录数据库迁移 %d：%w", item.version, err)
			}
		}
		return RequireSchemaVersion(tx)
	})
}

func ReadSchemaStatus(db *gorm.DB) (SchemaStatus, error) {
	status := SchemaStatus{Expected: CurrentSchemaVersion}
	if !db.Migrator().HasTable(&schemaMigration{}) {
		return status, nil
	}
	if err := db.Model(&schemaMigration{}).Select("COALESCE(MAX(version), 0)").Scan(&status.Current).Error; err != nil {
		return status, fmt.Errorf("读取数据库结构版本：%w", err)
	}
	if status.Current != status.Expected {
		return status, nil
	}
	if err := validateMigrationRecords(db); err != nil {
		return status, err
	}
	status.Ready = true
	return status, nil
}

func validateMigrationRecords(db *gorm.DB) error {
	plan, err := migrationsForDatabase(db)
	if err != nil {
		return err
	}
	for _, item := range plan {
		var applied schemaMigration
		if err := db.First(&applied, "version = ?", item.version).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("数据库缺少迁移记录 %d（%s）", item.version, item.name)
			}
			return fmt.Errorf("读取数据库迁移 %d：%w", item.version, err)
		}
		if err := validateMigrationRecord(applied, item); err != nil {
			return err
		}
	}
	return nil
}

func validateMigrationRecord(applied schemaMigration, expected migration) error {
	if applied.Name != expected.name {
		return fmt.Errorf("数据库迁移 %d 名称不一致：记录为 %s，程序期望 %s", expected.version, applied.Name, expected.name)
	}
	if applied.Checksum != expected.checksum {
		return fmt.Errorf("数据库迁移 %d 校验和不一致：记录为 %s，程序期望 %s", expected.version, applied.Checksum, expected.checksum)
	}
	return nil
}

func RequireSchemaVersion(db *gorm.DB) error {
	status, err := ReadSchemaStatus(db)
	if err != nil {
		return err
	}
	if status.Current < status.Expected {
		return fmt.Errorf("数据库结构版本过旧：当前 %d，程序要求 %d，请先执行 migrate-schema up", status.Current, status.Expected)
	}
	if status.Current > status.Expected {
		return fmt.Errorf("数据库结构版本 %d 高于程序支持的 %d，拒绝使用旧程序连接新数据库", status.Current, status.Expected)
	}
	return nil
}
