import { useEffect } from "react";
import { createPortal } from "react-dom";

import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import LoginPage from "@/pages/auth/login";
import RegisterPage from "@/pages/auth/register";
import "./auth-dialog.css";

export function AuthDialogHost() {
    const open = useAuthDialogStore((state) => state.open);
    const tab = useAuthDialogStore((state) => state.tab);
    const setTab = useAuthDialogStore((state) => state.setTab);
    const closeAuth = useAuthDialogStore((state) => state.closeAuth);
    const openAuth = useAuthDialogStore((state) => state.openAuth);

    useEffect(() => {
        const params = new URLSearchParams(window.location.search);
        const auth = params.get("auth");
        if (auth === "login" || auth === "register") {
            openAuth({ tab: auth, next: params.get("next") || undefined });
            params.delete("auth");
            const query = params.toString();
            window.history.replaceState(null, "", `${window.location.pathname}${query ? `?${query}` : ""}${window.location.hash}`);
        }
        if (params.get("oauth_error")) openAuth({ tab: "login" });
    }, [openAuth]);

    useEffect(() => {
        document.documentElement.classList.toggle("mc-auth-open", open);
        if (!open) return;
        const onKey = (event: KeyboardEvent) => {
            if (event.key === "Escape") closeAuth();
        };
        window.addEventListener("keydown", onKey);
        return () => {
            document.documentElement.classList.remove("mc-auth-open");
            window.removeEventListener("keydown", onKey);
        };
    }, [closeAuth, open]);

    if (!open || typeof document === "undefined") return null;
    return createPortal(
        <div className="portal-auth-overlay" role="presentation" onClick={closeAuth}>
            <div className="portal-auth-card" role="dialog" aria-modal="true" aria-labelledby="portal-auth-title" onClick={(event) => event.stopPropagation()}>
                <div className="portal-auth-tabs">
                    <button type="button" className={tab === "login" ? "is-active" : undefined} onClick={() => setTab("login")}>登录</button>
                    <button type="button" className={tab === "register" ? "is-active" : undefined} onClick={() => setTab("register")}>注册</button>
                    <button type="button" className="portal-auth-close" onClick={closeAuth} aria-label="关闭">×</button>
                </div>
                <h2 id="portal-auth-title" className="portal-auth-title">{tab === "login" ? "登录画布TV" : "注册画布TV"}</h2>
                {tab === "login" ? <LoginPage embedded /> : <RegisterPage embedded />}
            </div>
        </div>,
        document.body,
    );
}
