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

func TestHuifuWechatJspayCreateOrderPostsTNativeWithoutWxData(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
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
		if data["trade_type"] != huifuTradeTypeWechatNative || data["req_seq_id"] != "order-wx-1" || data["trans_amt"] != "1.00" {
			t.Fatalf("data = %#v", data)
		}
		if _, exists := data["wx_data"]; exists {
			t.Fatalf("wx_data must not be sent: %#v", data)
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
			"qr_code":    "weixin://wxpay/bizpayurl?pr=i8SfEeFzz",
			"trans_stat": "P", "hf_seq_id": "HFSEQWX1",
		})
	}))
	defer server.Close()

	provider := NewHuifuWechatJspayProvider(server.Client())
	provider.baseURL = server.URL
	provider.now = func() time.Time { return fixedNow }
	checkout, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-wx-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL: "https://merchant.example/api/payments/notify/huifu-wechat-native/cfg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Mode != "qr_code" || checkout.Value != "weixin://wxpay/bizpayurl?pr=i8SfEeFzz" {
		t.Fatalf("checkout = %#v", checkout)
	}
}

func TestHuifuWechatJspayCreateOrderAcceptsHTTPSQrCode(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000100", "resp_desc": "交易处理中",
			"qr_code": "https://wx.tenpay.com/cgi-bin/mmpayweb-bin/checkmweb?prepay_id=wx1", "trans_stat": "P",
		})
	}))
	defer server.Close()
	provider := NewHuifuWechatJspayProvider(server.Client())
	provider.baseURL = server.URL
	checkout, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-wx-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL: "https://merchant.example/api/payments/notify/huifu-wechat-native/cfg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Mode != "qr_code" || !strings.HasPrefix(checkout.Value, "https://wx.tenpay.com/") {
		t.Fatalf("checkout = %#v", checkout)
	}
}

func TestHuifuWechatJspayCreateOrderRejectsEmptyQrCode(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
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
		NotifyURL: "https://merchant.example/api/payments/notify/huifu-wechat-native/cfg",
	}); err == nil || !strings.Contains(err.Error(), "qr_code") {
		t.Fatalf("err = %v", err)
	}
}

func TestHuifuWechatJspayDescriptorIsWechatNativeQrCode(t *testing.T) {
	provider := NewHuifuWechatJspayProvider(http.DefaultClient)
	desc := provider.Descriptor()
	if desc.ID != ProviderHuifuWechatJspay || desc.PluginID != PluginHuifuWechatJspay || desc.CheckoutMode != "qr_code" {
		t.Fatalf("descriptor = %#v", desc)
	}
	if desc.Name != "斗拱微信正扫" {
		t.Fatalf("name = %s", desc.Name)
	}
}
