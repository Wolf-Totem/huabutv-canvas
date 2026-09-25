import { useCallback, useEffect, useState } from "react";
import { App, Button, Input, Modal } from "antd";

import { AdminPageFrame } from "./components/admin-shell";
import { AdminDataTable, AdminStatusBadge } from "./components/admin-ui";
import { listAdminPlazaWorks, takeDownPlazaWork, type PlazaWork } from "@/services/api/plaza";

export default function PlazaWorksPage() {
    const { message } = App.useApp();
    const [items, setItems] = useState<PlazaWork[]>([]);
    const [takeDownId, setTakeDownId] = useState<string | null>(null);
    const [takeDownNote, setTakeDownNote] = useState("");
    const load = useCallback(async () => {
        const result = await listAdminPlazaWorks({ page: 1, pageSize: 50 });
        setItems(result.works);
    }, []);
    useEffect(() => { void load().catch((error) => message.error(error instanceof Error ? error.message : "读取失败")); }, [load, message]);

    return (
        <AdminPageFrame title="广场作品" description="下架后目录、参观和成片全部 404，已复制画布保留">
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
                            render: (_, row: PlazaWork) => row.status === "listed" ? (
                                <Button size="small" danger onClick={() => { setTakeDownId(row.id); setTakeDownNote(""); }}>下架</Button>
                            ) : null,
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
        </AdminPageFrame>
    );
}
