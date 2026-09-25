import type { TFunction } from "i18next";

import type { SupportedLocale } from "@/i18n/languages";

const LOCALE_TAGS: Record<string, string> = {
    zh: "zh-CN",
    en: "en",
    id: "id-ID",
    vi: "vi-VN",
    th: "th-TH",
    fil: "fil-PH",
    ms: "ms-MY",
};

export function localeTag(code: string | undefined | null) {
    const normalized = String(code || "").split("-")[0];
    return LOCALE_TAGS[normalized] || "en";
}

export function pluginDisplayName(t: TFunction, id: string, fallback: string) {
    return t(`plugin.item.${id}.name`, { defaultValue: fallback });
}

export function pluginDisplayDescription(t: TFunction, id: string, name: string, fallback: string) {
    const specific = t(`plugin.item.${id}.description`, { defaultValue: "" });
    if (specific) return specific;
    if (fallback.includes("独立请求协议插件")) return t("plugin.protocolDescription", { name });
    return fallback;
}

export function htmlLang(locale: SupportedLocale) {
    return locale === "zh" ? "zh-CN" : locale;
}
