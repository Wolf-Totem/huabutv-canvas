package locale

import (
	"encoding/json"
	"strings"
)

const (
	FallbackLanguage     = "zh"
	DefaultOtherLanguage = "en"
)

var SupportedLanguages = []string{"zh", "en", "id", "vi", "th", "fil", "ms"}

var LanguageNames = map[string]string{
	"zh":  "中文",
	"en":  "English",
	"id":  "Bahasa Indonesia",
	"vi":  "Tiếng Việt",
	"th":  "ไทย",
	"fil": "Filipino",
	"ms":  "Bahasa Melayu",
}

// DefaultCountryLanguageMap 只用于首次推荐。可由系统设置 locale_country_map 覆盖。
var DefaultCountryLanguageMap = map[string]string{
	"CN": "zh",
	"ID": "id",
	"TH": "th",
	"VN": "vi",
	"PH": "fil",
	"MY": "ms",
}

func MergeJSONObjects(base, overlay []byte) ([]byte, error) {
	var left map[string]json.RawMessage
	var right map[string]json.RawMessage
	if err := json.Unmarshal(base, &left); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(overlay, &right); err != nil {
		return base, nil
	}
	if left == nil {
		left = map[string]json.RawMessage{}
	}
	for key, value := range right {
		left[key] = value
	}
	return json.Marshal(left)
}

func IsSupported(code string) bool {
	code = Normalize(code)
	for _, item := range SupportedLanguages {
		if item == code {
			return true
		}
	}
	return false
}

func Normalize(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	switch {
	case code == "zh-cn" || code == "zh_hans" || code == "cmn":
		return "zh"
	case strings.HasPrefix(code, "zh"):
		return "zh"
	case strings.HasPrefix(code, "en"):
		return "en"
	case strings.HasPrefix(code, "id"):
		return "id"
	case strings.HasPrefix(code, "vi"):
		return "vi"
	case strings.HasPrefix(code, "th"):
		return "th"
	case code == "tl" || strings.HasPrefix(code, "fil"):
		return "fil"
	case strings.HasPrefix(code, "ms"):
		return "ms"
	default:
		return code
	}
}

func Recommend(countryCode string, countryMap map[string]string) string {
	country := strings.ToUpper(strings.TrimSpace(countryCode))
	if country == "" || country == "XX" || country == "T1" {
		return DefaultOtherLanguage
	}
	if countryMap == nil {
		countryMap = DefaultCountryLanguageMap
	}
	mapped, ok := countryMap[country]
	if !ok {
		mapped = DefaultOtherLanguage
	}
	mapped = Normalize(mapped)
	if !IsSupported(mapped) {
		return FallbackLanguage
	}
	return mapped
}

func ParseCountryMapJSON(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return DefaultCountryLanguageMap
	}
	var overlay map[string]string
	if json.Unmarshal([]byte(raw), &overlay) != nil {
		return DefaultCountryLanguageMap
	}
	merged := map[string]string{}
	for key, value := range DefaultCountryLanguageMap {
		merged[key] = value
	}
	for key, value := range overlay {
		country := strings.ToUpper(strings.TrimSpace(key))
		if country == "" {
			continue
		}
		merged[country] = Normalize(value)
	}
	return merged
}
