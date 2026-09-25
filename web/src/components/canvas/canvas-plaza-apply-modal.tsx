import { useEffect, useMemo, useState } from "react";
import { App, Button, Input, Modal, Select, Switch } from "antd";

import { applyPlazaWork, getPlazaCategories, getPlazaSettings, listMyPlazaApplications, type PlazaApplication, type PlazaCategory } from "@/services/api/plaza";
import type { CanvasNodeData } from "@/types/canvas";

function mediaNodes(nodes: CanvasNodeData[]) {
    return nodes.filter((node) => (node.type === "image" || node.type === "video") && (node.metadata?.storageKey || node.metadata?.content));
}

export function CanvasPlazaApplyModal({
    projectId,
    title,
    nodes,
    open,
    onClose,
    beforeSubmit,
}: {
    projectId: string;
    title: string;
    nodes: CanvasNodeData[];
    open: boolean;
    onClose: () => void;
    beforeSubmit: () => Promise<boolean | void>;
}) {
    const { message } = App.useApp();
    const [applyEnabled, setApplyEnabled] = useState(false);
    const [categories, setCategories] = useState<PlazaCategory[]>([]);
    const [pending, setPending] = useState<PlazaApplication | null>(null);
    const [workTitle, setWorkTitle] = useState(title);
    const [subtitle, setSubtitle] = useState("");
    const [categoryId, setCategoryId] = useState("");
    const [coverNodeId, setCoverNodeId] = useState("");
    const [watchNodeId, setWatchNodeId] = useState("");
    const [allowWatch, setAllowWatch] = useState(true);
    const [allowProcessView, setAllowProcessView] = useState(true);
    const [allowCopy, setAllowCopy] = useState(true);
    const [originalityAck, setOriginalityAck] = useState(false);
    const [submitting, setSubmitting] = useState(false);
    const candidates = useMemo(() => mediaNodes(nodes), [nodes]);
    const videoNodes = candidates.filter((node) => node.type === "video");

    useEffect(() => {
        if (!open) return;
        setWorkTitle(title);
        const cover = videoNodes[0] || candidates[0];
        setCoverNodeId(cover?.id || "");
        setWatchNodeId(videoNodes[0]?.id || "");
        void Promise.all([getPlazaSettings(), getPlazaCategories(), listMyPlazaApplications()]).then(([settings, cats, mine]) => {
            setApplyEnabled(settings.settings.applyEnabled);
            setCategories(cats.categories.filter((item) => item.slug !== "all" && item.kind !== "campaign"));
            const current = mine.applications.find((item) => item.projectId === projectId && item.status === "pending");
            setPending(current || null);
        }).catch((error) => message.error(error instanceof Error ? error.message : "读取上架配置失败"));
    }, [candidates, message, open, projectId, title]);

    const submit = async () => {
        setSubmitting(true);
        try {
            const saved = await beforeSubmit();
            if (saved === false) return;
            const result = await applyPlazaWork(projectId, {
                title: workTitle.trim(),
                subtitle: subtitle.trim(),
                categoryId,
                coverNodeId,
                watchNodeId,
                allowWatch,
                allowProcessView,
                allowCopy,
                originalityAck,
            });
            setPending(result.application);
            message.success("已提交审核");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "提交失败");
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Modal title="上架作品广场" open={open} onCancel={onClose} footer={null} centered width={560} destroyOnHidden>
            {!applyEnabled ? <p className="text-sm text-foreground/60">管理员尚未开放申请上架。</p> : pending ? (
                <div className="space-y-3 text-sm">
                    <p>已提交审核，当前状态：<strong>{pending.status}</strong></p>
                    {pending.reviewNote ? <p>上次原因：{pending.reviewNote}</p> : null}
                    <p className="text-foreground/55">审核通过后会按提交时的草稿上架，之后改画布不会影响广场。</p>
                </div>
            ) : (
                <div className="space-y-4">
                    <label className="block space-y-1 text-sm">标题<Input value={workTitle} maxLength={40} onChange={(event) => setWorkTitle(event.target.value)} /></label>
                    <label className="block space-y-1 text-sm">副标题<Input value={subtitle} maxLength={160} onChange={(event) => setSubtitle(event.target.value)} /></label>
                    <label className="block space-y-1 text-sm">分类
                        <Select className="w-full" value={categoryId || undefined} placeholder="选择分类" options={categories.map((item) => ({ value: item.id, label: item.name }))} onChange={setCategoryId} />
                    </label>
                    <label className="block space-y-1 text-sm">封面节点
                        <Select className="w-full" value={coverNodeId || undefined} options={candidates.map((node) => ({ value: node.id, label: node.title || node.id }))} onChange={setCoverNodeId} />
                    </label>
                    <label className="block space-y-1 text-sm">成片节点
                        <Select className="w-full" allowClear value={watchNodeId || undefined} options={videoNodes.map((node) => ({ value: node.id, label: node.title || node.id }))} onChange={(value) => setWatchNodeId(value || "")} />
                    </label>
                    <div className="grid gap-2 text-sm">
                        <label className="flex items-center justify-between">允许观看成片 <Switch checked={allowWatch} onChange={setAllowWatch} /></label>
                        <label className="flex items-center justify-between">允许参观制作过程 <Switch checked={allowProcessView} onChange={setAllowProcessView} /></label>
                        <label className="flex items-center justify-between">允许复制项目 <Switch checked={allowCopy} onChange={setAllowCopy} /></label>
                        <label className="flex items-center gap-2"><input type="checkbox" checked={originalityAck} onChange={(event) => setOriginalityAck(event.target.checked)} />我确认这是原创或已获授权的作品</label>
                    </div>
                    <Button type="primary" block loading={submitting} disabled={!originalityAck || !categoryId} onClick={() => void submit()}>提交审核</Button>
                </div>
            )}
        </Modal>
    );
}
