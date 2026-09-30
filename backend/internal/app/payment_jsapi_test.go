package app

import "testing"

func TestRequireJsapiOpenID(t *testing.T) {
	if err := requireJsapiOpenID("qr_code", ""); err != nil {
		t.Fatalf("qr_code = %v", err)
	}
	if err := requireJsapiOpenID("redirect", ""); err != nil {
		t.Fatalf("redirect = %v", err)
	}
	if err := requireJsapiOpenID("jsapi", ""); err == nil {
		t.Fatal("jsapi without openid was accepted")
	}
	if err := requireJsapiOpenID("jsapi", "o-6yZ6Nc8Ahlx6qMQ85fH8gS6T3c"); err != nil {
		t.Fatalf("jsapi with openid = %v", err)
	}
}
