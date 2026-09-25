import { useCallback, useEffect, useMemo, useState } from "react";
import { App, Button, Input, Modal } from "antd";

import { AdminPageFrame } from "./components/admin-shell";
import { AdminDataTable, AdminStatusBadge } from "./components/admin-ui";
import { deletePlazaWork, listAdminPlazaWorks, seedAdminPlazaExternal, takeDownPlazaWork, type PlazaWork } from "@/services/api/plaza";

const KEEP_SLUG = "0251b9ae0e304f7fb96e353eecfe2204";

function publicPlazaUrl(slug: string) {
    if (typeof window === "undefined") return `/plaza/${slug}`;
    return `${window.location.origin}/plaza/${encodeURIComponent(slug)}`;
}

function sourcePlazaUrl(sourceProjectId?: string) {
    const id = String(sourceProjectId || "").replace(/^ext:/, "").trim();
    if (!id) return "";
    return `https://www.liblib.tv/detail/${id}`;
}

function parseImportUrls(raw: string) {
    const ids = Array.from(raw.matchAll(/\/(?:detail|plaza|canvas)\/([0-9a-f]{32})/gi)).map((item) => item[1].toLowerCase());
    const unique: string[] = [];
    const seen = new Set<string>();
    for (const id of ids) {
        if (seen.has(id) || id === KEEP_SLUG) continue;
        seen.add(id);
        unique.push(id);
    }
    return unique;
}

export default function PlazaWorksPage() {
    const { message } = App.useApp();
    const [items, setItems] = useState<PlazaWork[]>([]);
    const [takeDownId, setTakeDownId] = useState<string | null>(null);
    const [takeDownNote, setTakeDownNote] = useState("");
    const [deleteTarget, setDeleteTarget] = useState<PlazaWork | null>(null);
    const [deleteConfirm, setDeleteConfirm] = useState("");
    const [importText, setImportText] = useState("");
    const [importing, setImporting] = useState(false);
    const load = useCallback(async () => {
        const result = await listAdminPlazaWorks({ page: 1, pageSize: 100 });
        setItems(result.works);
    }, []);
    useEffect(() => { void load().catch((error) => message.error(error instanceof Error ? error.message : "读取失败")); }, [load, message]);
    const parsed = useMemo(() => parseImportUrls(importText), [importText]);

    return (
        <AdminPageFrame title="广场展示" description="管理作品广场对外展示地址。可粘贴公开画布链接批量导入，也可复制本站展示链接或彻底删除。">
            <div className="mb-4 space-y-2 rounded-xl border border-border p-4">
                <strong className="text-sm">导入展示地址</strong>
                <p className="text-xs text-foreground/55">每行一个公开画布详情链接。保留作品 {KEEP_SLUG} 不会被覆盖。导入后本站地址为 /plaza/对应编号。</p>
                <Input.TextArea
                    value={importText}
                    onChange={(event) => setImportText(event.target.value)}
                    rows={6}
                    placeholder="https://www.example.com/detail/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                />
                <div className="flex items-center gap-3">
                    <Button
                        type="primary"
                        loading={importing}
                        disabled={!parsed.length}
                        onClick={async () => {
                            setImporting(true);
                            try {
                                const report = await seedAdminPlazaExternal(parsed.map((uuid) => ({
                                    uuid,
                                    slug: uuid,
                                    title: uuid.slice(0, 8),
                                    categoryId: "plaza-cat-featured",
                                    projectUuid: uuid,
                                })));
                                message.success(`导入完成：成功 ${report.report.imported}，失败 ${report.report.failed}`);
                                setImportText("");
                                await load();
                            } catch (error) {
                                message.error(error instanceof Error ? error.message : "导入失败");
                            } finally {
                                setImporting(false);
                            }
                        }}
                    >
                        导入 {parsed.length} 条
                    </Button>
                    <span className="text-xs text-foreground/55">已识别 {parsed.length} 个编号</span>
                </div>
            </div>
            <AdminDataTable
                table={{
                    rowKey: "id",
                    dataSource: items,
                    columns: [
                        { title: "标题", dataIndex: "title" },
                        {
                            title: "本站展示地址",
                            render: (_, row: PlazaWork) => {
                                const href = publicPlazaUrl(row.slug);
                                return (
                                    <span style={{ display: "inline-flex", gap: 8, alignItems: "center" }}>
                                        <a href={href} target="_blank" rel="noreferrer">{href}</a>
                                        <Button size="small" onClick={() => { void navigator.clipboard.writeText(href); message.success("已复制"); }}>复制</Button>
                                    </span>
                                );
                            },
                        },
                        {
                            title: "来源地址",
                            render: (_, row: PlazaWork) => {
                                const href = sourcePlazaUrl(row.sourceProjectId);
                                return href ? <a href={href} target="_blank" rel="noreferrer">{href}</a> : "—";
                            },
                        },
                        { title: "状态", dataIndex: "status", render: (value: string) => <AdminStatusBadge label={value} /> },
                        { title: "过程", render: (_, row: PlazaWork) => row.allowProcessView ? "有" : "无" },
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
