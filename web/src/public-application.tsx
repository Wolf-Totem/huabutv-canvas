import { StrictMode, Suspense, lazy, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { Navigate, RouterProvider, createBrowserRouter } from "react-router";
import { App, ConfigProvider } from "antd";
import { I18nextProvider } from "react-i18next";
import "antd/dist/reset.css";

import { AuthDialogHost } from "@/components/auth/auth-dialog";
import { AuthScene } from "@/pages/auth/auth-scene";
import GuestHomePage from "@/pages/public-home/guest-home";
import i18n from "@/i18n/config";
import { getAntThemeConfig } from "@/lib/app-theme";
import { getAuthSession } from "@/services/api/auth";
import { useAppearanceStore } from "@/stores/use-appearance-store";
import { useUserStore } from "@/stores/use-user-store";
import "./styles/globals.css";

const ForgotPasswordPage = lazy(() => import("@/pages/auth/forgot-password"));

document.documentElement.dataset.publicShell = "1";
document.documentElement.classList.add("dark");
document.documentElement.style.colorScheme = "dark";

const router = createBrowserRouter([
    { path: "/", element: <GuestHomePage /> },
    { path: "/login", element: <Navigate to="/?auth=login" replace /> },
    { path: "/register", element: <Navigate to="/?auth=register" replace /> },
    {
        element: <AuthScene />,
        children: [
            { path: "/forgot-password", element: <Suspense fallback={null}><ForgotPasswordPage /></Suspense> },
        ],
    },
]);

function PublicRoot() {
    const appearance = useAppearanceStore((state) => state.appearance);

    useEffect(() => {
        let active = true;
        getAuthSession()
            .then((payload) => {
                if (!active) return;
                const store = useUserStore.getState();
                if (payload.user) {
                    store.setUser(payload.user);
                    store.setPermissions(payload.permissions);
                    store.setRuntimeLimits(payload.runtimeLimits);
                    store.setDrawingEngine(payload.drawingEngine);
                    store.setFeatures(payload.features);
                    store.setMembership(payload.membership);
                }
                store.setHydrated(true);
            })
            .catch(() => {
                if (active) useUserStore.getState().setHydrated(true);
            });
        return () => {
            active = false;
        };
    }, []);

    return (
        <I18nextProvider i18n={i18n}>
            <ConfigProvider theme={getAntThemeConfig(true, appearance.activeSkin)}>
                <App message={{ duration: 3, maxCount: 3 }}>
                    <AuthDialogHost />
                    <RouterProvider router={router} />
                </App>
            </ConfigProvider>
        </I18nextProvider>
    );
}

createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <PublicRoot />
    </StrictMode>,
);
