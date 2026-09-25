import { describe, expect, test } from "bun:test";

import { CINEMATIC_SKINS } from "../src/lib/cinematic-skins";
import { enabledLoaderSkins, resolveLoaderSkin } from "../src/lib/loader-skin";
import { DEFAULT_CLASSIC_SKIN } from "../src/lib/skin-themes";
import { DEFAULT_PUBLIC_APPEARANCE } from "../src/stores/use-appearance-store";

describe("loader cinematic skins", () => {
    test("catalog uses the required 10 ids and public asset paths", () => {
        expect(CINEMATIC_SKINS.map((skin) => skin.id)).toEqual(["apex", "nexus", "lumen", "prism", "radix", "helix", "ink", "ember", "veil", "echo"]);
        for (const skin of CINEMATIC_SKINS) {
            expect(skin.film).toBe(`/bg/skin-${skin.id}.mp4`);
            expect(skin.poster).toBe(`/bg/skin-${skin.id}.jpg`);
        }
    });

    test("disabled skins never appear on the restore loader", () => {
        const appearance = {
            ...DEFAULT_PUBLIC_APPEARANCE,
            activeSkin: DEFAULT_CLASSIC_SKIN,
            enabledSkins: CINEMATIC_SKINS.filter((skin) => skin.id === "apex" || skin.id === "ember").map((skin) => ({
                ...DEFAULT_CLASSIC_SKIN,
                id: skin.id,
                name: skin.nameZh,
                locked: false,
                film: skin.film,
                poster: skin.poster,
                fx: skin.fx,
                hue: skin.hue,
            })),
        };
        expect(enabledLoaderSkins(appearance).map((skin) => skin.id)).toEqual(["apex", "ember"]);
        expect(resolveLoaderSkin(appearance)?.id).toBe("apex");
    });
});
