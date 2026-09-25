import { describe, expect, test } from "bun:test";

import { applyStoryboardImport, parseStoryboardImportText, storyboardImportTemplateCsv } from "../src/lib/canvas/canvas-storyboard-import";
import { CanvasNodeType, type CanvasNodeData } from "../src/types/canvas";

function imageNode(id: string, title: string, extra: Partial<CanvasNodeData["metadata"]> = {}): CanvasNodeData {
    return {
        id,
        type: CanvasNodeType.Image,
        title,
        position: { x: 0, y: 0 },
        width: 160,
        height: 160,
        metadata: { content: "data:image/png;base64,xx", assetCategory: extra?.assetCategory, assetTags: extra?.assetTags, workflowKind: extra?.workflowKind, characterAssetId: extra?.characterAssetId, characterVersionId: extra?.characterVersionId },
    };
}

describe("storyboard prompt import", () => {
    test("parses the downloaded csv template", () => {
        const parsed = parseStoryboardImportText(storyboardImportTemplateCsv());
        expect(parsed.rows).toHaveLength(2);
        expect(parsed.rows[0].videoMotionPrompt).toContain("镜头缓推");
        expect(parsed.rows[0].assetNames).toEqual(["女主", "咖啡馆"]);
        expect(parsed.rows[1].dialogue).toBe("走吧。");
    });

    test("parses excel-style tsv paste", () => {
        const parsed = parseStoryboardImportText("序号\t时长\t视频提示词\t台词/旁白\t关联资产\n1\t8\t推进走廊\t开门\t男主,走廊");
        expect(parsed.rows).toHaveLength(1);
        expect(parsed.rows[0].durationSeconds).toBe(8);
        expect(parsed.rows[0].assetNames).toEqual(["男主", "走廊"]);
    });

    test("parses quoted csv cells with commas", () => {
        const parsed = parseStoryboardImportText('序号,视频提示词,关联资产\n1,"推近,再横移","女主,灯笼"');
        expect(parsed.rows[0].videoMotionPrompt).toBe("推近,再横移");
        expect(parsed.rows[0].assetNames).toEqual(["女主", "灯笼"]);
    });

    test("binds canvas assets by title and reports unmatched names", () => {
        const applied = applyStoryboardImport(parseStoryboardImportText(storyboardImportTemplateCsv()).rows, [
            imageNode("hero", "女主", { workflowKind: "character", characterAssetId: "c1", characterVersionId: "v1" }),
            imageNode("cafe", "咖啡馆", { assetCategory: "environment" }),
        ]);
        expect(applied.rows).toHaveLength(2);
        expect(applied.rows[0].assetBindings.map((item) => item.nodeId)).toEqual(["hero", "cafe"]);
        expect(applied.unmatchedAssets).toEqual(["街道"]);
        expect(applied.matchedAssetCount).toBe(3);
    });
});
