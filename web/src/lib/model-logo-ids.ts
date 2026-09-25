/** 后台编辑模型时优先展示的主流品牌 Logo（LobeHub 官方图标 ID）。 */
export const FEATURED_MODEL_LOGOS: Array<{ id: string; aliases: string }> = [
    { id: "OpenAI", aliases: "openai chatgpt gpt dall-e dalle" },
    { id: "Minimax", aliases: "minimax hailuo 海螺" },
    { id: "Volcengine", aliases: "volcengine 火山引擎 火山 ark" },
    { id: "Doubao", aliases: "doubao 豆包 dola seedance seedream" },
    { id: "Vidu", aliases: "vidu" },
    { id: "Grok", aliases: "grok xai" },
];

export function inferBrandIcon(modelName?: string): string {
    const name = String(modelName || "").trim().toLowerCase();
    if (!name) return "";
    if (name.includes("gpt") || name.includes("openai") || name.includes("dall")) return "OpenAI";
    if (name.includes("minimax") || name.includes("hailuo") || name.includes("h3-a")) return "Minimax";
    if (name.includes("vidu")) return "Vidu";
    if (name.includes("grok") || name.includes("xai")) return "Grok";
    if (name.includes("seedance") || name.includes("seedream") || name.includes("doubao") || name.includes("dola")) return "Doubao";
    if (name.includes("volc") || name.includes("jimeng") || name.includes("ark")) return "Volcengine";
    return "";
}
