export const CINEMATIC_SKIN_IDS = ["apex", "nexus", "lumen", "prism", "radix", "helix", "ink", "ember", "veil", "echo"] as const;
export type CinematicSkinID = (typeof CINEMATIC_SKIN_IDS)[number];

export type CinematicSkinMeta = {
    id: CinematicSkinID;
    name: string;
    nameZh: string;
    film: string;
    poster: string;
    fx: "rings" | "radar" | "rain" | "shards" | "scan" | "helix" | "ink" | "sparks" | "aurora" | "sonar";
    hue: number;
};

/** Matches the grok.me/studio skin tokens: canvas/surface/text/muted/accent/onAccent. */
export type CinematicPalette = {
    canvas: string;
    surface: string;
    text: string;
    muted: string;
    primary: string;
    onPrimary: string;
};

export const CINEMATIC_PALETTES: Record<CinematicSkinID, { light: CinematicPalette; dark: CinematicPalette }> = {
    apex: {
        dark: { canvas: "#070709", surface: "#1a1a20", text: "#f4f1ea", muted: "#8a8680", primary: "#d9e6f2", onPrimary: "#0b0c10" },
        light: { canvas: "#f3efe6", surface: "#fffdf8", text: "#1a1814", muted: "#6f6a64", primary: "#2c333c", onPrimary: "#f4f1ea" },
    },
    nexus: {
        dark: { canvas: "#031018", surface: "#0c2230", text: "#d8f6ff", muted: "#6a93a4", primary: "#5ee7ff", onPrimary: "#032028" },
        light: { canvas: "#dff4fa", surface: "#ffffff", text: "#073040", muted: "#4a7380", primary: "#0490a8", onPrimary: "#ffffff" },
    },
    lumen: {
        dark: { canvas: "#08080c", surface: "#1b1b24", text: "#f5f5f7", muted: "#8b8b99", primary: "#4d7dff", onPrimary: "#ffffff" },
        light: { canvas: "#f6f7f9", surface: "#ffffff", text: "#171717", muted: "#6b7280", primary: "#2563eb", onPrimary: "#ffffff" },
    },
    prism: {
        dark: { canvas: "#0e0a18", surface: "#221a34", text: "#f0eaff", muted: "#9a90b8", primary: "#c4b5ff", onPrimary: "#1a1230" },
        light: { canvas: "#f5f2fb", surface: "#ffffff", text: "#211b35", muted: "#716a86", primary: "#6656d9", onPrimary: "#ffffff" },
    },
    radix: {
        dark: { canvas: "#051410", surface: "#123028", text: "#d8fff0", muted: "#6fa392", primary: "#3ee0b0", onPrimary: "#042018" },
        light: { canvas: "#eaf6f2", surface: "#ffffff", text: "#142026", muted: "#4e6c62", primary: "#087f76", onPrimary: "#ffffff" },
    },
    helix: {
        dark: { canvas: "#070c18", surface: "#162244", text: "#dce8ff", muted: "#7d8eaa", primary: "#6ea8ff", onPrimary: "#071028" },
        light: { canvas: "#e8eef8", surface: "#ffffff", text: "#12182a", muted: "#5a6780", primary: "#1d4ed8", onPrimary: "#ffffff" },
    },
    ink: {
        dark: { canvas: "#12100e", surface: "#26201a", text: "#f3eadc", muted: "#8a8278", primary: "#e8dcc8", onPrimary: "#1a1612" },
        light: { canvas: "#f4ecde", surface: "#fffaf0", text: "#1c1814", muted: "#6e655c", primary: "#1c1814", onPrimary: "#fffaf0" },
    },
    ember: {
        dark: { canvas: "#140c08", surface: "#301c12", text: "#f8e6d2", muted: "#a08870", primary: "#e08a4a", onPrimary: "#2a1008" },
        light: { canvas: "#fbf4eb", surface: "#fffdf9", text: "#35261f", muted: "#806b61", primary: "#b94f2f", onPrimary: "#fffaf6" },
    },
    veil: {
        dark: { canvas: "#121214", surface: "#222228", text: "#ececec", muted: "#8a8d94", primary: "#c8ccd4", onPrimary: "#121214" },
        light: { canvas: "#ececef", surface: "#fbfbfc", text: "#1c1c20", muted: "#6a6d74", primary: "#3a3c42", onPrimary: "#ffffff" },
    },
    echo: {
        dark: { canvas: "#061016", surface: "#142834", text: "#d4f0ff", muted: "#6f8fa4", primary: "#3ec8e8", onPrimary: "#041820" },
        light: { canvas: "#e4f1f2", surface: "#ffffff", text: "#0e2a32", muted: "#4e6e76", primary: "#0e7490", onPrimary: "#ffffff" },
    },
};

export const CINEMATIC_SKINS: readonly CinematicSkinMeta[] = [
    { id: "apex", name: "Apex", nameZh: "奇点", film: "/bg/skin-apex.mp4", poster: "/bg/skin-apex.jpg", fx: "rings", hue: 38 },
    { id: "nexus", name: "Nexus", nameZh: "冰核", film: "/bg/skin-nexus.mp4", poster: "/bg/skin-nexus.jpg", fx: "radar", hue: 192 },
    { id: "lumen", name: "Lumen", nameZh: "霓虹", film: "/bg/skin-lumen.mp4", poster: "/bg/skin-lumen.jpg", fx: "rain", hue: 312 },
    { id: "prism", name: "Prism", nameZh: "棱镜", film: "/bg/skin-prism.mp4", poster: "/bg/skin-prism.jpg", fx: "shards", hue: 268 },
    { id: "radix", name: "Radix", nameZh: "全息", film: "/bg/skin-radix.mp4", poster: "/bg/skin-radix.jpg", fx: "scan", hue: 168 },
    { id: "helix", name: "Helix", nameZh: "螺旋", film: "/bg/skin-helix.mp4", poster: "/bg/skin-helix.jpg", fx: "helix", hue: 262 },
    { id: "ink", name: "Ink", nameZh: "墨核", film: "/bg/skin-ink.mp4", poster: "/bg/skin-ink.jpg", fx: "ink", hue: 220 },
    { id: "ember", name: "Ember", nameZh: "熔核", film: "/bg/skin-ember.mp4", poster: "/bg/skin-ember.jpg", fx: "sparks", hue: 22 },
    { id: "veil", name: "Veil", nameZh: "极光", film: "/bg/skin-veil.mp4", poster: "/bg/skin-veil.jpg", fx: "aurora", hue: 148 },
    { id: "echo", name: "Echo", nameZh: "深渊", film: "/bg/skin-echo.mp4", poster: "/bg/skin-echo.jpg", fx: "sonar", hue: 222 },
];

export const DEFAULT_CINEMATIC_SKIN_ID: CinematicSkinID = "apex";
export const DEFAULT_APPEARANCE_MODE = "dark" as const;
export const CINEMATIC_SKIN_STORAGE_KEY = "canvas.loader.skin";

export function cinematicSkinMeta(id: string | undefined | null) {
    return CINEMATIC_SKINS.find((skin) => skin.id === id) || null;
}

export function isCinematicSkinID(value: string | undefined | null): value is CinematicSkinID {
    return CINEMATIC_SKINS.some((skin) => skin.id === value);
}
