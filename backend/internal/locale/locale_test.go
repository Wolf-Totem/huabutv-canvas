package locale

import (
	"strings"
	"testing"
)

func TestRecommendKnownCountries(t *testing.T) {
	cases := map[string]string{"CN": "zh", "ID": "id", "TH": "th", "VN": "vi", "PH": "fil", "MY": "ms", "US": "en", "": "en", "XX": "en"}
	for country, want := range cases {
		if got := Recommend(country, nil); got != want {
			t.Fatalf("Recommend(%q) = %q, want %q", country, got, want)
		}
	}
}

func TestRecommendUnsupportedMappingFallsBackToZh(t *testing.T) {
	got := Recommend("JP", map[string]string{"JP": "ja"})
	if got != FallbackLanguage {
		t.Fatalf("Recommend unsupported = %q, want %q", got, FallbackLanguage)
	}
}

func TestLoadNamespaceHasPromptKeys(t *testing.T) {
	data, err := LoadNamespace("th", "common")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "language.prompt") {
		t.Fatalf("thai common catalog missing language.prompt: %s", data)
	}
}
