package paymentplugins

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testWechatSubAppID = "wx0a3601d75ac00022"
const testWechatSubOpenID = "o-6yZ6Nc8Ahlx6qMQ85fH8gS6T3c"

func testWechatJspayConfig(t *testing.T) Config {
	t.Helper()
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	config["subAppId"] = testWechatSubAppID
	return config
}

func TestHuifuWechatJspayValidateConfigRequiresSubAppId(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	provider := NewHuifuWechatJspayProvider(http.DefaultClient)
	if err := provider.ValidateConfig(config); err == nil || !strings.Contains(err.Error(), "subAppId") {
		t.Fatalf("err = %v", err)
	}
}

func TestHuifuWechatJspayCreateOrderPostsTMiniappWithWxData(t *testing.T) {
	config := testWechatJspayConfig(t)
	fixedNow := time.Date(2026, 9, 27, 15, 4, 5, 0, huifuLocation())
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != huifuJspayPath {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatal(err)
		}
		data := captured["data"].(map[string]any)
		if data["trade_type"] != huifuTradeTypeWechatMiniapp || data["req_seq_id"] != "order-wx-1" || data["trans_amt"] != "1.00" {
			t.Fatalf("data = %#v", data)
		}
		wxRaw, _ := data["wx_data"].(string)
		var wxData map[string]any
		if err := json.Unmarshal([]byte(wxRaw), &wxData); err != nil {
			t.Fatalf("wx_data = %v", data["wx_data"])
		}
		if wxData["sub_appid"] != testWechatSubAppID || wxData["sub_openid"] != testWechatSubOpenID {
			t.Fatalf("wx_data = %#v", wxData)
		}
		if _, exists := data["alipay_data"]; exists {
			t.Fatalf("alipay_data must not be sent: %#v", data)
		}
		if _, exists := data["hosting_data"]; exists {
			t.Fatalf("hosting_data must not be sent: %#v", data)
		}
		if captured["sign"] != huifuTestSign(t, config, data) {
			t.Fatalf("sign mismatch")
		}
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000000", "resp_desc": "交易成功",
			"pay_info":   testWechatPayInfoObject(),
			"trans_stat": "P", "hf_seq_id": "HFSEQWX1",
		})
	}))
	defer server.Close()

	provider := NewHuifuWechatJspayProvider(server.Client())
	provider.baseURL = server.URL
	provider.now = func() time.Time { return fixedNow }
	checkout, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-wx-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL:       "https://merchant.example/api/payments/notify/huifu-wechat-native/cfg",
		WeChatSubOpenID: testWechatSubOpenID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Mode != "jsapi" {
		t.Fatalf("checkout = %#v", checkout)
	}
	assertWechatPayInfo(t, checkout.Value)
}

func TestHuifuWechatJspayCreateOrderAcceptsPayInfoString(t *testing.T) {
	config := testWechatJspayConfig(t)
	payInfo, err := json.Marshal(testWechatPayInfoObject())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000100", "resp_desc": "交易处理中",
			"pay_info": string(payInfo), "trans_stat": "P",
		})
	}))
	defer server.Close()
	provider := NewHuifuWechatJspayProvider(server.Client())
	provider.baseURL = server.URL
	checkout, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-wx-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL:       "https://merchant.example/api/payments/notify/huifu-wechat-native/cfg",
		WeChatSubOpenID: testWechatSubOpenID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Mode != "jsapi" || checkout.Value != string(payInfo) {
		t.Fatalf("checkout = %#v", checkout)
	}
}

func TestHuifuWechatJspayCreateOrderRejectsMissingOpenID(t *testing.T) {
	config := testWechatJspayConfig(t)
	provider := NewHuifuWechatJspayProvider(http.DefaultClient)
	if _, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-wx-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL: "https://merchant.example/api/payments/notify/huifu-wechat-native/cfg",
	}); err == nil || !strings.Contains(err.Error(), "sub_openid") {
		t.Fatalf("err = %v", err)
	}
}

func TestHuifuWechatJspayCreateOrderRejectsEmptyPayInfo(t *testing.T) {
	config := testWechatJspayConfig(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000100", "resp_desc": "交易处理中", "trans_stat": "P",
		})
	}))
	defer server.Close()
	provider := NewHuifuWechatJspayProvider(server.Client())
	provider.baseURL = server.URL
	if _, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-wx-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL:       "https://merchant.example/api/payments/notify/huifu-wechat-native/cfg",
		WeChatSubOpenID: testWechatSubOpenID,
	}); err == nil || !strings.Contains(err.Error(), "pay_info") {
		t.Fatalf("err = %v", err)
	}
}

func TestHuifuWechatJspayDescriptorIsWechatMiniappJsapi(t *testing.T) {
	provider := NewHuifuWechatJspayProvider(http.DefaultClient)
	desc := provider.Descriptor()
	if desc.ID != ProviderHuifuWechatJspay || desc.PluginID != PluginHuifuWechatJspay || desc.CheckoutMode != "jsapi" {
		t.Fatalf("descriptor = %#v", desc)
	}
	if desc.Name != "斗拱微信小程序" {
		t.Fatalf("name = %s", desc.Name)
	}
}

func testWechatPayInfoObject() map[string]any {
	return map[string]any{
		"appId":     testWechatSubAppID,
		"timeStamp": "1695800000",
		"nonceStr":  "abc",
		"package":   "prepay_id=wx123",
		"signType":  "RSA",
		"paySign":   "SIG",
	}
}

func assertWechatPayInfo(t *testing.T, value string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		t.Fatalf("pay_info %q: %v", value, err)
	}
	if payload["appId"] != testWechatSubAppID || payload["package"] != "prepay_id=wx123" || payload["paySign"] != "SIG" {
		t.Fatalf("pay_info = %#v", payload)
	}
}
