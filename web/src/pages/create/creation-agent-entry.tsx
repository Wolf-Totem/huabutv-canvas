import { useRef, useState } from "react";
import { App, Button } from "antd";
import { ArrowUpRight, FolderOpen, Sparkles } from "lucide-react";
import { useNavigate } from "react-router";
import { useTranslation } from "react-i18next";
import { BrandLogo } from "@/components/brand/brand-logo";
import "@/components/canvas/canvas-cloud-agent.css";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import { useCanvasStore } from "@/stores/canvas/use-canvas-store";
import { useUserStore } from "@/stores/use-user-store";
import { confirmCreateCanvasIfOthersPending } from "@/lib/canvas/confirm-create-canvas";
import { createCanvasProjectWithRemoteSync, hasRemoteUserDataSyncSession, saveRemoteCanvasProjectNow } from "@/services/user-data-sync";

function requireLogin(next: string) {
    useAuthDialogStore.getState().openAuth({ tab: "login", next });
}

/** Agent 依赖真实画布 ID，先完成服务端保存，再进入同一个画布 Agent。 */
export function CreationAgentEntry() {
    const { t } = useTranslation("canvas");
    const navigate = useNavigate();
    const { message, modal } = App.useApp();
    const hydrated = useCanvasStore((state) => state.hydrated);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");
    const lock = useRef(false);
    const created = useRef<{ id: string; userId: string } | null>(null);
    const start = async () => {
        if (lock.current) return;
        const userId = useUserStore.getState().user?.id;
        if (!userId || !hasRemoteUserDataSyncSession()) {
            requireLogin("/create");
            return;
        }
        lock.current = true;
        setError("");
        try {
            if (!hydrated) throw new Error(t("agent.canvasRestoring"));
            if (created.current?.userId !== userId) created.current = null;
            if (!created.current) {
                const decision = await confirmCreateCanvasIfOthersPending(modal);
                if (useUserStore.getState().user?.id !== userId) return;
                if (!decision.proceed) {
                    navigate(`/canvas/${encodeURIComponent(decision.projectId)}?agent=1`);
                    return;
                }
            }
            setBusy(true);
            if (!created.current) {
                const result = await createCanvasProjectWithRemoteSync("Agent 创作");
                if (!result.id) throw new Error(t("agent.createFailed"));
                created.current = { id: result.id, userId };
                if (result.syncError) throw new Error(result.syncError instanceof Error ? result.syncError.message : t("agent.syncPending"));
            } else {
                await saveRemoteCanvasProjectNow(created.current.id);
            }
            if (useUserStore.getState().user?.id !== userId) return;
            navigate(`/canvas/${encodeURIComponent(created.current.id)}?agent=1`);
        } catch (cause) {
            if (userId && useUserStore.getState().user?.id === userId) {
                const detail = cause instanceof Error ? cause.message : t("agent.startFailed");
                setError(detail);
                message.error(detail);
            }
        } finally { lock.current = false; setBusy(false); }
    };
    return <section className="creation-agent-entry" aria-label={t("mode.agent")}>
        <div className="creation-agent-orb" aria-hidden>
            <span className="agent-welcome-logo-wrap">
                <BrandLogo className="agent-welcome-logo" alt="" fallback={<img src="/logo.png" alt="" className="agent-welcome-logo" draggable={false} />} />
            </span>
        </div>
        <div className="creation-agent-copy"><span className="creation-agent-eyebrow">CANVAS AGENT</span><h2>{t("agent.headline")}</h2><p>{t("agent.lead", { name: t("mode.agent") })}</p></div>
        <div className="creation-agent-actions"><Button type="primary" icon={<Sparkles />} loading={busy} onClick={() => void start()}>{created.current ? t("agent.retryEnter") : t("agent.startNew")}<ArrowUpRight /></Button><Button icon={<FolderOpen />} disabled={busy} onClick={() => {
            if (!useUserStore.getState().user) {
                requireLogin("/canvas?agent=1");
                return;
            }
            navigate("/canvas?agent=1");
        }}>{t("agent.continueExisting")}</Button></div>
        {error ? <p className="creation-agent-error" role="alert">{error}</p> : null}
    </section>;
}
