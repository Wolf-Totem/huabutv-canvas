const RELOAD_KEY = "canvas.chunk-reload";

export function isChunkLoadError(error: unknown) {
    const message = error instanceof Error ? error.message : String(error || "");
    return /Failed to fetch dynamically imported module|Importing a module script failed|error loading dynamically imported module|Loading chunk \S+ failed|Unable to preload CSS/i.test(message);
}

export function reloadForStaleChunk(error?: unknown) {
    if (error && !isChunkLoadError(error)) return false;
    try {
        if (sessionStorage.getItem(RELOAD_KEY) === "1") return false;
        sessionStorage.setItem(RELOAD_KEY, "1");
    } catch {
        return false;
    }
    window.location.reload();
    return true;
}

export function clearChunkReloadGuard() {
    try {
        sessionStorage.removeItem(RELOAD_KEY);
    } catch {
        /* ignore */
    }
}

export function importWithChunkRecovery<T>(loader: () => Promise<T>): Promise<T> {
    return loader().catch((error) => {
        if (reloadForStaleChunk(error)) return new Promise<T>(() => undefined);
        throw error;
    });
}
