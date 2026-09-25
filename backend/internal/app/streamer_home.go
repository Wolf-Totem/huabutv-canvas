package app

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

var htmlTagPattern = regexp.MustCompile(`(?is)<[^>]*>`)

type StreamerHomePayload struct {
	Hero     *StreamerHomeHero     `json:"hero,omitempty"`
	Features []StreamerHomeFeature `json:"features,omitempty"`
	CTA      *StreamerHomeCTA      `json:"cta,omitempty"`
	Footer   *StreamerHomeFooter   `json:"footer,omitempty"`
}

type StreamerHomeHero struct {
	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type StreamerHomeFeature struct {
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

type StreamerHomeCTA struct {
	Label string `json:"label,omitempty"`
	Href  string `json:"href,omitempty"`
}

type StreamerHomeFooter struct {
	Text string `json:"text,omitempty"`
}

type StreamerHomeRendered struct {
	TemplateID string                `json:"templateId"`
	Hero       *StreamerHomeHero     `json:"hero,omitempty"`
	Features   []StreamerHomeFeature `json:"features,omitempty"`
	CTA        *StreamerHomeCTA      `json:"cta,omitempty"`
	Footer     *StreamerHomeFooter   `json:"footer,omitempty"`
}

func validateHomeTemplateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" || id == "landing-simple" || id == "landing-feature" {
		return nil
	}
	return BadAuthRequest("未知的首页模板")
}

func NormalizeStreamerHomePayload(templateID string, raw json.RawMessage) (json.RawMessage, *StreamerHomePayload, error) {
	if err := validateHomeTemplateID(templateID); err != nil {
		return nil, nil, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		empty, _ := json.Marshal(StreamerHomePayload{})
		return empty, &StreamerHomePayload{}, nil
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	var payload StreamerHomePayload
	if err := decoder.Decode(&payload); err != nil {
		return nil, nil, BadAuthRequest("首页内容只允许 hero/features/cta/footer，禁止原始 HTML")
	}
	sanitizeHomePayload(&payload)
	if templateID == "landing-simple" {
		payload.Features = nil
		payload.Footer = nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	return encoded, &payload, nil
}

func RenderStreamerHome(templateID string, raw []byte) (json.RawMessage, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		templateID = "landing-simple"
	}
	if err := validateHomeTemplateID(templateID); err != nil {
		return nil, err
	}
	_, payload, err := NormalizeStreamerHomePayload(templateID, json.RawMessage(raw))
	if err != nil {
		return nil, err
	}
	if payload == nil || homePayloadEmpty(payload) {
		return nil, nil
	}
	rendered := StreamerHomeRendered{TemplateID: templateID, Hero: payload.Hero, CTA: payload.CTA}
	if templateID == "landing-feature" {
		rendered.Features = payload.Features
		rendered.Footer = payload.Footer
	}
	return json.Marshal(rendered)
}

func sanitizeHomePayload(payload *StreamerHomePayload) {
	if payload.Hero != nil {
		payload.Hero.Title = sanitizeHomeText(payload.Hero.Title, 80)
		payload.Hero.Subtitle = sanitizeHomeText(payload.Hero.Subtitle, 200)
		payload.Hero.ImageURL = sanitizeHomeURL(payload.Hero.ImageURL)
		if payload.Hero.Title == "" && payload.Hero.Subtitle == "" && payload.Hero.ImageURL == "" {
			payload.Hero = nil
		}
	}
	if len(payload.Features) > 6 {
		payload.Features = payload.Features[:6]
	}
	cleanFeatures := make([]StreamerHomeFeature, 0, len(payload.Features))
	for _, feature := range payload.Features {
		item := StreamerHomeFeature{Title: sanitizeHomeText(feature.Title, 40), Text: sanitizeHomeText(feature.Text, 160)}
		if item.Title == "" && item.Text == "" {
			continue
		}
		cleanFeatures = append(cleanFeatures, item)
	}
	payload.Features = cleanFeatures
	if payload.CTA != nil {
		payload.CTA.Label = sanitizeHomeText(payload.CTA.Label, 32)
		payload.CTA.Href = sanitizeHomeURL(payload.CTA.Href)
		if payload.CTA.Label == "" {
			payload.CTA = nil
		}
	}
	if payload.Footer != nil {
		payload.Footer.Text = sanitizeHomeText(payload.Footer.Text, 120)
		if payload.Footer.Text == "" {
			payload.Footer = nil
		}
	}
}

func homePayloadEmpty(payload *StreamerHomePayload) bool {
	return payload == nil || (payload.Hero == nil && len(payload.Features) == 0 && payload.CTA == nil && payload.Footer == nil)
}

func sanitizeHomeText(value string, max int) string {
	value = htmlTagPattern.ReplaceAllString(value, "")
	value = html.UnescapeString(value)
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "<", "")
	value = strings.ReplaceAll(value, ">", "")
	if max > 0 && utf8.RuneCountInString(value) > max {
		runes := []rune(value)
		value = string(runes[:max])
	}
	return value
}

func sanitizeHomeURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "/") {
		if strings.ContainsAny(value, "<>\"'") || strings.Contains(lower, "javascript:") {
			return ""
		}
		return value
	}
	return ""
}
