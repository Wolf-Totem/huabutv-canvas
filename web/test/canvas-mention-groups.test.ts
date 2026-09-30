import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { emptyQueryMentionCandidates, groupCanvasMentionReferences } from "@/lib/canvas/canvas-mention-groups";
import type { CanvasResourceReference } from "@/lib/canvas/canvas-resource-references";

function ref(partial: Partial<CanvasResourceReference> & Pick<CanvasResourceReference, "id" | "kind" | "label">): CanvasResourceReference {
    return {
        nodeId: partial.nodeId ?? partial.id,
        title: partial.title ?? partial.label,
        active: false,
        ...partial,
    };
}

describe("canvas mention groups", () => {
    const connectedImage = ref({ id: "img-on", kind: "image", label: "图片1", active: true, nodeId: "image-1" });
    const looseVideo = ref({ id: "vid-off", kind: "video", label: "视频1", active: false, nodeId: "video-1" });
    const skill = ref({ id: "skill-1", kind: "skill", label: "分镜技能", active: true, nodeId: "skill-node" });
    const libraryImage = ref({ id: "asset-1", kind: "image", label: "角色图", title: "角色图", active: false, assetId: "asset-1", category: "character", nodeId: "" });

    test("empty query lists current-canvas nodes with connected first, skills separate, and library out of keyboard candidates", () => {
        const groups = groupCanvasMentionReferences({
            canvasReferences: [looseVideo, skill, connectedImage],
            assetReferences: [libraryImage],
            query: "",
        });
        expect(groups.currentCanvas.map((item) => item.id)).toEqual(["img-on", "vid-off"]);
        expect(groups.skills.map((item) => item.id)).toEqual(["skill-1"]);
        expect(groups.allCanvases.map((item) => item.id)).toEqual(["asset-1"]);
        expect(emptyQueryMentionCandidates(groups).map((item) => item.id)).toEqual(["img-on", "vid-off", "skill-1"]);
        expect(emptyQueryMentionCandidates(groups).some((item) => item.assetId)).toBe(false);
    });

    test("search flattens matching canvas, skill and library items", () => {
        const groups = groupCanvasMentionReferences({
            canvasReferences: [looseVideo, skill, connectedImage],
            assetReferences: [libraryImage],
            query: "角色",
        });
        expect(groups.currentCanvas).toEqual([]);
        expect(groups.skills).toEqual([]);
        expect(groups.allCanvases.map((item) => item.id)).toEqual(["asset-1"]);
    });

    test("locks empty-query visual source to the full canvas list and the two product labels", () => {
        const editor = readFileSync(resolve(import.meta.dir, "../src/components/canvas/canvas-resource-mention-textarea.tsx"), "utf8");
        expect(editor).toContain("canvasReferences={onSelectReference ? canvasReferences : activeCanvasReferences}");
        expect(editor).toContain("当前画布");
        expect(editor).toContain("所有画布");
        expect(editor).toContain("选择后自动连线");
        expect(editor).toContain("emptyQueryMentionCandidates");
        expect(editor).not.toContain("<span>画布节点</span>");
        expect(editor).not.toContain("<span>素材库</span>");
    });
});
