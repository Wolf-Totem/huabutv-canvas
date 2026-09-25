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
	PermanentActive           bool       `json:"permanentActive"`
	AdvancedPlanSKU           string     `json:"advancedPlanSku"`
	AdvancedExpiresAt         *time.Time `json:"advancedExpiresAt,omitempty"`
	AdvancedRemainingSeconds  int64      `json:"advancedRemainingSeconds"`
	CanPurchaseAdvanced       bool       `json:"canPurchaseAdvanced"`
	CanPurchasePermanent      bool       `json:"canPurchasePermanent"`
	HasOpenMembershipOrder    bool       `json:"hasOpenMembershipOrder"`
	OpenMembershipOrderID     string     `json:"openMembershipOrderId,omitempty"`
	PersonalStorageAllowed      bool       `json:"personalStorageAllowed"`
	EffectiveStoredFileBytes    int64      `json:"effectiveStoredFileBytes"`
	QuotaSource                 string     `json:"quotaSource"`
	StorageOverrideBytes        *int64     `json:"storageOverrideBytes"`
	StorageBonusBytes           int64      `json:"storageBonusBytes"`
	StorageExpansionMessage     string     `json:"storageExpansionMessage"`
	SupportTicketURL            string     `json:"supportTicketUrl,omitempty"`
	SupportQQ                   string     `json:"supportQq,omitempty"`
	MinPurchasableAdvancedSKU   string     `json:"minPurchasableAdvancedSku,omitempty"`
	OnlinePaymentEnabled        bool       `json:"onlinePaymentEnabled"`
	RedeemEnabled               bool       `json:"redeemEnabled"`
}

type MembershipProductView struct {
	model.MembershipProduct
}

type UpdateMembershipProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AmountFen   int64  `json:"amountFen"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sortOrder"`
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
	Account    *model.CreditAccount   `json:"account"`
	Membership *MembershipPublicView  `json:"membership,omitempty"`
	Granted    *RedeemGrantedView     `json:"granted,omitempty"`
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

func (s *Service) PublicMembershipProducts() ([]model.MembershipProduct, error) {
	return s.repo.MembershipProducts(false)
}

func (s *Service) AdminMembershipProducts(actor *model.User) ([]model.MembershipProduct, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.repo.MembershipProducts(true)
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
	product.UpdatedBy = actor.ID
	product.UpdatedAt = time.Now()
	if err := s.repo.SaveMembershipProduct(product); err != nil {
		return nil, err
	}
	if err := s.appendAdminAudit(actor, "membership_product.update", "membership_product", product.ID, "更新订阅商品", map[string]any{"sku": product.SKU, "amountFen": product.AmountFen, "enabled": product.Enabled}); err != nil {
		return nil, err
	}
	return product, nil
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
	view := &MembershipPublicView{
		PermanentActive:      row.PermanentActive,
		AdvancedPlanSKU:      row.AdvancedPlanSKU,
		AdvancedExpiresAt:    row.AdvancedExpiresAt,
		StorageOverrideBytes: row.StorageOverrideBytes,
		StorageBonusBytes:    row.StorageBonusBytes,
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
	view.CanPurchasePermanent = !row.PermanentActive && !view.HasOpenMembershipOrder
	view.CanPurchaseAdvanced = !view.HasOpenMembershipOrder && (view.AdvancedRemainingSeconds <= 30*24*3600)
	if view.CanPurchaseAdvanced {
		view.MinPurchasableAdvancedSKU = model.MembershipSKUAdvancedMonth
	}
	resolved, err := s.ResolveEffectiveStoredFileBytes(userID)
	if err != nil {
		return nil, err
	}
	view.EffectiveStoredFileBytes = resolved.Bytes
	view.QuotaSource = resolved.Source
	isAdmin := false
	if user, userErr := s.repo.User(userID); userErr == nil && user.Role == model.UserRoleAdmin {
		isAdmin = true
	}
	allowed, allowErr := s.PersonalStorageAllowed(userID, isAdmin)
	if allowErr != nil {
		return nil, allowErr
	}
	view.PersonalStorageAllowed = allowed
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
	bonus := int64(0)
	if row != nil {
		bonus = row.StorageBonusBytes
	}
	if row != nil && row.StorageOverrideBytes != nil {
		return ResolvedStorageQuota{Bytes: applyStorageBonus(*row.StorageOverrideBytes, bonus), Source: model.QuotaSourceOverride}, nil
	}
	now := time.Now()
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
	return row != nil && row.PermanentActive, nil
}

func (s *Service) readOSSSettingValue() (ossSettingValue, error) {
	_, value, err := s.readOSSSetting()
	return value, err
}

func (s *Service) AssertCanPurchaseMembership(userID, sku string, occupied int64) error {
	if !model.IsMembershipSKU(sku) {
		return BadAuthRequest("未知订阅套餐")
	}
	if occupied > 0 {
		return FailedPrecondition("你有一笔未完成的订阅订单")
	}
	row, err := s.repo.UserMembership(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if row == nil {
		row = &model.UserMembership{UserID: userID}
	}
	if sku == model.MembershipSKUPermanent {
		if row.PermanentActive {
			return FailedPrecondition("已拥有永久订阅")
		}
		return nil
	}
	now := time.Now()
	if row.AdvancedExpiresAt != nil && row.AdvancedExpiresAt.After(now.Add(30*24*time.Hour)) {
		return FailedPrecondition("高级订阅剩余超过 30 天，暂不能购买新套餐")
	}
	return nil
}

func (s *Service) CreatePaymentOrder(ctx context.Context, actor *model.User, request CreatePaymentOrderRequest) (*PaymentOrderView, error) {
	kind := model.NormalizeProductKind(request.ProductKind)
	if kind == model.ProductKindMembership {
		return s.createMembershipPaymentOrder(ctx, actor, request)
	}
	return s.createCreditTopupPaymentOrder(ctx, actor, request)
}

func (s *Service) createCreditTopupPaymentOrder(ctx context.Context, actor *model.User, request CreatePaymentOrderRequest) (*PaymentOrderView, error) {
	if actor == nil {
		return nil, Unauthorized("请先登录")
	}
	if err := s.RequireFeature(FeatureCredits); err != nil {
		return nil, err
	}
	if err := s.requireOnlinePayment(); err != nil {
		return nil, err
	}
	return s.createPaymentOrderForProduct(ctx, actor, request, nil)
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
		_ = s.repo.SetPaymentOrderCreateFailure(order.ID, safePaymentError(err))
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
		_ = s.repo.SetPaymentOrderCreateFailure(order.ID, safePaymentError(err))
		return nil, WrapAppError(502, "支付渠道下单失败，请稍后重试", err)
	}
	if err := s.repo.SetPaymentOrderCheckout(order.ID, checkout.Mode, checkout.Value, checkout.ExpiresAt); err != nil {
		_ = s.repo.SetPaymentOrderCreateFailure(order.ID, safePaymentError(err))
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
	if occupied > 0 {
		return FailedPrecondition("你有一笔未完成的订阅订单")
	}
	row, err := repository.UserMembershipInTx(tx, userID)
	if err != nil {
		return err
	}
	if sku == model.MembershipSKUPermanent && row.PermanentActive {
		return FailedPrecondition("已拥有永久订阅")
	}
	if model.IsAdvancedMembershipSKU(sku) && row.AdvancedExpiresAt != nil && row.AdvancedExpiresAt.After(time.Now().Add(30*24*time.Hour)) {
		return FailedPrecondition("高级订阅剩余超过 30 天，暂不能购买新套餐")
	}
	return nil
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


