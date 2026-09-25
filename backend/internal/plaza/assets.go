package plaza

import (
	"bytes"
	"encoding/json"
	"io"
	"path"
	"strings"

	"infinite-canvas/backend/internal/assets"
	"infinite-canvas/backend/internal/canvas"
	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
)

func (s *Service) duplicateResource(sourceUserID, destUserID, sourceID, kind, fileName string) (*model.Resource, error) {
	resource, body, err := s.host.OpenResource(sourceUserID, sourceID)
	if err != nil {
		return nil, err
	}
	if body != nil {
		defer body.Close()
	}
	if resource == nil {
		return nil, kernel.NotFound("资源不存在")
	}
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if fileName == "" {
		fileName = sourceID
		if ext := path.Ext(resource.ObjectKey); ext != "" {
			fileName += ext
		}
	}
	copied, err := s.host.StoreResource(destUserID, kernel.FirstNonEmpty(kind, resource.Kind), fileName, resource.MimeType, int64(len(payload)), resource.Width, resource.Height, resource.DurationMs, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if copied == nil {
		return nil, kernel.NewAppError(500, "复制资源失败")
	}
	return copied, nil
}

func rewriteDocumentResourceIDs(value any, mapping map[string]string) any {
	switch item := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(item))
		for key, child := range item {
			out[key] = rewriteDocumentResourceIDs(child, mapping)
		}
		return out
	case []any:
		out := make([]any, len(item))
		for index, child := range item {
			out[index] = rewriteDocumentResourceIDs(child, mapping)
		}
		return out
	case string:
		resourceID := extractResourceID(item)
		if resourceID == "" {
			return item
		}
		next, ok := mapping[resourceID]
		if !ok {
			return item
		}
		if strings.HasPrefix(strings.TrimSpace(item), "resource:") {
			return "resource:" + next
		}
		return "resource:" + next
	default:
		return item
	}
}

func extractResourceID(value string) string {
	value = strings.TrimSpace(value)
	if id := assets.ResourceID(value); id != "" {
		return id
	}
	if index := strings.Index(value, "/assets/"); index >= 0 {
		remainder := value[index+len("/assets/"):]
		if end := strings.IndexAny(remainder, "/?#"); end >= 0 {
			remainder = remainder[:end]
		}
		return assets.ValidID(remainder)
	}
	return ""
}

func cloneDocument(doc map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func applyPlazaAssetURLs(doc map[string]any, workID string) {
	canvas.RewriteCanvasResourceURLs(doc, func(resourceID string) string {
		return plazaAssetURL(workID, resourceID)
	})
}

func applyAdminDraftAssetURLs(doc map[string]any, applicationID string) {
	canvas.RewriteCanvasResourceURLs(doc, func(resourceID string) string {
		return plazaAdminDraftAssetURL(applicationID, resourceID)
	})
}

func resourceIDList(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for id := range set {
		if strings.TrimSpace(id) == "" {
			continue
		}
		out = append(out, id)
	}
	return out
}

func nodeResourceID(node map[string]any) string {
	metadata, _ := node["metadata"].(map[string]any)
	if metadata == nil {
		return ""
	}
	return assets.ResourceID(kernel.StringValue(metadata["content"]))
}

func findNode(doc map[string]any, nodeID string) map[string]any {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" || doc == nil {
		return nil
	}
	rawNodes, _ := doc["nodes"].([]any)
	for _, raw := range rawNodes {
		node, _ := raw.(map[string]any)
		if kernel.StringValue(node["id"]) == nodeID {
			return node
		}
	}
	return nil
}

func displayableMediaCount(doc map[string]any) int {
	count := 0
	rawNodes, _ := doc["nodes"].([]any)
	for _, raw := range rawNodes {
		node, _ := raw.(map[string]any)
		nodeType := kernel.StringValue(node["type"])
		if nodeType != "image" && nodeType != "video" {
			continue
		}
		if nodeResourceID(node) != "" {
			count++
		}
	}
	return count
}
