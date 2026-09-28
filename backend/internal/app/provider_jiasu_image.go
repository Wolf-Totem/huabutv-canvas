package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"infinite-canvas/backend/internal/protocol"
)

var (
	pixelSizePattern   = regexp.MustCompile(`(?i)^\s*(\d+)\s*[x×]\s*(\d+)\s*$`)
	aspectRatioPattern = regexp.MustCompile(`^\s*\d+\s*[:：]\s*\d+\s*$`)
)

// applyJiasuImageProtocolFields 按佳速异步出图文档拆字段：
// model 保持调用方给的上游 ID；1K 写在 resolution；画幅写在 ratio；size 只表示像素。
func applyJiasuImageProtocolFields(request *protocol.GenerationRequest, config providerConfig) {
	if request == nil {
		return
	}
	if request.Extra == nil {
		request.Extra = map[string]any{}
	}
	size := strings.TrimSpace(config.Size)
	if width, height, ok := parsePixelSize(size); ok {
		pixels := fmt.Sprintf("%dx%d", width, height)
		ratio := imageRatioFromPixels(width, height)
		request.Extra["size"] = pixels
		request.Extra["ratio"] = ratio
		request.AspectRatio = ratio
		if request.Resolution == "" {
			request.Resolution = imageResolutionFromPixels(width, height)
		}
	} else if looksLikeAspectRatio(size) {
		request.AspectRatio = size
		request.Extra["ratio"] = size
	}
	if bucket := imageResolutionFromQuality(config.Quality); bucket != "" {
		request.Resolution = bucket
	}
	if request.Resolution != "" {
		request.Extra["resolution"] = request.Resolution
	}
}

func annotateJiasuImageModelEndpointError(config providerConfig, err error) error {
	if err == nil || strings.TrimSpace(config.InterfaceType) != "jiasu-image" {
		return err
	}
	message := err.Error()
	if !strings.Contains(message, "不支持此接口") {
		return err
	}
	model := firstNonEmpty(config.ProviderModelKey, config.Model)
	return fmt.Errorf("%s。佳速异步出图请把已绑定 POST /v1/images/create 的 id 填进「上游模型 ID」（文档示例 gpt-image-2.5-1k），1K 写在 resolution 而不是模型名。当前发送 model=%s", message, model)
}

func parsePixelSize(value string) (int, int, bool) {
	match := pixelSizePattern.FindStringSubmatch(value)
	if match == nil {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(match[1])
	height, errH := strconv.Atoi(match[2])
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func looksLikeAspectRatio(value string) bool {
	return aspectRatioPattern.MatchString(value)
}

func imageRatioFromPixels(width, height int) string {
	candidates := [][2]int{{1, 1}, {16, 9}, {9, 16}, {3, 4}, {4, 3}, {3, 2}, {2, 3}, {4, 5}, {5, 4}, {21, 9}, {2, 1}, {1, 2}}
	value := float64(width) / float64(height)
	best := "1:1"
	bestDiff := 1e9
	for _, pair := range candidates {
		expected := float64(pair[0]) / float64(pair[1])
		diff := value - expected
		if diff < 0 {
			diff = -diff
		}
		rel := diff / expected
		if rel < bestDiff {
			bestDiff = rel
			best = fmt.Sprintf("%d:%d", pair[0], pair[1])
		}
	}
	return best
}

func imageResolutionFromPixels(width, height int) string {
	pixels := width * height
	switch {
	case pixels <= 2_000_000:
		return "1K"
	case pixels <= 4_300_000:
		return "2K"
	default:
		return "4K"
	}
}

func imageResolutionFromQuality(quality string) string {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case "1k":
		return "1K"
	case "2k":
		return "2K"
	case "4k":
		return "4K"
	default:
		return ""
	}
}
