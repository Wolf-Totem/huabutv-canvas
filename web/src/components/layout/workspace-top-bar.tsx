import { Coins, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { Link, useLocation } from "react-router";

import { useTranslation } from "react-i18next";
import { LocaleSwitcher } from "@/components/i18n/locale-switcher";
import { SystemAnnouncementCenter } from "@/components/layout/system-announcement-center";
import { WorkspaceAccountMenu } from "@/components/layout/workspace-account-menu";
import { WorkspaceTopBarExtensionSlot } from "@/components/layout/workspace-top-bar-extension";
import { AnimatedThemeToggler } from "@/components/ui/animated-theme-toggler";
import { useThemeStore } from "@/stores/use-theme-store";
import { useUserStore } from "@/stores/use-user-store";
import { useAppearanceStore } from "@/stores/use-appearance-store";
import { useWalletBalance } from "@/hooks/use-wallet-balance";
import { openWorkspaceWallet } from "@/lib/workspace-wallet";

const PAGE_TITLE_KEYS: Record<string, string> = {
    home: "nav.create",
    create: "nav.create",
    projects: "nav.projects",
    canvas: "nav.canvas",
    plaza: "nav.plaza",
    tasks: "nav.history",
    assets: "nav.assets",
    skills: "nav.skills",
    plugins: "nav.plugins",
    settings: "nav.settings",
};

export function WorkspaceTopBar({ sidebarOpen, onToggleSidebar }: { sidebarOpen: boolean; onToggleSidebar: () => void }) {
    const { t } = useTranslation("common");
    const { t: ts } = useTranslation("sidebar");
    const brandName = useAppearanceStore((state) => state.appearance.brandName);
    const theme = useThemeStore((state) => state.theme);
    const setTheme = useThemeStore((state) => state.setTheme);
    const user = useUserStore((state) => state.user);
    const creditsEnabled = useUserStore((state) => state.features.creditsEnabled);
    const { availableMicrocredits } = useWalletBalance(user?.id, creditsEnabled);
    const { pathname } = useLocation();
    const slug = pathname.split("/").filter(Boolean)[0];
    const pageTitle = slug ? (PAGE_TITLE_KEYS[slug] ? ts(PAGE_TITLE_KEYS[slug]) : brandName) : ts("nav.create");
    const balance = availableMicrocredits === null ? "--" : (availableMicrocredits / 1_000_000).toLocaleString(undefined, { maximumFractionDigits: 2 });

    return (
        <header className="app-workspace-topbar">
            <button type="button" className="app-workspace-mobile-menu app-workspace-topbar-icon-button" aria-label={sidebarOpen ? t("topbar.collapseSidebar") : t("topbar.expandSidebar")} onClick={onToggleSidebar}>
                {sidebarOpen ? <PanelLeftClose className="size-4" /> : <PanelLeftOpen className="size-4" />}
            </button>
            <nav className="app-workspace-topbar-breadcrumb" aria-label={ts("breadcrumb")}>
                <Link to="/" className="font-medium text-foreground/65 transition-colors hover:text-foreground">{brandName}</Link>
                <span aria-hidden="true">/</span>
                <span className="truncate font-medium text-foreground">{pageTitle}</span>
            </nav>
            <WorkspaceTopBarExtensionSlot />
            <div className="app-workspace-topbar-actions">
                {creditsEnabled ? <button type="button" className="app-workspace-topbar-credit-pill" aria-label={t("topbar.openCredits", { balance })} onClick={() => openWorkspaceWallet()}>
                    <Coins aria-hidden="true" />
                    <span>{t("topbar.credits")}</span>
                    <strong>{balance}</strong>
                </button> : null}
                <LocaleSwitcher compact />
                {user ? <SystemAnnouncementCenter userId={user.id} className="app-workspace-topbar-icon-button" autoOpen /> : null}
                <AnimatedThemeToggler className="app-workspace-topbar-icon-button" theme={theme} onThemeChange={setTheme} aria-label={t("topbar.toggleTheme")} />
                <WorkspaceAccountMenu />
            </div>
        </header>
    );
}
