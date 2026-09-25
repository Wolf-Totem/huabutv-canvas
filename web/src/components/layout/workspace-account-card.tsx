import { Button } from "antd";
import { ArrowUpRight, Coins, LogOut, RefreshCw, Settings, ShieldCheck } from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { useWalletBalance } from "@/hooks/use-wallet-balance";
import { useWorkspaceLogout } from "@/hooks/use-workspace-logout";
import { localeTag } from "@/i18n/display";
import { canAccessAdmin, hasPermission, PERMISSIONS } from "@/lib/access";
import { roleLabel } from "@/lib/user-role";
import { agentConsoleURL } from "@/lib/public-hosts";
import { useUserStore } from "@/stores/use-user-store";
import { UserAvatar } from "./user-avatar";
import "./workspace-account-card.css";

function followAccountLink(event: { preventDefault: () => void }, path: string, onNavigate: () => void) {
    onNavigate();
    if (document.documentElement.dataset.publicShell === "1") {
        event.preventDefault();
        window.location.assign(path);
    }
}

/** 同一账户卡片用于顶部和侧栏；余额与退出均复用真实服务。 */
export function WorkspaceAccountCard({ onWallet, onNavigate }: { onWallet: () => void; onNavigate: () => void }) {
    const { t, i18n } = useTranslation("common");
    const user = useUserStore((state) => state.user);
    const permissions = useUserStore((state) => state.permissions);
    const creditsEnabled = useUserStore((state) => state.features.creditsEnabled);
    const { availableMicrocredits, refreshing, refresh } = useWalletBalance(user?.id, creditsEnabled);
    const { handleLogout, loggingOut } = useWorkspaceLogout();
    if (!user) return null;
    return <section className="workspace-account-card" aria-label={t("account.mine")}>
        <header className="workspace-account-card-identity">
            <UserAvatar user={user} className="workspace-account-card-avatar" />
            <div><strong>{user.displayName || user.username}</strong><span>@{user.username}</span></div>
            <em>{roleLabel(user.role)}</em>
        </header>
        {creditsEnabled ? <div className="workspace-account-card-wallet">
            <div className="workspace-account-card-balance"><span><Coins />{t("account.credits")}</span><strong>{availableMicrocredits === null ? "—" : (availableMicrocredits / 1_000_000).toLocaleString(localeTag(i18n.language), { maximumFractionDigits: 2 })}</strong></div>
            {availableMicrocredits === null ? <Button size="small" loading={refreshing} icon={<RefreshCw />} onClick={() => void refresh()}>{t("account.refresh")}</Button> : <button type="button" onClick={onWallet}>{t("account.topup")}<ArrowUpRight /></button>}
        </div> : <div className="workspace-account-card-wallet">
            <button type="button" onClick={onWallet}>{t("account.subscribe")}<ArrowUpRight /></button>
        </div>}
        <nav className="workspace-account-card-actions" aria-label={t("account.actions")}>
            <Link to="/settings" onClick={(event) => followAccountLink(event, "/settings", onNavigate)}><Settings /><span>{t("account.settings")}</span><ArrowUpRight /></Link>
            {hasPermission(user.role, permissions, PERMISSIONS.agentConsole) ? <a href={agentConsoleURL()} onClick={onNavigate}><ShieldCheck /><span>代理后台</span><ArrowUpRight /></a> : null}
            {canAccessAdmin(user.role, permissions) ? <Link to="/admin" onClick={(event) => followAccountLink(event, "/admin", onNavigate)}><ShieldCheck /><span>{t("account.adminConsole")}</span><ArrowUpRight /></Link> : null}
            <Button danger type="text" icon={<LogOut />} loading={loggingOut} onClick={() => void handleLogout()}>{t("action.logout")}</Button>
        </nav>
    </section>;
}
