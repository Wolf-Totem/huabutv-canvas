import type { ReactNode } from "react";
import { lazy, Suspense, useLayoutEffect, useMemo } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { App, ConfigProvider } from "antd";
import enUS from "antd/locale/en_US";
import idID from "antd/locale/id_ID";
import thTH from "antd/locale/th_TH";
import viVN from "antd/locale/vi_VN";
import zhCN from "antd/locale/zh_CN";
import { I18nextProvider, useTranslation } from "react-i18next";

import { AuthSessionHydrator } from "@/components/auth/auth-session-hydrator";
import { I18nRuntime } from "@/components/i18n/i18n-runtime";
import { FullScreenLoader } from "@/components/ui/aceternity/full-screen-loader";
import i18n from "@/i18n/config";
import { getAntThemeConfig } from "@/lib/app-theme";
import { applySkinTheme } from "@/lib/skin-themes";
import { appQueryClient } from "@/lib/query-client";
import { useActiveTheme } from "@/stores/canvas/use-canvas-theme-store";
import { resolveWorkspaceSkin } from "@/lib/workspace-skin";
import { applyAppearanceMetadata, useAppearanceStore } from "@/stores/use-appearance-store";
import { useWorkspaceSkinStore } from "@/stores/use-workspace-skin-store";
import { useUserStore } from "@/stores/use-user-store";

const ClientRootInit = lazy(() => import("@/components/layout/client-root-init").then((module) => ({ default: module.ClientRootInit })));

function antdLocale(language: string) {
    if (language.startsWith("zh")) return zhCN;
    if (language.startsWith("id")) return idID;
    if (language.startsWith("vi")) return viVN;
    if (language.startsWith("th")) return thTH;
    return enUS;
}

function ClientRootBoundary({ children }: { children: ReactNode }) {
    const authenticated = useUserStore((state) => Boolean(state.user));
    if (!authenticated) return children;
    return <Suspense fallback={<FullScreenLoader />}><ClientRootInit>{children}</ClientRootInit></Suspense>;
}

export function AppProviders({ children }: { children: ReactNode }) {
    const theme = useActiveTheme();
    const dark = theme === "dark";
    const appearance = useAppearanceStore((state) => state.appearance);
    const workspaceSkinId = useWorkspaceSkinStore((state) => state.skinId);
    const workspaceSkin = useMemo(() => resolveWorkspaceSkin(appearance, workspaceSkinId), [appearance, workspaceSkinId]);

    useLayoutEffect(() => {
        document.documentElement.classList.toggle("dark", dark);
        document.documentElement.style.colorScheme = theme;
        applySkinTheme(workspaceSkin, theme);
        applyAppearanceMetadata({ ...appearance, activeSkin: workspaceSkin, skinId: workspaceSkin.id });
    }, [appearance, dark, theme, workspaceSkin]);

    const isolateDevRepro = import.meta.env.DEV && typeof window !== "undefined" && window.location.pathname === "/dev/director-repro";

    return (
        <I18nextProvider i18n={i18n}>
            <QueryClientProvider client={appQueryClient}>
                {isolateDevRepro ? children : (
                    <AuthSessionHydrator>
                        <I18nRuntime>
                            <ThemedApp dark={dark} appearanceSkin={workspaceSkin as unknown}>{children}</ThemedApp>
                        </I18nRuntime>
                    </AuthSessionHydrator>
                )}
            </QueryClientProvider>
        </I18nextProvider>
    );
}

function ThemedApp({ children, dark, appearanceSkin }: { children: ReactNode; dark: boolean; appearanceSkin: unknown }) {
    const { i18n: i18nInstance } = useTranslation();
    return (
        <ConfigProvider locale={antdLocale(i18nInstance.language)} theme={getAntThemeConfig(dark, appearanceSkin)}>
            <App message={{ duration: 3, maxCount: 3 }} notification={{ duration: 4.5, maxCount: 3, placement: "topRight" }}>
                <ClientRootBoundary>{children}</ClientRootBoundary>
            </App>
        </ConfigProvider>
    );
}
