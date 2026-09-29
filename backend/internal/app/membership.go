package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/payment"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/gorm"
)

const (
	commerceMethodsSettingKey = "commerce_methods"
	supportContactSettingKey  = "support_contact"
)

type CommerceMethods struct {
	OnlinePaymentEnabled bool `json:"onlinePaymentEnabled"`
	RedeemEnabled        bool `json:"redeemEnabled"`
}

type SupportContactSetting struct {
	TicketURL string `json:"ticketUrl"`
	QQ        string `json:"qq"`
}

type MembershipPublicView struct {
	PermanentActive           bool                     `json:"permanentActive"`
	AdvancedPlanSKU           string                   `json:"advancedPlanSku"`
	AdvancedExpiresAt         *time.Time               `json:"advancedExpiresAt,omitempty"`
	AdvancedRemainingSeconds  int64                    `json:"advancedRemainingSeconds"`
	Tier                      string                   `json:"tier,omitempty"`
	CanPurchaseAdvanced       bool                     `json:"canPurchaseAdvanced"`
	CanPurchasePermanent      bool                     `json:"canPurchasePermanent"`
	CanPurchaseVip            bool                     `json:"canPurchaseVip"`
	CanPurchaseSvip           bool                     `json:"canPurchaseSvip"`
	HasOpenMembershipOrder    bool                     `json:"hasOpenMembershipOrder"`
	OpenMembershipOrderID     string                   `json:"openMembershipOrderId,omitempty"`
	PersonalStorageAllowed    bool                     `json:"personalStorageAllowed"`
	PersonalBucketEnabled     bool                     `json:"personalBucketEnabled"`
	StorageDisplay            string                   `json:"storageDisplay,omitempty"`
	EffectiveStoredFileBytes  int64                    `json:"effectiveStoredFileBytes"`
	QuotaSource               string                   `json:"quotaSource"`
	StorageOverrideBytes      *int64                   `json:"storageOverrideBytes"`
	StorageBonusBytes         int64                    `json:"storageBonusBytes"`
	StorageExpansionMessage   string                   `json:"storageExpansionMessage"`
	SupportTicketURL          string                   `json:"supportTicketUrl,omitempty"`
	SupportQQ                 string                   `json:"supportQq,omitempty"`
	MinPurchasableAdvancedSKU string                   `json:"minPurchasableAdvancedSku,omitempty"`
	OnlinePaymentEnabled      bool                     `json:"onlinePaymentEnabled"`
	RedeemEnabled             bool                     `json:"redeemEnabled"`
	FeatureShowcase           []MembershipFeatureLine  `json:"featureShowcase,omitempty"`
	FreeShowcase              model.MembershipShowcase `json:"freeShowcase"`
	DefaultStoredFileBytes    int64                    `json:"defaultStoredFileBytes"`
	StorageBonusExpiresAt     *time.Time               `json:"storageBonusExpiresAt,omitempty"`
}

type MembershipFeatureLine = model.MembershipFeatureLine
type MembershipShowcase = model.MembershipShowcase

type MembershipProductView struct {
	model.MembershipProduct
}

type UpdateMembershipProductRequest struct {
	Name                string                         `json:"name"`
	Description         string                         `json:"description"`
	AmountFen           int64                          `json:"amountFen"`
	Enabled             bool                           `json:"enabled"`
	SortOrder           int                            `json:"sortOrder"`
	OriginalAmountFen   *int64                         `json:"originalAmountFen"`
	CreditsMicrocredits *int64                         `json:"creditsMicrocredits"`
	StorageQuotaBytes   *int64                         `json:"storageQuotaBytes"`
	Badge               *string                        `json:"badge"`
	Highlighted         *bool                          `json:"highlighted"`
	EntryLabel          *string                        `json:"entryLabel"`
	Audience            *string                        `json:"audience"`
	AddOnLabel          *string                        `json:"addOnLabel"`
	FeatureLines        *[]model.MembershipFeatureLine `json:"featureLines"`
	SyncShowcaseToTier  bool                           `json:"syncShowcaseToTier"`
}

type AdminStorageQuotaRequest struct {
	StorageOverrideBytes *int64 `json:"storageOverrideBytes"`
}

func (req *AdminStorageQuotaRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	msg, ok := raw["storageOverrideBytes"]
	if !ok {
		return errors.New("请提供 storageOverrideBytes")
	}
	if strings.TrimSpace(string(msg)) == "null" {
		req.StorageOverrideBytes = nil
		return nil
	}
	var value int64
	if err := json.Unmarshal(msg, &value); err != nil {
		return err
	}
	req.StorageOverrideBytes = &value
	return nil
}

type AdminMembershipGrantRequest struct {
	SKU            string `json:"sku"`
	PlanSKU        string `json:"planSku"`
	IdempotencyKey string `json:"idempotencyKey"`
	Note           string `json:"note"`
}

type RedeemOutcome struct {
	Account    *model.CreditAccount  `json:"account"`
	Membership *MembershipPublicView `json:"membership,omitempty"`
	Granted    *RedeemGrantedView    `json:"granted,omitempty"`
}

type RedeemGrantedView struct {
	Kind                string `json:"kind"`
	PlanSKU             string `json:"planSku,omitempty"`
	CreditsMicrocredits int64  `json:"creditsMicrocredits"`
	StorageQuotaBytes   int64  `json:"storageQuotaBytes,omitempty"`
}

func defaultCommerceMethods() CommerceMethods {
	return CommerceMethods{OnlinePaymentEnabled: true, RedeemEnabled: true}
}

func (s *Service) CommerceMethods() (CommerceMethods, error) {
	raw, err := s.repo.SystemSetting(commerceMethodsSettingKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultCommerceMethods(), nil
		}
		return CommerceMethods{}, err
	}
	methods := defaultCommerceMethods()
	if strings.TrimSpace(raw.ValueJSON) == "" {
		return methods, nil
	}
	if err := json.Unmarshal([]byte(raw.ValueJSON), &methods); err != nil {
		return defaultCommerceMethods(), nil
	}
	return methods, nil
}

func (s *Service) UpdateCommerceMethods(actor *model.User, methods CommerceMethods) (CommerceMethods, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return CommerceMethods{}, err
	}
	encoded, err := json.Marshal(methods)
	if err != nil {
		return CommerceMethods{}, err
	}
	if err := s.repo.SaveSystemSetting(&model.SystemSetting{Key: commerceMethodsSettingKey, ValueJSON: string(encoded), UpdatedBy: actor.ID}); err != nil {
		return CommerceMethods{}, err
	}
	if err := s.appendAdminAudit(actor, "commerce_methods.update", "system_setting", commerceMethodsSettingKey, "更新充值通道开关", map[string]any{"onlinePaymentEnabled": methods.OnlinePaymentEnabled, "redeemEnabled": methods.RedeemEnabled}); err != nil {
		return CommerceMethods{}, err
	}
	return methods, nil
}

func (s *Service) SupportContact() (SupportContactSetting, error) {
	raw, err := s.repo.SystemSetting(supportContactSettingKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SupportContactSetting{}, nil
		}
		return SupportContactSetting{}, err
	}
	var setting SupportContactSetting
	if strings.TrimSpace(raw.ValueJSON) == "" {
		return setting, nil
	}
	_ = json.Unmarshal([]byte(raw.ValueJSON), &setting)
	setting.TicketURL = strings.TrimSpace(setting.TicketURL)
	setting.QQ = strings.TrimSpace(setting.QQ)
	return setting, nil
}

func (s *Service) UpdateSupportContact(actor *model.User, setting SupportContactSetting) (SupportContactSetting, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return SupportContactSetting{}, err
	}
	setting.TicketURL = strings.TrimSpace(setting.TicketURL)
	setting.QQ = strings.TrimSpace(setting.QQ)
	encoded, err := json.Marshal(setting)
	if err != nil {
		return SupportContactSetting{}, err
	}
	if err := s.repo.SaveSystemSetting(&model.SystemSetting{Key: supportContactSettingKey, ValueJSON: string(encoded), UpdatedBy: actor.ID}); err != nil {
		return SupportContactSetting{}, err
	}
	return setting, nil
}

func (s *Service) requireOnlinePayment() error {
	methods, err := s.CommerceMethods()
	if err != nil {
		return err
	}
	if !methods.OnlinePaymentEnabled {
		return Forbidden("在线支付暂未开放")
	}
	return nil
}

func (s *Service) requireRedeemEnabled() error {
	methods, err := s.CommerceMethods()
	if err != nil {
		return err
	}
	if !methods.RedeemEnabled {
		return Forbidden("兑换码暂未开放")
	}
	return nil
}

func (s *Service) MembershipProducts(includeDisabled bool) ([]model.MembershipProduct, error) {
	return s.repo.MembershipProducts(includeDisabled)
}

func (s *Service) PublicMembershipProducts() ([]MembershipProductView, error) {
	products, err := s.repo.MembershipProducts(false)
	if err != nil {
		return nil, err
	}
	views := make([]MembershipProductView, 0, len(products))
	for _, product := range products {
		if !model.IsCatalogMembershipSKU(product.SKU) {
			continue
		}
		item := product
		if strings.TrimSpace(item.Tier) == "" {
			item.Tier = model.MembershipSKUTier(item.SKU)
		}
		model.ApplyDefaultMembershipShowcase(&item, false)
		views = append(views, MembershipProductView{MembershipProduct: item})
	}
	return views, nil
}

func (s *Service) MembershipFreeShowcase() (model.MembershipShowcase, error) {
	raw, err := s.repo.SystemSetting(model.MembershipFreeShowcaseSettingKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DefaultFreeMembershipShowcase(), nil
		}
		return model.MembershipShowcase{}, err
	}
	showcase := model.DefaultFreeMembershipShowcase()
	if strings.TrimSpace(raw.ValueJSON) == "" {
		return showcase, nil
	}
	if err := json.Unmarshal([]byte(raw.ValueJSON), &showcase); err != nil {
		return model.DefaultFreeMembershipShowcase(), nil
	}
	return model.NormalizeMembershipShowcase(showcase), nil
}

func (s *Service) UpdateMembershipFreeShowcase(actor *model.User, showcase model.MembershipShowcase) (model.MembershipShowcase, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return model.MembershipShowcase{}, err
	}
	showcase = model.NormalizeMembershipShowcase(showcase)
	if strings.TrimSpace(showcase.Title) == "" {
		showcase.Title = "免费使用"
	}
	encoded, err := json.Marshal(showcase)
	if err != nil {
		return model.MembershipShowcase{}, err
	}
	if err := s.repo.SaveSystemSetting(&model.SystemSetting{Key: model.MembershipFreeShowcaseSettingKey, ValueJSON: string(encoded), UpdatedBy: actor.ID}); err != nil {
		return model.MembershipShowcase{}, err
	}
	if err := s.appendAdminAudit(actor, "membership_free_showcase.update", "system_setting", model.MembershipFreeShowcaseSettingKey, "更新未开通展示", map[string]any{"title": showcase.Title}); err != nil {
		return model.MembershipShowcase{}, err
	}
	return showcase, nil
}

func (s *Service) AdminMembershipProducts(actor *model.User) ([]model.MembershipProduct, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	products, err := s.repo.MembershipProducts(true)
	if err != nil {
		return nil, err
	}
	for i := range products {
		if model.IsCatalogMembershipSKU(products[i].SKU) {
			model.ApplyDefaultMembershipShowcase(&products[i], false)
		}
	}
	return products, nil
}

func (s *Service) UpdateMembershipProduct(actor *model.User, id string, req UpdateMembershipProductRequest) (*model.MembershipProduct, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	product, err := s.repo.MembershipProduct(id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, BadAuthRequest("请填写套餐名称")
	}
	if req.AmountFen < 0 {
		return nil, BadAuthRequest("价格不能为负数")
	}
	product.Name = truncateRunes(name, 120)
	product.Description = truncateRunes(strings.TrimSpace(req.Description), 500)
	product.AmountFen = req.AmountFen
	product.Enabled = req.Enabled
	product.SortOrder = req.SortOrder
	if req.OriginalAmountFen != nil {
		if *req.OriginalAmountFen < 0 {
			return nil, BadAuthRequest("划线价不能为负数")
		}
		product.OriginalAmountFen = *req.OriginalAmountFen
	}
	if req.CreditsMicrocredits != nil {
		if *req.CreditsMicrocredits < 0 || *req.CreditsMicrocredits > maxTopupCreditsMicrocredits {
			return nil, BadAuthRequest("赠送积分必须为 0 至 10 亿积分")
		}
		product.CreditsMicrocredits = *req.CreditsMicrocredits
	}
	if req.StorageQuotaBytes != nil {
		if *req.StorageQuotaBytes < 0 || *req.StorageQuotaBytes > model.MaxMembershipStorageB {
			return nil, BadAuthRequest("套餐容量需为 0 至 3TiB")
		}
		product.StorageQuotaBytes = *req.StorageQuotaBytes
	}
	if req.Badge != nil {
		product.Badge = truncateRunes(strings.TrimSpace(*req.Badge), 40)
	}
	if req.Highlighted != nil {
		product.Highlighted = *req.Highlighted
	}
	if req.EntryLabel != nil {
		product.EntryLabel = model.NormalizeShowcaseLabel(*req.EntryLabel)
	}
	if req.Audience != nil {
		product.Audience = model.NormalizeShowcaseLabel(*req.Audience)
	}
	if req.AddOnLabel != nil {
		product.AddOnLabel = model.NormalizeShowcaseLabel(*req.AddOnLabel)
	}
	if req.FeatureLines != nil {
		product.FeatureLines = model.NormalizeMembershipFeatureLines(*req.FeatureLines)
	}
	if strings.TrimSpace(product.Tier) == "" {
		product.Tier = model.MembershipSKUTier(product.SKU)
	}
	product.UpdatedBy = actor.ID
	product.UpdatedAt = time.Now()
	if err := s.repo.SaveMembershipProduct(product); err != nil {
		return nil, err
	}
	if req.SyncShowcaseToTier {
		if err := s.syncMembershipShowcaseToTier(actor.ID, product); err != nil {
			return nil, err
		}
	}
	if err := s.appendAdminAudit(actor, "membership_product.update", "membership_product", product.ID, "更新订阅商品", map[string]any{
		"sku": product.SKU, "amountFen": product.AmountFen, "creditsMicrocredits": product.CreditsMicrocredits,
		"storageQuotaBytes": product.StorageQuotaBytes, "enabled": product.Enabled, "syncShowcaseToTier": req.SyncShowcaseToTier,
	}); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *Service) syncMembershipShowcaseToTier(actorID string, source *model.MembershipProduct) error {
	if source == nil {
		return nil
	}
	tier := source.EffectiveTier()
	if tier == "" {
		return nil
	}
	products, err := s.repo.MembershipProducts(true)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, item := range products {
		if item.ID == source.ID || item.EffectiveTier() != tier {
			continue
		}
		item.EntryLabel = source.EntryLabel
		item.Audience = source.Audience
		item.AddOnLabel = source.AddOnLabel
		item.FeatureLines = append([]model.MembershipFeatureLine(nil), source.FeatureLines...)
		item.UpdatedBy = actorID
		item.UpdatedAt = now
		if err := s.repo.SaveMembershipProduct(&item); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) PublicMembership(userID string) (*MembershipPublicView, error) {
	row, err := s.repo.UserMembership(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if row == nil {
		row = &model.UserMembership{UserID: userID}
	}
	now := time.Now()
	bonus := row.StorageBonusBytes
	var bonusExpires *time.Time
	if summary, summaryErr := s.repo.StorageBonusSummary(userID, now); summaryErr == nil {
		if summary.GrantCount > 0 {
			bonus = summary.ActiveBytes
			bonusExpires = summary.EarliestExpiry
			if row.StorageBonusBytes != bonus {
				_ = s.repo.SyncStorageBonusCache(userID, now)
			}
		}
	}
	view := &MembershipPublicView{
		PermanentActive:       row.PermanentActive,
		AdvancedPlanSKU:       row.AdvancedPlanSKU,
		AdvancedExpiresAt:     row.AdvancedExpiresAt,
		StorageOverrideBytes:  row.StorageOverrideBytes,
		StorageBonusBytes:     bonus,
		StorageBonusExpiresAt: bonusExpires,
	}
	if row.AdvancedExpiresAt != nil && row.AdvancedExpiresAt.After(now) {
		view.AdvancedRemainingSeconds = int64(row.AdvancedExpiresAt.Sub(now).Seconds())
	} else {
		view.AdvancedPlanSKU = ""
	}
	occupied, err := s.repo.OccupiedMembershipPaymentOrderCount(userID)
	if err != nil {
		return nil, err
	}
	view.HasOpenMembershipOrder = occupied > 0
	if occupied > 0 {
		if order, orderErr := s.repo.OccupiedMembershipPaymentOrder(userID); orderErr == nil {
			view.OpenMembershipOrderID = order.ID
		}
	}
	view.Tier = model.ActiveMembershipTier(*row, now)
	view.CanPurchasePermanent = model.MembershipPurchaseBlockReason(*row, model.MembershipSKUPermanent, occupied, now) == ""
	view.CanPurchaseVip = model.MembershipPurchaseBlockReason(*row, model.MembershipSKUVipMonth, occupied, now) == ""
	view.CanPurchaseSvip = model.MembershipPurchaseBlockReason(*row, model.MembershipSKUSvipMonth, occupied, now) == ""
	view.CanPurchaseAdvanced = view.CanPurchaseVip
	if view.CanPurchaseVip {
		view.MinPurchasableAdvancedSKU = model.MembershipSKUVipMonth
	} else if view.CanPurchaseSvip {
		view.MinPurchasableAdvancedSKU = model.MembershipSKUSvipMonth
	}
	if showcase, showcaseErr := s.MembershipFreeShowcase(); showcaseErr == nil {
		view.FreeShowcase = showcase
	} else {
		view.FreeShowcase = model.DefaultFreeMembershipShowcase()
	}
	resolved, err := s.ResolveEffectiveStoredFileBytes(userID)
	if err != nil {
		return nil, err
	}
	view.EffectiveStoredFileBytes = resolved.Bytes
	view.QuotaSource = resolved.Source
	if policy, policyErr := s.RuntimePolicy(); policyErr == nil {
		view.DefaultStoredFileBytes = gigabytes(policy.Resource.StoredFileGB)
	}
	isAdmin := false
	if user, userErr := s.repo.User(userID); userErr == nil && user.Role == model.UserRoleAdmin {
		isAdmin = true
	}
	allowed, allowErr := s.PersonalStorageAllowed(userID, isAdmin)
	if allowErr != nil {
		return nil, allowErr
	}
	view.PersonalStorageAllowed = allowed
	personalEnabled, personalErr := s.usingPersonalResourceOSS(userID)
	if personalErr != nil {
		return nil, personalErr
	}
	view.PersonalBucketEnabled = personalEnabled
	if personalEnabled {
		view.StorageDisplay = model.StorageDisplayPersonal
	} else {
		view.StorageDisplay = model.StorageDisplayPlatform
	}
	methods, _ := s.CommerceMethods()
	view.OnlinePaymentEnabled = methods.OnlinePaymentEnabled
	view.RedeemEnabled = methods.RedeemEnabled
	contact, _ := s.SupportContact()
	view.SupportTicketURL = strings.TrimSpace(contact.TicketURL)
	if view.SupportTicketURL == "" {
		view.SupportQQ = strings.TrimSpace(contact.QQ)
	}
	if view.SupportTicketURL == "" && view.SupportQQ == "" {
		view.StorageExpansionMessage = "存储扩容暂未开放在线购买"
	}
	return view, nil
}

type ResolvedStorageQuota struct {
	Bytes  int64
	Source string
}

func applyStorageBonus(base, bonus int64) int64 {
	if base < 0 {
		base = 0
	}
	if bonus <= 0 {
		return base
	}
	if base >= model.MaxMembershipStorageB {
		return model.MaxMembershipStorageB
	}
	remain := model.MaxMembershipStorageB - base
	if bonus > remain {
		return model.MaxMembershipStorageB
	}
	return base + bonus
}

func (s *Service) ResolveEffectiveStoredFileBytes(userID string) (ResolvedStorageQuota, error) {
	policy, err := s.RuntimePolicy()
	if err != nil {
		return ResolvedStorageQuota{}, err
	}
	fallback := gigabytes(policy.Resource.StoredFileGB)
	row, err := s.repo.UserMembership(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ResolvedStorageQuota{}, err
	}
	now := time.Now()
	bonus := int64(0)
	if row != nil {
		bonus = row.StorageBonusBytes
	}
	if summary, summaryErr := s.repo.StorageBonusSummary(userID, now); summaryErr == nil && summary.GrantCount > 0 {
		bonus = summary.ActiveBytes
		if row != nil && row.StorageBonusBytes != bonus {
			_ = s.repo.SyncStorageBonusCache(userID, now)
		}
	}
	if row != nil && row.StorageOverrideBytes != nil {
		return ResolvedStorageQuota{Bytes: applyStorageBonus(*row.StorageOverrideBytes, bonus), Source: model.QuotaSourceOverride}, nil
	}
	if row != nil && row.AdvancedExpiresAt != nil && row.AdvancedExpiresAt.After(now) && row.PlanStorageQuotaBytes > 0 {
		return ResolvedStorageQuota{Bytes: applyStorageBonus(row.PlanStorageQuotaBytes, bonus), Source: model.QuotaSourcePlan}, nil
	}
	return ResolvedStorageQuota{Bytes: applyStorageBonus(fallback, bonus), Source: model.QuotaSourceGlobal}, nil
}

func (s *Service) PersonalStorageAllowed(userID string, isAdmin bool) (bool, error) {
	oss, err := s.readOSSSettingValue()
	if err != nil {
		return false, err
	}
	if !oss.AllowUserS3 {
		return false, nil
	}
	if isAdmin {
		return true, nil
	}
	if user, userErr := s.repo.User(userID); userErr == nil && user.Role == model.UserRoleAdmin {
		return true, nil
	}
	row, err := s.repo.UserMembership(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	return model.PersonalMembershipEligible(*row, time.Now()), nil
}

func (s *Service) readOSSSettingValue() (ossSettingValue, error) {
	_, value, err := s.readOSSSetting()
	return value, err
}

func (s *Service) AssertCanPurchaseMembership(userID, sku string, occupied int64) error {
	row, err := s.repo.UserMembership(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if row == nil {
		row = &model.UserMembership{UserID: userID}
	}
	return membershipPurchaseError(*row, sku, occupied, time.Now())
}

func membershipPurchaseError(row model.UserMembership, sku string, occupied int64, now time.Time) error {
	reason := model.MembershipPurchaseBlockReason(row, sku, occupied, now)
	switch reason {
	case "":
		return nil
	case model.MembershipPurchaseBlockUnknownSKU:
		return BadAuthRequest(reason)
	default:
		return FailedPrecondition(reason)
	}
}

func (s *Service) CreatePaymentOrder(ctx context.Context, actor *model.User, request CreatePaymentOrderRequest) (*PaymentOrderView, error) {
	kind := model.NormalizeProductKind(request.ProductKind)
	if kind == model.ProductKindMembership {
		return s.createMembershipPaymentOrder(ctx, actor, request)
	}
	return s.createTopupPaymentOrder(ctx, actor, request)
}

func (s *Service) createCreditTopupPaymentOrder(ctx context.Context, actor *model.User, request CreatePaymentOrderRequest) (*PaymentOrderView, error) {
	return s.createTopupPaymentOrder(ctx, actor, request)
}

func (s *Service) createTopupPaymentOrder(ctx context.Context, actor *model.User, request CreatePaymentOrderRequest) (*PaymentOrderView, error) {
	if actor == nil {
		return nil, Unauthorized("请先登录")
	}
	product, err := s.repo.TopupProduct(strings.TrimSpace(request.ProductID))
	if err != nil || !product.Enabled {
		return nil, BadAuthRequest("充值商品不存在或已停用")
	}
	kind := model.NormalizeTopupKind(product.Kind)
	if kind == model.ProductKindCreditTopup {
		if err := s.RequireFeature(FeatureCredits); err != nil {
			return nil, err
		}
	}
	if err := s.requireOnlinePayment(); err != nil {
		return nil, err
	}
	return s.createPaymentOrderForProduct(ctx, actor, request, product)
}

func (s *Service) createMembershipPaymentOrder(ctx context.Context, actor *model.User, request CreatePaymentOrderRequest) (*PaymentOrderView, error) {
	if actor == nil {
		return nil, Unauthorized("请先登录")
	}
	if err := s.requireOnlinePayment(); err != nil {
		return nil, err
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		return nil, BadAuthRequest("支付幂等标识不能为空")
	}
	if len(idempotencyKey) > 120 {
		return nil, BadAuthRequest("支付幂等标识过长")
	}
	productID := strings.TrimSpace(request.ProductID)
	providerID := strings.TrimSpace(request.ProviderID)
	product, err := s.repo.MembershipProduct(productID)
	if err != nil || !product.Enabled {
		return nil, BadAuthRequest("订阅商品不存在或已停用")
	}
	if product.AmountFen <= 0 {
		return nil, FailedPrecondition("订阅商品未定价")
	}
	provider, ok := s.paymentRegistry.Get(providerID)
	if !ok {
		return nil, BadAuthRequest("未知支付渠道")
	}
	view, config, err := s.paymentProviderView(provider.Descriptor())
	if err != nil {
		return nil, err
	}
	if !view.Enabled || !view.Configured || config == nil {
		return nil, Forbidden("支付渠道未启用或尚未配置")
	}
	now := time.Now()
	order := &model.PaymentOrder{
		ID: newID(), UserID: actor.ID, IdempotencyKey: idempotencyKey, MerchantOrderNo: newID(),
		ProductID: product.ID, ProductName: product.Name, ProviderID: provider.Descriptor().ID,
		PluginID: provider.Descriptor().PluginID, PluginVersion: provider.Descriptor().PluginVersion,
		ProviderConfigID: config.ID, ProviderConfigVersion: config.Version,
		AmountFen: product.AmountFen, Currency: "CNY", CreditsMicrocredits: product.CreditsMicrocredits,
		ProductKind: model.ProductKindMembership, PlanSKU: product.SKU, MembershipDurationDays: product.DurationDays,
		StorageQuotaBytes: product.StorageQuotaBytes, Status: model.PaymentOrderCreated, CheckoutMode: provider.Descriptor().CheckoutMode,
		ExpiresAt: now.Add(time.Duration(config.CloseAfterMinutes) * time.Minute),
	}
	order, created, err := s.repo.CreateMembershipPaymentOrder(order, func(tx *gorm.DB) error {
		occupied, err := repository.OccupiedMembershipCountInTx(tx, actor.ID)
		if err != nil {
			return err
		}
		return s.assertCanPurchaseMembershipTx(tx, actor.ID, product.SKU, occupied)
	})
	if errors.Is(err, repository.ErrPaymentOrderStateConflict) {
		return nil, NewAppError(http.StatusConflict, "支付幂等标识已用于不同的商品或支付渠道")
	}
	if err != nil {
		return nil, err
	}
	if !created {
		result := paymentOrderView(*order)
		return &result, nil
	}
	values, err := s.decryptPaymentConfig(config)
	if err != nil {
		s.abandonUncreatedPaymentOrder(order.ID, err)
		return nil, err
	}
	baseURL := strings.TrimRight(values["publicBaseUrl"], "/")
	checkout, err := provider.CreateOrder(ctx, values, payment.CreateRequest{
		MerchantOrderNo: order.MerchantOrderNo, Description: product.Name, AmountFen: order.AmountFen,
		Currency: order.Currency, ExpiresAt: order.ExpiresAt,
		NotifyURL: baseURL + "/api/payments/notify/" + url.PathEscape(order.ProviderID) + "/" + url.PathEscape(config.ID),
		ReturnURL: baseURL + "/api/payments/return/" + url.PathEscape(order.ProviderID) + "?orderId=" + url.QueryEscape(order.ID),
	})
	if err != nil {
		s.abandonUncreatedPaymentOrder(order.ID, err)
		return nil, WrapAppError(502, "支付渠道下单失败，请稍后重试", err)
	}
	if err := s.repo.SetPaymentOrderCheckout(order.ID, checkout.Mode, checkout.Value, checkout.ExpiresAt); err != nil {
		_ = s.closePaymentOrder(ctx, order)
		s.abandonUncreatedPaymentOrder(order.ID, err)
		return nil, err
	}
	fresh, err := s.repo.PaymentOrder(order.ID)
	if err != nil {
		return nil, err
	}
	result := paymentOrderView(*fresh)
	return &result, nil
}

func (s *Service) assertCanPurchaseMembershipTx(tx *gorm.DB, userID, sku string, occupied int64) error {
	row, err := repository.UserMembershipInTx(tx, userID)
	if err != nil {
		return err
	}
	return membershipPurchaseError(*row, sku, occupied, time.Now())
}

func (s *Service) UpdateUserStorageQuota(actor *model.User, userID string, override *int64) error {
	if err := s.RequireAdmin(actor); err != nil {
		return err
	}
	if override != nil && (*override < 0 || *override > model.MaxMembershipStorageB) {
		return BadAuthRequest("存储覆盖超出允许范围")
	}
	if err := s.repo.UpdateUserStorageOverride(userID, actor.ID, override); err != nil {
		return err
	}
	return s.appendAdminAudit(actor, "membership.storage_override", "user", userID, "更新用户存储覆盖", map[string]any{"storageOverrideBytes": override})
}

func (s *Service) AdminGrantMembership(actor *model.User, userID string, req AdminMembershipGrantRequest) (*MembershipPublicView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		return nil, BadAuthRequest("请填写幂等标识")
	}
	sku := strings.TrimSpace(req.SKU)
	if sku == "" {
		sku = strings.TrimSpace(req.PlanSKU)
	}
	product, err := s.repo.MembershipProductBySKU(sku)
	if err != nil {
		return nil, BadAuthRequest("未知订阅套餐")
	}
	snap := model.MembershipGrantSnapshot{
		ProductID: product.ID, PlanSKU: product.SKU, CreditsMicrocredits: product.CreditsMicrocredits,
		StorageQuotaBytes: product.StorageQuotaBytes, DurationDays: product.DurationDays,
		Source: model.MembershipGrantSourceAdmin, AdminIdempotencyKey: key, Note: truncateRunes(strings.TrimSpace(req.Note), 500), ActorUserID: actor.ID,
	}
	if err := s.repo.AdminGrantMembership(userID, snap, nil); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			return s.PublicMembership(userID)
		}
		return nil, err
	}
	if err := s.appendAdminAudit(actor, "membership.grant", "user", userID, "赠送订阅", map[string]any{"sku": sku}); err != nil {
		return nil, err
	}
	return s.PublicMembership(userID)
}

func (s *Service) AssertUploadFitsAccountQuota(userID string, size int64) error {
	storedLimit, skipStored, err := s.platformStoredLimitForUpload(userID)
	if err != nil {
		return err
	}
	if skipStored {
		return nil
	}
	if storedLimit > 0 && size > storedLimit {
		return QuotaExceeded(fmt.Sprintf("文件超过账号存储总量上限 %s", formatStorageLimit(storedLimit)))
	}
	return nil
}
