import { getPublicAppearance, type PublicAppearance } from "@/services/api/appearance";
import { commitPublicAppearance, DEFAULT_PUBLIC_APPEARANCE } from "@/stores/use-appearance-store";

const APPEARANCE_BOOTSTRAP_TIMEOUT_MS = 4_000;
export const APPEARANCE_CACHE_KEY = "infinite-canvas:public-appearance";

export function restoreCachedAppearance() {
    if (typeof localStorage === "undefined") return false;
    try {
        const raw = localStorage.getItem(APPEARANCE_CACHE_KEY);
        if (!raw) return false;
        const parsed = JSON.parse(raw) as PublicAppearance;
        if (!parsed || typeof parsed !== "object") return false;
        commitPublicAppearance(parsed);
        return true;
    } catch {
        return false;
    }
}

function cachePublicAppearance(appearance: PublicAppearance) {
    if (typeof localStorage === "undefined") return;
    try {
        localStorage.setItem(APPEARANCE_CACHE_KEY, JSON.stringify(appearance));
    } catch {
        // 配额满或隐私模式：忽略，下次仍走网络。
    }
}

export async function resolvePublicAppearance(fetchAppearance: (signal: AbortSignal) => Promise<PublicAppearance> = getPublicAppearance) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), APPEARANCE_BOOTSTRAP_TIMEOUT_MS);
    try {
        return await fetchAppearance(controller.signal);
    } catch {
        return DEFAULT_PUBLIC_APPEARANCE;
    } finally {
        clearTimeout(timer);
    }
}

export async function bootstrapAppearance(fetchAppearance?: (signal: AbortSignal) => Promise<PublicAppearance>) {
    const appearance = commitPublicAppearance(await resolvePublicAppearance(fetchAppearance));
    cachePublicAppearance(appearance);
    return appearance;
}
