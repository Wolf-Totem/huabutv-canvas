package app

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"infinite-canvas/backend/internal/agentcontext"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
)

// cloudAgentContextCompactionOperation 是"压缩历史"这次独立模型调用的任务操作名。
// 压缩不是本轮的一步，不计入步数上限，失败也不会让整轮判死。
const cloudAgentContextCompactionOperation = "cloud_agent_context_compaction"

const (
	cloudAgentMaxCompactionsPerRun = 4
	cloudAgentContextKeepPairs     = 2
)

// cloudAgentContextCompaction 是"暂停步进循环去压历史"的状态面。
type cloudAgentContextCompaction struct {
	Status      string `json:"status"`
	SourceBytes int    `json:"sourceBytes"`
	TurnCount   int    `json:"turnCount"`
	Resume      bool   `json:"resume,omitempty"`
}

func cloudAgentContextShouldCompact(historyMessages, encodedBytes int) bool {
	return agentcontext.ShouldCompact(historyMessages, encodedBytes, cloudAgentHistoryKeepRounds*2, cloudAgentHistoryMaxBytes)
}

func cloudAgentCompactionSourceFacts(state *cloudAgentRuntime) (int, int) {
	raw, err := json.Marshal(state.Canonical.Messages)
	if err != nil {
		raw = nil
	}
	return len(raw), cloudAgentConversationTurnCount(state.Canonical.Messages)
}

func cloudAgentJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func cloudAgentCheckpointFailure(op string, err error) error {
	return fmt.Errorf("%w: %s: %v", errCloudAgentCheckpoint, op, err)
}

// cloudAgentRequestCompaction 在历史条数或字节到线时暂停步进，先压成检查点。
// 本运行时没有渠道窗口 token 预算，判据沿用跨轮历史闸门。
func (s *Service) cloudAgentRequestCompaction(run *model.CloudAgentExecution, state *cloudAgentRuntime, force bool) (bool, error) {
	if run == nil || state == nil || state.ContextCompaction != nil {
		return false, nil
	}
	if state.ActiveTaskID != "" {
		return false, nil
	}
	if state.ContextCompactionCount >= cloudAgentMaxCompactionsPerRun {
		return false, nil
	}
	sourceBytes, turnCount := cloudAgentCompactionSourceFacts(state)
	if !force && !cloudAgentContextShouldCompact(len(state.Canonical.Messages), sourceBytes) {
		return false, nil
	}
	state.ContextCompaction = &cloudAgentContextCompaction{Status: "requested", SourceBytes: sourceBytes, TurnCount: turnCount, Resume: true}
	state.event(run.ID, "context_compaction_requested", map[string]any{
		"reason": "threshold", "basis": "bytes", "sourceBytes": sourceBytes, "turnCount": turnCount,
		"text": "对话上下文过长，正在压缩历史后再继续",
	})
	return true, s.repo.MutateCloudAgent(run.UserID, run.ID, run.Revision, func(current *model.CloudAgentExecution, _ *repository.Repository) error {
		return cloudAgentSave(current, state)
	})
}

func cloudAgentConversationTurnCount(messages []map[string]any) int {
	count := 0
	for _, message := range messages {
		if stringField(message, "role") != "user" {
			continue
		}
		content := stringField(message, "content")
		if stringField(message, cloudAgentContextSourceKey) == "checkpoint" {
			if checkpoint, err := agentcontext.ParseFrame(content); err == nil {
				if checkpoint.CompactedTurnCount > count {
					count = checkpoint.CompactedTurnCount
				}
				continue
			}
		}
		count++
	}
	return count
}

func cloudAgentContextFacts(events []CloudAgentEvent) []map[string]any {
	start := max(0, len(events)-200)
	facts := make([]map[string]any, 0, len(events)-start)
	for _, event := range events[start:] {
		switch event.Type {
		case "tool_completed", "tool_failed", "generation_task_created", "approval_decided", "run_failed", "canvas_updated":
		default:
			continue
		}
		if stringValue(event.Payload["toolName"]) == "skills_load" {
			continue
		}
		fact := map[string]any{"event": event.Type, "seq": event.Seq}
		for _, key := range []string{"toolName", "callId", "nodeId", "nodeIds", "referenceNodeIds", "taskId", "title", "summary", "status", "decision", "phase", "taskSubmitted", "reason", "operation"} {
			if value, ok := event.Payload[key]; ok {
				fact[key] = value
			}
		}
		if result, ok := event.Payload["result"].(map[string]any); ok {
			for _, key := range []string{"nodeId", "nodeIds", "referenceNodeIds", "taskId", "title", "summary", "status", "phase", "taskSubmitted"} {
				if value, exists := result[key]; exists {
					fact[key] = value
				}
			}
		}
		if event.Type == "canvas_updated" {
			ids := make([]string, 0, 3)
			for _, action := range creationMaps(event.Payload["actions"]) {
				if id := stringValue(action["nodeId"]); id != "" && !cloudAgentContainsString(ids, id) {
					ids = append(ids, id)
					if len(ids) == 3 {
						break
					}
				}
			}
			if len(ids) > 0 {
				fact["nodeIds"] = ids
			}
			if event.Payload["requiresRefresh"] == true {
				fact["requiresRefresh"] = true
			}
		}
		facts = append(facts, fact)
	}
	return facts
}

func cloudAgentContextCompactionPrompt(state *cloudAgentRuntime) string {
	turnCount := cloudAgentConversationTurnCount(state.Canonical.Messages)
	if state.ContextCompaction != nil && state.ContextCompaction.TurnCount > turnCount {
		turnCount = state.ContextCompaction.TurnCount
	}
	return agentcontext.BuildPrompt(agentcontext.Source{
		ConversationJSON: cloudAgentJSON(state.Canonical.Messages),
		OperationsJSON:   cloudAgentJSON(cloudAgentContextFacts(state.Events)),
		CreativeJSON:     cloudAgentJSON(state.CreativeAnchor),
		PreferencesJSON:  cloudAgentJSON(state.Profile.Layers),
		DecisionsJSON:    cloudAgentJSON(state.Decisions),
		TurnCount:        turnCount,
	})
}

func cloudAgentFallbackCheckpoint(state *cloudAgentRuntime) agentcontext.Checkpoint {
	checkpoint := agentcontext.Checkpoint{Version: agentcontext.Version}
	if state.ContextCompaction != nil {
		checkpoint.CompactedTurnCount = state.ContextCompaction.TurnCount
	}
	var history []string
	for _, message := range state.Canonical.Messages {
		role := stringField(message, "role")
		content := strings.TrimSpace(stringField(message, "content"))
		if stringField(message, cloudAgentContextSourceKey) == "checkpoint" {
			if previous, err := agentcontext.ParseFrame(content); err == nil {
				checkpoint = previous
				continue
			}
		}
		if (role == "user" || role == "assistant") && content != "" {
			history = append(history, role+": "+truncateRunes(content, 1800))
		}
	}
	if len(history) > 8 {
		history = history[len(history)-8:]
	}
	currentHistory := strings.Join(history, "\n")
	checkpoint.HistorySummary = truncateRunes(strings.TrimSpace(checkpoint.HistorySummary+"\n"+currentHistory), 10000)
	checkpoint.ScriptDesign = truncateRunes(strings.TrimSpace(checkpoint.ScriptDesign+"\n"+state.CreativeAnchor.UserPrompt+"\n"+currentHistory), 10000)
	checkpoint.CurrentWork = truncateRunes(state.Request.Prompt, 3000)
	checkpoint.NextStep = "继续当前工作；执行前重新读取画布和任务状态，并优先处理未完成任务。"
	for _, asset := range state.CreativeAnchor.ReferenceAssets {
		if asset.VisualIdentity != "inspected" {
			continue
		}
		checkpoint.Decisions = append(checkpoint.Decisions, fmt.Sprintf("已查看过画面 %s（%s），无需重复看图", asset.NodeID, asset.Type))
	}
	checkpoint.Constraints = append(checkpoint.Constraints, fmt.Sprintf("权限模式：%s；本轮预算上限：%.4f credits", state.Request.PermissionMode, state.Request.Budget.MaxCredits))
	for _, layer := range state.Profile.Layers {
		if content := strings.TrimSpace(layer.Content); content != "" {
			checkpoint.UserPreferences = append(checkpoint.UserPreferences, layer.Scope+": "+truncateRunes(content, 2400))
		}
	}
	keys := make([]string, 0, len(state.Decisions))
	for key := range state.Decisions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		checkpoint.Decisions = append(checkpoint.Decisions, key+": "+truncateRunes(state.Decisions[key], 1200))
	}
	for _, fact := range cloudAgentContextFacts(state.Events) {
		encoded := truncateRunes(cloudAgentJSON(fact), 1200)
		checkpoint.OperationHistory = append(checkpoint.OperationHistory, encoded)
		if fact["event"] == "generation_task_created" || stringValue(fact["status"]) == "running" {
			checkpoint.PendingTasks = append(checkpoint.PendingTasks, encoded)
		}
	}
	if len(checkpoint.OperationHistory) > 24 {
		checkpoint.OperationHistory = checkpoint.OperationHistory[len(checkpoint.OperationHistory)-24:]
	}
	if len(checkpoint.PendingTasks) > 12 {
		checkpoint.PendingTasks = checkpoint.PendingTasks[len(checkpoint.PendingTasks)-12:]
	}
	return checkpoint
}

func cloudAgentBoundCheckpoint(checkpoint agentcontext.Checkpoint) agentcontext.Checkpoint {
	checkpoint.HistorySummary = truncateRunes(checkpoint.HistorySummary, 3000)
	checkpoint.ScriptDesign = truncateRunes(checkpoint.ScriptDesign, 4000)
	checkpoint.CurrentWork = truncateRunes(checkpoint.CurrentWork, 800)
	checkpoint.NextStep = truncateRunes(checkpoint.NextStep, 500)
	bound := func(values []string, maxItems, maxRunes int) []string {
		if len(values) > maxItems {
			values = values[len(values)-maxItems:]
		}
		result := make([]string, 0, len(values))
		for _, value := range values {
			result = append(result, truncateRunes(value, maxRunes))
		}
		return result
	}
	checkpoint.OperationHistory = bound(checkpoint.OperationHistory, 10, 150)
	checkpoint.PendingTasks = bound(checkpoint.PendingTasks, 6, 150)
	checkpoint.Decisions = bound(checkpoint.Decisions, 8, 150)
	checkpoint.Constraints = bound(checkpoint.Constraints, 8, 150)
	checkpoint.UserPreferences = bound(checkpoint.UserPreferences, 6, 300)
	return checkpoint
}

func cloudAgentCompleteTurnTail(messages []map[string]any, pairs int) []providerTextMessage {
	complete := make([]providerTextMessage, 0, max(0, pairs)*2)
	var pending *providerTextMessage
	for _, message := range messages {
		role, content := stringField(message, "role"), strings.TrimSpace(stringField(message, "content"))
		if (role != "user" && role != "assistant") || content == "" || stringField(message, cloudAgentContextSourceKey) == "runtime" || stringField(message, cloudAgentContextSourceKey) == "continuation" {
			continue
		}
		if stringField(message, cloudAgentContextSourceKey) == "checkpoint" {
			if _, err := agentcontext.ParseFrame(content); err == nil {
				continue
			}
		}
		if role == "user" {
			candidate := providerTextMessage{Role: role, Content: content}
			pending = &candidate
			continue
		}
		if _, hasCalls := message["tool_calls"]; hasCalls || pending == nil {
			continue
		}
		complete = append(complete, *pending, providerTextMessage{Role: role, Content: content})
		pending = nil
	}
	start := max(0, len(complete)-max(0, pairs)*2)
	result := append([]providerTextMessage(nil), complete[start:]...)
	if pending != nil {
		result = append(result, *pending)
	}
	return result
}

func cloudAgentRetainedTurnCount(retained []providerTextMessage) int {
	count := 0
	for _, message := range retained {
		if message.Role == "user" {
			count++
		}
	}
	return count
}

func cloudAgentCheckpointHistory(checkpoint agentcontext.Checkpoint, recent []providerTextMessage) ([]providerTextMessage, error) {
	framed, err := agentcontext.Frame(checkpoint)
	if err != nil {
		return nil, err
	}
	history := []providerTextMessage{{Role: "user", Content: framed, AgentContextSource: "checkpoint"}, {Role: "assistant", Content: agentcontext.Acknowledgement}}
	return append(history, recent...), nil
}

func (s *Service) enqueueCloudAgentContextCompaction(run *model.CloudAgentExecution, state *cloudAgentRuntime) error {
	prompt := cloudAgentContextCompactionPrompt(state)
	input := map[string]any{
		"mode": "text", "prompt": prompt,
		"config":      map[string]any{"channelId": state.Request.ChannelID, "channelModelKey": state.Request.ChannelModelKey, "model": firstNonEmpty(state.Request.ChannelModelKey, state.Request.Model), "systemPrompt": "只执行服务端上下文压缩合同。不要调用工具，不要生成面向用户的回复。"},
		"textOptions": map[string]any{"stream": false, "thinking": false},
	}
	req := CreateTaskRequest{ProjectID: state.Request.CanvasID, Type: "canvas_text", Operation: cloudAgentContextCompactionOperation, Prompt: prompt, Model: state.Request.Model, LogicalModelID: state.Request.LogicalModelID, Input: input}
	return s.enqueueCloudAgentTask(run, state, req, nil)
}

func (s *Service) advanceCloudAgentContextCompaction(run *model.CloudAgentExecution, state *cloudAgentRuntime, task *model.Task) error {
	if task.Status == model.TaskStatusQueued || task.Status == model.TaskStatusRunning {
		return nil
	}
	checkpoint := cloudAgentFallbackCheckpoint(state)
	mode, reason := "fallback", "压缩模型任务未成功，已使用服务端保底检查点"
	if task.Status == model.TaskStatusSucceeded {
		var result struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(task.ResultJSON), &result); err == nil {
			if parsed, parseErr := agentcontext.Parse(result.Text); parseErr == nil {
				parsed.OperationHistory = checkpoint.OperationHistory
				parsed.PendingTasks = checkpoint.PendingTasks
				checkpoint, mode, reason = parsed, "model", ""
			} else {
				reason = "压缩模型输出不符合检查点合同，已使用服务端保底检查点"
			}
		} else {
			reason = "压缩模型结果损坏，已使用服务端保底检查点"
		}
	}
	if state.ContextCompaction != nil {
		checkpoint.CompactedTurnCount = state.ContextCompaction.TurnCount
	}
	return s.persistCloudAgentContextCheckpoint(run, state, checkpoint, mode, reason)
}

func (s *Service) persistCloudAgentContextCheckpoint(run *model.CloudAgentExecution, state *cloudAgentRuntime, checkpoint agentcontext.Checkpoint, mode, reason string) error {
	return s.writeCloudAgentContextCheckpoint(run, state, checkpoint, mode, reason, false)
}

func (s *Service) writeCloudAgentContextCheckpoint(run *model.CloudAgentExecution, state *cloudAgentRuntime, checkpoint agentcontext.Checkpoint, mode, reason string, keepTerminal bool) error {
	checkpoint = cloudAgentBoundCheckpoint(checkpoint)
	turnsBefore := cloudAgentConversationTurnCount(state.Canonical.Messages)
	recent := cloudAgentCompleteTurnTail(state.Canonical.Messages, cloudAgentContextKeepPairs)
	history, err := cloudAgentCheckpointHistory(checkpoint, recent)
	if err != nil {
		return cloudAgentCheckpointFailure("compaction checkpoint history", err)
	}
	return s.repo.MutateCloudAgent(run.UserID, run.ID, run.Revision, func(current *model.CloudAgentExecution, _ *repository.Repository) error {
		state.ContextCheckpoint = &checkpoint
		state.TextHistory = history
		state.Canonical.Messages = make([]map[string]any, 0, len(history))
		for _, message := range history {
			entry := map[string]any{"role": message.Role, "content": message.Content}
			if message.AgentContextSource != "" {
				entry[cloudAgentContextSourceKey] = message.AgentContextSource
			}
			state.Canonical.Messages = append(state.Canonical.Messages, entry)
		}
		state.SkillReads = nil
		state.ProfileReads = nil
		state.ActiveTaskID, state.ActiveTextDraft = "", ""
		resume := state.ContextCompaction != nil && state.ContextCompaction.Resume
		state.ContextCompaction = nil
		if resume {
			state.ContextCompactionCount++
		} else if !keepTerminal {
			current.Status = "completed"
		}
		payload := map[string]any{
			"mode": mode, "resume": resume,
			"compactedTurnCount": checkpoint.CompactedTurnCount,
			"historyMessages":    len(history),
			"droppedTurns":       max(0, turnsBefore-cloudAgentRetainedTurnCount(recent)),
			"text":               "已压缩历史上下文",
		}
		if reason != "" {
			payload["reason"] = reason
		}
		state.event(run.ID, "context_compacted", payload)
		return cloudAgentSave(current, state)
	})
}
