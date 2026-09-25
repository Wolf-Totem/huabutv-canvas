import { useCallback, useEffect, useState } from "react";
import { App, Button, Input, Modal } from "antd";

import { AdminPageFrame } from "./components/admin-shell";
import { AdminDataTable, AdminStatusBadge } from "./components/admin-ui";
import { deletePlazaWork, listAdminPlazaWorks, takeDownPlazaWork, type PlazaWork } from "@/services/api/plaza";

export default function PlazaWorksPage() {
    const { message } = App.useApp();
    const [items, setItems] = useState<PlazaWork[]>([]);
    const [takeDownId, setTakeDownId] = useState<string | null>(null);
    const [takeDownNote, setTakeDownNote] = useState("");
    const [deleteTarget, setDeleteTarget] = useState<PlazaWork | null>(null);
    const [deleteConfirm, setDeleteConfirm] = useState("");
    const load = useCallback(async () => {
        const result = await listAdminPlazaWorks({ page: 1, pageSize: 100 });
        setItems(result.works);
    }, []);
    useEffect(() => { void load().catch((error) => message.error(error instanceof Error ? error.message : "读取失败")); }, [load, message]);

    return (
        <AdminPageFrame title="广场作品" description="下架后目录、参观和成片全部 404。删除会清掉广场记录和对应 plaza 画布，已复制到用户名下的画布保留。">
            <AdminDataTable
                table={{
                    rowKey: "id",
                    dataSource: items,
                    columns: [
                        { title: "标题", dataIndex: "title" },
                        { title: "作者", render: (_, row: PlazaWork) => row.author?.displayName },
                        { title: "状态", dataIndex: "status", render: (value: string) => <AdminStatusBadge label={value} /> },
                        { title: "复制", dataIndex: "copyCount" },
                        { title: "点赞", dataIndex: "likeCount" },
                        {
                            title: "操作",
                            render: (_, row: PlazaWork) => (
                                <span style={{ display: "inline-flex", gap: 8 }}>
                                    {row.status === "listed" ? (
                                        <Button size="small" danger onClick={() => { setTakeDownId(row.id); setTakeDownNote(""); }}>下架</Button>
                                    ) : null}
                                    <Button size="small" danger onClick={() => { setDeleteTarget(row); setDeleteConfirm(""); }}>删除</Button>
                                </span>
                            ),
                        },
                    ],
                }}
            />
            <Modal title="下架作品" open={Boolean(takeDownId)} onCancel={() => setTakeDownId(null)} onOk={async () => {
                if (!takeDownId) return;
                await takeDownPlazaWork(takeDownId, takeDownNote);
                message.success("已下架");
                setTakeDownId(null);
                await load();
            }}>
                <Input.TextArea value={takeDownNote} onChange={(event) => setTakeDownNote(event.target.value)} placeholder="原因" />
            </Modal>
            <Modal
                title="彻底删除作品"
                open={Boolean(deleteTarget)}
                okButtonProps={{ danger: true, disabled: deleteConfirm !== (deleteTarget?.title || "") }}
                onCancel={() => setDeleteTarget(null)}
                onOk={async () => {
                    if (!deleteTarget || deleteConfirm !== deleteTarget.title) return;
                    await deletePlazaWork(deleteTarget.id);
                    message.success("已删除");
                    setDeleteTarget(null);
                    await load();
                }}
            >
                <p>输入作品标题以确认删除。广场记录和对应 plaza 画布会清掉，用户自己复制出去的画布保留。</p>
                <p><strong>{deleteTarget?.title}</strong></p>
                <Input value={deleteConfirm} onChange={(event) => setDeleteConfirm(event.target.value)} placeholder="输入完整标题" />
            </Modal>
        </AdminPageFrame>
    );
}
