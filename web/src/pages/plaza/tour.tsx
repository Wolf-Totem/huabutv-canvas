import { useEffect, useState } from "react";
import { App, Button } from "antd";
import { useNavigate, useParams } from "react-router";

import { FullScreenLoader } from "@/components/ui/aceternity/full-screen-loader";
import { featuredBySlug, featuredTourProject } from "@/lib/plaza-catalog";
import { copyPlazaWork, getPlazaSnapshot, getPlazaWork, recordPlazaEvent } from "@/services/api/plaza";
import { createCanvasProjectWithRemoteSync, hasRemoteUserDataSyncSession } from "@/services/user-data-sync";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import type { CanvasProject } from "@/stores/canvas/use-canvas-store";
import { useUserStore } from "@/stores/use-user-store";
import { ReadOnlyCanvasView } from "@/pages/canvas/read-only-canvas";

export default function PlazaTourPage() {
    const { slug = "" } = useParams();
    const navigate = useNavigate();
    const { message } = App.useApp();
    const user = useUserStore((state) => state.user);
    const [project, setProject] = useState<CanvasProject | null>(null);
    const [title, setTitle] = useState("作品参观");
    const [workId, setWorkId] = useState("");
    const [allowCopy, setAllowCopy] = useState(true);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        let active = true;
        setLoading(true);
        getPlazaWork(slug).then(async ({ work }) => {
            const snapshot = await getPlazaSnapshot(work.id);
            if (!active) return;
            setTitle(work.title);
            setWorkId(work.id);
            setAllowCopy(work.allowCopy);
            setProject(snapshot.project);
            void recordPlazaEvent(work.id, "tour");
        }).catch(() => {
            if (!active) setProject(null);
        }).finally(() => {
            if (active) setLoading(false);
        });
        return () => { active = false; };
    }, [slug]);

    const onCopy = async () => {
        if (!user || !hasRemoteUserDataSyncSession()) {
            useAuthDialogStore.getState().openAuth({ tab: "login", next: `/plaza/${encodeURIComponent(slug)}/tour` });
            return;
        }
        try {
            if (workId) {
                const result = await copyPlazaWork(workId);
                message.success("已复制到你的画布");
                navigate(`/canvas/${result.projectId}`);
                return;
            }
            const featured = featuredBySlug(slug);
            if (!featured) throw new Error("作品不存在");
            const snapshot = featuredTourProject(featured);
            const result = await createCanvasProjectWithRemoteSync(featured.title, undefined, { nodes: snapshot.nodes, connections: snapshot.connections });
            if (result.syncError) message.warning("已复制到本机画布，云端稍后同步");
            else message.success("已复制到你的画布");
            navigate(`/canvas/${result.id}`);
        } catch (err) {
            message.error(err instanceof Error ? err.message : "复制失败");
        }
    };

    if (loading && !project) return <FullScreenLoader label="正在打开制作过程" detail="读取只读画布快照" />;
    if (!project) {
        return (
            <div className="grid h-screen place-items-center bg-[#111] px-5 text-white">
                <div className="space-y-3 text-center">
                    <h1 className="text-xl font-semibold">制作过程</h1>
                    <p className="text-sm text-white/60">暂时无法打开这个作品的节点图。</p>
                    <Button onClick={() => navigate(`/plaza/${slug}`)}>返回预览</Button>
                </div>
            </div>
        );
    }

    return <ReadOnlyCanvasView title={title} project={project} mode="tour" onCopy={allowCopy ? onCopy : undefined} headerRight={<Button onClick={() => navigate(`/plaza/${slug}`)}>返回预览</Button>} />;
}
