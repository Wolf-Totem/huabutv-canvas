import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { isTrayCanvasMediaNode, trayCanvasNodeTitle } from "@/lib/canvas/canvas-tray-media";
import { CanvasNodeType, type CanvasNodeData } from "@/types/canvas";

function source(path: string) {
    return readFileSync(resolve(import.meta.dir, path), "utf8");
}

function node(type: CanvasNodeData["type"], metadata: CanvasNodeData["metadata"] = {}): CanvasNodeData {
    return { id: `${type}-1`, type, title: "", position: { x: 0, y: 0 }, width: 320, height: 180, metadata };
}

describe("canvas asset tray media", () => {
    test("current-canvas predicate includes video, audio and drawing without requiring content on drawings", () => {
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Image, { content: "https://example.com/a.png" }))).toBe(true);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Image, { storageKey: "resource:img" }))).toBe(true);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Image))).toBe(false);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Video, { storageKey: "resource:vid" }))).toBe(true);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Audio, { content: "https://example.com/a.mp3" }))).toBe(true);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Audio))).toBe(false);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Drawing, { drawingId: "draw-1" }))).toBe(true);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Drawing, { drawingPreviewUrl: "blob:preview" }))).toBe(true);
        expect(isTrayCanvasMediaNode(node(CanvasNodeType.Text, { content: "hello" }))).toBe(false);
        expect(trayCanvasNodeTitle(node(CanvasNodeType.Video, { prompt: "夜戏" }))).toBe("夜戏");
    });

    test("locks tray labels, insert helper and media focus", () => {
        const tray = source("../src/components/canvas/canvas-asset-tray.tsx");
        const upload = source("../src/pages/canvas/use-canvas-upload.ts");
        const viewport = source("../src/pages/canvas/use-canvas-viewport-controller.ts");
        const project = source("../src/pages/canvas/project.tsx");

        expect(tray).toContain("所有画布");
        expect(tray).toContain("当前画布");
        expect(tray).toContain("个人素材库");
        expect(tray).toContain("application/x-infinite-canvas-asset");
        expect(tray).toContain("application/x-infinite-canvas-image-asset");
        expect(tray).toContain("<Music2");
        expect(tray).not.toContain("onInsertAssetImage");
        expect(upload).toContain("insertLibraryAssetAt");
        expect(upload).toContain("setNodes((current) => [...current, node])");
        expect(upload).toContain("selectInsertedNode(node.id, \"close\")");
        expect(viewport).toContain("focusCanvasMediaNode");
        expect(viewport).toContain("setDialogNodeId(null)");
        expect(viewport).toContain("setToolbarNodeId(node.id)");
        expect(project).toContain("onInsertLibraryAsset");
        expect(project).toContain("insertLibraryAssetAt(localAssetToInsertPayload(asset), getCanvasCenter())");
        expect(project).toContain("onFocusCanvasMedia={focusCanvasMediaNode}");
        expect(project).not.toContain("onFocusCanvasImage={focusCanvasNode}");
    });
});
