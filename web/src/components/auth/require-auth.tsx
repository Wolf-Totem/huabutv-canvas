import type { ReactNode } from "react";
import { useEffect } from "react";
import { useLocation } from "react-router";

import { FullScreenLoader } from "@/components/ui/aceternity/full-screen-loader";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import { useUserStore } from "@/stores/use-user-store";

export function RequireAuth({ children }: { children: ReactNode }) {
    const location = useLocation();
    const hydrated = useUserStore((state) => state.hydrated);
    const user = useUserStore((state) => state.user);

    if (!hydrated) return <FullScreenLoader />;
    if (!user) return <AuthRequired next={`${location.pathname}${location.search}`} />;
    return children;
}

function AuthRequired({ next }: { next: string }) {
    const openAuth = useAuthDialogStore((state) => state.openAuth);
    useEffect(() => {
        openAuth({ tab: "login", next });
    }, [next, openAuth]);
    return (
        <div className="grid h-full min-h-[50vh] place-items-center px-6 text-center">
            <div className="space-y-3">
                <p className="text-base font-medium">请先登录</p>
                <p className="text-sm text-foreground/60">登录后即可继续使用短剧 Agent 和自由画布。</p>
                <button type="button" className="rounded-full bg-foreground px-5 py-2 text-sm text-background" onClick={() => openAuth({ tab: "login", next })}>
                    打开登录
                </button>
            </div>
        </div>
    );
}
