import { CanvasNodeType, type CanvasNodeData, type StoryboardAssetBinding, type StoryboardAssetRole, type StoryboardRow } from "@/types/canvas";

export type StoryboardAssetCatalogItem = {
    id: string;
    title: string;
    type: "image" | "video" | "audio" | "character";
    category?: string;
    tags: string[];
    prompt: string;
    characterAssetId?: string;
    characterVersionId?: string;
};

const OUTPUT_WORKFLOW_KINDS = new Set(["shot", "action_board", "final"]);
const STORYBOARD_ASSET_ROLES = new Set<StoryboardAssetRole>(["character", "environment", "wardrobe", "prop", "weapon", "style", "motion", "audio"]);

export function buildStoryboardAssetCatalog(nodes: CanvasNodeData[]): StoryboardAssetCatalogItem[] {
    return nodes.flatMap((node): StoryboardAssetCatalogItem[] => {
        const type = storyboardAssetType(node);
        if (!type || OUTPUT_WORKFLOW_KINDS.has(node.metadata?.workflowKind || "")) return [];
        if (!node.metadata?.content && !node.metadata?.storageKey && !node.metadata?.assetId && type !== "character") return [];
        const prompt = compactStoryboardAssetText(node.metadata?.prompt || node.metadata?.workflowDescription || node.metadata?.characterPrompt || "");
        return [{
            id: node.id,
            title: compactStoryboardAssetText(node.title, 120) || "未命名资产",
            type,
            category: node.metadata?.assetCategory,
            tags: Array.from(new Set((node.metadata?.assetTags || []).map((tag) => compactStoryboardAssetText(tag, 64)).filter(Boolean))).slice(0, 12),
            prompt,
            characterAssetId: node.metadata?.characterAssetId,
            characterVersionId: node.metadata?.characterVersionId,
        }];
    }).slice(0, 60);
}

export function storyboardAssetRoleForNode(node: CanvasNodeData): StoryboardAssetRole | null {
    if (node.metadata?.workflowKind === "character" || node.metadata?.assetCategory === "character") return "character";
    if (node.type === CanvasNodeType.Audio) return "audio";
    if (node.type === CanvasNodeType.Video) return "motion";
    const category = node.metadata?.assetCategory;
    if (category === "environment" || category === "prop" || category === "wardrobe" || category === "weapon" || category === "style") return category;
    if (node.type === CanvasNodeType.Image || node.type === CanvasNodeType.Drawing) return "character";
    return null;
}

export function removeStoryboardRowBinding(rows: StoryboardRow[], rowId: string, nodeId: string): StoryboardRow[] {
    return rows.map((row) => row.id !== rowId ? row : {
        ...row,
        assetBindings: (row.assetBindings || []).filter((binding) => binding.nodeId !== nodeId),
    });
}

export function normalizeStoryboardAssetBindings(bindings: StoryboardAssetBinding[] | undefined, nodes?: CanvasNodeData[]) {
    const nodeIds = nodes ? new Set(nodes.map((node) => node.id)) : null;
    const seen = new Set<string>();
    return (bindings || []).flatMap((binding): StoryboardAssetBinding[] => {
        const nodeId = String(binding?.nodeId || "").trim();
        if (!nodeId || seen.has(nodeId) || !STORYBOARD_ASSET_ROLES.has(binding.role) || (nodeIds && !nodeIds.has(nodeId))) return [];
        seen.add(nodeId);
        return [{ nodeId, role: binding.role, priority: Math.max(0, Math.min(100, Math.round(Number(binding.priority) || 0))) }];
    }).sort((left, right) => right.priority - left.priority);
}

function storyboardAssetType(node: CanvasNodeData): StoryboardAssetCatalogItem["type"] | null {
    if (node.metadata?.workflowKind === "character" && node.metadata.characterAssetId && node.metadata.characterVersionId) return "character";
    if (node.type === CanvasNodeType.Image || node.type === CanvasNodeType.Drawing) return "image";
    if (node.type === CanvasNodeType.Video) return "video";
    if (node.type === CanvasNodeType.Audio) return "audio";
    return null;
}

function compactStoryboardAssetText(value: string, limit = 600) {
    const normalized = value.replace(/\s+/g, " ").trim();
    return normalized.length > limit ? `${normalized.slice(0, limit)}…` : normalized;
}

const MIN_STORYBOARD_MATCH_RUNES = 2;

export type StoryboardMatchCatalogItem = {
    node: CanvasNodeData;
    title: string;
    role: StoryboardAssetRole;
    aliases: string[];
};

export type StoryboardAssetMatchHit = {
    nodeId: string;
    title: string;
    role: StoryboardAssetRole;
    query: string;
};

export type StoryboardRowAssetMatch = {
    rowId: string;
    shotNumber: number;
    added: StoryboardAssetMatchHit[];
    existing: StoryboardAssetMatchHit[];
    ambiguous: { query: string; titles: string[] }[];
};

export function normalizeStoryboardMatchText(value: string) {
    return value.replace(/\s+/g, "").toLowerCase();
}

export function buildStoryboardMatchCatalog(nodes: CanvasNodeData[]): StoryboardMatchCatalogItem[] {
    return nodes.flatMap((node) => {
        const role = storyboardAssetRoleForNode(node);
        if (!role || OUTPUT_WORKFLOW_KINDS.has(node.metadata?.workflowKind || "")) return [];
        if (!node.metadata?.content && !node.metadata?.storageKey && !node.metadata?.assetId && role !== "character" && node.metadata?.workflowKind !== "character") return [];
        const title = compactStoryboardAssetText(node.title, 120) || "未命名资产";
        const aliases = [title, node.metadata?.assetCategory || "", ...(node.metadata?.assetTags || []), node.metadata?.characterName || "", node.metadata?.workflowKind === "character" ? "角色" : ""].map((item) => item.trim()).filter(Boolean);
        return [{ node, title, role, aliases }];
    });
}

export function matchStoryboardAssetName(name: string, catalog: StoryboardMatchCatalogItem[]) {
    const query = normalizeStoryboardMatchText(name);
    if (!query) return undefined;
    const exact = catalog.filter((item) => normalizeStoryboardMatchText(item.title) === query || item.aliases.some((alias) => normalizeStoryboardMatchText(alias) === query));
    if (exact.length === 1) return exact[0];
    if (exact.length > 1) return undefined;
    const fuzzy = catalog.filter((item) => {
        const title = normalizeStoryboardMatchText(item.title);
        return title.includes(query) || query.includes(title) || item.aliases.some((alias) => {
            const value = normalizeStoryboardMatchText(alias);
            return value.includes(query) || query.includes(value);
        });
    });
    return fuzzy.length === 1 ? fuzzy[0] : undefined;
}

function storyboardRowHaystack(row: StoryboardRow) {
    return normalizeStoryboardMatchText([row.videoMotionPrompt, row.plotDescription, row.imageGenerationPrompt, row.dialogue, ...(row.characters || []).map((character) => character.characterName)].join(" "));
}

export function matchStoryboardCopy(rows: StoryboardRow[], nodes: CanvasNodeData[], mode: "append" | "replace" = "append"): StoryboardRowAssetMatch[] {
    const catalog = buildStoryboardMatchCatalog(nodes);
    const sorted = [...catalog].sort((left, right) => normalizeStoryboardMatchText(right.title).length - normalizeStoryboardMatchText(left.title).length);
    return rows.map((row) => {
        const haystack = storyboardRowHaystack(row);
        const covered: Array<[number, number]> = [];
        const grouped = new Map<string, StoryboardMatchCatalogItem[]>();
        for (const item of sorted) {
            const needles = Array.from(new Set([item.title, ...item.aliases].map((value) => normalizeStoryboardMatchText(value)).filter((value) => Array.from(value).length >= MIN_STORYBOARD_MATCH_RUNES)));
            for (const needle of needles) {
                let from = 0;
                while (from < haystack.length) {
                    const index = haystack.indexOf(needle, from);
                    if (index < 0) break;
                    const end = index + needle.length;
                    if (!covered.some(([start, stop]) => index < stop && end > start)) {
                        covered.push([index, end]);
                        grouped.set(needle, [...(grouped.get(needle) || []), item]);
                    }
                    from = index + 1;
                }
            }
        }
        const added: StoryboardAssetMatchHit[] = [];
        const ambiguous: StoryboardRowAssetMatch["ambiguous"] = [];
        const seen = new Set<string>();
        grouped.forEach((items, query) => {
            const unique = new Map(items.map((item) => [item.node.id, item]));
            if (unique.size !== 1) {
                ambiguous.push({ query, titles: Array.from(unique.values()).map((item) => item.title) });
                return;
            }
            const hit = unique.values().next().value;
            if (!hit || seen.has(hit.node.id)) return;
            seen.add(hit.node.id);
            added.push({ nodeId: hit.node.id, title: hit.title, role: hit.role, query });
        });
        const existing = (mode === "replace" ? [] : row.assetBindings || []).flatMap((binding) => {
            const item = catalog.find((candidate) => candidate.node.id === binding.nodeId);
            return item ? [{ nodeId: binding.nodeId, title: item.title, role: binding.role, query: item.title }] : [];
        });
        const mergedAdded = added.filter((hit) => !existing.some((item) => item.nodeId === hit.nodeId));
        return { rowId: row.id, shotNumber: row.shotNumber, added: mergedAdded, existing, ambiguous };
    });
}

export function applyStoryboardCopyMatches(rows: StoryboardRow[], matches: StoryboardRowAssetMatch[], mode: "append" | "replace" = "append"): StoryboardRow[] {
    const byRow = new Map(matches.map((item) => [item.rowId, item]));
    return rows.map((row) => {
        const match = byRow.get(row.id);
        if (!match) return row;
        const current = mode === "replace" ? [] : [...(row.assetBindings || [])];
        const seen = new Set(current.map((binding) => binding.nodeId));
        match.added.forEach((hit) => {
            if (seen.has(hit.nodeId) || current.length >= 8) return;
            seen.add(hit.nodeId);
            current.push({ nodeId: hit.nodeId, role: hit.role, priority: hit.role === "character" ? 100 : 80 });
        });
        return { ...row, assetBindings: normalizeStoryboardAssetBindings(current) };
    });
}
