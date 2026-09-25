package app

import (
	"io"
	"net/http"
	"strings"

	"infinite-canvas/backend/internal/locale"
	"infinite-canvas/backend/internal/model"
)

const localeCountryMapSettingKey = "locale_country_map"

type LocaleRecommendation struct {
	Country     string `json:"country"`
	Recommended string `json:"recommended"`
	Supported   []string `json:"supported"`
}

func (s *Service) RecommendLocale(countryCode string) (LocaleRecommendation, error) {
	countryMap, err := s.localeCountryMap()
	if err != nil {
		return LocaleRecommendation{}, err
	}
	country := strings.ToUpper(strings.TrimSpace(countryCode))
	return LocaleRecommendation{
		Country:     country,
		Recommended: locale.Recommend(country, countryMap),
		Supported:   locale.SupportedLanguages,
	}, nil
}

func (s *Service) UpdateUserLocale(user *model.User, code string) (*model.User, error) {
	if user == nil {
		return nil, Unauthorized("请先登录")
	}
	normalized := locale.Normalize(code)
	if !locale.IsSupported(normalized) {
		return nil, BadAuthRequest("不支持的语种")
	}
	if err := s.repo.UpdateUserLocale(user.ID, normalized); err != nil {
		return nil, err
	}
	user.Locale = normalized
	return user, nil
}

func (s *Service) LocaleNamespaceJSON(lng, ns string) ([]byte, error) {
	base, err := locale.LoadNamespace(lng, ns)
	if err != nil {
		return nil, err
	}
	overlay, err := s.localeNamespaceFromOSS(lng, ns)
	if err != nil || len(overlay) == 0 {
		return base, nil
	}
	merged, mergeErr := locale.MergeJSONObjects(base, overlay)
	if mergeErr != nil {
		return base, nil
	}
	return merged, nil
}

func (s *Service) localeCountryMap() (map[string]string, error) {
	setting, err := s.repo.SystemSetting(localeCountryMapSettingKey)
	if err != nil {
		return locale.DefaultCountryLanguageMap, nil
	}
	return locale.ParseCountryMapJSON(setting.ValueJSON), nil
}

func (s *Service) localeNamespaceFromOSS(lng, ns string) ([]byte, error) {
	_, value, err := s.readOSSSetting()
	if err != nil || !value.Enabled {
		return nil, err
	}
	key := strings.Trim(value.PathPrefix, "/")
	if key != "" {
		key += "/"
	}
	key += "i18n/" + locale.Normalize(lng) + "/" + ns + ".json"
	stream, err := getOSSObjectRange(value, key, "")
	if err != nil {
		return nil, err
	}
	defer stream.body.Close()
	return io.ReadAll(io.LimitReader(stream.body, 1<<20))
}

func RequestCountryCode(header http.Header) string {
	for _, name := range []string{"CF-IPCountry", "Cf-Ipcountry", "X-Country-Code", "X-Appengine-Country"} {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}
