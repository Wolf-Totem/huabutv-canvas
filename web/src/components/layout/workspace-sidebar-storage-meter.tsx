import { TicketPlus } from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";

import { useAccountFileStorageUsage } from "@/hooks/use-account-file-storage-usage";
import { accountStorageMeter } from "@/lib/account-storage-usage";
import { cn } from "@/lib/utils";
import { preloadWorkspaceRoute } from "@/lib/workspace-route-modules";
import { openWorkspaceWallet } from "@/lib/workspace-wallet";

export function WorkspaceSidebarStorageMeter({ collapsed }: { collapsed: boolean }) {
    const { t } = useTranslation("sidebar");
    const query = useAccountFileStorageUsage();
    const meter = accountStorageMeter(query.data);
    const usedText = query.data ? t("storage.used", { value: meter.usedLabel }) : query.isError ? t("storage.unavailable") : t("storage.loading");
    const remainingText = query.data ? (meter.full ? t("storage.full") : t("storage.remaining", { value: meter.remainingLabel })) : "";
    const totalText = query.data ? t("storage.total", { value: meter.totalLabel }) : "";
    const summary = query.data ? `${usedText}，${remainingText}，${totalText}` : usedText;

    if (query.isError && !query.data) {
        return (
            <div className={cn("app-workspace-sidebar-storage is-error", collapsed && "is-collapsed")}>
                {collapsed ? (
                    <button type="button" title="容量统计暂时不可用，点击重试" aria-label="重试加载账号容量" onClick={() => void query.refetch()} />
                ) : (
                    <>
                        <span>容量暂不可用</span>
                        <button type="button" onClick={() => void query.refetch()}>
                            重试
                        </button>
                    </>
                )}
            </div>
        );
    }

    return (
        <div className={cn("app-workspace-sidebar-storage-wrap", collapsed && "is-collapsed")}>
        <Link
            to="/assets"
            className={cn(
                "app-workspace-sidebar-storage",
                collapsed && "is-collapsed",
                meter.tone === "warn" && "is-warn",
                meter.tone === "critical" && "is-critical",
                query.data?.usedBytes ? "has-usage" : null,
                query.isPending && !query.data && "is-pending",
            )}
            title={`${summary}。包含素材文件和 Agent 会话附件`}
            aria-label={`账号容量，${summary}`}
            aria-busy={query.isPending && !query.data}
            onFocus={() => preloadWorkspaceRoute("/assets")}
            onPointerEnter={() => preloadWorkspaceRoute("/assets")}
        >
            {collapsed ? null : (
                <span className="app-workspace-sidebar-storage-copy">
                    <span className="app-workspace-sidebar-storage-meta">
                        <span className="app-workspace-sidebar-storage-used">{usedText}</span>
                        {totalText ? <span className="app-workspace-sidebar-storage-total">{totalText}</span> : null}
                    </span>
                    {remainingText ? <span className="app-workspace-sidebar-storage-remain">{remainingText}</span> : null}
                </span>
            )}
            <span
                className="app-workspace-sidebar-storage-track"
                role="progressbar"
                aria-label="账号文件容量使用进度"
                aria-valuemin={0}
                aria-valuemax={query.data?.totalBytes ?? 0}
                aria-valuenow={query.data ? Math.min(query.data.usedBytes, query.data.totalBytes) : 0}
                aria-valuetext={summary}
            >
                <span style={{ width: `${meter.percent}%` }} />
            </span>
        </Link>
        <button
            type="button"
            className={cn("app-workspace-sidebar-storage-topup", collapsed && "is-collapsed")}
            title="用兑换码充值容量"
            aria-label="用兑换码充值容量"
            onClick={() => openWorkspaceWallet({ focusRedeem: true })}
        >
            {collapsed ? <TicketPlus className="size-3.5" /> : t("storage.recharge")}
        </button>
        </div>
    );
}
