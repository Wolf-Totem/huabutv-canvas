import { describe, expect, it } from "bun:test";

import { applyStoryboardCopyMatches, buildStoryboardAssetCatalog, matchStoryboardCopy, removeStoryboardRowBinding, storyboardAssetRoleForNode } from "@/lib/canvas/canvas-storyboard-assets";
import { reconcileStoryboardTargetConnections, replaceStoryboardAssetReferenceConnections, storyboardComposerContent, storyboardRowReferenceNodeIds } from "@/lib/canvas/canvas-storyboard-materializer";
import { CanvasNodeType, type CanvasConnection, type CanvasNodeData, type StoryboardRow } from "@/types/canvas";

const node = (id: string, type: CanvasNodeType, metadata: CanvasNodeData["metadata"] = {}): CanvasNodeData => ({
    id,
    type,
    title: id,
    position: { x: 0, y: 0 },
    width: 320,
    height: 180,
    metadata,
});

const row: StoryboardRow = {
    id: "row-1",
    shotNumber: 1,
    durationSeconds: 6,
    plotDescription: "雨夜追逐",
    dialogue: "快走",
    characters: [{ characterName: "林岚", characterAssetId: "character-asset" }],
    narrativeIntent: "",
    viewerPOV: "",
    performanceBlocking: "",
    shotSize: "",
    emotion: "",
    lightingAndAtmosphere: "",
    audioEffects: "",
    camera: "",
    motion: "",
    timeBeats: "",
    imageGenerationPrompt: "",
    videoMotionPrompt: "快速跟拍",
    mustHave: [],
    optionalDetails: [],
    continuityOut: "",
    negativePrompt: "",
    assetBindings: [{ nodeId: "prop", role: "prop", priority: 80 }],
    imageNodeId: "first-frame",
    status: "idle",
};

describe("storyboard asset catalog", () => {
    it("sends reusable image, video, audio and character assets but excludes generated shots", () => {
        const assets = buildStoryboardAssetCatalog([
            node("image", CanvasNodeType.Image, { content: "data:image/png;base64,x", assetCategory: "environment" }),
            node("video", CanvasNodeType.Video, { storageKey: "resource:video" }),
            node("audio", CanvasNodeType.Audio, { content: "data:audio/wav;base64,x" }),
            node("character", CanvasNodeType.Image, { workflowKind: "character", characterAssetId: "character-asset", characterVersionId: "v1" }),
            node("shot", CanvasNodeType.Image, { content: "data:image/png;base64,x", workflowKind: "shot" }),
            node("text", CanvasNodeType.Text, { content: "story" }),
        ]);

        expect(assets.map((asset) => [asset.id, asset.type])).toEqual([
            ["image", "image"],
            ["video", "video"],
            ["audio", "audio"],
            ["character", "character"],
        ]);
    });
});

describe("storyboard target materializer", () => {
    const script = node("script", CanvasNodeType.Script, { storyboard: { rows: [row], visibleColumns: [], referenceNodeIds: ["project-style"] } });
    const nodes = [script, node("project-style", CanvasNodeType.Image, { content: "style" }), node("prop", CanvasNodeType.Image, { content: "prop" }), node("character", CanvasNodeType.Image, { workflowKind: "character", characterAssetId: "character-asset" }), node("manual", CanvasNodeType.Video, { content: "video" }), node("direct-manual", CanvasNodeType.Audio, { content: "audio" }), node("first-frame", CanvasNodeType.Image, { content: "frame", workflowKind: "shot" }), node("target", CanvasNodeType.Video)];
    const connections: CanvasConnection[] = [{ id: "manual-row-input", fromNodeId: "manual", toNodeId: "script", toHandleId: "row:row-1" }];

    it("combines stable bindings and produces position mention tokens", () => {
        const references = storyboardRowReferenceNodeIds(script, row, nodes, connections, true);
        expect(references).toEqual(["project-style", "prop", "character", "manual", "first-frame"]);
        expect(storyboardComposerContent("快速跟拍", references, nodes)).toBe("参考资产：@图片1 @图片2 @角色1 @视频1 @图片3\n快速跟拍");

        const withDirectManualInput = storyboardRowReferenceNodeIds(script, row, nodes, [...connections, { id: "direct", fromNodeId: "direct-manual", toNodeId: "target" }], false, "target");
        expect(withDirectManualInput).toContain("direct-manual");
    });

    it("reconciles managed edges without deleting manual connections", () => {
        const manualTargetEdge: CanvasConnection = { id: "manual-target", fromNodeId: "manual", toNodeId: "target" };
        const created = reconcileStoryboardTargetConnections([manualTargetEdge], script, row, "target", ["prop", "character"]);
        expect(created.filter((edge) => edge.relation === "storyboard-output")).toHaveLength(1);
        expect(created.filter((edge) => edge.relation === "storyboard-asset-reference").map((edge) => edge.fromNodeId).sort()).toEqual(["character", "prop"]);

        const reconciled = reconcileStoryboardTargetConnections(created, script, row, "target", ["character"]);
        expect(reconciled.some((edge) => edge.id === "manual-target")).toBe(true);
        expect(reconciled.some((edge) => edge.relation === "storyboard-asset-reference" && edge.fromNodeId === "prop")).toBe(false);
    });
});

describe("storyboard copy matching", () => {
    it("treats a named canvas image as a character binding", () => {
        expect(storyboardAssetRoleForNode(node("zhang", CanvasNodeType.Image, { content: "data:image/png;base64,x" }))).toBe("character");
    });

    it("binds 张三 from shot copy onto that row only", () => {
        const zhang = { ...node("zhang", CanvasNodeType.Image, { content: "data:image/png;base64,x", storageKey: "resource:zhang" }), title: "张三" };
        const idle = { ...row, assetBindings: [] };
        const dancing = { ...row, id: "row-2", shotNumber: 2, videoMotionPrompt: "张三在跳舞", assetBindings: [] };
        const matches = matchStoryboardCopy([idle, dancing], [zhang]);
        expect(matches[0].added).toEqual([]);
        expect(matches[1].added.map((hit) => hit.nodeId)).toEqual(["zhang"]);
        const applied = applyStoryboardCopyMatches([idle, dancing], matches);
        expect(applied[0].assetBindings).toEqual([]);
        expect(applied[1].assetBindings[0]).toMatchObject({ nodeId: "zhang", role: "character" });
    });

    it("does not strip an existing binding when the shot copy no longer names it", () => {
        const zhang = { ...node("zhang", CanvasNodeType.Image, { content: "data:image/png;base64,x" }), title: "张三" };
        const bound = { ...row, videoMotionPrompt: "画一个小猫", assetBindings: [{ nodeId: "zhang", role: "character" as const, priority: 100 }] };
        const matches = matchStoryboardCopy([bound], [zhang]);
        expect(matches[0].added).toEqual([]);
        expect(matches[0].existing.map((hit) => hit.nodeId)).toEqual(["zhang"]);
        expect(applyStoryboardCopyMatches([bound], matches)[0].assetBindings.map((item) => item.nodeId)).toEqual(["zhang"]);
    });
});

describe("storyboard row unbind", () => {
    it("removes one row binding without touching other rows", () => {
        const bound = [{ nodeId: "zhang", role: "character" as const, priority: 100 }];
        const rows = [
            { ...row, assetBindings: bound },
            { ...row, id: "row-2", shotNumber: 2, videoMotionPrompt: "张三在跳舞", assetBindings: bound },
        ];
        const next = removeStoryboardRowBinding(rows, "row-2", "zhang");
        expect(next[0].assetBindings).toEqual(bound);
        expect(next[1].assetBindings).toEqual([]);
    });

    it("can remove a stale nodeId and drop the matching asset edge", () => {
        const rows = [
            { ...row, assetBindings: [{ nodeId: "missing", role: "character", priority: 100 }] },
            { ...row, id: "row-2", shotNumber: 2, assetBindings: [{ nodeId: "zhang", role: "character", priority: 100 }] },
        ];
        const next = removeStoryboardRowBinding(rows, "row-1", "missing");
        expect(next[0].assetBindings).toEqual([]);
        expect(next[1].assetBindings.map((item) => item.nodeId)).toEqual(["zhang"]);
        const edges = replaceStoryboardAssetReferenceConnections(
            [
                { id: "stale", fromNodeId: "missing", toNodeId: "script", toHandleId: "row:row-1", relation: "storyboard-asset-reference", storyboardRowId: "row-1" },
                { id: "keep", fromNodeId: "zhang", toNodeId: "script", toHandleId: "row:row-2", relation: "storyboard-asset-reference", storyboardRowId: "row-2" },
            ],
            "script",
            next,
        );
        expect(edges.some((edge) => edge.fromNodeId === "missing")).toBe(false);
        expect(edges.some((edge) => edge.fromNodeId === "zhang" && edge.storyboardRowId === "row-2")).toBe(true);
    });

    it("drops the unbound asset from that shot's generation references", () => {
        const script = node("script", CanvasNodeType.Script, { storyboard: { rows: [], visibleColumns: ["assets"], referenceNodeIds: [] } });
        const zhang = { ...node("zhang", CanvasNodeType.Image, { content: "data:image/png;base64,x" }), title: "张三" };
        const unbound = { ...row, assetBindings: [] };
        expect(storyboardRowReferenceNodeIds(script, unbound, [script, zhang], [], false)).not.toContain("zhang");
    });
});
