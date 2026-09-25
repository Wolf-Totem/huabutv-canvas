import { App, Button, Input, Table } from "antd";
import { useEffect, useState } from "react";

import { approveAdminPayout, listAdminPayouts, rejectAdminPayout, type StreamerPayout } from "@/services/api/streamer";

const statusLabel: Record<StreamerPayout["status"], string> = {
    pending: "待审",
    approved: "已同意（请线下打款）",
    rejected: "已退回",
};

export default function PayoutsPanel() {
    const { message, modal } = App.useApp();
    const [items, setItems] = useState<StreamerPayout[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);

    const load = async (next = page) => {
        const data = await listAdminPayouts(next, 20);
        setItems(data.items || []);
        setTotal(data.total || 0);
        setPage(next);
    };

    useEffect(() => {
        void load(1).catch((error) => message.error(error instanceof Error ? error.message : "读取提现失败"));
    }, [message]);

    return (
        <Table
            rowKey="id"
            dataSource={items}
            pagination={{ current: page, pageSize: 20, total, onChange: (next) => void load(next) }}
            columns={[
                { title: "代理", dataIndex: "displayName" },
                { title: "金额", dataIndex: "amountCredits", render: (value: number) => Number(value || 0).toFixed(2) },
                { title: "支付宝账号", dataIndex: "alipayAccount" },
                { title: "姓名", dataIndex: "alipayRealName" },
                { title: "状态", dataIndex: "status", render: (value: StreamerPayout["status"]) => statusLabel[value] || value },
                { title: "原因", dataIndex: "rejectReason" },
                { title: "申请时间", dataIndex: "createdAt" },
                {
                    title: "操作",
                    render: (_, row) => row.status !== "pending" ? null : (
                        <div className="flex flex-wrap gap-2">
                            <Button
                                size="small"
                                type="primary"
                                onClick={() => {
                                    modal.confirm({
                                        title: "同意提现？",
                                        content: "只扣钱包记账，需要你线下转到这个支付宝账号。",
                                        onOk: () => approveAdminPayout(row.id).then(() => load(page)).then(() => message.success("已同意，请线下打款")),
                                    });
                                }}
                            >
                                同意
                            </Button>
                            <Button
                                size="small"
                                onClick={() => {
                                    let reason = "";
                                    modal.confirm({
                                        title: "退回复核",
                                        content: <Input.TextArea placeholder="原因" onChange={(event) => { reason = event.target.value; }} />,
                                        onOk: () => {
                                            if (!reason.trim()) {
                                                message.error("请填写原因");
                                                return Promise.reject();
                                            }
                                            return rejectAdminPayout(row.id, reason.trim()).then(() => load(page)).then(() => message.success("已退回"));
                                        },
                                    });
                                }}
                            >
                                退回复核
                            </Button>
                        </div>
                    ),
                },
            ]}
        />
    );
}
