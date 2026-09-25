import { create } from "zustand";

import { WORKSPACE_SKIN_STORAGE_KEY, readStoredWorkspaceSkinId, writeStoredWorkspaceSkinId } from "@/lib/workspace-skin";

type WorkspaceSkinStore = {
    skinId: string;
    setSkinId: (skinId: string) => void;
};

export const useWorkspaceSkinStore = create<WorkspaceSkinStore>((set) => ({
    skinId: typeof window === "undefined" ? "" : readStoredWorkspaceSkinId(),
    setSkinId: (skinId) => {
        const next = String(skinId || "").trim();
        writeStoredWorkspaceSkinId(next);
        set({ skinId: next });
    },
}));

export { WORKSPACE_SKIN_STORAGE_KEY };
