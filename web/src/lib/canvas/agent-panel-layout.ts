export type AgentPanelLayout = { left: number; top: number; width: number; height: number };
export type AgentPanelViewport = { width: number; height: number };
export type AgentPanelGesture = "move" | "north" | "south" | "east" | "west" | "northwest" | "northeast" | "southwest" | "southeast";

export const AGENT_PANEL_LAYOUT_KEY = "canvas:agent-panel-layout:v1";
const MARGIN = 12;
export const AGENT_PANEL_MIN_WIDTH = 240;
export const AGENT_PANEL_MIN_HEIGHT = 200;
const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max);

function minSize(limit: number, max: number) {
    return Math.min(limit, max);
}

export function clampAgentPanelLayout(layout: AgentPanelLayout, viewport: AgentPanelViewport): AgentPanelLayout {
    const maxWidth = Math.max(1, viewport.width - MARGIN * 2);
    const maxHeight = Math.max(1, viewport.height - MARGIN * 2);
    const width = clamp(layout.width, minSize(AGENT_PANEL_MIN_WIDTH, maxWidth), maxWidth);
    const height = clamp(layout.height, minSize(AGENT_PANEL_MIN_HEIGHT, maxHeight), maxHeight);
    return {
        width,
        height,
        left: clamp(layout.left, MARGIN, Math.max(MARGIN, viewport.width - width - MARGIN)),
        top: clamp(layout.top, MARGIN, Math.max(MARGIN, viewport.height - height - MARGIN)),
    };
}

export function restoreAgentPanelLayout(raw: string | null, viewport: AgentPanelViewport): AgentPanelLayout {
    const fallback = { width: 448, height: 720, left: viewport.width - 464, top: viewport.height - 732 };
    if (!raw) return clampAgentPanelLayout(fallback, viewport);
    try {
        const parsed: unknown = JSON.parse(raw);
        if (parsed && typeof parsed === "object" && ["left", "top", "width", "height"].every((key) => typeof (parsed as Record<string, unknown>)[key] === "number" && Number.isFinite((parsed as Record<string, unknown>)[key]))) {
            return clampAgentPanelLayout(parsed as AgentPanelLayout, viewport);
        }
    } catch {
        // UI 偏好损坏不影响对话或服务端数据，恢复可见的默认窗口。
    }
    return clampAgentPanelLayout(fallback, viewport);
}

export function changeAgentPanelLayout(start: AgentPanelLayout, gesture: AgentPanelGesture, dx: number, dy: number, viewport: AgentPanelViewport): AgentPanelLayout {
    if (gesture === "move") return clampAgentPanelLayout({ ...start, left: start.left + dx, top: start.top + dy }, viewport);
    const maxWidth = Math.max(1, viewport.width - MARGIN * 2);
    const maxHeight = Math.max(1, viewport.height - MARGIN * 2);
    const minW = minSize(AGENT_PANEL_MIN_WIDTH, maxWidth);
    const minH = minSize(AGENT_PANEL_MIN_HEIGHT, maxHeight);
    const startRight = start.left + start.width;
    const startBottom = start.top + start.height;
    const west = gesture === "west" || gesture === "northwest" || gesture === "southwest";
    const east = gesture === "east" || gesture === "northeast" || gesture === "southeast";
    const north = gesture === "north" || gesture === "northwest" || gesture === "northeast";
    const south = gesture === "south" || gesture === "southwest" || gesture === "southeast";
    let left = start.left;
    let top = start.top;
    let width = start.width;
    let height = start.height;
    if (west) {
        left = clamp(start.left + dx, Math.max(MARGIN, startRight - maxWidth), startRight - minW);
        width = startRight - left;
    } else if (east) {
        const desired = clamp(start.width + dx, minW, maxWidth);
        const maxRight = viewport.width - MARGIN;
        if (start.left + desired <= maxRight) {
            width = desired;
        } else {
            width = desired;
            left = clamp(maxRight - width, MARGIN, maxRight - minW);
            width = Math.min(width, maxRight - left);
        }
    }
    if (north) {
        top = clamp(start.top + dy, Math.max(MARGIN, startBottom - maxHeight), startBottom - minH);
        height = startBottom - top;
    } else if (south) {
        const desired = clamp(start.height + dy, minH, maxHeight);
        const maxBottom = viewport.height - MARGIN;
        if (start.top + desired <= maxBottom) {
            height = desired;
        } else {
            height = desired;
            top = clamp(maxBottom - height, MARGIN, maxBottom - minH);
            height = Math.min(height, maxBottom - top);
        }
    }
    return clampAgentPanelLayout({ left, top, width, height }, viewport);
}
