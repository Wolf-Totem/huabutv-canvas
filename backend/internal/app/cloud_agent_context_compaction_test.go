package app

import (
	"encoding/json"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/agentcontext"
	"infinite-canvas/backend/internal/model"
)

func TestCloudAgentContextShouldCompactUsesHistoryGates(t *testing.T) {
	if cloudAgentContextShouldCompact(0, 0) {
		t.Fatal("empty history should not compact")
	}
	if !cloudAgentContextShouldCompact(cloudAgentHistoryKeepRounds*2, 10) {
		t.Fatal("message count at the cross-run gate should compact")
	}
	if !cloudAgentContextShouldCompact(2, cloudAgentHistoryMaxBytes) {
		t.Fatal("byte count at the cross-run gate should compact")
	}
}

func TestCloudAgentFallbackCheckpointKeepsRecentTurns(t *testing.T) {
	state := cloudAgentRuntime{
		Request:   agentTestRequest(),
		Decisions: map[string]string{"style": "冷色调"},
		Canonical: canonicalAgentRequest{Messages: []map[string]any{
			{"role": "user", "content": "先写第一幕"},
			{"role": "assistant", "content": "第一幕已起草"},
			{"role": "user", "content": "继续第三幕，保持蓝色冷调"},
			{"role": "assistant", "content": "好，接着补镜头"},
		}},
		ContextCompaction: &cloudAgentContextCompaction{TurnCount: 4, Resume: true},
	}
	checkpoint := cloudAgentBoundCheckpoint(cloudAgentFallbackCheckpoint(&state))
	if checkpoint.Version != agentcontext.Version || checkpoint.CompactedTurnCount != 4 {
		t.Fatalf("checkpoint contract = %+v", checkpoint)
	}
	if !strings.Contains(checkpoint.HistorySummary, "第三幕") {
		t.Fatalf("fallback lost recent user goal: %+v", checkpoint)
	}
	if !strings.Contains(strings.Join(checkpoint.Decisions, "\n"), "冷色调") {
		t.Fatalf("fallback lost decisions: %+v", checkpoint)
	}
	recent := cloudAgentCompleteTurnTail(state.Canonical.Messages, 2)
	history, err := cloudAgentCheckpointHistory(checkpoint, recent)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) < 4 || history[0].AgentContextSource != "checkpoint" {
		t.Fatalf("checkpoint history = %+v", history)
	}
	if _, err := agentcontext.ParseFrame(history[0].Content); err != nil {
		t.Fatalf("framed checkpoint is unreadable: %v", err)
	}
}

func TestCloudAgentRequestCompactionPausesStep(t *testing.T) {
	s, db, _, _ := creationTestService(t)
	if err := db.Create(&model.CanvasProject{ID: "agent-canvas", UserID: "user", PayloadJSON: `{"nodes":[]}`}).Error; err != nil {
		t.Fatal(err)
	}
	root, err := s.CreateCloudAgentRun("user", agentTestRequest(), "")
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.repo.CloudAgent("user", root.ID)
	if err != nil {
		t.Fatal(err)
	}
	state, err := cloudAgentDecode(run)
	if err != nil {
		t.Fatal(err)
	}
	state.ActiveTaskID = ""
	for i := 0; i < cloudAgentHistoryKeepRounds*2; i++ {
		state.Canonical.Messages = append(state.Canonical.Messages,
			map[string]any{"role": "user", "content": strings.Repeat("用户补充设定", 80)},
			map[string]any{"role": "assistant", "content": strings.Repeat("已记下", 40)},
		)
	}
	requested, err := s.cloudAgentRequestCompaction(run, &state, false)
	if err != nil {
		t.Fatal(err)
	}
	if !requested || state.ContextCompaction == nil || state.ContextCompaction.Status != "requested" {
		t.Fatalf("compaction not requested: requested=%v state=%+v", requested, state.ContextCompaction)
	}
	persisted, err := s.repo.CloudAgent("user", root.ID)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := cloudAgentDecode(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ContextCompaction == nil || decoded.ContextCompaction.Status != "requested" {
		t.Fatalf("compaction state did not persist: %+v", decoded.ContextCompaction)
	}
}

func TestCloudAgentPersistCheckpointReplacesHistory(t *testing.T) {
	s, db, _, _ := creationTestService(t)
	if err := db.Create(&model.CanvasProject{ID: "agent-canvas", UserID: "user", PayloadJSON: `{"nodes":[]}`}).Error; err != nil {
		t.Fatal(err)
	}
	root, err := s.CreateCloudAgentRun("user", agentTestRequest(), "")
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.repo.CloudAgent("user", root.ID)
	if err != nil {
		t.Fatal(err)
	}
	state, err := cloudAgentDecode(run)
	if err != nil {
		t.Fatal(err)
	}
	state.ActiveTaskID = ""
	state.Canonical.Messages = []map[string]any{
		{"role": "user", "content": "写开场"},
		{"role": "assistant", "content": "开场完成"},
		{"role": "user", "content": "补第三幕"},
		{"role": "assistant", "content": "第三幕草稿"},
	}
	state.ContextCompaction = &cloudAgentContextCompaction{Status: "running", TurnCount: 2, Resume: true}
	checkpoint := cloudAgentFallbackCheckpoint(&state)
	if err := s.persistCloudAgentContextCheckpoint(run, &state, checkpoint, "fallback", "test"); err != nil {
		t.Fatal(err)
	}
	persisted, err := s.repo.CloudAgent("user", root.ID)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := cloudAgentDecode(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ContextCompaction != nil {
		t.Fatalf("compaction state was not cleared: %+v", decoded.ContextCompaction)
	}
	if decoded.ContextCompactionCount != 1 {
		t.Fatalf("compaction count = %d", decoded.ContextCompactionCount)
	}
	if len(decoded.Canonical.Messages) == 0 || stringField(decoded.Canonical.Messages[0], cloudAgentContextSourceKey) != "checkpoint" {
		raw, _ := json.Marshal(decoded.Canonical.Messages)
		t.Fatalf("history was not replaced with a checkpoint: %s", raw)
	}
}
