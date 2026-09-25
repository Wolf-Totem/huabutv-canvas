export const SUPPORTED_LOCALES = ["zh", "en", "id", "vi", "th", "fil", "ms"] as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

export const FALLBACK_LOCALE: SupportedLocale = "zh";
export const DEFAULT_OTHER_LOCALE: SupportedLocale = "en";
export const I18N_NAMESPACES = ["common", "sidebar", "canvas", "setting"] as const;

export const LOCALE_STORAGE_KEY = "canvas.locale";
export const LOCALE_CHOICE_SESSION_KEY = "canvas.locale.choice";

export function shouldPromptIpLocaleChoice(enabled: boolean, saved: string, recommended: string) {
    return enabled && Boolean(saved) && saved !== recommended;
}

export function resolveInitialLocale(input: { ipLocalePromptEnabled: boolean; userLocale?: string | null; visitorLocale?: string | null; recommended?: string | null }): SupportedLocale {
    const saved = normalizeLocale(input.userLocale);
    if (saved) return saved;
    const visitor = normalizeLocale(input.visitorLocale);
    if (visitor) return visitor;
    if (input.ipLocalePromptEnabled) {
        const recommended = normalizeLocale(input.recommended);
        if (recommended) return recommended;
    }
    return FALLBACK_LOCALE;
}

export const COUNTRY_LANGUAGE_MAP: Record<string, SupportedLocale> = {
    CN: "zh",
    ID: "id",
    TH: "th",
    VN: "vi",
    PH: "fil",
    MY: "ms",
};

export function isSupportedLocale(value: string | null | undefined): value is SupportedLocale {
    return SUPPORTED_LOCALES.includes(String(value || "") as SupportedLocale);
}

export function normalizeLocale(value: string | null | undefined): SupportedLocale | "" {
    const code = String(value || "").trim().toLowerCase();
    if (!code) return "";
    if (code.startsWith("zh")) return "zh";
    if (code.startsWith("en")) return "en";
    if (code.startsWith("id")) return "id";
    if (code.startsWith("vi")) return "vi";
    if (code.startsWith("th")) return "th";
    if (code === "tl" || code.startsWith("fil")) return "fil";
    if (code.startsWith("ms")) return "ms";
    return isSupportedLocale(code) ? code : "";
}
