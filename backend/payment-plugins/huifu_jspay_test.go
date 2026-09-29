package paymentplugins

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHuifuJspayCreateOrderPostsAlipayNativeWithoutAlipayData(t *testing.T) {
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
		if captured["sys_id"] != "sys-1" || captured["product_id"] != "YYZY" {
			t.Fatalf("envelope = %#v", captured)
		}
		if data["trade_type"] != huifuTradeTypeAlipayNative || data["req_seq_id"] != "order-1" || data["trans_amt"] != "1.00" || data["huifu_id"] != "6666000109133323" {
			t.Fatalf("data = %#v", data)
		}
		if data["req_date"] != "20260927" || data["time_expire"] != "20260927153405" {
			t.Fatalf("time fields = %#v", data)
		}
		if data["notify_url"] != "https://merchant.example/api/payments/notify/huifu-aggregate-native/cfg" {
			t.Fatalf("notify_url = %v", data["notify_url"])
		}
		if _, exists := data["alipay_data"]; exists {
			t.Fatalf("alipay_data must not be sent: %#v", data)
		}
		if _, exists := data["wx_data"]; exists {
			t.Fatalf("wx_data must not be sent: %#v", data)
		}
		if _, exists := data["hosting_data"]; exists {
			t.Fatalf("hosting_data must not be sent: %#v", data)
		}
		if captured["sign"] != huifuTestSign(t, config, data) {
			t.Fatalf("sign mismatch")
		}
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000000", "resp_desc": "交易成功",
			"qr_code":    "https://qr.alipay.com/bax03232ftw69valbwmg000d",
			"trans_stat": "P", "hf_seq_id": "HFSEQ1",
		})
	}))
	defer server.Close()

	provider := NewHuifuJspayProvider(server.Client())
	provider.baseURL = server.URL
	provider.now = func() time.Time { return fixedNow }
	checkout, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		ExpiresAt: fixedNow.Add(30 * time.Minute),
		NotifyURL: "https://merchant.example/api/payments/notify/huifu-aggregate-native/cfg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Mode != "qr_code" || checkout.Value != "https://qr.alipay.com/bax03232ftw69valbwmg000d" {
		t.Fatalf("checkout = %#v", checkout)
	}
}

func TestHuifuJspayCreateOrderAcceptsProcessingWhenQrCodePresent(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000100", "resp_desc": "交易处理中",
			"qr_code": "https://qr.alipay.com/bax-processing", "trans_stat": "P",
		})
	}))
	defer server.Close()
	provider := NewHuifuJspayProvider(server.Client())
	provider.baseURL = server.URL
	checkout, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL: "https://merchant.example/api/payments/notify/huifu-aggregate-native/cfg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Mode != "qr_code" || checkout.Value != "https://qr.alipay.com/bax-processing" {
		t.Fatalf("checkout = %#v", checkout)
	}
}

func TestHuifuJspayCreateOrderRejectsProcessingWithoutQrCode(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000100", "resp_desc": "交易处理中", "trans_stat": "P",
		})
	}))
	defer server.Close()
	provider := NewHuifuJspayProvider(server.Client())
	provider.baseURL = server.URL
	if _, err := provider.CreateOrder(context.Background(), config, CreateRequest{
		MerchantOrderNo: "order-1", Description: "100 积分", AmountFen: 100, Currency: "CNY",
		NotifyURL: "https://merchant.example/api/payments/notify/huifu-aggregate-native/cfg",
	}); err == nil || !strings.Contains(err.Error(), "qr_code") {
		t.Fatalf("err = %v", err)
	}
}

func TestHuifuJspayQueryOrderMapsTransStatSuccess(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != huifuScanpayQueryPath {
			t.Fatalf("path = %s", request.URL.Path)
		}
		body, _ := io.ReadAll(request.Body)
		var envelope map[string]any
		_ = json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["org_req_seq_id"] != "order-1" || data["huifu_id"] != "6666000109133323" {
			t.Fatalf("query data = %#v", data)
		}
		if _, exists := data["req_seq_id"]; exists {
			t.Fatalf("scanpay query must not send req_seq_id: %#v", data)
		}
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000000", "resp_desc": "操作成功",
			"org_req_seq_id": "order-1", "org_hf_seq_id": "HFSEQ1",
			"trans_stat": "S", "trans_amt": "1.00", "end_time": "20260927153000",
			"org_req_date": "20260927",
		})
	}))
	defer server.Close()
	provider := NewHuifuJspayProvider(server.Client())
	provider.baseURL = server.URL
	result, err := provider.QueryOrder(context.Background(), config, QueryRequest{MerchantOrderNo: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Paid || result.AmountFen != 100 || result.ProviderTradeNo != "HFSEQ1" || result.MerchantOrderNo != "order-1" {
		t.Fatalf("result = %#v", result)
	}
	if result.PaidAt.IsZero() || result.PaidAt.Format("20060102150405") != "20260927153000" {
		t.Fatalf("paidAt = %v", result.PaidAt)
	}
}

func TestHuifuJspayQueryOrderPendingIsNotPaid(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000000", "trans_stat": "P", "trans_amt": "1.00",
			"org_req_seq_id": "order-1", "hf_seq_id": "HFSEQ1",
		})
	}))
	defer server.Close()
	provider := NewHuifuJspayProvider(server.Client())
	provider.baseURL = server.URL
	result, err := provider.QueryOrder(context.Background(), config, QueryRequest{MerchantOrderNo: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Paid || result.Closed || result.ProviderTradeNo != "HFSEQ1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHuifuJspayCloseOrderSkipsWhenAlreadyPaid(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		writeHuifuJSON(t, writer, config, map[string]any{
			"resp_code": "00000000", "trans_stat": "S", "trans_amt": "1.00", "org_req_seq_id": "order-1",
		})
	}))
	defer server.Close()
	provider := NewHuifuJspayProvider(server.Client())
	provider.baseURL = server.URL
	result, err := provider.CloseOrder(context.Background(), config, CloseRequest{MerchantOrderNo: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Paid || result.Closed {
		t.Fatalf("result = %#v", result)
	}
	if len(paths) != 1 || paths[0] != huifuScanpayQueryPath {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestHuifuJspayCloseOrderDoesNotTreatCloseSuccessAsPaid(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if request.URL.Path == huifuScanpayQueryPath {
			writeHuifuJSON(t, writer, config, map[string]any{
				"resp_code": "00000000", "trans_stat": "P", "trans_amt": "1.00", "org_req_date": "20260927",
			})
			return
		}
		if request.URL.Path != huifuScanpayClosePath {
			t.Fatalf("path = %s", request.URL.Path)
		}
		body, _ := io.ReadAll(request.Body)
		var envelope map[string]any
		_ = json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["org_req_seq_id"] != "order-1" || data["huifu_id"] != "6666000109133323" {
			t.Fatalf("close data = %#v", data)
		}
		writeHuifuJSON(t, writer, config, map[string]any{"resp_code": "00000000", "trans_stat": "S", "org_trans_stat": "P"})
	}))
	defer server.Close()
	provider := NewHuifuJspayProvider(server.Client())
	provider.baseURL = server.URL
	result, err := provider.CloseOrder(context.Background(), config, CloseRequest{MerchantOrderNo: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Closed || result.Paid {
		t.Fatalf("result = %#v", result)
	}
	if len(paths) != 2 || paths[1] != huifuScanpayClosePath {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestHuifuJspayVerifyNotificationUsesOfficialRespData(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	respData := `{"huifu_id":"6666000109133323","req_seq_id":"order-1","trans_amt":"1.00","trans_stat":"S","hf_seq_id":"HFSEQ1","end_time":"20260927153000"}`
	signature, err := rsaSHA256Sign(privateKey, []byte(respData))
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"resp_data": {respData}, "sign": {signature}}
	provider := NewHuifuJspayProvider(http.DefaultClient)
	notification, err := provider.VerifyNotification(context.Background(), config, nil, []byte(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	if !notification.Paid || notification.MerchantOrderNo != "order-1" || notification.AmountFen != 100 || notification.ProviderTradeNo != "HFSEQ1" {
		t.Fatalf("notification = %#v", notification)
	}
}

func TestHuifuJspayDoesNotRequireProjectID(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	config := testHuifuJspayConfig(privateKey, publicKey)
	if _, ok := config["projectId"]; ok {
		t.Fatal("jspay config must not require projectId")
	}
	provider := NewHuifuJspayProvider(http.DefaultClient)
	if err := provider.ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}

func TestHuifuJspayDownloadTradeBillIsNotOnScanpayAPI(t *testing.T) {
	privateKey, publicKey := testRSAKeyPair(t)
	provider := NewHuifuJspayProvider(http.DefaultClient)
	_, err := provider.DownloadTradeBill(context.Background(), testHuifuJspayConfig(privateKey, publicKey), time.Now())
	if !errors.Is(err, ErrTradeBillNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestHuifuJspayDescriptorIsAggregateNativeQrCode(t *testing.T) {
	provider := NewHuifuJspayProvider(http.DefaultClient)
	desc := provider.Descriptor()
	if desc.ID != ProviderHuifuJspay || desc.PluginID != PluginHuifuJspay || desc.CheckoutMode != "qr_code" {
		t.Fatalf("descriptor = %#v", desc)
	}
}

func testHuifuJspayConfig(merchantPrivate *rsa.PrivateKey, huifuPublic *rsa.PublicKey) Config {
	return Config{
		"publicBaseUrl":      "https://merchant.example",
		"sysId":              "sys-1",
		"productId":          "YYZY",
		"huifuId":            "6666000109133323",
		"merchantPrivateKey": testPrivatePEM(merchantPrivate),
		"huifuPublicKey":     testPublicPEM(huifuPublic),
		"gateway":            "https://api.huifu.com",
	}
}
