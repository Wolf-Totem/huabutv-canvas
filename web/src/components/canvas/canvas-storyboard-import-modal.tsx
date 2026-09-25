import { App, Button, Input, Modal, Radio, Table } from "antd";
import { Download, Upload } from "lucide-react";
import { useMemo, useRef, useState } from "react";

import {
    applyStoryboardImport,
    parseStoryboardImportText,
    storyboardImportTemplateCsv,
    type StoryboardImportDraftRow,
} from "@/lib/canvas/canvas-storyboard-import";
import type { CanvasNodeData, StoryboardRow } from "@/types/canvas";

type ImportMode = "replace" | "append";

export function CanvasStoryboardImportModal({
    open,
    nodes,
    existingRowCount,
    onClose,
    onImport,
}: {
    open: boolean;
    nodes: CanvasNodeData[];
    existingRowCount: number;
    onClose: () => void;
    onImport: (rows: StoryboardRow[], mode: ImportMode) => void;
}) {
    const { message } = App.useApp();
    const fileRef = useRef<HTMLInputElement>(null);
    const [mode, setMode] = useState<ImportMode>(existingRowCount ? "append" : "replace");
    const [paste, setPaste] = useState("");
    const [drafts, setDrafts] = useState<StoryboardImportDraftRow[]>([]);
    const [warnings, setWarnings] = useState<string[]>([]);
    const applied = useMemo(() => (drafts.length ? applyStoryboardImport(drafts, nodes) : null), [drafts, nodes]);

    const loadText = (text: string) => {
        const parsed = parseStoryboardImportText(text);
        setDrafts(parsed.rows);
        setWarnings(parsed.warnings);
        if (!parsed.rows.length) message.error(parsed.warnings[0] || "没有读到镜头");
        else if (parsed.warnings.length) message.warning(parsed.warnings[0]);
    };

    const pickFile = async (file: File) => {
        try {
            loadText(await file.text());
        } catch {
            message.error("读取文件失败");
        }
    };

    const downloadTemplate = () => {
        const blob = new Blob([storyboardImportTemplateCsv()], { type: "text/csv;charset=utf-8" });
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = "分镜提示词模板.csv";
        link.click();
        URL.revokeObjectURL(url);
    };

    const confirm = () => {
        if (!applied?.rows.length) {
            message.error("请先上传或粘贴分镜表");
            return;
        }
        onImport(applied.rows, mode);
        const unmatched = applied.unmatchedAssets;
        if (unmatched.length) message.warning(`已导入 ${applied.rows.length} 镜，未匹配资产：${unmatched.slice(0, 6).join("、")}${unmatched.length > 6 ? "…" : ""}`);
        else message.success(`已导入 ${applied.rows.length} 镜，自动关联 ${applied.matchedAssetCount} 个资产`);
        onClose();
        setPaste("");
        setDrafts([]);
        setWarnings([]);
    };

    return (
        <Modal
            title="导入分镜提示词"
            open={open}
            onCancel={onClose}
            width={880}
            centered
            destroyOnHidden
            footer={[
                <Button key="cancel" onClick={onClose}>取消</Button>,
                <Button key="ok" type="primary" disabled={!applied?.rows.length} onClick={confirm}>
                    {mode === "append" ? "追加导入" : "覆盖导入"}
                </Button>,
            ]}
        >
            <div className="flex flex-col gap-3" data-canvas-no-zoom>
                <p className="m-0 text-sm text-foreground/65">
                    按固定模板准备镜头：每行一个镜头。关联资产、角色列填写画布上已有节点的名称（多个用逗号或顿号分隔），导入后会自动挂到该镜。
                </p>
                <div className="flex flex-wrap items-center gap-2">
                    <Button icon={<Download className="size-3.5" />} onClick={downloadTemplate}>下载 CSV 模板</Button>
                    <Button icon={<Upload className="size-3.5" />} onClick={() => fileRef.current?.click()}>上传 CSV / JSON</Button>
                    <input
                        ref={fileRef}
                        type="file"
                        accept=".csv,.tsv,.txt,.json,.md,text/csv,application/json"
                        className="hidden"
                        onChange={(event) => {
                            const file = event.target.files?.[0];
                            event.target.value = "";
                            if (file) void pickFile(file);
                        }}
                    />
                    <Radio.Group value={mode} onChange={(event) => setMode(event.target.value)}>
                        <Radio.Button value="append">追加到现有镜头</Radio.Button>
                        <Radio.Button value="replace">覆盖当前分镜表</Radio.Button>
                    </Radio.Group>
                </div>
                <Input.TextArea
                    value={paste}
                    rows={5}
                    placeholder="也可从 Excel 复制带表头的单元格，粘贴到这里"
                    onChange={(event) => setPaste(event.target.value)}
                    onPaste={(event) => {
                        const text = event.clipboardData.getData("text");
                        if (text.trim()) {
                            event.preventDefault();
                            setPaste(text);
                            loadText(text);
                        }
                    }}
                />
                {paste.trim() && !drafts.length ? <Button onClick={() => loadText(paste)}>解析粘贴内容</Button> : null}
                {warnings.length ? <div className="text-xs text-amber-600">{warnings.join("；")}</div> : null}
                {applied?.unmatchedAssets.length ? <div className="text-xs text-amber-600">未匹配资产：{applied.unmatchedAssets.join("、")}</div> : null}
                {applied?.rows.length ? (
                    <Table
                        size="small"
                        pagination={false}
                        rowKey={(_row, index) => String(index)}
                        scroll={{ y: 280 }}
                        dataSource={applied.rows}
                        columns={[
                            { title: "序号", dataIndex: "shotNumber", width: 64 },
                            { title: "时长", dataIndex: "durationSeconds", width: 72, render: (value: number) => `${value}s` },
                            { title: "视频提示词", dataIndex: "videoMotionPrompt", ellipsis: true },
                            { title: "台词", dataIndex: "dialogue", width: 160, ellipsis: true },
                            { title: "资产", dataIndex: "assetBindings", width: 88, render: (value: StoryboardRow["assetBindings"]) => `${value?.length || 0} 个` },
                        ]}
                    />
                ) : null}
            </div>
        </Modal>
    );
}
