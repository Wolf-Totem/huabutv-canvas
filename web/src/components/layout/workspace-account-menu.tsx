import { Popover } from "antd";
import { Switch } from "@/components/ui/base/switch";
import { LogIn, Moon, Sun } from "lucide-react";
import { useState } from "react";

import { useTranslation } from "react-i18next";
import { AppChangelogButton } from "@/components/layout/app-changelog-modal";
import { WorkspaceAccountCard } from "./workspace-account-card";
import { UserAvatar } from "./user-avatar";
import { openWorkspaceWallet } from "@/lib/workspace-wallet";
import { useThemeStore } from "@/stores/use-theme-store";
import { useUserStore } from "@/stores/use-user-store";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";

/** 顶部与侧栏复用同一账户卡片；顶部额外保留版本和主题偏好。 */
export function WorkspaceAccountMenu() {
    const { t } = useTranslation("common");
    const theme = useThemeStore((state) => state.theme);
    const setTheme = useThemeStore((state) => state.setTheme);
    const user = useUserStore((state) => state.user);
    const hydrated = useUserStore((state) => state.hydrated);
    const [menuOpen, setMenuOpen] = useState(false);

    if (!hydrated) {
        return <span className="size-9 animate-pulse rounded-[var(--r-md)] bg-foreground/[.06]" aria-hidden />;
    }

    return user ? (
        <><Popover
            trigger="click"
            placement="bottomRight"
            rootClassName="workspace-account-popover"
            open={menuOpen}
            onOpenChange={setMenuOpen}
            content={(
                <div className="workspace-topbar-account-menu">
                    <WorkspaceAccountCard onNavigate={() => setMenuOpen(false)} onWallet={() => { setMenuOpen(false); openWorkspaceWallet(); }} />

                    <div className="workspace-topbar-account-section">
                        <AppChangelogButton className="flex h-8 w-full items-center gap-2 rounded px-2 text-[var(--fs-label)] text-foreground/58 hover:bg-surface-hover hover:text-foreground [&_svg]:size-3.5" showLabel showVersion versionClassName="ml-auto text-[var(--fs-micro)] tabular-nums text-foreground/32" />
                    </div>

                    <div className="workspace-topbar-account-theme">
                        {theme === "dark" ? <Moon className="size-3.5 text-foreground/45" /> : <Sun className="size-3.5 text-foreground/45" />}
                        <span className="ml-2 flex-1 text-xs text-foreground/65">{t("theme.dark")}</span>
                        <Switch size="sm" checked={theme === "dark"} onChange={(checked) => setTheme(checked ? "dark" : "light")} aria-label={t("theme.dark")} />
                    </div>
                </div>
            )}
        >
            <button type="button" className="app-workspace-topbar-icon-button app-workspace-account-trigger" aria-label={t("account.menu")} title={user.displayName || user.username}>
                <UserAvatar user={user} className="size-6" />
            </button>
        </Popover></>
    ) : (
        <button type="button" className="app-workspace-topbar-icon-button" aria-label={t("action.login")} title={t("action.login")} onClick={() => useAuthDialogStore.getState().openAuth({ tab: "login" })}>
            <LogIn />
        </button>
    );
}
