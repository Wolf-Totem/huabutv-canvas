import { CINEMATIC_SKIN_STORAGE_KEY, DEFAULT_CINEMATIC_SKIN_ID, cinematicSkinMeta, type CinematicSkinMeta } from "@/lib/cinematic-skins";
import type { PublicAppearance } from "@/services/api/appearance";
import type { SkinDefinition } from "@/lib/skin-themes";

export type LoaderSkin = {
    id: string;
    name: string;
    film?: string;
    poster?: string;
    fx?: string;
    hue?: number;
};

function withCinematicMeta(skin: Pick<SkinDefinition, "id" | "name" | "film" | "poster" | "fx" | "hue"> | LoaderSkin): LoaderSkin {
    const meta = cinematicSkinMeta(skin.id);
    return {
        id: skin.id,
        name: skin.name || meta?.nameZh || skin.id,
        film: skin.film || meta?.film,
        poster: skin.poster || meta?.poster,
        fx: skin.fx || meta?.fx,
        hue: skin.hue ?? meta?.hue,
    };
}

export function enabledLoaderSkins(appearance: PublicAppearance): LoaderSkin[] {
    const listed = appearance.enabledSkins?.length ? appearance.enabledSkins : [];
    return listed.map(withCinematicMeta).filter((skin) => Boolean(skin.film || skin.poster));
}

export function resolveLoaderSkin(appearance: PublicAppearance): LoaderSkin | null {
    const enabled = enabledLoaderSkins(appearance);
    if (!enabled.length) return null;
    let local = "";
    try {
        local = window.localStorage.getItem(CINEMATIC_SKIN_STORAGE_KEY) || "";
    } catch {
        local = "";
    }
    const fromLocal = enabled.find((skin) => skin.id === local);
    if (fromLocal) return fromLocal;
    const active = withCinematicMeta(appearance.activeSkin);
    const fromActive = enabled.find((skin) => skin.id === active.id);
    if (fromActive) return fromActive;
    return enabled.find((skin) => skin.id === DEFAULT_CINEMATIC_SKIN_ID) || enabled[0];
}

export function cinematicFallback(id = DEFAULT_CINEMATIC_SKIN_ID): CinematicSkinMeta {
    return cinematicSkinMeta(id) || cinematicSkinMeta(DEFAULT_CINEMATIC_SKIN_ID)!;
}
