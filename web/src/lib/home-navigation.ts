export type HomeNavItem = {
    label: string;
    href: string;
    openInNewTab?: boolean;
};

export const DEFAULT_HOME_NAV_ITEMS: HomeNavItem[] = [
    { label: "工作台", href: "#workbench" },
    { label: "贡献者", href: "#contributors" },
    { label: "GitHub", href: "https://github.com/ddcat-ai/open-ai-canvas", openInNewTab: true },
];

export const DEFAULT_HOME_CTA_LABEL = "开始创作";
export const DEFAULT_HOME_CTA_HREF = "/create";

export function normalizeHomeNavItems(items?: HomeNavItem[] | null): HomeNavItem[] {
    if (!items) return DEFAULT_HOME_NAV_ITEMS.map((item) => ({ ...item }));
    return items
        .map((item) => ({
            label: String(item?.label || "").trim(),
            href: String(item?.href || "").trim(),
            openInNewTab: Boolean(item?.openInNewTab),
        }))
        .filter((item) => item.label || item.href);
}

export function validHomeHref(raw: string) {
    const value = String(raw || "").trim();
    if (!value || value.length > 300 || /[\s]/.test(value)) return false;
    const lower = value.toLowerCase();
    if (lower.startsWith("javascript:") || lower.startsWith("data:") || lower.startsWith("vbscript:")) return false;
    if (value.startsWith("#")) return value.length > 1;
    if (value.startsWith("/") && !value.startsWith("//")) return true;
    try {
        const parsed = new URL(value);
        return parsed.protocol === "http:" || parsed.protocol === "https:";
    } catch {
        return false;
    }
}
