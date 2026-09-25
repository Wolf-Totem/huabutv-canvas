package locale

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed catalog/*/*.json
var embeddedCatalog embed.FS

func LoadNamespace(lng, ns string) ([]byte, error) {
	lng = Normalize(lng)
	ns = strings.TrimSpace(ns)
	if !IsSupported(lng) {
		lng = FallbackLanguage
	}
	if ns == "" || strings.ContainsAny(ns, `/\.`) {
		return nil, fmt.Errorf("invalid locale namespace")
	}
	data, err := embeddedCatalog.ReadFile(path.Join("catalog", lng, ns+".json"))
	if err != nil {
		if lng != FallbackLanguage {
			return embeddedCatalog.ReadFile(path.Join("catalog", FallbackLanguage, ns+".json"))
		}
		return nil, err
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("locale catalog is not valid json")
	}
	return data, nil
}

func EmbeddedNamespaces() []string {
	var names []string
	_ = fs.WalkDir(embeddedCatalog, "catalog", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") {
			return err
		}
		names = append(names, p)
		return nil
	})
	return names
}
