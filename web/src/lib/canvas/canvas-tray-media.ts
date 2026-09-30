import { CanvasNodeType, type CanvasNodeData } from "@/types/canvas";

export function isTrayCanvasMediaNode(node: CanvasNodeData) {
    if (node.type === CanvasNodeType.Image || node.type === CanvasNodeType.Video) {
        return Boolean(node.metadata?.content || node.metadata?.storageKey);
    }
    if (node.type === CanvasNodeType.Audio) return Boolean(node.metadata?.content);
    if (node.type === CanvasNodeType.Drawing) return Boolean(node.metadata?.drawingId || node.metadata?.drawingPreviewUrl);
    return false;
}

export function trayCanvasNodeTitle(node: CanvasNodeData) {
    if (node.title?.trim()) return node.title;
    if (node.metadata?.prompt?.trim()) return node.metadata.prompt;
    if (node.type === CanvasNodeType.Video) return "视频节点";
    if (node.type === CanvasNodeType.Audio) return "音频节点";
    if (node.type === CanvasNodeType.Drawing) return "绘图节点";
    return "图片节点";
}
