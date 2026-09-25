package canvas

import (
	"encoding/json"
	"errors"

	"infinite-canvas/backend/internal/assets"
	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
)

// SanitizeCanvasDocument 清洗画布文档：丢掉密钥/任务态，节点媒体改为 resource:{id}。
// 分享和广场共用这一层；分享再改写成 token URL，广场再改写成公开 asset URL。
func SanitizeCanvasDocument(project *model.CanvasProject) (map[string]any, map[string]bool, error) {
	var source map[string]any
	if project == nil || json.Unmarshal([]byte(project.PayloadJSON), &source) != nil {
		return nil, nil, errors.New("画布数据格式无效")
	}
	allowedResources := map[string]bool{}
	result := map[string]any{
		"id":             kernel.StringValue(source["id"]),
		"title":          kernel.DefaultString(kernel.StringValue(source["title"]), project.Title),
		"createdAt":      source["createdAt"],
		"updatedAt":      source["updatedAt"],
		"backgroundMode": source["backgroundMode"],
		"showImageInfo":  source["showImageInfo"],
		"viewport":       scrubPublicCanvasValue(source["viewport"]),
		"connections":    publicCanvasConnections(source["connections"]),
		"chatSessions":   []any{},
		"activeChatId":   nil,
		"directorScenes": []any{},
	}
	rawNodes, _ := source["nodes"].([]any)
	nodes := make([]any, 0, len(rawNodes))
	for _, rawNode := range rawNodes {
		if node, ok := sanitizeCanvasNode(rawNode, allowedResources); ok {
			nodes = append(nodes, node)
		}
	}
	result["nodes"] = nodes
	return result, allowedResources, nil
}

// RewriteCanvasResourceURLs 把节点 metadata.content 里的 resource:{id} 改写成调用方提供的 URL。
func RewriteCanvasResourceURLs(doc map[string]any, rewrite func(resourceID string) string) {
	if doc == nil || rewrite == nil {
		return
	}
	rawNodes, _ := doc["nodes"].([]any)
	for _, raw := range rawNodes {
		node, _ := raw.(map[string]any)
		metadata, _ := node["metadata"].(map[string]any)
		if metadata == nil {
			continue
		}
		resourceID := assets.ResourceID(kernel.StringValue(metadata["content"]))
		if resourceID == "" {
			continue
		}
		metadata["content"] = rewrite(resourceID)
	}
}

func sanitizeCanvasNode(value any, allowedResources map[string]bool) (map[string]any, bool) {
	node, _ := value.(map[string]any)
	if node == nil || kernel.StringValue(node["id"]) == "" || kernel.StringValue(node["type"]) == "" {
		return nil, false
	}
	result := map[string]any{
		"id": node["id"], "type": node["type"], "title": node["title"], "position": scrubPublicCanvasValue(node["position"]),
		"width": node["width"], "height": node["height"], "parentId": node["parentId"],
	}
	metadata, _ := node["metadata"].(map[string]any)
	publicMetadata := map[string]any{}
	for key, child := range metadata {
		if !publicCanvasMetadataKeys[key] {
			continue
		}
		publicMetadata[key] = scrubPublicCanvasValue(child)
	}
	if resourceID := assets.ResourceID(kernel.StringValue(metadata["storageKey"])); resourceID != "" {
		allowedResources[resourceID] = true
		publicMetadata["content"] = "resource:" + resourceID
	} else if resourceID := assets.ResourceID(kernel.StringValue(metadata["content"])); resourceID != "" {
		allowedResources[resourceID] = true
		publicMetadata["content"] = "resource:" + resourceID
	} else if nodeType := kernel.StringValue(node["type"]); nodeType == "image" || nodeType == "video" || nodeType == "audio" {
		delete(publicMetadata, "content")
	}
	delete(publicMetadata, "storageKey")
	result["metadata"] = publicMetadata
	return result, true
}
