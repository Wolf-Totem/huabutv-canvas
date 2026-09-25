import { describe, expect, test } from "bun:test";

import { COUNTRY_LANGUAGE_MAP, FALLBACK_LOCALE, isSupportedLocale, normalizeLocale } from "../src/i18n/languages";

describe("locale mapping", () => {
    test("maps configured countries", () => {
        expect(COUNTRY_LANGUAGE_MAP.CN).toBe("zh");
        expect(COUNTRY_LANGUAGE_MAP.PH).toBe("fil");
    });

    test("normalizes aliases", () => {
        expect(normalizeLocale("zh-CN")).toBe("zh");
        expect(normalizeLocale("tl")).toBe("fil");
        expect(isSupportedLocale("ms")).toBe(true);
        expect(isSupportedLocale("ja")).toBe(false);
        expect(FALLBACK_LOCALE).toBe("zh");
    });
});
