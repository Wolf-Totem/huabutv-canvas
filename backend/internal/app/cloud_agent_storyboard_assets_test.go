package app

import (
	"encoding/json"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
)

func addCanvasImageNode(t *testing.T, s *Service, canvas *model.CanvasProject, id, title, prompt string) {
	t.Helper()
	stored, err := s.repo.CanvasProjectForUser("user", canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := creationDocument(stored.PayloadJSON)
	if err != nil {
		t.Fatal(err)
	}
	doc["nodes"] = append(creationMaps(doc["nodes"]), map[string]any{
		"id": id, "type": "image", "title": title,
		"position": map[string]any{"x": 0, "y": 0}, "width": 320.0, "height": 180.0,
		"metadata": map[string]any{"content": "data:image/png;base64,x", "prompt": prompt, "status": "success", "storageKey": "resource:ref-one"},
	})
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	stored.PayloadJSON = string(raw)
	if err := s.repo.UpsertCanvasProject(stored); err != nil {
		t.Fatal(err)
	}
	*canvas = *stored
}

func TestCloudAgentStoryboardBindAssetsToAllRows(t *testing.T) {
	s, canvas := cloudAgentStoryboardFixture(t)
	rows := createCloudAgentStoryboardForTest(t, s, canvas)
	addCanvasImageNode(t, s, canvas, "zhang-san", "张三", "古风角色")
	policy, err := s.RuntimePolicy()
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := creationDocument(canvas.PayloadJSON)
	call := cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "bind-all", map[string]any{
		"snapshotHash": cloudAgentCanvasHash(doc), "nodeId": "storyboard-1", "action": "bind_assets_all_rows",
		"assets": []map[string]any{{"nodeId": "zhang-san", "role": "character"}},
	})
	result, err := applyCloudAgentStoryboardMutation(s.repo, "user", canvas.ID, call, policy)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.(map[string]any)["summary"].(string), "全部") {
		t.Fatalf("bind-all summary = %+v", result)
	}
	stored, err := s.repo.CanvasProjectForUser("user", canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedDoc, _ := creationDocument(stored.PayloadJSON)
	_, _, next, err := storyboardNodeFromDocument(storedDoc, "storyboard-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != len(rows) {
		t.Fatalf("row count changed: %d", len(next))
	}
	for _, row := range next {
		bindings := creationMaps(row["assetBindings"])
		if len(bindings) != 1 || stringValue(bindings[0]["nodeId"]) != "zhang-san" || stringValue(bindings[0]["role"]) != "character" {
			t.Fatalf("row missing 张三 binding: %+v", row)
		}
	}
	edges := creationMaps(storedDoc["connections"])
	if len(edges) < 2 {
		t.Fatalf("expected row connections, got %+v", edges)
	}
	for _, edge := range edges {
		if stringValue(edge["relation"]) != "storyboard-asset-reference" || stringValue(edge["fromNodeId"]) != "zhang-san" {
			t.Fatalf("unexpected edge: %+v", edge)
		}
	}
}

func TestCloudAgentStoryboardBindRejectsUnknownNodeAndPatchField(t *testing.T) {
	s, canvas := cloudAgentStoryboardFixture(t)
	rows := createCloudAgentStoryboardForTest(t, s, canvas)
	policy, err := s.RuntimePolicy()
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := creationDocument(canvas.PayloadJSON)
	hash := cloudAgentCanvasHash(doc)
	if _, err := applyCloudAgentStoryboardMutation(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "bind-missing", map[string]any{
		"snapshotHash": hash, "nodeId": "storyboard-1", "action": "bind_assets", "rowId": stringValue(rows[0]["id"]),
		"assets": []map[string]any{{"nodeId": "library-only"}},
	}), policy); err == nil || !strings.Contains(err.Error(), "可绑定素材") {
		t.Fatalf("library id should be rejected: %v", err)
	}
	if _, err := prepareCloudAgentStoryboardEdit(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "patch-bind", map[string]any{
		"snapshotHash": hash, "nodeId": "storyboard-1", "action": "update", "rowId": stringValue(rows[0]["id"]),
		"patch": map[string]any{"assetBindings": []any{}},
	})); err == nil {
		t.Fatal("patch still must not accept assetBindings")
	}
}

func TestCloudAgentStoryboardMatchAssetsFromCopy(t *testing.T) {
	s, canvas := cloudAgentStoryboardFixture(t)
	rows := createCloudAgentStoryboardForTest(t, s, canvas)
	addCanvasImageNode(t, s, canvas, "zhang-san", "张三", "古风角色")
	policy, err := s.RuntimePolicy()
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := creationDocument(canvas.PayloadJSON)
	hash := cloudAgentCanvasHash(doc)
	rowID := stringValue(rows[1]["id"])
	if _, err := applyCloudAgentStoryboardMutation(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "write-copy", map[string]any{
		"snapshotHash": hash, "nodeId": "storyboard-1", "action": "update", "rowId": rowID,
		"patch": map[string]any{"videoMotionPrompt": "张三在跳舞"},
	}), policy); err != nil {
		t.Fatal(err)
	}
	stored, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	*canvas = *stored
	doc, _ = creationDocument(canvas.PayloadJSON)
	dry, err := applyCloudAgentStoryboardMutation(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "match-dry", map[string]any{
		"snapshotHash": cloudAgentCanvasHash(doc), "nodeId": "storyboard-1", "action": "match_assets", "dryRun": true,
	}), policy)
	if err != nil {
		t.Fatal(err)
	}
	payload := dry.(map[string]any)
	if payload["dryRun"] != true {
		t.Fatalf("dry run flag missing: %+v", payload)
	}
	afterDry, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	if afterDry.PayloadJSON != canvas.PayloadJSON {
		t.Fatal("dry run wrote the canvas")
	}
	if _, err := applyCloudAgentStoryboardMutation(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "match-write", map[string]any{
		"snapshotHash": cloudAgentCanvasHash(doc), "nodeId": "storyboard-1", "action": "match_assets",
	}), policy); err != nil {
		t.Fatal(err)
	}
	written, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	writtenDoc, _ := creationDocument(written.PayloadJSON)
	_, _, next, err := storyboardNodeFromDocument(writtenDoc, "storyboard-1")
	if err != nil {
		t.Fatal(err)
	}
	second := creationMaps(next[1]["assetBindings"])
	if len(second) != 1 || stringValue(second[0]["nodeId"]) != "zhang-san" {
		t.Fatalf("match did not bind 张三 onto the dancing shot: %+v", next[1])
	}
}

func TestCloudAgentStoryboardUnbindAndStaleBind(t *testing.T) {
	s, canvas := cloudAgentStoryboardFixture(t)
	rows := createCloudAgentStoryboardForTest(t, s, canvas)
	addCanvasImageNode(t, s, canvas, "zhang-san", "张三", "古风角色")
	policy, err := s.RuntimePolicy()
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := creationDocument(canvas.PayloadJSON)
	rowID := stringValue(rows[0]["id"])
	if _, err := applyCloudAgentStoryboardMutation(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "bind-one", map[string]any{
		"snapshotHash": cloudAgentCanvasHash(doc), "nodeId": "storyboard-1", "action": "bind_assets", "rowId": rowID,
		"assets": []map[string]any{{"nodeId": "zhang-san"}},
	}), policy); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareCloudAgentStoryboardEdit(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "stale-bind", map[string]any{
		"snapshotHash": "stale", "nodeId": "storyboard-1", "action": "bind_assets", "rowId": rowID,
		"assets": []map[string]any{{"nodeId": "zhang-san"}},
	})); err == nil || !strings.Contains(err.Error(), "画布已变化") {
		t.Fatalf("stale bind: %v", err)
	}
	stored, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	*canvas = *stored
	doc, _ = creationDocument(canvas.PayloadJSON)
	if _, err := applyCloudAgentStoryboardMutation(s.repo, "user", canvas.ID, cloudAgentStoryboardCall(t, "canvas_edit_storyboard", "unbind", map[string]any{
		"snapshotHash": cloudAgentCanvasHash(doc), "nodeId": "storyboard-1", "action": "unbind_assets", "rowId": rowID,
		"assets": []map[string]any{{"nodeId": "zhang-san"}},
	}), policy); err != nil {
		t.Fatal(err)
	}
	cleared, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	clearedDoc, _ := creationDocument(cleared.PayloadJSON)
	_, _, next, _ := storyboardNodeFromDocument(clearedDoc, "storyboard-1")
	if len(creationMaps(next[0]["assetBindings"])) != 0 {
		t.Fatalf("unbind left bindings: %+v", next[0])
	}
}
