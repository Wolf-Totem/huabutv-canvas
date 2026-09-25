import { App, Button, Form, Input, InputNumber, Modal, Select, Switch, Table } from "antd";
import { useEffect, useState } from "react";

import {
    createAdminStreamer,
    disableAdminStreamer,
    enableAdminStreamer,
    getAdminStreamerSkin,
    listAdminAgentShareModels,
    listAdminStreamers,
    rotateAdminStreamerCode,
    updateAdminAgentShareModel,
    updateAdminStreamer,
    updateAdminStreamerSkin,
    uploadAdminStreamerHero,
    type AgentShareModel,
    type StreamerAdmin,
    type StreamerSkin,
} from "@/services/api/streamer";
import { listAdminUsers } from "@/services/api/auth";

const capabilityLabel: Record<string, string> = { text: "文本", image: "图片", video: "视频", audio: "音频" };

export default function StreamersPanel() {
    const { message } = App.useApp();
    const [items, setItems] = useState<StreamerAdmin[]>([]);
    const [userOptions, setUserOptions] = useState<Array<{ value: string; label: string }>>([]);
    const [saving, setSaving] = useState(false);
    const [form] = Form.useForm();
    const [editForm] = Form.useForm();
    const [skinForm] = Form.useForm();
    const [editing, setEditing] = useState<StreamerAdmin | null>(null);
    const [skinTarget, setSkinTarget] = useState<StreamerAdmin | null>(null);
    const [skin, setSkin] = useState<StreamerSkin | null>(null);
    const [shareModels, setShareModels] = useState<AgentShareModel[]>([]);
    const watchedVideo = Form.useWatch("heroVideoUrl", skinForm);
    const watchedPoster = Form.useWatch("heroPosterUrl", skinForm);

    const reload = async () => {
        const data = await listAdminStreamers();
        setItems(data.items || []);
    };

    const loadShares = async () => {
        const data = await listAdminAgentShareModels();
        setShareModels(data.items || []);
    };

    useEffect(() => {
        void reload().catch((error) => message.error(error instanceof Error ? error.message : "读取主播失败"));
        void loadShares().catch(() => undefined);
        void listAdminUsers({ page: 1, pageSize: 100 })
            .then((page) => {
                const rows = page.users || [];
                setUserOptions(rows.map((row) => ({ value: row.id, label: `${row.displayName || row.username} (${row.username})` })));
            })
            .catch(() => undefined);
    }, [message]);

    const openEdit = (row: StreamerAdmin) => {
        setEditing(row);
        editForm.setFieldsValue({
            displayName: row.displayName,
            slug: row.slug,
            inviteCode: row.inviteCode,
            serialNo: row.serialNo,
            note: row.note,
            textRebateBps: row.textRebateBps ?? row.rebateRateBps,
            imageRebateBps: row.imageRebateBps ?? row.rebateRateBps,
            videoRebateBps: row.videoRebateBps ?? row.rebateRateBps,
            customChannelsEnabled: row.customChannelsEnabled,
        });
    };

    return (
        <div className="grid gap-8">
            <Form
                form={form}
                layout="inline"
                onFinish={async (values) => {
                    setSaving(true);
                    try {
                        await createAdminStreamer({
                            userId: values.userId,
                            slug: values.slug,
                            displayName: values.displayName,
                            inviteCode: values.inviteCode,
                            rebateRateBps: Number(values.rebateRateBps ?? 1000),
                        });
                        form.resetFields();
                        await reload();
                        message.success("代理已创建");
                    } catch (error) {
                        message.error(error instanceof Error ? error.message : "创建失败");
                    } finally {
                        setSaving(false);
                    }
                }}
            >
                <Form.Item name="userId" rules={[{ required: true, message: "选择用户" }]}>
                    <Select showSearch optionFilterProp="label" placeholder="选择已有用户" style={{ minWidth: 240 }} options={userOptions} />
                </Form.Item>
                <Form.Item name="slug" rules={[{ required: true, message: "填写 slug" }]}>
                    <Input placeholder="子域 slug，如 zhangsan" />
                </Form.Item>
                <Form.Item name="displayName">
                    <Input placeholder="展示名" />
                </Form.Item>
                <Form.Item name="inviteCode">
                    <Input placeholder="邀请码，可空自动生成" />
                </Form.Item>
                <Form.Item name="rebateRateBps" initialValue={1000}>
                    <InputNumber min={0} max={10000} placeholder="默认返利万分比" />
                </Form.Item>
                <Button type="primary" htmlType="submit" loading={saving}>
                    创建代理
                </Button>
            </Form>

            <Table
                rowKey="id"
                dataSource={items}
                pagination={false}
                columns={[
                    { title: "序号", dataIndex: "serialNo", width: 80 },
                    { title: "名字", dataIndex: "displayName" },
                    { title: "域名", dataIndex: "slug", render: (value: string) => `${value}.j11.net` },
                    { title: "邀请码", dataIndex: "inviteCode" },
                    { title: "文本/图/视频", render: (_, row) => `${((row.textRebateBps ?? row.rebateRateBps) / 100).toFixed(1)}% / ${((row.imageRebateBps ?? row.rebateRateBps) / 100).toFixed(1)}% / ${((row.videoRebateBps ?? row.rebateRateBps) / 100).toFixed(1)}%` },
                    { title: "自定义渠道", dataIndex: "customChannelsEnabled", render: (value: boolean) => (value ? "开" : "关") },
                    { title: "总返利", dataIndex: "rebateTotalCredits", render: (value: number) => Number(value || 0).toFixed(2) },
                    { title: "可提现", dataIndex: "rebateWithdrawableCredits", render: (value: number) => Number(value || 0).toFixed(2) },
                    { title: "已提现", dataIndex: "rebateWithdrawnCredits", render: (value: number) => Number(value || 0).toFixed(2) },
                    { title: "状态", dataIndex: "status" },
                    {
                        title: "操作",
                        render: (_, row) => (
                            <div className="flex flex-wrap gap-2">
                                <Button size="small" onClick={() => openEdit(row)}>编辑</Button>
                                {row.status === "active" ? (
                                    <Button size="small" onClick={() => void disableAdminStreamer(row.id).then(reload)}>禁用</Button>
                                ) : (
                                    <Button size="small" onClick={() => void enableAdminStreamer(row.id).then(reload)}>启用</Button>
                                )}
                                <Button size="small" onClick={() => void rotateAdminStreamerCode(row.id).then(reload)}>换码</Button>
                                <Button
                                    size="small"
                                    onClick={() => {
                                        setSkinTarget(row);
                                        void getAdminStreamerSkin(row.id).then((data) => {
                                            setSkin(data.skin);
                                            skinForm.setFieldsValue({
                                                heroVideoUrl: data.skin.heroVideoUrl || "",
                                                heroPosterUrl: data.skin.heroPosterUrl || "",
                                            });
                                        });
                                    }}
                                >
                                    背景视频
                                </Button>
                            </div>
                        ),
                    },
                ]}
            />

            <section className="rounded-2xl border border-white/10 p-5">
                <h3 className="text-lg font-semibold">全站模型代理定价比</h3>
                <p className="mb-4 text-sm opacity-70">打开后，所有代理在该模型上的返利再乘这个百分比。关 = 100%。</p>
                <Table
                    rowKey="id"
                    dataSource={shareModels}
                    pagination={false}
                    size="small"
                    columns={[
                        { title: "模型", dataIndex: "name" },
                        { title: "类型", dataIndex: "capability", render: (value: string) => capabilityLabel[value] || value },
                        {
                            title: "打开",
                            dataIndex: "agentShareEnabled",
                            render: (value: boolean, row) => (
                                <Switch
                                    checked={value}
                                    onChange={(checked) => {
                                        void updateAdminAgentShareModel(row.id, { enabled: checked })
                                            .then(loadShares)
                                            .catch((error) => message.error(error instanceof Error ? error.message : "保存失败"));
                                    }}
                                />
                            ),
                        },
                        {
                            title: "比例 %",
                            dataIndex: "agentShareBps",
                            render: (value: number, row) => (
                                <InputNumber
                                    min={1}
                                    max={100}
                                    value={Math.round((value || 10000) / 100)}
                                    disabled={!row.agentShareEnabled}
                                    onBlur={(event) => {
                                        const next = Number((event.target as HTMLInputElement).value);
                                        if (!Number.isFinite(next)) return;
                                        void updateAdminAgentShareModel(row.id, { shareBps: Math.round(next * 100) })
                                            .then(loadShares)
                                            .catch((error) => message.error(error instanceof Error ? error.message : "保存失败"));
                                    }}
                                />
                            ),
                        },
                    ]}
                />
            </section>

            <Modal
                title={editing ? `编辑代理 ${editing.displayName}` : "编辑代理"}
                open={Boolean(editing)}
                onCancel={() => setEditing(null)}
                onOk={() => editForm.submit()}
                okText="保存"
                destroyOnHidden
            >
                <Form
                    form={editForm}
                    layout="vertical"
                    onFinish={async (values) => {
                        if (!editing) return;
                        try {
                            await updateAdminStreamer(editing.id, {
                                displayName: values.displayName,
                                slug: values.slug,
                                inviteCode: values.inviteCode,
                                serialNo: Number(values.serialNo),
                                note: values.note || "",
                                textRebateBps: Number(values.textRebateBps),
                                imageRebateBps: Number(values.imageRebateBps),
                                videoRebateBps: Number(values.videoRebateBps),
                                customChannelsEnabled: Boolean(values.customChannelsEnabled),
                            });
                            message.success("代理已保存");
                            setEditing(null);
                            await reload();
                        } catch (error) {
                            message.error(error instanceof Error ? error.message : "保存失败");
                        }
                    }}
                >
                    <Form.Item name="displayName" label="名字" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="serialNo" label="代理序号" rules={[{ required: true }]}><InputNumber min={1} className="w-full" /></Form.Item>
                    <Form.Item name="slug" label="代理域名" extra="保存后落地页为 {slug}.j11.net"><Input /></Form.Item>
                    <Form.Item name="inviteCode" label="邀请码"><Input /></Form.Item>
                    <Form.Item name="note" label="备注"><Input.TextArea rows={3} maxLength={500} /></Form.Item>
                    <div className="grid grid-cols-3 gap-3">
                        <Form.Item name="textRebateBps" label="文本万分比"><InputNumber min={0} max={10000} className="w-full" /></Form.Item>
                        <Form.Item name="imageRebateBps" label="图片万分比"><InputNumber min={0} max={10000} className="w-full" /></Form.Item>
                        <Form.Item name="videoRebateBps" label="视频万分比"><InputNumber min={0} max={10000} className="w-full" /></Form.Item>
                    </div>
                    <Form.Item name="customChannelsEnabled" label="自定义渠道" valuePropName="checked">
                        <Switch />
                    </Form.Item>
                    <p className="text-xs opacity-50">万分比 1000 = 10%。全站自定义渠道关掉时，这里打开也不会生效。</p>
                </Form>
            </Modal>

            {skinTarget ? (
                <Form
                    form={skinForm}
                    layout="vertical"
                    className="max-w-2xl"
                    onFinish={async (values) => {
                        try {
                            const result = await updateAdminStreamerSkin(skinTarget.id, {
                                heroVideoUrl: values.heroVideoUrl || "",
                                heroPosterUrl: values.heroPosterUrl || "",
                            });
                            setSkin(result.skin);
                            message.success("专属背景已保存。其它首页内容与官网相同。");
                            await reload();
                        } catch (error) {
                            message.error(error instanceof Error ? error.message : "保存失败");
                        }
                    }}
                >
                    <h3>代理 {skinTarget.displayName} 的首页背景</h3>
                    <p className="mb-4 text-sm opacity-70">落地页 {`https://${skinTarget.slug}.j11.net/`} 与官网同一套。这里只换背景视频，留空则用官方视频。</p>
                    <Form.Item name="heroVideoUrl" label="背景视频 URL（https，mp4/webm）">
                        <Input placeholder="留空则使用官网首页视频" />
                    </Form.Item>
                    <Form.Item label="或上传视频">
                        <Input
                            type="file"
                            accept="video/mp4,video/webm"
                            onChange={(event) => {
                                const file = event.target.files?.[0];
                                event.target.value = "";
                                if (!file) return;
                                void uploadAdminStreamerHero(skinTarget.id, "video", file)
                                    .then((result) => {
                                        setSkin(result.skin);
                                        skinForm.setFieldsValue({ heroVideoUrl: result.skin.heroVideoUrl || "" });
                                        message.success("背景视频已上传");
                                    })
                                    .catch((error) => message.error(error instanceof Error ? error.message : "上传失败"));
                            }}
                        />
                    </Form.Item>
                    <Form.Item name="heroPosterUrl" label="封面图 URL（可选）">
                        <Input placeholder="减动效或视频未出首帧时显示" />
                    </Form.Item>
                    <Form.Item label="或上传封面">
                        <Input
                            type="file"
                            accept="image/png,image/jpeg,image/webp"
                            onChange={(event) => {
                                const file = event.target.files?.[0];
                                event.target.value = "";
                                if (!file) return;
                                void uploadAdminStreamerHero(skinTarget.id, "poster", file)
                                    .then((result) => {
                                        setSkin(result.skin);
                                        skinForm.setFieldsValue({ heroPosterUrl: result.skin.heroPosterUrl || "" });
                                        message.success("封面已上传");
                                    })
                                    .catch((error) => message.error(error instanceof Error ? error.message : "上传失败"));
                            }}
                        />
                    </Form.Item>
                    {(() => {
                        const videoSrc = watchedVideo || (skin?.heroVideoResourceId && skinTarget ? `/api/admin/streamers/${skinTarget.id}/hero-video?v=${encodeURIComponent(skin.updatedAt || "")}` : "");
                        const posterSrc = watchedPoster || (skin?.heroPosterResourceId && skinTarget ? `/api/admin/streamers/${skinTarget.id}/hero-poster?v=${encodeURIComponent(skin.updatedAt || "")}` : "");
                        if (!videoSrc && !posterSrc) return null;
                        return (
                            <video
                                key={videoSrc || posterSrc}
                                className="mb-4 max-h-56 w-full rounded-xl object-cover"
                                src={videoSrc || undefined}
                                poster={posterSrc || undefined}
                                muted
                                loop
                                playsInline
                                autoPlay
                                controls
                            />
                        );
                    })()}
                    <div className="flex flex-wrap gap-2">
                        <Button type="primary" htmlType="submit">保存 URL</Button>
                        <Button
                            onClick={() => {
                                void updateAdminStreamerSkin(skinTarget.id, { clearHeroVideo: true, clearHeroPoster: true })
                                    .then((result) => {
                                        setSkin(result.skin);
                                        skinForm.setFieldsValue({ heroVideoUrl: "", heroPosterUrl: "" });
                                        message.success("已恢复官网背景");
                                    })
                                    .catch((error) => message.error(error instanceof Error ? error.message : "恢复失败"));
                            }}
                        >
                            恢复官网默认
                        </Button>
                        <Button onClick={() => { setSkinTarget(null); setSkin(null); }}>关闭</Button>
                    </div>
                </Form>
            ) : null}
        </div>
    );
}
