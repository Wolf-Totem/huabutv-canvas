package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const appearanceMediaHoldsKey = "appearance_media_holds"

const (
	appearanceMediaImageMaxBytes int64 = 10 << 20
	appearanceMediaVideoMaxBytes int64 = 256 << 20
	appearanceMediaHoldTTL             = 24 * time.Hour
)

type AppearanceMediaUpload struct {
	Resource    *model.Resource `json:"resource"`
	DisplayURL  string          `json:"displayUrl"`
	OriginalURL string          `json:"originalUrl"`
	Compression string          `json:"compression"`
}

type AppearanceMediaDelivery struct {
	Resource    *model.Resource
	RedirectURL string
	Stream      *ResourceStream
	Compression string
}

type appearanceMediaHold struct {
	ResourceID string    `json:"resourceId"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func (s *Service) UploadAppearanceMedia(actor *model.User, header *multipart.FileHeader) (*AppearanceMediaUpload, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	mimeType, kind, err := validateAppearanceMediaUpload(header)
	if err != nil {
		return nil, err
	}
	header.Header.Set("Content-Type", mimeType)
	resource, err := s.UploadResource(actor.ID, header, kind, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	resource.PublicURL = ""
	if err := s.holdAppearanceMedia(resource.ID); err != nil {
		return nil, err
	}
	return &AppearanceMediaUpload{
		Resource:    resource,
		DisplayURL:  appearanceMediaURL(resource.ID, "display", 0),
		OriginalURL: appearanceMediaURL(resource.ID, "original", 0),
		Compression: "original",
	}, nil
}

func (s *Service) AppearanceMedia(id, variant string, width int) (*AppearanceMediaDelivery, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, NotFound("外观资源不存在")
	}
	_, value, err := s.readAppearance()
	if err != nil {
		return nil, NotFound("外观资源不存在")
	}
	referenced, err := landingHeroResourceIDs(value.Landing)
	if err != nil {
		return nil, NotFound("外观资源不存在")
	}
	if _, ok := referenced[id]; !ok {
		return nil, NotFound("外观资源不存在")
	}
	resource, err := s.repo.Resource(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFound("外观资源不存在")
		}
		return nil, err
	}
	if resource.Status != model.ResourceStatusReady {
		return nil, NotFound("外观资源不存在")
	}
	variant = strings.ToLower(strings.TrimSpace(variant))
	if variant == "" {
		variant = "display"
	}
	if variant != "display" && variant != "original" {
		return nil, BadAuthRequest("媒体参数无效")
	}
	width = quantizeAppearanceMediaWidth(width)
	if resource.Provider != "local" {
		setting, ossErr := s.ossSettingForResource(resource.UserID, resource)
		if ossErr == nil && strings.TrimSpace(setting.CDNBaseURL) != "" {
			redirectURL, cdnErr := ossCDNObjectURL(setting.CDNBaseURL, resource.ObjectKey)
			if cdnErr != nil {
				return nil, cdnErr
			}
			compression := "original"
			if variant == "display" && resource.Kind == "image" && aliyunStyleCDN(setting.CDNBaseURL) {
				redirectURL = appendOSSImageProcess(redirectURL, width)
				compression = "oss-process"
			}
			return &AppearanceMediaDelivery{Resource: resource, RedirectURL: redirectURL, Compression: compression}, nil
		}
	}
	stream, err := s.openResourceRange(resource.UserID, resource, "")
	if err != nil {
		return nil, err
	}
	return &AppearanceMediaDelivery{Resource: resource, Stream: stream, Compression: "original"}, nil
}

func (s *Service) holdAppearanceMedia(resourceID string) error {
	holds, err := s.readAppearanceMediaHolds()
	if err != nil {
		return err
	}
	now := time.Now()
	next := make([]appearanceMediaHold, 0, len(holds)+1)
	for _, hold := range holds {
		if hold.ResourceID == "" || hold.ExpiresAt.Before(now) || hold.ResourceID == resourceID {
			continue
		}
		next = append(next, hold)
	}
	next = append(next, appearanceMediaHold{ResourceID: resourceID, ExpiresAt: now.Add(appearanceMediaHoldTTL)})
	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	setting, err := s.repo.SystemSetting(appearanceMediaHoldsKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = &model.SystemSetting{Key: appearanceMediaHoldsKey}
	} else if err != nil {
		return err
	}
	setting.ValueJSON = string(encoded)
	return s.repo.SaveSystemSetting(setting)
}

func (s *Service) readAppearanceMediaHolds() ([]appearanceMediaHold, error) {
	setting, err := s.repo.SystemSetting(appearanceMediaHoldsKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(setting.ValueJSON) == "" {
		return nil, nil
	}
	var holds []appearanceMediaHold
	if err := json.Unmarshal([]byte(setting.ValueJSON), &holds); err != nil {
		return nil, nil
	}
	return holds, nil
}

func appearanceMediaHoldIDs(holds []appearanceMediaHold, now time.Time) map[string]struct{} {
	out := map[string]struct{}{}
	for _, hold := range holds {
		if hold.ResourceID == "" || hold.ExpiresAt.Before(now) {
			continue
		}
		out[hold.ResourceID] = struct{}{}
	}
	return out
}

func landingHeroResourceIDs(raw json.RawMessage) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if len(bytesTrimSpace(raw)) == 0 {
		return out, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	showcase, _ := doc["heroShowcase"].(map[string]any)
	if showcase == nil {
		return out, nil
	}
	collectLandingMediaIDs(out, showcase["create"])
	if banners, ok := showcase["banners"].([]any); ok {
		for _, item := range banners {
			collectLandingMediaIDs(out, item)
		}
	}
	if tiles, ok := showcase["tiles"].([]any); ok {
		for _, item := range tiles {
			collectLandingMediaIDs(out, item)
		}
	}
	return out, nil
}

func collectLandingMediaIDs(out map[string]struct{}, value any) {
	item, _ := value.(map[string]any)
	if item == nil {
		return
	}
	for _, key := range []string{"imageResourceId", "previewResourceId"} {
		id, _ := item[key].(string)
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = struct{}{}
		}
	}
}

func normalizeAppearanceLanding(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytesTrimSpace(raw)) == 0 {
		return raw, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, BadAuthRequest("首页外观 JSON 无效")
	}
	showcase, _ := doc["heroShowcase"].(map[string]any)
	if showcase != nil {
		if create, ok := showcase["create"].(map[string]any); ok {
			normalizeLandingHref(create)
		}
		if banners, ok := showcase["banners"].([]any); ok {
			for _, item := range banners {
				if row, ok := item.(map[string]any); ok {
					normalizeLandingHref(row)
				}
			}
		}
		if tiles, ok := showcase["tiles"].([]any); ok {
			for _, item := range tiles {
				if row, ok := item.(map[string]any); ok {
					normalizeLandingHref(row)
				}
			}
		}
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func normalizeLandingHref(row map[string]any) {
	href, _ := row["href"].(string)
	if strings.TrimSpace(href) == "" {
		row["href"] = "/create"
	}
}

func projectPublicLanding(raw json.RawMessage) json.RawMessage {
	if len(bytesTrimSpace(raw)) == 0 {
		return raw
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return raw
	}
	showcase, _ := doc["heroShowcase"].(map[string]any)
	if showcase == nil {
		return raw
	}
	projectLandingMediaURLs(showcase["create"])
	if banners, ok := showcase["banners"].([]any); ok {
		for _, item := range banners {
			projectLandingMediaURLs(item)
		}
	}
	if tiles, ok := showcase["tiles"].([]any); ok {
		for _, item := range tiles {
			projectLandingMediaURLs(item)
		}
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return raw
	}
	return encoded
}

func projectLandingMediaURLs(value any) {
	item, _ := value.(map[string]any)
	if item == nil {
		return
	}
	if id, _ := item["imageResourceId"].(string); strings.TrimSpace(id) != "" {
		item["imageUrl"] = appearanceMediaURL(strings.TrimSpace(id), "display", 0)
	}
	if id, _ := item["previewResourceId"].(string); strings.TrimSpace(id) != "" {
		item["previewUrl"] = appearanceMediaURL(strings.TrimSpace(id), "display", 0)
	}
}

func appearanceMediaURL(id, variant string, width int) string {
	query := url.Values{}
	if variant != "" && variant != "display" {
		query.Set("variant", variant)
	} else {
		query.Set("variant", "display")
	}
	if width > 0 {
		query.Set("w", strconv.Itoa(width))
	}
	return "/api/public/appearance/media/" + url.PathEscape(id) + "?" + query.Encode()
}

func quantizeAppearanceMediaWidth(width int) int {
	if width <= 0 {
		return 1400
	}
	best := 800
	bestDelta := absInt(width - 800)
	for _, candidate := range []int{1400, 1920} {
		delta := absInt(width - candidate)
		if delta < bestDelta {
			best = candidate
			bestDelta = delta
		}
	}
	return best
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func aliyunStyleCDN(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.Contains(host, "aliyuncs.com") || strings.HasSuffix(host, "cdn.j11.net") || strings.Contains(host, "oss-cn-") || strings.HasSuffix(host, "j11.net")
}

func appendOSSImageProcess(raw string, width int) string {
	spec := fmt.Sprintf("image/resize,w_%d,m_lfit/format,webp/quality,q_80", width)
	if strings.Contains(raw, "x-oss-process=") {
		return raw
	}
	if strings.Contains(raw, "?") {
		return raw + "&x-oss-process=" + spec
	}
	return raw + "?x-oss-process=" + spec
}

func validateAppearanceMediaUpload(header *multipart.FileHeader) (string, string, error) {
	if header == nil || header.Size <= 0 {
		return "", "", BadAuthRequest("请选择要上传的图片或视频")
	}
	file, err := header.Open()
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	buffer := make([]byte, 512)
	read, readErr := file.Read(buffer)
	if readErr != nil && read == 0 {
		return "", "", BadAuthRequest("外观资源内容无法读取")
	}
	mimeType := detectAppearanceMIME(AppearanceAssetPoster, buffer[:read], header.Size)
	kind := "image"
	maxBytes := appearanceMediaImageMaxBytes
	if _, isImage := appearanceAllowedMIMETypes(AppearanceAssetPoster)[mimeType]; !isImage {
		mimeType = detectAppearanceMIME(AppearanceAssetVideo, buffer[:read], header.Size)
		kind = "video"
		maxBytes = appearanceMediaVideoMaxBytes
		if _, isVideo := appearanceAllowedMIMETypes(AppearanceAssetVideo)[mimeType]; !isVideo {
			return "", "", BadAuthRequest("只支持 PNG、JPEG、WebP 图片或 MP4、WebM 视频")
		}
	}
	if header.Size > maxBytes {
		return "", "", BadAuthRequest(fmt.Sprintf("文件大小必须在 %dMB 以内", maxBytes>>20))
	}
	return mimeType, kind, nil
}

func bytesTrimSpace(raw json.RawMessage) json.RawMessage {
	return json.RawMessage(strings.TrimSpace(string(raw)))
}
