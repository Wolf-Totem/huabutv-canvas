import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

import { resolveInitialLocale, shouldPromptIpLocaleChoice } from "../src/i18n/languages";

const langs = ["zh", "en", "id", "vi", "th", "fil", "ms"];
const nss = ["common", "sidebar", "canvas", "setting"];
const root = join(import.meta.dir, "../../backend/internal/locale/catalog");

describe("i18n catalogs", () => {
    test("every locale has required namespaces and language prompt keys", () => {
        for (const lng of langs) {
            for (const ns of nss) {
                const json = JSON.parse(readFileSync(join(root, lng, `${ns}.json`), "utf8"));
                expect(typeof json).toBe("object");
            }
            const common = JSON.parse(readFileSync(join(root, lng, "common.json"), "utf8"));
            expect(common["language.prompt"]).toContain("{{saved}}");
            expect(common["language.useSaved"]).toBeTruthy();
            expect(common["language.useRecommended"]).toBeTruthy();
            const setting = JSON.parse(readFileSync(join(root, lng, "setting.json"), "utf8"));
            expect(setting["channel.connection.jiasu"]).toBeTruthy();
            expect(setting["channel.preset.jiasuName"]).toBeTruthy();
            expect(setting["channel.connectionType"]).toBeTruthy();
            const email = JSON.parse(readFileSync(join(root, lng, "email.json"), "utf8"));
            expect(email["register.subject"]).toContain("{{brand}}");
            expect(email["reset.title"]).toBeTruthy();
        }
    });

    test("IP language prompt stays off unless the admin switch is enabled", () => {
        expect(shouldPromptIpLocaleChoice(false, "zh", "en")).toBe(false);
        expect(shouldPromptIpLocaleChoice(true, "zh", "en")).toBe(true);
        expect(shouldPromptIpLocaleChoice(true, "zh", "zh")).toBe(false);
        expect(shouldPromptIpLocaleChoice(true, "", "en")).toBe(false);
        expect(resolveInitialLocale({ ipLocalePromptEnabled: false, recommended: "en" })).toBe("zh");
        expect(resolveInitialLocale({ ipLocalePromptEnabled: true, recommended: "en" })).toBe("en");
        expect(resolveInitialLocale({ ipLocalePromptEnabled: false, visitorLocale: "id", recommended: "en" })).toBe("id");
        expect(resolveInitialLocale({ ipLocalePromptEnabled: true, userLocale: "zh", recommended: "en" })).toBe("zh");
    });
});
