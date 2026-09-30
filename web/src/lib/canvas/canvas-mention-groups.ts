import type { CanvasResourceReference } from "@/lib/canvas/canvas-resource-references";

export type CanvasMentionGroupId = "current-canvas" | "skills" | "all-canvases";

export const DEFAULT_CANVAS_MENTION_GROUP_LABELS = {
    currentCanvas: "当前画布",
    allCanvases: "所有画布",
} as const;

function matchesMentionQuery(reference: CanvasResourceReference, query: string) {
    const needle = query.trim().toLowerCase();
    if (!needle) return true;
    return `${reference.label} ${reference.title} ${reference.kind} ${reference.category || ""} ${reference.text || ""}`.toLowerCase().includes(needle);
}

export function groupCanvasMentionReferences(args: {
    canvasReferences: CanvasResourceReference[];
    assetReferences: CanvasResourceReference[];
    query: string;
}) {
    const currentCanvas = args.canvasReferences
        .filter((item) => item.kind !== "skill" && !item.assetId)
        .slice()
        .sort((left, right) => Number(Boolean(right.active)) - Number(Boolean(left.active)));
    const skills = args.canvasReferences.filter((item) => item.kind === "skill");
    const allCanvases = args.assetReferences;
    if (!args.query.trim()) return { currentCanvas, skills, allCanvases };
    return {
        currentCanvas: currentCanvas.filter((item) => matchesMentionQuery(item, args.query)),
        skills: skills.filter((item) => matchesMentionQuery(item, args.query)),
        allCanvases: allCanvases.filter((item) => matchesMentionQuery(item, args.query)),
    };
}

export function emptyQueryMentionCandidates(groups: ReturnType<typeof groupCanvasMentionReferences>) {
    return [...groups.currentCanvas, ...groups.skills];
}
