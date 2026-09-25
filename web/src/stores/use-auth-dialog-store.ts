import { create } from "zustand";

export type AuthDialogTab = "login" | "register";

type AuthDialogState = {
    open: boolean;
    tab: AuthDialogTab;
    next: string;
    openAuth: (input?: { tab?: AuthDialogTab; next?: string }) => void;
    setTab: (tab: AuthDialogTab) => void;
    closeAuth: () => void;
};

function safeNext(value?: string) {
    const next = String(value || "").trim();
    if (!next.startsWith("/") || next.startsWith("//")) {
        if (typeof window !== "undefined") return `${window.location.pathname}${window.location.search}` || "/";
        return "/";
    }
    return next;
}

export const useAuthDialogStore = create<AuthDialogState>()((set) => ({
    open: false,
    tab: "login",
    next: "/",
    openAuth: (input) => set({ open: true, tab: input?.tab || "login", next: safeNext(input?.next) }),
    setTab: (tab) => set({ tab }),
    closeAuth: () => set({ open: false }),
}));
