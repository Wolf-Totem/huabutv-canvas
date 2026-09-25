import type { ReactNode } from "react";
import { useEffect } from "react";

import { getAuthSession, type AuthSessionPayload } from "@/services/api/auth";
import { FullScreenLoader } from "@/components/ui/aceternity/full-screen-loader";
import { preloadWorkspaceRoute } from "@/lib/workspace-route-modules";
import { useUserStore } from "@/stores/use-user-store";

function isPublicLandingPath() {
    const path = window.location.pathname.replace(/\/+$/, "") || "/";
    return path === "/";
}

export function AuthSessionHydrator({ children }: { children: ReactNode }) {
    const hydrated = useUserStore((state) => state.hydrated);
    const publicLanding = isPublicLandingPath();

    useEffect(() => {
        let cancelled = false;
        getAuthSession()
            .then(async (payload) => {
                if (cancelled) return;
                if (!payload.user) {
                    applyAnonymousSession(payload);
                    return;
                }
                if (publicLanding) {
                    const store = useUserStore.getState();
                    store.setUser(payload.user);
                    store.setPermissions(payload.permissions);
                    store.setRuntimeLimits(payload.runtimeLimits);
                    store.setDrawingEngine(payload.drawingEngine);
                    store.setFeatures(payload.features);
                    store.setMembership(payload.membership);
                    store.setHydrated(true);
                    return;
                }
                const { applyUserSession } = await import("@/lib/user-session");
                if (cancelled) return;
                await applyUserSession(payload);
                preloadWorkspaceRoute(window.location.pathname);
            })
            .catch(() => {
                if (!cancelled) applyAnonymousSession({ user: null, logicalModels: [] });
            });
        return () => {
            cancelled = true;
        };
    }, [publicLanding]);

    if (publicLanding) return children;
    return hydrated ? children : <FullScreenLoader label="正在打开" detail="读取登录状态" />;
}

function applyAnonymousSession(payload: AuthSessionPayload) {
    const store = useUserStore.getState();
    store.clearSession();
    store.setRuntimeLimits(payload.runtimeLimits);
    store.setDrawingEngine(payload.drawingEngine);
    store.setFeatures(payload.features);
    store.setHydrated(true);
}
