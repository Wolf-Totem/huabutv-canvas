package app

import (
	"errors"
	"testing"

	"infinite-canvas/backend/internal/payment"
)

func TestPaymentCloseErrorMapsHuifuOneMinuteRule(t *testing.T) {
	err := paymentCloseError(&payment.ProviderError{Code: "20000001", Message: "不允许关闭一分钟以内的订单"})
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError", err)
	}
	if appErr.Message != "原订单刚创建，请稍后再换支付方式" {
		t.Fatalf("message = %q", appErr.Message)
	}
}

func TestPaymentCloseErrorKeepsGenericCloseFailure(t *testing.T) {
	err := paymentCloseError(&payment.ProviderError{Code: "90000000", Message: "业务执行失败"})
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError", err)
	}
	if appErr.Message != "支付渠道关单失败，请稍后重试" {
		t.Fatalf("message = %q", appErr.Message)
	}
}

func TestPaymentCloseErrorMapsOneMinuteRuleByMessage(t *testing.T) {
	err := paymentCloseError(&payment.ProviderError{Code: "provider_error", Message: "不允许关闭一分钟以内的订单"})
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError", err)
	}
	if appErr.Message != "原订单刚创建，请稍后再换支付方式" {
		t.Fatalf("message = %q", appErr.Message)
	}
}
