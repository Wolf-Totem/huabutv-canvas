export const AGENT_PRODUCT_NAME = "画布TV智能 Agent";
export const LAST_AGENT_CANVAS_KEY = "canvas.agent.lastCanvas";

export function lastAgentCanvasStorageKey(userId: string) {
    return `${LAST_AGENT_CANVAS_KEY}.${userId}`;
}

export function readLastAgentCanvasId(userId: string) {
    try {
        return window.localStorage.getItem(lastAgentCanvasStorageKey(userId)) || "";
    } catch {
        return "";
    }
}

export function writeLastAgentCanvasId(userId: string, canvasId: string) {
    try {
        window.localStorage.setItem(lastAgentCanvasStorageKey(userId), canvasId);
    } catch {
        /* ignore */
    }
}
