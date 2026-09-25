import { describe, expect, test } from "bun:test";

import { availableWorkspaceSkins, resolveWorkspaceSkin } from "../src/lib/workspace-skin";
import { DEFAULT_PUBLIC_APPEARANCE } from "../src/stores/use-appearance-store";

describe("workspace skin picker", () => {
    test("falls back to the official cinematic default when a stored id is gone", () => {
        const skins = DEFAULT_PUBLIC_APPEARANCE.workspaceSkins || [];
        const appearance = {
            ...DEFAULT_PUBLIC_APPEARANCE,
            skinId: "apex",
            activeSkin: skins[0],
            workspaceSkins: skins,
        };
        expect(resolveWorkspaceSkin(appearance, "classic").id).toBe("apex");
        expect(resolveWorkspaceSkin(appearance, "nexus").id).toBe("nexus");
        expect(availableWorkspaceSkins(appearance).map((skin) => skin.id)).toEqual([
            "apex", "nexus", "lumen", "prism", "radix", "helix", "ink", "ember", "veil", "echo",
        ]);
        expect(new Set(skins.map((skin) => skin.tokens.dark.canvas)).size).toBe(skins.length);
        expect(new Set(skins.map((skin) => skin.tokens.dark.primary)).size).toBeGreaterThan(6);
        expect(skins.find((skin) => skin.id === "prism")?.tokens.dark.primary).toBe("#c4b5ff");
    });
});
