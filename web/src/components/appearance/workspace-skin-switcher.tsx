import { Popover } from "antd";
import { Palette } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import { availableWorkspaceSkins, resolveWorkspaceSkin } from "@/lib/workspace-skin";
import { skinSwatches } from "@/lib/skin-themes";
import { useAppearanceStore } from "@/stores/use-appearance-store";
import { useWorkspaceSkinStore } from "@/stores/use-workspace-skin-store";
import { cn } from "@/lib/utils";

export function WorkspaceSkinSwitcher({ className, compact = false }: { className?: string; compact?: boolean }) {
    const { t } = useTranslation("common");
    const appearance = useAppearanceStore((state) => state.appearance);
    const selectedId = useWorkspaceSkinStore((state) => state.skinId);
    const setSkinId = useWorkspaceSkinStore((state) => state.setSkinId);
    const skins = useMemo(() => availableWorkspaceSkins(appearance), [appearance]);
    const current = useMemo(() => resolveWorkspaceSkin(appearance, selectedId), [appearance, selectedId]);

    if (skins.length < 2) return null;

    const list = (
        <div className="workspace-skin-menu" role="listbox" aria-label={t("theme.skin")}>
            {skins.map((skin) => {
                const swatches = skinSwatches(skin).slice(0, 8);
                const selected = skin.id === current.id;
                return (
                    <button
                        key={skin.id}
                        type="button"
                        role="option"
                        aria-selected={selected}
                        className={cn("workspace-skin-option", selected && "is-selected")}
                        onClick={() => setSkinId(skin.id)}
                    >
                        {skin.poster ? <img className="workspace-skin-poster" src={skin.poster} alt="" /> : (
                        <span className="workspace-skin-swatches" aria-hidden="true">
                            {swatches.map((color) => (
                                <i key={color} style={{ background: color }} />
                            ))}
                        </span>
                        )}
                        <span className="workspace-skin-copy">
                            <strong>{skin.name}</strong>
                            {skin.description ? <em>{skin.description}</em> : null}
                        </span>
                    </button>
                );
            })}
        </div>
    );

    if (compact) {
        return (
            <Popover trigger="click" placement="bottomRight" overlayClassName="workspace-skin-popover" content={list}>
                <button type="button" className={cn("app-workspace-topbar-skin", className)} title={t("theme.skin")} aria-label={t("theme.skin")}>
                    <Palette className="size-3.5 shrink-0 opacity-70" aria-hidden />
                    <span>{current.name}</span>
                </button>
            </Popover>
        );
    }

    return (
        <div className={cn("workspace-skin-account", className)}>
            <div className="mb-2 flex items-center gap-2 px-2 text-xs text-foreground/65">
                <Palette className="size-3.5" aria-hidden />
                <span>{t("theme.skin")}</span>
            </div>
            {list}
        </div>
    );
}
