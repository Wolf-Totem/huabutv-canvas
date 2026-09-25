import { useEffect, useMemo, useState } from "react";

import { BrandLogoFrame } from "@/components/brand/brand-logo";
import { StudioAtmosphere } from "@/components/studio/atmosphere";
import { resolveLoaderSkin } from "@/lib/loader-skin";
import { cn } from "@/lib/utils";
import { useAppearanceStore } from "@/stores/use-appearance-store";
import { useThemeStore } from "@/stores/use-theme-store";
import "@/styles/studio-atmosphere.css";

type FullScreenLoaderProps = {
    label?: string;
    detail?: string;
    className?: string;
};

export function FullScreenLoader({ label = "正在打开页面", detail = "请稍候", className }: FullScreenLoaderProps) {
    const appearance = useAppearanceStore((state) => state.appearance);
    const theme = useThemeStore((state) => state.theme);
    const mode = appearance.defaultMode === "light" || appearance.defaultMode === "dark" ? appearance.defaultMode : theme;
    const loaderSkin = useMemo(() => resolveLoaderSkin(appearance), [appearance]);

    return (
        <div
            data-full-screen-loader
            data-skin={loaderSkin?.id || appearance.skinId}
            data-mode={mode}
            role="status"
            aria-live="polite"
            aria-label={`${label}，${detail}`}
            className={cn("full-screen-loader", loaderSkin && "has-studio-fx", className)}
        >
            <StudioAtmosphere skin={loaderSkin} mode={mode} />
            <div className="full-screen-loader-scene" aria-hidden="true">
                <span className="full-screen-loader-guide is-horizontal" />
                <span className="full-screen-loader-guide is-vertical" />
                <span className="full-screen-loader-frame is-left"><i /><i /><i /></span>
                <span className="full-screen-loader-frame is-right"><i /><i /><i /></span>
                <span className="full-screen-loader-script"><i /><i /><i /><b /></span>
                <span className="full-screen-loader-timeline"><i /><i /><i /><i /><b /></span>
                <span className="full-screen-loader-orbit" />
                <BrandLogoFrame className="full-screen-loader-logo" logoClassName="full-screen-loader-logo-image" alt="" fallback={<span className="full-screen-loader-logo-fallback" />} />
            </div>
            <div className="full-screen-loader-copy"><strong>{label}</strong><span>{detail}</span><LoadingSignal /></div>
        </div>
    );
}

export function WorkspaceRouteLoader({ label = "正在打开页面" }: { label?: string }) {
    const [visible, setVisible] = useState(false);

    useEffect(() => {
        const timer = window.setTimeout(() => setVisible(true), 140);
        return () => window.clearTimeout(timer);
    }, []);

    return (
        <section data-workspace-route-loader className={cn("workspace-route-loader", visible && "is-visible")} role="status" aria-live="polite" aria-label={label}>
            <div className="workspace-route-loader-content">
                <span className="workspace-route-loader-mark"><BrandLogoFrame className="workspace-route-loader-logo" logoClassName="size-6" alt="" fallback={<span className="full-screen-loader-logo-fallback" />} /></span>
                <LoadingSignal />
                <span>{label}</span>
            </div>
        </section>
    );
}

function LoadingSignal() {
    return <span className="loading-signal" aria-hidden="true"><i /><i /><i /></span>;
}
