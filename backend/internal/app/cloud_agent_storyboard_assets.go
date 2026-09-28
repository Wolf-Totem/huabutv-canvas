package app

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"infinite-canvas/backend/internal/repository"
)

const (
	cloudAgentStoryboardMaxAssetsPerRow = 8
	cloudAgentStoryboardMinMatchRunes   = 2
)

var cloudAgentStoryboardAssetRoles = map[string]int{
	"character":   100,
	"environment": 90,
	"wardrobe":    80,
	"prop":        80,
	"weapon":      80,
	"style":       60,
	"motion":      70,
	"audio":       70,
}

type cloudAgentStoryboardAssetRef struct {
	NodeID string `json:"nodeId"`
	Role   string `json:"role,omitempty"`
}

type cloudAgentStoryboardAssetCatalogItem struct {
	ID      string
	Title   string
	Role    string
	Aliases []string
	Node    map[string]any
}

func cloudAgentStoryboardDryRun(call cloudAgentCall) bool {
	var args struct {
		Action string `json:"action"`
		DryRun bool   `json:"dryRun"`
	}
	if decodeCloudAgentJSONObject(call.Function.Arguments, &args) != nil {
		return false
	}
	return args.Action == "match_assets" && args.DryRun
}

func cloudAgentStoryboardAssetActions(action string) bool {
	switch action {
	case "bind_assets", "unbind_assets", "bind_assets_all_rows", "match_assets":
		return true
	default:
		return false
	}
}

func cloudAgentStoryboardStaleSnapshot() error {
	return &cloudAgentFieldArgumentError{
		error: &cloudAgentArgumentError{creationConflict("画布已变化，本次未写入；请重新读取并重新申请审批")},
		Field: "snapshotHash", Issue: "stale_snapshot",
	}
}

func cloudAgentStoryboardAssetRoleForNode(node map[string]any) (string, bool) {
	nodeType := stringValue(node["type"])
	meta, _ := node["metadata"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	kind := stringValue(meta["workflowKind"])
	if kind == "shot" || kind == "action_board" || kind == "final" {
		return "", false
	}
	category := stringValue(meta["assetCategory"])
	if kind == "character" || category == "character" {
		return "character", true
	}
	if nodeType == "audio" {
		return "audio", true
	}
	if nodeType == "video" {
		return "motion", true
	}
	if _, ok := cloudAgentStoryboardAssetRoles[category]; ok && category != "" {
		return category, true
	}
	if nodeType == "image" || nodeType == "drawing" {
		hasMedia := stringValue(meta["content"]) != "" || stringValue(meta["storageKey"]) != "" || stringValue(meta["assetId"]) != ""
		if kind == "character" || hasMedia {
			return "character", true
		}
	}
	return "", false
}

func cloudAgentStoryboardAssetCatalog(doc map[string]any) []cloudAgentStoryboardAssetCatalogItem {
	items := make([]cloudAgentStoryboardAssetCatalogItem, 0)
	for _, node := range creationMaps(doc["nodes"]) {
		role, ok := cloudAgentStoryboardAssetRoleForNode(node)
		if !ok {
			continue
		}
		id := stringValue(node["id"])
		title := strings.TrimSpace(stringValue(node["title"]))
		if title == "" {
			title = "未命名资产"
		}
		meta, _ := node["metadata"].(map[string]any)
		aliases := []string{title}
		if category := stringValue(meta["assetCategory"]); category != "" {
			aliases = append(aliases, category)
		}
		for _, tag := range creationMaps(meta["assetTags"]) {
			if name := strings.TrimSpace(stringValue(tag["name"])); name != "" {
				aliases = append(aliases, name)
			} else if raw := strings.TrimSpace(fmt.Sprint(tag)); raw != "" && raw != "map[]" {
				aliases = append(aliases, raw)
			}
		}
		if tags, ok := meta["assetTags"].([]any); ok {
			for _, tag := range tags {
				if name, ok := tag.(string); ok && strings.TrimSpace(name) != "" {
					aliases = append(aliases, strings.TrimSpace(name))
				}
			}
		}
		if stringValue(meta["workflowKind"]) == "character" {
			aliases = append(aliases, "角色")
			if name := strings.TrimSpace(stringValue(meta["characterName"])); name != "" {
				aliases = append(aliases, name)
			}
		}
		items = append(items, cloudAgentStoryboardAssetCatalogItem{ID: id, Title: title, Role: role, Aliases: aliases, Node: node})
	}
	return items
}

func cloudAgentStoryboardCatalogByID(catalog []cloudAgentStoryboardAssetCatalogItem) map[string]cloudAgentStoryboardAssetCatalogItem {
	index := map[string]cloudAgentStoryboardAssetCatalogItem{}
	for _, item := range catalog {
		index[item.ID] = item
	}
	return index
}

func cloudAgentNormalizeStoryboardRole(role, fallback string) (string, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		role = fallback
	}
	if _, ok := cloudAgentStoryboardAssetRoles[role]; !ok {
		return "", cloudAgentFieldError("assets", "invalid_value", "素材角色无效：只能是 character、environment、wardrobe、prop、weapon、style、motion 或 audio")
	}
	return role, nil
}

func cloudAgentResolveStoryboardAssets(doc map[string]any, refs []cloudAgentStoryboardAssetRef) ([]map[string]any, error) {
	if len(refs) == 0 {
		return nil, cloudAgentFieldError("assets", "required", "需要至少一个画布节点")
	}
	if len(refs) > cloudAgentStoryboardMaxAssetsPerRow {
		return nil, cloudAgentFieldError("assets", "item_count", fmt.Sprintf("每行最多绑定 %d 个素材", cloudAgentStoryboardMaxAssetsPerRow))
	}
	catalog := cloudAgentStoryboardCatalogByID(cloudAgentStoryboardAssetCatalog(doc))
	resolved := make([]map[string]any, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		id := strings.TrimSpace(ref.NodeID)
		if err := validateCloudAgentID(id, "素材节点ID", 80); err != nil {
			return nil, cloudAgentFieldError("assets", "invalid_value", "素材节点ID无效")
		}
		if seen[id] {
			continue
		}
		item, ok := catalog[id]
		if !ok {
			return nil, cloudAgentFieldError("assets", "invalid_value", fmt.Sprintf("节点 %s 不在当前画布或不是可绑定素材；请先把素材放到画布上，不要使用素材库ID", id))
		}
		role, err := cloudAgentNormalizeStoryboardRole(ref.Role, item.Role)
		if err != nil {
			return nil, err
		}
		seen[id] = true
		resolved = append(resolved, map[string]any{
			"nodeId": id, "role": role, "priority": float64(cloudAgentStoryboardAssetRoles[role]), "title": item.Title,
		})
	}
	if len(resolved) == 0 {
		return nil, cloudAgentFieldError("assets", "required", "需要至少一个画布节点")
	}
	return resolved, nil
}

func cloudAgentRowAssetBindings(row map[string]any) []map[string]any {
	return creationMaps(row["assetBindings"])
}

func cloudAgentMergeRowAssetBindings(row map[string]any, incoming []map[string]any, replace bool) error {
	current := cloudAgentRowAssetBindings(row)
	if replace {
		current = nil
	}
	seen := map[string]bool{}
	next := make([]map[string]any, 0, len(current)+len(incoming))
	for _, binding := range current {
		id := stringValue(binding["nodeId"])
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		next = append(next, binding)
	}
	for _, binding := range incoming {
		id := stringValue(binding["nodeId"])
		if id == "" || seen[id] {
			continue
		}
		if len(next) >= cloudAgentStoryboardMaxAssetsPerRow {
			return cloudAgentFieldError("assets", "item_count", fmt.Sprintf("每行最多绑定 %d 个素材", cloudAgentStoryboardMaxAssetsPerRow))
		}
		seen[id] = true
		next = append(next, binding)
	}
	row["assetBindings"] = mapsAsAny(next)
	return nil
}

func cloudAgentRemoveRowAssetBindings(row map[string]any, nodeIDs []string) {
	if len(nodeIDs) == 0 {
		row["assetBindings"] = []any{}
		return
	}
	drop := map[string]bool{}
	for _, id := range nodeIDs {
		drop[strings.TrimSpace(id)] = true
	}
	kept := make([]map[string]any, 0)
	for _, binding := range cloudAgentRowAssetBindings(row) {
		if drop[stringValue(binding["nodeId"])] {
			continue
		}
		kept = append(kept, binding)
	}
	row["assetBindings"] = mapsAsAny(kept)
}

func cloudAgentEnsureStoryboardAssetConnections(doc map[string]any, userID, scriptID, rowID string, bindings []map[string]any) {
	wanted := map[string]map[string]any{}
	for _, binding := range bindings {
		id := stringValue(binding["nodeId"])
		if id == "" {
			continue
		}
		wanted[id] = binding
	}
	edges := creationMaps(doc["connections"])
	kept := make([]map[string]any, 0, len(edges)+len(wanted))
	seen := map[string]bool{}
	for _, edge := range edges {
		if stringValue(edge["toNodeId"]) == scriptID && stringValue(edge["relation"]) == "storyboard-asset-reference" && stringValue(edge["storyboardRowId"]) == rowID {
			from := stringValue(edge["fromNodeId"])
			if _, ok := wanted[from]; !ok {
				continue
			}
			seen[from] = true
		}
		kept = append(kept, edge)
	}
	for assetID := range wanted {
		if seen[assetID] {
			continue
		}
		kept = append(kept, map[string]any{
			"id":              cloudAgentID(userID, fmt.Sprintf("storyboard-asset:%s:%s:%s", scriptID, rowID, assetID)),
			"fromNodeId":      assetID,
			"toNodeId":        scriptID,
			"toHandleId":      "row:" + rowID,
			"relation":        "storyboard-asset-reference",
			"storyboardRowId": rowID,
		})
	}
	doc["connections"] = mapsAsAny(kept)
}

func cloudAgentSyncStoryboardAssetConnections(doc map[string]any, userID, scriptID string, rows []map[string]any) {
	for _, row := range rows {
		cloudAgentEnsureStoryboardAssetConnections(doc, userID, scriptID, stringValue(row["id"]), cloudAgentRowAssetBindings(row))
	}
}

func cloudAgentNormalizeMatchText(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
}

func cloudAgentStoryboardRowHaystack(row map[string]any) string {
	parts := []string{
		stringValue(row["videoMotionPrompt"]),
		stringValue(row["plotDescription"]),
		stringValue(row["imageGenerationPrompt"]),
		stringValue(row["dialogue"]),
	}
	for _, character := range creationMaps(row["characters"]) {
		parts = append(parts, stringValue(character["characterName"]))
	}
	return cloudAgentNormalizeMatchText(strings.Join(parts, " "))
}

func cloudAgentMatchCatalogInText(haystack string, catalog []cloudAgentStoryboardAssetCatalogItem) (unique []cloudAgentStoryboardAssetCatalogItem, ambiguous []map[string]any, unmatched []string) {
	if haystack == "" {
		return nil, nil, nil
	}
	sorted := append([]cloudAgentStoryboardAssetCatalogItem{}, catalog...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return utf8.RuneCountInString(cloudAgentNormalizeMatchText(sorted[i].Title)) > utf8.RuneCountInString(cloudAgentNormalizeMatchText(sorted[j].Title))
	})
	type span struct{ start, end int }
	covered := []span{}
	hits := map[string][]cloudAgentStoryboardAssetCatalogItem{}
	coveredQuery := map[string]bool{}
	overlaps := func(start, end int) bool {
		for _, item := range covered {
			if start < item.end && end > item.start {
				return true
			}
		}
		return false
	}
	for _, item := range sorted {
		needles := append([]string{item.Title}, item.Aliases...)
		for _, needleRaw := range needles {
			needle := cloudAgentNormalizeMatchText(needleRaw)
			if utf8.RuneCountInString(needle) < cloudAgentStoryboardMinMatchRunes {
				continue
			}
			from := 0
			for {
				idx := strings.Index(haystack[from:], needle)
				if idx < 0 {
					break
				}
				start := from + idx
				end := start + len(needle)
				if !overlaps(start, end) {
					covered = append(covered, span{start, end})
					hits[needle] = append(hits[needle], item)
					coveredQuery[needle] = true
				}
				from = start + 1
				if from >= len(haystack) {
					break
				}
			}
		}
	}
	for query, items := range hits {
		seen := map[string]cloudAgentStoryboardAssetCatalogItem{}
		for _, item := range items {
			seen[item.ID] = item
		}
		deduped := make([]cloudAgentStoryboardAssetCatalogItem, 0, len(seen))
		titles := make([]string, 0, len(seen))
		for _, item := range seen {
			deduped = append(deduped, item)
			titles = append(titles, item.Title)
		}
		if len(deduped) == 1 {
			unique = append(unique, deduped[0])
			continue
		}
		sort.Strings(titles)
		ambiguous = append(ambiguous, map[string]any{"query": query, "titles": titles})
	}
	_ = coveredQuery
	return unique, ambiguous, unmatched
}

func cloudAgentMatchStoryboardAssets(rows []map[string]any, catalog []cloudAgentStoryboardAssetCatalogItem, replace bool) (next []map[string]any, report []map[string]any, added int) {
	next = make([]map[string]any, len(rows))
	report = make([]map[string]any, 0, len(rows))
	for i, row := range rows {
		clone := map[string]any{}
		for key, value := range row {
			clone[key] = value
		}
		haystack := cloudAgentStoryboardRowHaystack(row)
		unique, ambiguous, _ := cloudAgentMatchCatalogInText(haystack, catalog)
		incoming := make([]map[string]any, 0, len(unique))
		addedTitles := make([]string, 0, len(unique))
		for _, item := range unique {
			incoming = append(incoming, map[string]any{
				"nodeId": item.ID, "role": item.Role, "priority": float64(cloudAgentStoryboardAssetRoles[item.Role]), "title": item.Title,
			})
			addedTitles = append(addedTitles, item.Title)
		}
		before := len(cloudAgentRowAssetBindings(clone))
		_ = cloudAgentMergeRowAssetBindings(clone, incoming, replace)
		after := len(cloudAgentRowAssetBindings(clone))
		if after > before {
			added += after - before
		}
		next[i] = clone
		report = append(report, map[string]any{
			"rowId":      stringValue(row["id"]),
			"shotNumber": row["shotNumber"],
			"added":      addedTitles,
			"ambiguous":  ambiguous,
			"existing":   before,
		})
	}
	return next, report, added
}

func cloudAgentEnrichStoryboardAssetBindings(doc map[string]any, rows []map[string]any) {
	catalog := cloudAgentStoryboardCatalogByID(cloudAgentStoryboardAssetCatalog(doc))
	for _, row := range rows {
		enriched := make([]map[string]any, 0)
		for _, binding := range cloudAgentRowAssetBindings(row) {
			item := map[string]any{"nodeId": stringValue(binding["nodeId"]), "role": stringValue(binding["role"]), "priority": binding["priority"]}
			if asset, ok := catalog[stringValue(binding["nodeId"])]; ok {
				item["title"] = asset.Title
			}
			enriched = append(enriched, item)
		}
		row["assetBindings"] = mapsAsAny(enriched)
	}
}

func (plan *cloudAgentStoryboardMutationPlan) bindPreview(title, description string, items []cloudAgentApprovalPreviewItem) {
	plan.Preview = cloudAgentApprovalPreview{Kind: "canvas_mutation", Title: title, Description: description, Items: items}
}

func prepareCloudAgentStoryboardAssetEdit(repo *repository.Repository, userID, canvasID string, call cloudAgentCall, args cloudAgentStoryboardEditArgs) (*cloudAgentStoryboardMutationPlan, error) {
	if args.SnapshotHash == "" || args.NodeID == "" {
		return nil, BadAuthRequest("编辑分镜需要快照和节点ID")
	}
	canvas, err := repo.CanvasProjectForUser(userID, canvasID)
	if err != nil {
		return nil, err
	}
	doc, err := creationDocument(canvas.PayloadJSON)
	if err != nil {
		return nil, err
	}
	beforeHash := cloudAgentCanvasHash(doc)
	if beforeHash != args.SnapshotHash {
		return nil, cloudAgentStoryboardStaleSnapshot()
	}
	node, storyboard, rows, err := storyboardNodeFromDocument(doc, args.NodeID)
	if err != nil {
		return nil, err
	}
	scriptID := stringValue(node["id"])
	scriptTitle := stringValue(node["title"])
	plan := &cloudAgentStoryboardMutationPlan{Canvas: canvas, Document: doc, BeforeJSON: canvas.PayloadJSON, BeforeSnapshotHash: beforeHash, DryRun: args.DryRun}

	rowIndex := func(rowID string) (int, error) {
		if err := validateCloudAgentID(rowID, "分镜行ID", 120); err != nil {
			return -1, BadAuthRequest("修改镜头必须使用最新读取结果中的真实 rowId")
		}
		for i, row := range rows {
			if stringValue(row["id"]) == rowID {
				return i, nil
			}
		}
		return -1, BadAuthRequest("分镜行不存在，请先读取真实行ID")
	}

	summaries := func(bindings []map[string]any, shot any) []cloudAgentApprovalPreviewItem {
		items := make([]cloudAgentApprovalPreviewItem, 0, len(bindings))
		for _, binding := range bindings {
			items = append(items, cloudAgentApprovalPreviewItem{
				Operation: "bind_assets", NodeID: scriptID, NodeTitle: scriptTitle, NodeType: "script", NodeTypeLabel: "分镜脚本",
				TargetNodeID: stringValue(binding["nodeId"]), TargetNodeTitle: stringValue(binding["title"]),
				Summary: fmt.Sprintf("第 %v 镜 ← %s（%s）", shot, stringValue(binding["title"]), stringValue(binding["role"])),
			})
		}
		return items
	}

	switch args.Action {
	case "bind_assets":
		index, err := rowIndex(args.RowID)
		if err != nil {
			return nil, err
		}
		incoming, err := cloudAgentResolveStoryboardAssets(doc, args.Assets)
		if err != nil {
			return nil, err
		}
		if err := cloudAgentMergeRowAssetBindings(rows[index], incoming, false); err != nil {
			return nil, err
		}
		cloudAgentEnsureStoryboardAssetConnections(doc, userID, scriptID, args.RowID, cloudAgentRowAssetBindings(rows[index]))
		plan.bindPreview("确认绑定分镜素材", "Agent 准备把画布素材挂到指定镜头。批准后才会写入。", summaries(incoming, rows[index]["shotNumber"]))
	case "bind_assets_all_rows":
		if len(rows) == 0 {
			return nil, BadAuthRequest("分镜表没有镜头行")
		}
		incoming, err := cloudAgentResolveStoryboardAssets(doc, args.Assets)
		if err != nil {
			return nil, err
		}
		items := []cloudAgentApprovalPreviewItem{}
		for _, row := range rows {
			if err := cloudAgentMergeRowAssetBindings(row, incoming, false); err != nil {
				return nil, err
			}
			cloudAgentEnsureStoryboardAssetConnections(doc, userID, scriptID, stringValue(row["id"]), cloudAgentRowAssetBindings(row))
			items = append(items, summaries(incoming, row["shotNumber"])...)
		}
		plan.bindPreview("确认绑定全部分镜素材", fmt.Sprintf("Agent 准备把素材挂到全部 %d 个镜头。批准后才会写入。", len(rows)), items)
	case "unbind_assets":
		index, err := rowIndex(args.RowID)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(args.Assets))
		for _, ref := range args.Assets {
			ids = append(ids, strings.TrimSpace(ref.NodeID))
		}
		cloudAgentRemoveRowAssetBindings(rows[index], ids)
		cloudAgentEnsureStoryboardAssetConnections(doc, userID, scriptID, args.RowID, cloudAgentRowAssetBindings(rows[index]))
		summary := "清空该镜素材绑定"
		if len(ids) > 0 {
			summary = fmt.Sprintf("从第 %v 镜移除 %d 个素材", rows[index]["shotNumber"], len(ids))
		}
		plan.bindPreview("确认移除分镜素材", "Agent 准备移除镜头上的素材绑定。批准后才会写入。", []cloudAgentApprovalPreviewItem{{
			Operation: "unbind_assets", NodeID: scriptID, NodeTitle: scriptTitle, NodeType: "script", NodeTypeLabel: "分镜脚本", Summary: summary,
		}})
	case "match_assets":
		replace := strings.EqualFold(args.Mode, "replace")
		target := rows
		if args.RowID != "" {
			index, err := rowIndex(args.RowID)
			if err != nil {
				return nil, err
			}
			target = []map[string]any{rows[index]}
		}
		next, report, added := cloudAgentMatchStoryboardAssets(target, cloudAgentStoryboardAssetCatalog(doc), replace)
		if args.RowID != "" {
			index, err := rowIndex(args.RowID)
			if err != nil {
				return nil, err
			}
			rows[index] = next[0]
		} else {
			copy(rows, next)
		}
		result := map[string]any{"nodeId": scriptID, "dryRun": args.DryRun, "mode": map[bool]string{true: "replace", false: "append"}[replace], "added": added, "rows": report, "snapshotHash": beforeHash}
		plan.Result = result
		if args.DryRun {
			plan.DryRun = true
			plan.bindPreview("预演分镜素材匹配", "仅预演，不会写入画布。", nil)
			return plan, nil
		}
		cloudAgentSyncStoryboardAssetConnections(doc, userID, scriptID, rows)
		plan.bindPreview("确认按文案匹配分镜素材", fmt.Sprintf("将根据镜头文案写入 %d 处新的素材绑定。歧义项不会写入。", added), []cloudAgentApprovalPreviewItem{{
			Operation: "match_assets", NodeID: scriptID, NodeTitle: scriptTitle, NodeType: "script", NodeTypeLabel: "分镜脚本",
			Summary: fmt.Sprintf("按文案匹配并绑定 %d 处素材", added),
		}})
	default:
		return nil, BadAuthRequest("分镜操作必须是 append、update、remove、bind_assets、unbind_assets、bind_assets_all_rows 或 match_assets")
	}
	storyboard["rows"] = mapsAsAny(rows)
	meta, _ := node["metadata"].(map[string]any)
	meta["storyboard"] = storyboard
	return plan, nil
}
