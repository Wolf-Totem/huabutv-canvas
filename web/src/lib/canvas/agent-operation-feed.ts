import { agentToolCategory, agentToolName, agentToolStatus, friendlyAgentToolSummary } from "./agent-tool-presentation";

/**
 * Agent 的操作记录（连续的 `role === "tool"` 消息）折成一行：默认只报最新一步，
 * 点击展开完整记录（组件见 `canvas-cloud-agent-chat-ui` 的 `AgentOperationFeed`）。
 */

export type AgentFeedRecord = {
    id: string;
    role: string;
    title?: string;
    text: string;
    reasoning?: boolean;
    detail?: unknown;
    planItems?: readonly unknown[];
    question?: unknown;
};

export type AgentFeedSegment<T> =
    | { kind: "operations"; key: string; items: T[] }
    | { kind: "reasoning"; key: string; items: T[] }
    | { kind: "message"; key: string; item: T };

function record(value: unknown): Record<string, unknown> {
    return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}

export function isAgentCarrierRecord(item: AgentFeedRecord): boolean {
    if (item.role === "user" || item.text || item.title) return false;
    return Boolean(item.planItems?.length) || Boolean(item.question);
}

export function isAgentOperationRecord(item: AgentFeedRecord): boolean {
    return item.role === "tool" && !isAgentCarrierRecord(item) && record(item.detail).status !== "pending";
}

export function isAgentReasoningRecord(item: AgentFeedRecord): boolean {
    return item.role === "assistant" && item.reasoning === true;
}

export function agentOperationLabel(item: AgentFeedRecord): string {
    const name = agentToolName(item.title || "工具执行", item.detail);
    return friendlyAgentToolSummary(name, item.text, item.detail, agentToolStatus(name, item.text, item.detail) === "pending");
}

export function agentOperationFailed(item: AgentFeedRecord): boolean {
    const status = agentToolStatus(agentToolName(item.title || "工具执行", item.detail), item.text, item.detail);
    return status === "failed" || status === "rejected";
}

export function agentOperationCategory(item: AgentFeedRecord) {
    return agentToolCategory(agentToolName(item.title || "工具执行", item.detail), item.detail);
}

export function agentOperationSegmentLabel(items: readonly AgentFeedRecord[]): string {
    const latest = items[items.length - 1];
    if (!latest) return "";
    if (agentOperationCategory(latest) !== "vision") return agentOperationLabel(latest);
    const viewed = new Set<string>();
    let latestTitle = "";
    for (const item of items) {
        if (agentOperationCategory(item) !== "vision") continue;
        const result = record(record(item.detail).result);
        if (result.reuseObservation === true || result.repeat === true) continue;
        if (typeof result.nodeId === "string" && result.nodeId) {
            viewed.add(result.nodeId);
            if (typeof result.title === "string" && result.title) latestTitle = result.title;
        }
    }
    if (!viewed.size) return agentOperationLabel(latest);
    return `查看了 ${viewed.size} 张画面${latestTitle ? ` · 最新《${latestTitle}》` : ""}`;
}

export function buildAgentFeedSegments<T extends AgentFeedRecord>(messages: readonly T[]): AgentFeedSegment<T>[] {
    const segments: AgentFeedSegment<T>[] = [];
    for (const item of messages) {
        if (isAgentCarrierRecord(item)) continue;
        if (isAgentReasoningRecord(item)) {
            const last = segments[segments.length - 1];
            if (last?.kind === "reasoning") last.items.push(item);
            else segments.push({ kind: "reasoning", key: item.id, items: [item] });
            continue;
        }
        if (isAgentOperationRecord(item)) {
            const last = segments[segments.length - 1];
            if (last?.kind === "operations") last.items.push(item);
            else segments.push({ kind: "operations", key: item.id, items: [item] });
            continue;
        }
        segments.push({ kind: "message", key: item.id, item });
    }
    return segments;
}
