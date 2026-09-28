import { describe, expect, test } from "bun:test";

import { canvasConnectionHitsSelection, hitCanvasConnectionsInSelection, unbindStoryboardAssetsForDeletedConnections } from "@/lib/canvas/canvas-connection-selection";
import { CanvasNodeType, type CanvasConnection, type CanvasDisplayConnection, type CanvasNodeData } from "@/types/canvas";

function node(id: string, x: number, y: number, width = 120, height = 80): CanvasNodeData {
    return { id, type: CanvasNodeType.Image, title: id, position: { x, y }, width, height };
}

function entry(id: string, from: CanvasNodeData, to: CanvasNodeData, extra: Partial<CanvasConnection> = {}): CanvasDisplayConnection {
    return { connection: { id, fromNodeId: from.id, toNodeId: to.id, ...extra }, from, to };
}

describe("canvas connection marquee hits", () => {
    const left = node("a", 0, 0);
    const right = node("b", 400, 0);
    const line = entry("edge", left, right);

    test("contain requires the whole curve box inside the selection", () => {
        expect(canvasConnectionHitsSelection(line, { left: -10, top: -10, right: 600, bottom: 120 }, "contain")).toBe(true);
        expect(canvasConnectionHitsSelection(line, { left: 100, top: -10, right: 200, bottom: 120 }, "contain")).toBe(false);
    });

    test("intersect selects a curve the box only crosses", () => {
        expect(canvasConnectionHitsSelection(line, { left: 200, top: -20, right: 260, bottom: 120 }, "intersect")).toBe(true);
        expect(canvasConnectionHitsSelection(line, { left: 0, top: 200, right: 80, bottom: 280 }, "intersect")).toBe(false);
    });

    test("collects every visible connection the box hits", () => {
        const other = entry("other", node("c", 0, 300), node("d", 400, 300));
        const hits = hitCanvasConnectionsInSelection([line, other], { left: -10, top: -10, right: 600, bottom: 120 }, "contain");
        expect([...hits]).toEqual(["edge"]);
    });
});

describe("unbind storyboard assets when deleting reference edges", () => {
    test("clears the matching row binding and ignores ordinary edges", () => {
        const script: CanvasNodeData = {
            id: "script",
            type: CanvasNodeType.Script,
            title: "分镜",
            position: { x: 0, y: 0 },
            width: 480,
            height: 240,
            metadata: {
                storyboard: {
                    rows: [
                        { id: "row-1", shotNumber: 1, durationSeconds: 4, plotDescription: "", dialogue: "", characters: [], narrativeIntent: "", viewerPOV: "", performanceBlocking: "", shotSize: "", emotion: "", lightingAndAtmosphere: "", audioEffects: "", camera: "", motion: "", timeBeats: "", imageGenerationPrompt: "", videoMotionPrompt: "", mustHave: [], optionalDetails: [], continuityOut: "", negativePrompt: "", assetBindings: [{ nodeId: "zhang", role: "character", priority: 100 }] },
                        { id: "row-2", shotNumber: 2, durationSeconds: 4, plotDescription: "", dialogue: "", characters: [], narrativeIntent: "", viewerPOV: "", performanceBlocking: "", shotSize: "", emotion: "", lightingAndAtmosphere: "", audioEffects: "", camera: "", motion: "", timeBeats: "", imageGenerationPrompt: "", videoMotionPrompt: "", mustHave: [], optionalDetails: [], continuityOut: "", negativePrompt: "", assetBindings: [{ nodeId: "zhang", role: "character", priority: 100 }] },
                    ],
                    visibleColumns: ["assets"],
                    referenceNodeIds: [],
                },
            },
        };
        const next = unbindStoryboardAssetsForDeletedConnections([script, node("zhang", 0, 0)], [
            { id: "keep-shape", fromNodeId: "zhang", toNodeId: "script" },
            { id: "bind", fromNodeId: "zhang", toNodeId: "script", relation: "storyboard-asset-reference", toHandleId: "row:row-2", storyboardRowId: "row-2" },
        ]);
        expect(next[0].metadata?.storyboard?.rows[0].assetBindings).toEqual([{ nodeId: "zhang", role: "character", priority: 100 }]);
        expect(next[0].metadata?.storyboard?.rows[1].assetBindings).toEqual([]);
    });
});
