import { useCallback, useEffect, useState } from "react";
import { App, Button, Input, Modal, Switch } from "antd";
import { useNavigate } from "react-router";

import { AdminPageFrame } from "./components/admin-shell";
import { AdminDataTable, AdminStatusBadge } from "./components/admin-ui";
import {
    approvePlazaApplication,
    getAdminPlazaSettings,
    listAdminPlazaApplications,
    rejectPlazaApplication,
    updateAdminPlazaSettings,
    type PlazaApplication,
    type PlazaSettings,
} from "@/services/api/plaza";

export default function PlazaApplicationsPage() {
    const { message, modal } = App.useApp();
    const navigate = useNavigate();
    const [items, setItems] = useState<PlazaApplication[]>([]);
    const [settings, setSettings] = useState<PlazaSettings | null>(null);
    const [status, setStatus] = useState("pending");
    const [rejectId, setRejectId] = useState<string | null>(null);
    const [rejectNote, setRejectNote] = useState("");
    const load = useCallback(async () => {
        const [list, current] = await Promise.all([listAdminPlazaApplications({ status, page: 1, pageSize: 50 }), getAdminPlazaSettings()]);
        setItems(list.applications);
        setSettings(current.settings);
    }, [status]);

    useEffect(() => { void load().catch((error) => message.error(error instanceof Error ? error.message : "读取失败")); }, [load, message]);

    const saveSettings = async (next: PlazaSettings) => {
        const result = await updateAdminPlazaSettings(next);
        setSettings(result.settings);
        message.success("已保存广场开关");
    };

    return (
        <AdminPageFrame title="作品广场审核" description="只审核申请时的草稿快照，通过后拷贝媒体上架">
            {settings ? (
                <div className="mb-6 grid gap-3 rounded-xl border border-border p-4 sm:grid-cols-2 lg:grid-cols-4">
                    <label className="flex items-center justify-between text-sm">开放前台 <Switch checked={settings.enabled} onChange={(enabled) => void saveSettings({ ...settings, enabled })} /></label>
                    <label className="flex items-center justify-between text-sm">允许申请 <Switch checked={settings.applyEnabled} onChange={(applyEnabled) => void saveSettings({ ...settings, applyEnabled })} /></label>
                    <label className="flex items-center justify-between text-sm">允许复制 <Switch checked={settings.copyEnabled} onChange={(copyEnabled) => void saveSettings({ ...settings, copyEnabled })} /></label>
                    <label className="flex items-center justify-between text-sm">游客参观 <Switch checked={settings.publicWatch} onChange={(publicWatch) => void saveSettings({ ...settings, publicWatch })} /></label>
                </div>
            ) : null}
            <div className="mb-4 flex gap-2">
                {["pending", "approved", "rejected", "withdrawn"].map((item) => (
                    <Button key={item} type={status === item ? "primary" : "default"} onClick={() => setStatus(item)}>{item}</Button>
                ))}
            </div>
            <AdminDataTable
                table={{
                    rowKey: "id",
                    dataSource: items,
                    columns: [
                        { title: "标题", dataIndex: "title" },
                        { title: "作者", render: (_, row: PlazaApplication) => row.author?.displayName || row.userId },
                        { title: "状态", dataIndex: "status", render: (value: string) => <AdminStatusBadge label={value} /> },
                        { title: "节点", dataIndex: "nodeCount" },
                        { title: "提交时间", dataIndex: "submittedAt", render: (value: string) => value ? new Date(value).toLocaleString("zh-CN") : "" },
                        {
                            title: "操作",
                            render: (_, row: PlazaApplication) => (
                                <div className="flex flex-wrap gap-2">
                                    <Button size="small" onClick={() => navigate(`/admin/plaza/applications/${row.id}/preview`)}>预览草稿</Button>
                                    {row.status === "pending" ? (
                                        <>
                                            <Button size="small" type="primary" onClick={() => modal.confirm({ title: "通过该申请？", content: "将按申请草稿拷贝媒体并上架，不再读取作者当前画布。", onOk: async () => { await approvePlazaApplication(row.id); message.success("已通过"); await load(); } })}>通过</Button>
                                            <Button size="small" danger onClick={() => { setRejectId(row.id); setRejectNote(""); }}>拒绝</Button>
                                        </>
                                    ) : null}
                                </div>
                            ),
                        },
                    ],
                }}
            />
            <Modal title="拒绝申请" open={Boolean(rejectId)} onCancel={() => setRejectId(null)} onOk={async () => {
                if (!rejectId) return;
                await rejectPlazaApplication(rejectId, rejectNote);
                message.success("已拒绝");
                setRejectId(null);
                await load();
            }}>
                <Input.TextArea value={rejectNote} onChange={(event) => setRejectNote(event.target.value)} placeholder="拒绝原因，作者可见" />
            </Modal>
        </AdminPageFrame>
    );
}
