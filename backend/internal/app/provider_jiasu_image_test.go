package app

import (
	"errors"
	"strings"
	"testing"
)

func TestProtocolRequestMapsJiasuImageSizeToRatioAndResolution(t *testing.T) {
	request := protocolRequestFromInput(canvasGenerationInput{
		Mode: "image",
		Config: providerConfig{
			InterfaceType: "jiasu-image",
			Model:         "gpt-image-2.5-1k",
			Size:          "1024x1024",
			Quality:       "auto",
		},
	})
	if request.AspectRatio != "1:1" {
		t.Fatalf("aspect ratio = %q, want 1:1 (pixel size must not be sent as ratio)", request.AspectRatio)
	}
	if request.Resolution != "1K" || request.Extra["resolution"] != "1K" {
		t.Fatalf("resolution = %q extra=%v", request.Resolution, request.Extra)
	}
	if request.Extra["size"] != "1024x1024" || request.Extra["ratio"] != "1:1" {
		t.Fatalf("extra = %#v", request.Extra)
	}
}

func TestProtocolRequestMapsJiasuImageQualityBucketOverPixels(t *testing.T) {
	request := protocolRequestFromInput(canvasGenerationInput{
		Mode: "image",
		Config: providerConfig{
			InterfaceType: "jiasu-image",
			Model:         "gpt-image-2.5-1k",
			Size:          "1824x1024",
			Quality:       "2k",
		},
	})
	if request.AspectRatio != "16:9" {
		t.Fatalf("aspect ratio = %q", request.AspectRatio)
	}
	if request.Resolution != "2K" {
		t.Fatalf("quality 2k should win over 1K pixels, got %q", request.Resolution)
	}
}

func TestProtocolRequestLeavesOtherImageProtocolsAlone(t *testing.T) {
	request := protocolRequestFromInput(canvasGenerationInput{
		Mode:   "image",
		Config: providerConfig{InterfaceType: "openai-image", Size: "1024x1024"},
	})
	if request.AspectRatio != "1024x1024" {
		t.Fatalf("openai-image still uses size as aspect ratio field, got %q", request.AspectRatio)
	}
	if request.Extra["resolution"] != nil {
		t.Fatalf("unexpected jiasu fields on openai-image: %#v", request.Extra)
	}
}

func TestAnnotateJiasuImageEndpointError(t *testing.T) {
	err := annotateJiasuImageModelEndpointError(providerConfig{InterfaceType: "jiasu-image", Model: "gpt-image-2-1k"}, errors.New("该模型不支持此接口"))
	if err == nil || !strings.Contains(err.Error(), "gpt-image-2.5-1k") || !strings.Contains(err.Error(), "gpt-image-2-1k") {
		t.Fatalf("hint missing: %v", err)
	}
	passthrough := annotateJiasuImageModelEndpointError(providerConfig{InterfaceType: "jiasu-image"}, errors.New("timeout"))
	if passthrough.Error() != "timeout" {
		t.Fatalf("unrelated error rewritten: %v", passthrough)
	}
}
