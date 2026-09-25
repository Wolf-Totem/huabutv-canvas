import { useEffect, useState } from "react";
import { App, Button, Form, Input, InputNumber, Table, Tabs } from "antd";

import { canvasWorkspaceURL } from "@/lib/public-hosts";
import {
    createAgentPayout,
    getStreamerMe,
    getStreamerRebates,
    getStreamerSummary,
    getStreamerUsers,
    listAgentPayouts,
    type StreamerConsoleMe,
    type StreamerConsoleRebate,
    type StreamerConsoleUser,
    type StreamerConsoleSummary,
    type StreamerPayout,
} from "@/services/api/streamer";
import { ApiError } from "@/services/api/request";

const pageSize = 20;

const payoutStatus: Record<StreamerPayout["status"], string> = {
    pending: "审核中",
    approved: "已提现",
    rejected: "已退回",
};

const capabilityLabel: Record<string, string> = {
    text: "文本",
    image: "图片",
    video: "视频",
    audio: "音频",
};

const tablePagination = (current: number, total: number, onChange: (page: number) => void) => ({
    current,
    pageSize,
    total,
    showSizeChanger: false,
    showTotal: (count: number) => `共 ${count} 条`,
    onChange,
});

export default function AgentConsolePage() {
    const { message } = App.useApp();
    const [tab, setTab] = useState("overview");
    const [me, setMe] = useState<StreamerConsoleMe | null>(null);
    const [summary, setSummary] = useState<StreamerConsoleSummary | null>(null);
    const [users, setUsers] = useState<StreamerConsoleUser[]>([]);
    const [userTotal, setUserTotal] = useState(0);
    const [userPage, setUserPage] = useState(1);
    const [rebates, setRebates] = useState<StreamerConsoleRebate[]>([]);
    const [rebateTotal, setRebateTotal] = useState(0);
    const [rebatePage, setRebatePage] = useState(1);
    const [rebateType, setRebateType] = useState("");
    const [payouts, setPayouts] = useState<StreamerPayout[]>([]);
    const [payoutTotal, setPayoutTotal] = useState(0);
    const [payoutPage, setPayoutPage] = useState(1);
    const [forbidden, setForbidden] = useState("");
    const [loadingOverview, setLoadingOverview] = useState(true);
    const [loadingUsers, setLoadingUsers] = useState(false);
    const [loadingRebates, setLoadingRebates] = useState(false);
    const [loadingPayouts, setLoadingPayouts] = useState(false);
    const [form] = Form.useForm();

    const loadOverview = async () => {
        setLoadingOverview(true);
        try {
            const [meData, summaryData] = await Promise.all([getStreamerMe(), getStreamerSummary()]);
            setMe(meData);
            setSummary(summaryData);
            setForbidden("");
            form.setFieldsValue({
                alipayAccount: meData.alipayAccount || "",
                alipayRealName: meData.alipayRealName || "",
            });
        } catch (error) {
            const text = error instanceof Error ? error.message : "无法打开代理后台";
            if (error instanceof ApiError && (error.status === 403 || error.status === 401)) setForbidden(text);
            else message.error(text);
        } finally {
            setLoadingOverview(false);
        }
    };

    const loadUsers = async (page = userPage) => {
        setLoadingUsers(true);
        try {
            const data = await getStreamerUsers(page, pageSize);
            setUsers(data.items || []);
            setUserTotal(data.total || 0);
            setUserPage(page);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取名下用户失败");
        } finally {
            setLoadingUsers(false);
        }
    };

    const loadRebates = async (page = rebatePage, capability = rebateType) => {
        setLoadingRebates(true);
        try {
            const data = await getStreamerRebates(page, pageSize, capability);
            setRebates(data.items || []);
            setRebateTotal(data.total || 0);
            setRebatePage(page);
            setRebateType(capability);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取返利记录失败");
        } finally {
            setLoadingRebates(false);
        }
    };

    const loadPayouts = async (page = payoutPage) => {
        setLoadingPayouts(true);
        try {
            const data = await listAgentPayouts(page, pageSize);
            setPayouts(data.items || []);
            setPayoutTotal(data.total || 0);
            setPayoutPage(page);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取提现记录失败");
        } finally {
            setLoadingPayouts(false);
        }
    };

    useEffect(() => {
        void loadOverview();
    }, []);

    useEffect(() => {
        if (tab === "users") void loadUsers(userPage);
        if (tab === "rebates") void loadRebates(rebatePage, rebateType);
        if (tab === "payouts") void loadPayouts(payoutPage);
    }, [tab]);

    if (forbidden) {
        return (
            <main className="mx-auto max-w-lg px-6 py-20 text-center">
                <h1 className="text-2xl font-semibold">无权限</h1>
                <p className="mt-3 text-sm opacity-70">{forbidden}。这是主播只读后台，不是管理后台。</p>
                <a href={canvasWorkspaceURL()} className="mt-6 inline-block text-sm underline">返回画布</a>
            </main>
        );
    }

    const withdrawable = summary?.rebateWithdrawableCredits ?? 0;

    return (
        <main className="mx-auto flex min-h-dvh max-w-5xl flex-col gap-6 px-6 py-10">
            <header className="flex flex-wrap items-end justify-between gap-4">
                <div>
                    <p className="text-xs uppercase tracking-[0.2em] opacity-50">Streamer console</p>
                    <h1 className="mt-1 text-3xl font-semibold">代理后台</h1>
                    <p className="mt-2 text-sm opacity-70">查看邀请、名下用户、返利记录和提现。不能改价或改域名。</p>
                </div>
                <a href={me?.canvasUrl || canvasWorkspaceURL()} className="text-sm underline">打开画布</a>
            </header>

            <Tabs
                activeKey={tab}
                onChange={setTab}
                items={[
                    {
                        key: "overview",
                        label: "概览",
                        children: (
                            <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                                <Stat label="名下用户" value={summary ? String(summary.referredUserCount) : "—"} />
                                <Stat label="累计消耗" value={summary ? summary.consumedCredits.toFixed(2) : "—"} />
                                <Stat label="总返利" value={summary ? Number(summary.rebateTotalCredits || summary.rebateCredits || 0).toFixed(2) : "—"} />
                                <Stat label="可提现" value={summary ? Number(summary.rebateWithdrawableCredits || 0).toFixed(2) : "—"} />
                                <Stat label="审核中" value={summary ? Number(summary.rebatePendingCredits || 0).toFixed(2) : "—"} />
                                <Stat label="已提现 / 总提现" value={summary ? `${Number(summary.rebateWithdrawnCredits || 0).toFixed(2)} / ${Number(summary.rebateRequestedCredits || 0).toFixed(2)}` : "—"} />
                            </section>
                        ),
                    },
                    {
                        key: "invite",
                        label: "邀请",
                        children: me ? (
                            <dl className="grid gap-2 text-sm">
                                <div>序号：{me.serialNo || "—"}</div>
                                <div>名字：{me.displayName || "—"}</div>
                                <div>子域：{me.host}</div>
                                <div>邀请码：{me.inviteCode}</div>
                                <div>文本 / 图片 / 视频返利：{((me.textRebateBps ?? me.rebateRateBps ?? 0) / 100).toFixed(1)}% / {((me.imageRebateBps ?? me.rebateRateBps ?? 0) / 100).toFixed(1)}% / {((me.videoRebateBps ?? me.rebateRateBps ?? 0) / 100).toFixed(1)}%</div>
                                <div>
                                    落地页：
                                    <a className="underline" href={me.landingUrl}>{me.landingUrl}</a>
                                </div>
                            </dl>
                        ) : (
                            <p className="text-sm opacity-60">{loadingOverview ? "读取中…" : "暂无资料"}</p>
                        ),
                    },
                    {
                        key: "users",
                        label: "名下用户",
                        children: (
                            <Table
                                rowKey="userIdMasked"
                                loading={loadingUsers}
                                dataSource={users}
                                pagination={tablePagination(userPage, userTotal, (next) => void loadUsers(next))}
                                columns={[
                                    { title: "用户", dataIndex: "displayNameMasked" },
                                    { title: "编号", dataIndex: "userIdMasked" },
                                    { title: "注册时间", dataIndex: "registeredAt", render: formatTime },
                                    { title: "消耗", dataIndex: "consumedCredits", render: formatCredits },
                                    { title: "消费返利", dataIndex: "rebateCredits", render: formatCredits },
                                    { title: "剩余", dataIndex: "remainingCredits", render: formatCredits },
                                ]}
                            />
                        ),
                    },
                    {
                        key: "rebates",
                        label: "返利记录",
                        children: (
                            <div className="grid gap-4">
                                <Tabs
                                    size="small"
                                    activeKey={rebateType || "all"}
                                    onChange={(next) => void loadRebates(1, next === "all" ? "" : next)}
                                    items={[
                                        { key: "all", label: "全部" },
                                        { key: "text", label: "文本" },
                                        { key: "image", label: "图片" },
                                        { key: "video", label: "视频" },
                                    ]}
                                />
                                <Table
                                    rowKey="id"
                                    loading={loadingRebates}
                                    dataSource={rebates}
                                    pagination={tablePagination(rebatePage, rebateTotal, (next) => void loadRebates(next, rebateType))}
                                    columns={[
                                        { title: "时间", dataIndex: "createdAt", render: formatTime },
                                        { title: "类型", dataIndex: "capability", render: (value: string) => capabilityLabel[value] || value || "—" },
                                        { title: "用户", dataIndex: "sourceUserMasked" },
                                        { title: "消耗", dataIndex: "consumedCredits", render: formatCredits },
                                        { title: "模型比", dataIndex: "modelShareBps", render: formatPercent },
                                        { title: "代理比", dataIndex: "agentRebateBps", render: formatPercent },
                                        { title: "入账返利", dataIndex: "rebateCredits", render: formatCredits },
                                    ]}
                                />
                            </div>
                        ),
                    },
                    {
                        key: "payouts",
                        label: "提现",
                        children: (
                            <div className="grid gap-8">
                                <section className="max-w-xl">
                                    <h2 className="text-lg font-semibold">申请提现</h2>
                                    <p className="mt-2 text-sm opacity-60">同意后由官方线下转到支付宝，系统只记账。同时只能有一笔审核中。可提现 {withdrawable.toFixed(2)}</p>
                                    <Form
                                        form={form}
                                        className="mt-4 grid gap-3"
                                        layout="vertical"
                                        onFinish={async (values) => {
                                            try {
                                                await createAgentPayout({
                                                    amountCredits: Number(values.amountCredits),
                                                    alipayAccount: values.alipayAccount,
                                                    alipayRealName: values.alipayRealName,
                                                });
                                                message.success("已提交提现申请");
                                                form.setFieldValue("amountCredits", undefined);
                                                await Promise.all([loadOverview(), loadPayouts(1)]);
                                            } catch (error) {
                                                message.error(error instanceof Error ? error.message : "提交失败");
                                            }
                                        }}
                                    >
                                        <Form.Item name="amountCredits" label="金额（积分）" rules={[{ required: true, message: "填写金额" }]}>
                                            <InputNumber min={0.01} max={Math.max(withdrawable, 0.01)} step={0.01} className="w-full" />
                                        </Form.Item>
                                        <Form.Item name="alipayAccount" label="支付宝账号" rules={[{ required: true, message: "填写支付宝账号" }]}>
                                            <Input />
                                        </Form.Item>
                                        <Form.Item name="alipayRealName" label="真实姓名" rules={[{ required: true, message: "填写姓名" }]}>
                                            <Input />
                                        </Form.Item>
                                        <Button type="primary" htmlType="submit" disabled={withdrawable <= 0}>提交申请</Button>
                                    </Form>
                                </section>
                                <section>
                                    <h2 className="mb-3 text-lg font-semibold">提现记录</h2>
                                    <Table
                                        rowKey="id"
                                        loading={loadingPayouts}
                                        dataSource={payouts}
                                        pagination={tablePagination(payoutPage, payoutTotal, (next) => void loadPayouts(next))}
                                        columns={[
                                            { title: "时间", dataIndex: "createdAt", render: formatTime },
                                            { title: "金额", dataIndex: "amountCredits", render: formatCredits },
                                            { title: "支付宝", dataIndex: "alipayAccount" },
                                            { title: "姓名", dataIndex: "alipayRealName" },
                                            { title: "状态", dataIndex: "status", render: (value: StreamerPayout["status"]) => payoutStatus[value] || value },
                                            { title: "原因", dataIndex: "rejectReason" },
                                        ]}
                                    />
                                </section>
                            </div>
                        ),
                    },
                ]}
            />
            <p className="text-xs opacity-40">新返利进入可提现钱包，不能当创作积分花。主播本人消耗不算业绩。</p>
        </main>
    );
}

function Stat({ label, value }: { label: string; value: string }) {
    return (
        <div className="rounded-2xl border border-white/10 p-4">
            <div className="text-xs opacity-50">{label}</div>
            <div className="mt-1 text-2xl font-semibold">{value}</div>
        </div>
    );
}

function formatCredits(value: number) {
    return Number(value || 0).toFixed(2);
}

function formatPercent(value: number) {
    if (!Number.isFinite(value) || value <= 0) return "—";
    return `${(value / 100).toFixed(1)}%`;
}

function formatTime(value: string) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString("zh-CN");
}
