package paymentplugins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	huifuJspayPath              = "/v3/trade/payment/jspay"
	huifuScanpayQueryPath       = "/v3/trade/payment/scanpay/query"
	huifuScanpayClosePath       = "/v2/trade/payment/scanpay/close"
	huifuTradeTypeAlipayNative  = "A_NATIVE"
	huifuTradeTypeWechatMiniapp = "T_MINIAPP"
	huifuProcessingCode         = "00000100"
	huifuJspayDefaultExpire     = 2 * time.Hour
	huifuJspayGoodsDescLimit    = 127
	huifuJspaySeqLimit          = 128
)

type huifuJspayChannel struct {
	tradeType    string
	name         string
	providerID   string
	pluginID     string
	checkoutMode string
}

var (
	huifuAlipayNativeChannel = huifuJspayChannel{
		tradeType:    huifuTradeTypeAlipayNative,
		name:         "斗拱支付宝正扫",
		providerID:   ProviderHuifuJspay,
		pluginID:     PluginHuifuJspay,
		checkoutMode: "qr_code",
	}
	huifuWechatMiniappChannel = huifuJspayChannel{
		tradeType:    huifuTradeTypeWechatMiniapp,
		name:         "斗拱微信小程序",
		providerID:   ProviderHuifuWechatJspay,
		pluginID:     PluginHuifuWechatJspay,
		checkoutMode: "jsapi",
	}
)

type HuifuJspayProvider struct {
	client  *http.Client
	now     func() time.Time
	baseURL string
	channel huifuJspayChannel
}

func NewHuifuJspayProvider(client *http.Client) *HuifuJspayProvider {
	return newHuifuJspayProvider(client, huifuAlipayNativeChannel)
}

func NewHuifuWechatJspayProvider(client *http.Client) *HuifuJspayProvider {
	return newHuifuJspayProvider(client, huifuWechatMiniappChannel)
}

func newHuifuJspayProvider(client *http.Client, channel huifuJspayChannel) *HuifuJspayProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &HuifuJspayProvider{client: client, now: time.Now, channel: channel}
}

func (p *HuifuJspayProvider) Descriptor() Descriptor {
	channel := p.channel
	if channel.providerID == "" {
		channel = huifuAlipayNativeChannel
	}
	checkoutMode := channel.checkoutMode
	if checkoutMode == "" {
		checkoutMode = "qr_code"
	}
	return Descriptor{
		ID: channel.providerID, PluginID: channel.pluginID, PluginVersion: "1.0.0",
		Name: channel.name, Icon: "assets/icon.svg", CheckoutMode: checkoutMode,
		IdentityFields: []string{"sysId", "huifuId"},
		NotificationSuccess: NotificationResponse{
			Status: 200, ContentType: "text/plain; charset=utf-8", Body: "RECV_ORD_ID_",
		},
		NotificationFailure: NotificationResponse{
			Status: 400, ContentType: "text/plain; charset=utf-8", Body: "fail",
		},
	}
}

func (p *HuifuJspayProvider) ValidateConfig(config Config) error {
	for _, key := range []string{"sysId", "productId", "huifuId", "merchantPrivateKey", "huifuPublicKey", "gateway"} {
		if strings.TrimSpace(config[key]) == "" {
			return fmt.Errorf("斗拱聚合正扫配置缺少 %s", key)
		}
	}
	if _, err := parseRSAPrivateKey(config["merchantPrivateKey"]); err != nil {
		return fmt.Errorf("斗拱商户私钥无效：%w", err)
	}
	if _, err := parseRSAPublicKey(config["huifuPublicKey"]); err != nil {
		return fmt.Errorf("斗拱汇付公钥无效：%w", err)
	}
	if _, err := huifuGatewayOrigin(config); err != nil {
		return err
	}
	if p.tradeType() == huifuTradeTypeWechatMiniapp && strings.TrimSpace(config["subAppId"]) == "" {
		return errors.New("斗拱微信小程序配置缺少 subAppId")
	}
	return nil
}

func (p *HuifuJspayProvider) CreateOrder(ctx context.Context, config Config, request CreateRequest) (Checkout, error) {
	if err := p.ValidateConfig(config); err != nil {
		return Checkout{}, err
	}
	if request.AmountFen <= 0 || request.Currency != "CNY" || request.MerchantOrderNo == "" || request.NotifyURL == "" {
		return Checkout{}, errors.New("斗拱聚合正扫下单参数无效")
	}
	if err := huifuValidateNotifyURL(request.NotifyURL); err != nil {
		return Checkout{}, err
	}
	if p.tradeType() == huifuTradeTypeWechatMiniapp {
		return p.createWechatMiniappOrder(ctx, config, request)
	}
	return p.createNativeQrOrder(ctx, config, request)
}

func (p *HuifuJspayProvider) createNativeQrOrder(ctx context.Context, config Config, request CreateRequest) (Checkout, error) {
	now := p.now().In(huifuLocation())
	goodsDesc := huifuTruncate(request.Description, huifuJspayGoodsDescLimit)
	if goodsDesc == "" {
		goodsDesc = "钱包支付"
	}
	data := map[string]string{
		"req_date":   now.Format("20060102"),
		"req_seq_id": huifuTruncate(request.MerchantOrderNo, huifuJspaySeqLimit),
		"huifu_id":   strings.TrimSpace(config["huifuId"]),
		"trans_amt":  formatFen(request.AmountFen),
		"goods_desc": goodsDesc,
		"trade_type": p.tradeType(),
		"notify_url": request.NotifyURL,
	}
	if !request.ExpiresAt.IsZero() {
		data["time_expire"] = request.ExpiresAt.In(huifuLocation()).Format("20060102150405")
	}
	var envelope huifuEnvelope
	if err := p.call(ctx, config, huifuJspayPath, data, &envelope); err != nil {
		return Checkout{}, err
	}
	if envelope.Data.RespCode != huifuSuccessCode && envelope.Data.RespCode != huifuProcessingCode {
		return Checkout{}, huifuBusinessError(envelope.Data.RespCode, envelope.Data.RespDesc)
	}
	qrCode := strings.TrimSpace(envelope.Data.QrCode)
	if qrCode == "" {
		return Checkout{}, errors.New("斗拱聚合正扫未返回 qr_code")
	}
	if !p.validQrCode(qrCode) {
		return Checkout{}, errors.New("斗拱聚合正扫 qr_code 不是有效 http(s) 地址")
	}
	return Checkout{Mode: "qr_code", Value: qrCode, ExpiresAt: huifuJspayExpiresAt(now, envelope.Data.TimeExpire, request.ExpiresAt)}, nil
}

func (p *HuifuJspayProvider) createWechatMiniappOrder(ctx context.Context, config Config, request CreateRequest) (Checkout, error) {
	subOpenID := strings.TrimSpace(request.WeChatSubOpenID)
	if subOpenID == "" {
		return Checkout{}, errors.New("斗拱微信小程序下单缺少 sub_openid")
	}
	wxData, err := huifuJSON(map[string]string{
		"sub_appid":  strings.TrimSpace(config["subAppId"]),
		"sub_openid": subOpenID,
	})
	if err != nil {
		return Checkout{}, err
	}
	now := p.now().In(huifuLocation())
	goodsDesc := huifuTruncate(request.Description, huifuJspayGoodsDescLimit)
	if goodsDesc == "" {
		goodsDesc = "钱包支付"
	}
	data := map[string]string{
		"req_date":   now.Format("20060102"),
		"req_seq_id": huifuTruncate(request.MerchantOrderNo, huifuJspaySeqLimit),
		"huifu_id":   strings.TrimSpace(config["huifuId"]),
		"trans_amt":  formatFen(request.AmountFen),
		"goods_desc": goodsDesc,
		"trade_type": huifuTradeTypeWechatMiniapp,
		"wx_data":    string(wxData),
		"notify_url": request.NotifyURL,
	}
	if !request.ExpiresAt.IsZero() {
		data["time_expire"] = request.ExpiresAt.In(huifuLocation()).Format("20060102150405")
	}
	var envelope huifuEnvelope
	if err := p.call(ctx, config, huifuJspayPath, data, &envelope); err != nil {
		return Checkout{}, err
	}
	if envelope.Data.RespCode != huifuSuccessCode && envelope.Data.RespCode != huifuProcessingCode {
		return Checkout{}, huifuBusinessError(envelope.Data.RespCode, envelope.Data.RespDesc)
	}
	payInfo, err := huifuPayInfoValue(envelope.Data.PayInfo)
	if err != nil {
		return Checkout{}, err
	}
	return Checkout{Mode: "jsapi", Value: payInfo, ExpiresAt: huifuJspayExpiresAt(now, envelope.Data.TimeExpire, request.ExpiresAt)}, nil
}

func huifuJspayExpiresAt(now time.Time, timeExpire string, requestExpiresAt time.Time) time.Time {
	expiresAt := now.Add(huifuJspayDefaultExpire)
	if parsed, err := time.ParseInLocation("20060102150405", strings.TrimSpace(timeExpire), huifuLocation()); err == nil {
		return parsed
	}
	if !requestExpiresAt.IsZero() {
		return requestExpiresAt
	}
	return expiresAt
}

func (p *HuifuJspayProvider) QueryOrder(ctx context.Context, config Config, request QueryRequest) (Result, error) {
	if err := p.ValidateConfig(config); err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(request.MerchantOrderNo) == "" {
		return Result{}, errors.New("斗拱聚合正扫查单缺少商户订单号")
	}
	order, err := p.queryOriginal(ctx, config, request.MerchantOrderNo)
	if err != nil {
		return Result{}, err
	}
	return huifuResult(request.MerchantOrderNo, order), nil
}

func (p *HuifuJspayProvider) CloseOrder(ctx context.Context, config Config, request CloseRequest) (Result, error) {
	if err := p.ValidateConfig(config); err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(request.MerchantOrderNo) == "" {
		return Result{}, errors.New("斗拱聚合正扫关单缺少商户订单号")
	}
	order, err := p.queryOriginal(ctx, config, request.MerchantOrderNo)
	if err != nil {
		return Result{}, err
	}
	result := huifuResult(request.MerchantOrderNo, order)
	if result.Paid {
		return result, nil
	}
	now := p.now().In(huifuLocation())
	orgDate := strings.TrimSpace(order.OrgReqDate)
	if orgDate == "" {
		orgDate = now.Format("20060102")
	}
	data := map[string]string{
		"req_seq_id":     huifuNewSeq("C", request.MerchantOrderNo, now),
		"req_date":       now.Format("20060102"),
		"huifu_id":       strings.TrimSpace(config["huifuId"]),
		"org_req_date":   orgDate,
		"org_req_seq_id": huifuTruncate(request.MerchantOrderNo, huifuJspaySeqLimit),
	}
	var envelope huifuEnvelope
	if err := p.call(ctx, config, huifuScanpayClosePath, data, &envelope); err != nil {
		return Result{}, err
	}
	if envelope.Data.RespCode != huifuSuccessCode && envelope.Data.RespCode != huifuProcessingCode {
		if envelope.Data.RespCode == "23000000" || envelope.Data.RespCode == "10000016" {
			return p.QueryOrder(ctx, config, QueryRequest{MerchantOrderNo: request.MerchantOrderNo})
		}
		return Result{}, huifuBusinessError(envelope.Data.RespCode, envelope.Data.RespDesc)
	}
	if envelope.Data.OrgTransStat == huifuTransSuccess {
		result.Paid = true
		result.Closed = false
		result.ProviderStatus = envelope.Data.OrgTransStat
		if tradeNo := firstNonEmpty(envelope.Data.OrgHfSeqID, result.ProviderTradeNo); tradeNo != "" {
			result.ProviderTradeNo = tradeNo
		}
		return result, nil
	}
	// 扫码关单的 trans_stat 是关单状态，不是支付状态。
	result.ProviderStatus = firstNonEmpty(envelope.Data.TransStat, envelope.Data.OrgTransStat, result.ProviderStatus)
	result.Closed = envelope.Data.TransStat == huifuCloseSuccess
	result.Paid = false
	return result, nil
}

func (p *HuifuJspayProvider) VerifyNotification(_ context.Context, config Config, _ http.Header, rawBody []byte) (Notification, error) {
	if err := p.ValidateConfig(config); err != nil {
		return Notification{}, err
	}
	return huifuVerifyNotification(config, rawBody)
}

func (p *HuifuJspayProvider) DownloadTradeBill(context.Context, Config, time.Time) ([]BillRecord, error) {
	return nil, fmt.Errorf("%w: 聚合正扫接口未提供交易账单下载", ErrTradeBillNotFound)
}

func (p *HuifuJspayProvider) queryOriginal(ctx context.Context, config Config, merchantOrderNo string) (huifuData, error) {
	now := p.now().In(huifuLocation())
	dates := []string{now.Format("20060102"), now.AddDate(0, 0, -1).Format("20060102")}
	var last error
	for _, orgDate := range dates {
		data := map[string]string{
			"huifu_id":       strings.TrimSpace(config["huifuId"]),
			"org_req_date":   orgDate,
			"org_req_seq_id": huifuTruncate(merchantOrderNo, huifuJspaySeqLimit),
		}
		var envelope huifuEnvelope
		if err := p.call(ctx, config, huifuScanpayQueryPath, data, &envelope); err != nil {
			if errors.Is(err, ErrOrderNotFound) {
				last = err
				continue
			}
			return huifuData{}, err
		}
		if envelope.Data.RespCode == "23000001" || envelope.Data.RespCode == "20000004" || envelope.Data.RespCode == "99010003" {
			last = fmt.Errorf("%w: %s", ErrOrderNotFound, envelope.Data.RespDesc)
			continue
		}
		if envelope.Data.RespCode != huifuSuccessCode && envelope.Data.RespCode != huifuProcessingCode {
			return huifuData{}, huifuBusinessError(envelope.Data.RespCode, envelope.Data.RespDesc)
		}
		if envelope.Data.OrgReqDate == "" {
			envelope.Data.OrgReqDate = orgDate
		}
		return envelope.Data, nil
	}
	if last == nil {
		last = ErrOrderNotFound
	}
	return huifuData{}, last
}

func (p *HuifuJspayProvider) call(ctx context.Context, config Config, path string, data map[string]string, output *huifuEnvelope) error {
	origin, err := p.origin(config)
	if err != nil {
		return err
	}
	return huifuCall(ctx, p.client, origin, config, path, data, output)
}

func (p *HuifuJspayProvider) origin(config Config) (string, error) {
	if strings.TrimSpace(p.baseURL) != "" {
		return strings.TrimRight(p.baseURL, "/"), nil
	}
	return huifuGatewayOrigin(config)
}

func (p *HuifuJspayProvider) tradeType() string {
	if p != nil && strings.TrimSpace(p.channel.tradeType) != "" {
		return p.channel.tradeType
	}
	return huifuTradeTypeAlipayNative
}

func (p *HuifuJspayProvider) validQrCode(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" || parsed.Scheme == "http"
}
