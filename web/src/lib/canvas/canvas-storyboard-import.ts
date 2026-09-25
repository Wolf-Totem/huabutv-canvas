import { createStoryboardRow } from "@/lib/canvas/canvas-project-domain";
import { bindingForConnectedNode } from "@/lib/canvas/canvas-storyboard-materializer";
import type { CanvasNodeData, StoryboardAssetBinding, StoryboardCharacterReference, StoryboardRow } from "@/types/canvas";

export const STORYBOARD_IMPORT_TEMPLATE_HEADERS = ["序号", "时长", "视频提示词", "台词/旁白", "关联资产", "图片提示词", "画面描述", "角色"] as const;

export const STORYBOARD_IMPORT_TEMPLATE_EXAMPLE: string[][] = [
    ["1", "6", "镜头缓推，女主转身看向窗外，暖光洒在侧脸", "今天必须出发。", "女主,咖啡馆", "女主站在咖啡馆窗边，午后暖色光", "女主在窗边犹豫", "女主"],
    ["2", "4", "切外景，镜头跟随女主走出店门", "走吧。", "女主,街道", "女主推开咖啡馆玻璃门", "女主离开咖啡馆", "女主"],
];

type ImportField =
    | "shotNumber"
    | "durationSeconds"
    | "videoMotionPrompt"
    | "dialogue"
    | "assets"
    | "imageGenerationPrompt"
    | "plotDescription"
    | "characters"
    | "camera"
    | "motion"
    | "shotSize"
    | "negativePrompt"
    | "narrativeIntent"
    | "lightingAndAtmosphere"
    | "timeBeats"
    | "continuityOut";

const HEADER_ALIASES: Record<string, ImportField> = {
    序号: "shotNumber",
    镜号: "shotNumber",
    镜头号: "shotNumber",
    "#": "shotNumber",
    shot: "shotNumber",
    shotnumber: "shotNumber",
    时长: "durationSeconds",
    秒: "durationSeconds",
    duration: "durationSeconds",
    durationseconds: "durationSeconds",
    视频提示词: "videoMotionPrompt",
    运动提示词: "videoMotionPrompt",
    视频: "videoMotionPrompt",
    videomotionprompt: "videoMotionPrompt",
    prompt: "videoMotionPrompt",
    "台词/旁白": "dialogue",
    台词: "dialogue",
    旁白: "dialogue",
    dialogue: "dialogue",
    关联资产: "assets",
    资产: "assets",
    参考资产: "assets",
    参考: "assets",
    assets: "assets",
    图片提示词: "imageGenerationPrompt",
    首帧提示词: "imageGenerationPrompt",
    imagegenerationprompt: "imageGenerationPrompt",
    画面描述: "plotDescription",
    剧情: "plotDescription",
    分镜: "plotDescription",
    plotdescription: "plotDescription",
    角色: "characters",
    人物: "characters",
    characters: "characters",
    镜头设计: "camera",
    机位: "camera",
    camera: "camera",
    运镜: "motion",
    motion: "motion",
    景别: "shotSize",
    shotsize: "shotSize",
    负面要求: "negativePrompt",
    负向: "negativePrompt",
    negativeprompt: "negativePrompt",
    镜头意图: "narrativeIntent",
    narrativeintent: "narrativeIntent",
    光影氛围: "lightingAndAtmosphere",
    时间节拍: "timeBeats",
    连续性出口: "continuityOut",
};

export type StoryboardImportDraftRow = {
    shotNumber: number;
    durationSeconds: number;
    videoMotionPrompt: string;
    dialogue: string;
    imageGenerationPrompt: string;
    plotDescription: string;
    camera: string;
    motion: string;
    shotSize: string;
    negativePrompt: string;
    narrativeIntent: string;
    lightingAndAtmosphere: string;
    timeBeats: string;
    continuityOut: string;
    assetNames: string[];
    characterNames: string[];
};

export type StoryboardImportParseResult = {
    rows: StoryboardImportDraftRow[];
    warnings: string[];
};

export type StoryboardImportApplyResult = {
    rows: StoryboardRow[];
    unmatchedAssets: string[];
    matchedAssetCount: number;
};

export function storyboardImportTemplateCsv() {
    return `\uFEFF${serializeCsv([ [...STORYBOARD_IMPORT_TEMPLATE_HEADERS], ...STORYBOARD_IMPORT_TEMPLATE_EXAMPLE ])}`;
}

export function parseStoryboardImportText(raw: string): StoryboardImportParseResult {
    const text = raw.replace(/^\uFEFF/, "").trim();
    if (!text) return { rows: [], warnings: ["文件是空的"] };

    if (text.startsWith("{") || text.startsWith("[")) {
        try {
            return draftsFromJson(JSON.parse(text));
        } catch {
            return { rows: [], warnings: ["JSON 无法解析，请检查格式"] };
        }
    }

    const table = text.includes("|") && looksLikeMarkdownTable(text) ? parseMarkdownTable(text) : parseDelimitedTable(text);
    if (table.length < 2) return { rows: [], warnings: ["没有读到表头和镜头行。请使用下载的 CSV 模板，或从 Excel 复制带表头的单元格。"] };

    const headerMap = mapHeaders(table[0]);
    if (!headerMap.length) return { rows: [], warnings: ["表头无法识别。请保留模板中的「序号 / 时长 / 视频提示词 / 台词/旁白 / 关联资产」列名。"] };

    const warnings: string[] = [];
    if (!headerMap.some((item) => item.field === "videoMotionPrompt" || item.field === "plotDescription" || item.field === "imageGenerationPrompt")) {
        warnings.push("没有识别到提示词列，导入后镜头可能是空的");
    }

    const rows = table.slice(1).flatMap((cells, index) => {
        if (cells.every((cell) => !cell.trim())) return [];
        const record = Object.fromEntries(headerMap.map(({ field, column }) => [field, cells[column] || ""])) as Partial<Record<ImportField, string>>;
        return [draftFromRecord(record, index)];
    });

    if (!rows.length) warnings.push("表头下面没有有效镜头行");
    return { rows, warnings };
}

export function applyStoryboardImport(drafts: StoryboardImportDraftRow[], nodes: CanvasNodeData[]): StoryboardImportApplyResult {
    const catalog = buildImportAssetCatalog(nodes);
    const unmatched = new Set<string>();
    let matchedAssetCount = 0;
    const rows = drafts.map((draft, index) => {
        const bindings: StoryboardAssetBinding[] = [];
        const seen = new Set<string>();
        const names = [...draft.assetNames, ...draft.characterNames];
        names.forEach((name) => {
            const match = matchImportAsset(name, catalog);
            if (!match) {
                unmatched.add(name);
                return;
            }
            if (seen.has(match.node.id)) return;
            seen.add(match.node.id);
            const binding = bindingForConnectedNode(match.node);
            if (binding) {
                bindings.push(binding);
                matchedAssetCount += 1;
            }
        });
        const characters: StoryboardCharacterReference[] = draft.characterNames.map((characterName) => {
            const match = matchImportAsset(characterName, catalog.filter((item) => item.kind === "character"));
            return {
                characterName,
                characterAssetId: match?.node.metadata?.characterAssetId,
                characterVersionId: match?.node.metadata?.characterVersionId,
                characterImageNodeId: match?.node.id,
            };
        });
        return createStoryboardRow(index + 1, {
            durationSeconds: draft.durationSeconds,
            videoMotionPrompt: draft.videoMotionPrompt,
            dialogue: draft.dialogue,
            imageGenerationPrompt: draft.imageGenerationPrompt,
            plotDescription: draft.plotDescription,
            camera: draft.camera,
            motion: draft.motion,
            shotSize: draft.shotSize,
            negativePrompt: draft.negativePrompt,
            narrativeIntent: draft.narrativeIntent,
            lightingAndAtmosphere: draft.lightingAndAtmosphere,
            timeBeats: draft.timeBeats,
            continuityOut: draft.continuityOut,
            assetBindings: bindings,
            characters,
            shotNumber: draft.shotNumber || index + 1,
        });
    }).map((row, index) => ({ ...row, shotNumber: index + 1 }));
    return { rows, unmatchedAssets: Array.from(unmatched), matchedAssetCount };
}

function draftFromRecord(record: Partial<Record<ImportField, string>>, index: number): StoryboardImportDraftRow {
    const shotNumber = Math.max(1, Math.round(Number(record.shotNumber) || index + 1));
    const durationSeconds = Math.max(1, Math.min(60, Math.round(Number(record.durationSeconds) || 6)));
    return {
        shotNumber,
        durationSeconds,
        videoMotionPrompt: compactCell(record.videoMotionPrompt),
        dialogue: compactCell(record.dialogue),
        imageGenerationPrompt: compactCell(record.imageGenerationPrompt),
        plotDescription: compactCell(record.plotDescription),
        camera: compactCell(record.camera),
        motion: compactCell(record.motion),
        shotSize: compactCell(record.shotSize),
        negativePrompt: compactCell(record.negativePrompt),
        narrativeIntent: compactCell(record.narrativeIntent),
        lightingAndAtmosphere: compactCell(record.lightingAndAtmosphere),
        timeBeats: compactCell(record.timeBeats),
        continuityOut: compactCell(record.continuityOut),
        assetNames: splitNames(record.assets),
        characterNames: splitNames(record.characters),
    };
}

function draftsFromJson(value: unknown): StoryboardImportParseResult {
    const source = Array.isArray(value) ? value : value && typeof value === "object" && Array.isArray((value as { rows?: unknown }).rows) ? (value as { rows: unknown[] }).rows : null;
    if (!source) return { rows: [], warnings: ["JSON 需要是镜头数组，或带 rows 数组的对象"] };
    const rows = source.flatMap((item, index) => {
        if (!item || typeof item !== "object") return [];
        const record = item as Record<string, unknown>;
        return [draftFromRecord({
            shotNumber: String(record.shotNumber ?? record["序号"] ?? ""),
            durationSeconds: String(record.durationSeconds ?? record["时长"] ?? ""),
            videoMotionPrompt: String(record.videoMotionPrompt ?? record["视频提示词"] ?? ""),
            dialogue: String(record.dialogue ?? record["台词/旁白"] ?? record["台词"] ?? ""),
            assets: Array.isArray(record.assets) ? record.assets.map(String).join(",") : String(record.assets ?? record["关联资产"] ?? ""),
            imageGenerationPrompt: String(record.imageGenerationPrompt ?? record["图片提示词"] ?? ""),
            plotDescription: String(record.plotDescription ?? record["画面描述"] ?? ""),
            characters: Array.isArray(record.characters)
                ? record.characters.map((character) => (typeof character === "string" ? character : String((character as { characterName?: string }).characterName || ""))).join(",")
                : String(record.characters ?? record["角色"] ?? ""),
            camera: String(record.camera ?? ""),
            motion: String(record.motion ?? ""),
            shotSize: String(record.shotSize ?? ""),
            negativePrompt: String(record.negativePrompt ?? ""),
            narrativeIntent: String(record.narrativeIntent ?? ""),
            lightingAndAtmosphere: String(record.lightingAndAtmosphere ?? ""),
            timeBeats: String(record.timeBeats ?? ""),
            continuityOut: String(record.continuityOut ?? ""),
        }, index)];
    });
    return { rows, warnings: rows.length ? [] : ["JSON 里没有有效镜头"] };
}

function mapHeaders(headers: string[]) {
    return headers.flatMap((header, column) => {
        const field = HEADER_ALIASES[normalizeHeader(header)];
        return field ? [{ field, column }] : [];
    });
}

function normalizeHeader(value: string) {
    return value.replace(/\s+/g, "").replace(/[／]/g, "/").toLowerCase();
}

function splitNames(value?: string) {
    return Array.from(new Set((value || "").split(/[,，、/;；|]+/).map((item) => item.trim()).filter(Boolean)));
}

function compactCell(value?: string) {
    return (value || "").replace(/\r/g, "").trim();
}

type ImportAsset = { node: CanvasNodeData; title: string; aliases: string[]; kind: "character" | "other" };

function buildImportAssetCatalog(nodes: CanvasNodeData[]): ImportAsset[] {
    return nodes.flatMap((node) => {
        if (!bindingForConnectedNode(node)) return [];
        const aliases = [
            node.title,
            node.metadata?.assetCategory || "",
            ...(node.metadata?.assetTags || []),
            node.metadata?.workflowKind === "character" ? "角色" : "",
        ].map((item) => item.trim()).filter(Boolean);
        return [{
            node,
            title: node.title || "未命名资产",
            aliases,
            kind: node.metadata?.workflowKind === "character" || node.metadata?.assetCategory === "character" ? "character" : "other",
        }];
    });
}

function matchImportAsset(name: string, catalog: ImportAsset[]) {
    const query = normalizeMatch(name);
    if (!query) return undefined;
    const exact = catalog.filter((item) => normalizeMatch(item.title) === query || item.aliases.some((alias) => normalizeMatch(alias) === query));
    if (exact.length) return exact[0];
    const fuzzy = catalog.filter((item) => {
        const title = normalizeMatch(item.title);
        return title.includes(query) || query.includes(title) || item.aliases.some((alias) => {
            const value = normalizeMatch(alias);
            return value.includes(query) || query.includes(value);
        });
    });
    return fuzzy.length === 1 ? fuzzy[0] : undefined;
}

function normalizeMatch(value: string) {
    return value.replace(/\s+/g, "").toLowerCase();
}

export function parseDelimitedTable(text: string): string[][] {
    const source = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
    const firstLine = source.split("\n")[0] || "";
    const delimiter = delimiterOf(firstLine);
    const rows: string[][] = [];
    let row: string[] = [];
    let cell = "";
    let quoted = false;
    for (let index = 0; index < source.length; index += 1) {
        const char = source[index];
        const next = source[index + 1];
        if (quoted) {
            if (char === '"' && next === '"') {
                cell += '"';
                index += 1;
            } else if (char === '"') {
                quoted = false;
            } else {
                cell += char;
            }
            continue;
        }
        if (char === '"') {
            quoted = true;
            continue;
        }
        if (char === delimiter) {
            row.push(cell);
            cell = "";
            continue;
        }
        if (char === "\n") {
            row.push(cell);
            rows.push(row);
            row = [];
            cell = "";
            continue;
        }
        cell += char;
    }
    row.push(cell);
    if (row.some((item) => item.length)) rows.push(row);
    return rows;
}

function delimiterOf(line: string) {
    const tabs = (line.match(/\t/g) || []).length;
    const commas = (line.match(/,/g) || []).length;
    return tabs > commas ? "\t" : ",";
}

function looksLikeMarkdownTable(text: string) {
    const lines = text.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
    return lines.length >= 2 && lines[0].includes("|") && /^\|?[\s:|-]+$/.test(lines[1]) && lines[1].includes("-");
}

function parseMarkdownTable(text: string): string[][] {
    return text.split(/\r?\n/).map((line) => line.trim()).filter((line) => line.includes("|") && !/^\|?[\s:|-]+$/.test(line)).map((line) => {
        const cells = line.split("|").map((cell) => cell.trim());
        if (cells[0] === "") cells.shift();
        if (cells.at(-1) === "") cells.pop();
        return cells;
    });
}

export function serializeCsv(rows: string[][]) {
    return rows.map((row) => row.map(escapeCsvCell).join(",")).join("\n");
}

function escapeCsvCell(value: string) {
    return /[",\n]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value;
}
