import type { PublicAppearance } from "@/services/api/appearance";
import { DEFAULT_CLASSIC_SKIN, normalizeSkinDefinition, type SkinDefinition } from "@/lib/skin-themes";

export const WORKSPACE_SKIN_STORAGE_KEY = "open_ai_canvas:workspace_skin";

export function availableWorkspaceSkins(appearance: PublicAppearance): SkinDefinition[] {
    const listed = appearance.workspaceSkins?.length ? appearance.workspaceSkins : [appearance.activeSkin || DEFAULT_CLASSIC_SKIN];
    const skins = listed.map((skin) => normalizeSkinDefinition(skin)).filter((skin) => Boolean(skin.id));
    return skins.length ? skins : [DEFAULT_CLASSIC_SKIN];
}

export function readStoredWorkspaceSkinId() {
    try {
        return window.localStorage.getItem(WORKSPACE_SKIN_STORAGE_KEY) || "";
    } catch {
        return "";
    }
}

export function writeStoredWorkspaceSkinId(id: string) {
    try {
        if (id) window.localStorage.setItem(WORKSPACE_SKIN_STORAGE_KEY, id);
        else window.localStorage.removeItem(WORKSPACE_SKIN_STORAGE_KEY);
    } catch {
        // Ignore quota / private-mode failures; the in-memory store still applies.
    }
}

export function resolveWorkspaceSkin(appearance: PublicAppearance, selectedId = ""): SkinDefinition {
    const skins = availableWorkspaceSkins(appearance);
    const wanted = selectedId.trim() || readStoredWorkspaceSkinId();
    return skins.find((skin) => skin.id === wanted) || skins.find((skin) => skin.id === appearance.skinId) || appearance.activeSkin || skins[0] || DEFAULT_CLASSIC_SKIN;
}
