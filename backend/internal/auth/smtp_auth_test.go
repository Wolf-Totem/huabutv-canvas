package auth

import (
	"net/smtp"
	"testing"
)

func TestImplicitTLSPlainAuthAllowsAlreadyEncryptedSockets(t *testing.T) {
	auth := implicitTLSPlainAuth{username: "api_token", password: "secret", host: "smtp.mx.cloudflare.net"}
	mech, resp, err := auth.Start(&smtp.ServerInfo{Name: "smtp.mx.cloudflare.net", TLS: false})
	if err != nil {
		t.Fatalf("implicit TLS AUTH PLAIN must not require Client.tls: %v", err)
	}
	if mech != "PLAIN" {
		t.Fatalf("mech=%s", mech)
	}
	if string(resp) != "\x00api_token\x00secret" {
		t.Fatalf("unexpected PLAIN payload")
	}

	std := smtp.PlainAuth("", "api_token", "secret", "smtp.mx.cloudflare.net")
	if _, _, err := std.Start(&smtp.ServerInfo{Name: "smtp.mx.cloudflare.net", TLS: false}); err == nil {
		t.Fatal("stdlib PlainAuth unexpectedly accepted a non-TLS Client flag")
	}
}
