import { Modal } from "antd";
import { Tooltip } from "@/components/ui/base/tooltip";
import { useEffect, useMemo, useState } from "react";

import { Image as ImageIcon, Music2, Play, UserRound, X } from "lucide-react";

import { canvasNodeVideoPreviewUrl } from "@/lib/canvas/canvas-media-preview";
import { isStoryboardPreviewAsset } from "@/lib/canvas/canvas-storyboard-materializer";
import { resolveMediaUrl } from "@/services/file-storage";
import { CanvasNodeType, type CanvasNodeData, type StoryboardAssetBinding } from "@/types/canvas";

const ROLE_LABELS: Record<StoryboardAssetBinding["role"], string> = {
    character: "角色",
    environment: "场景",
    wardrobe: "服装",
    prop: "道具",
    weapon: "武器",
    style: "风格",
    motion: "动态",
    audio: "音频",
};

export function StoryboardAssetsCell({ bindings, nodes, limit = 4, onRemove }: { bindings: StoryboardAssetBinding[]; nodes: CanvasNodeData[]; limit?: number; onRemove?: (nodeId: string) => void }) {
    const [previewNode, setPreviewNode] = useState<CanvasNodeData | null>(null);
    const nodeById = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
    const assets = bindings.map((binding) => ({ binding, node: nodeById.get(binding.nodeId) })).filter((item) => !item.node || isStoryboardPreviewAsset(item.node));
    const visible = assets.slice(0, limit);
    const hiddenCount = Math.max(0, assets.length - visible.length);

    if (!assets.length) return <span className="text-[var(--fs-caption)] text-foreground/35">未关联</span>;
    return (
        <>
            <div className="flex min-w-0 items-center gap-1.5" aria-label={`已关联 ${assets.length} 个资产`}>
                {visible.map(({ binding, node }) => {
                    const title = node?.title || "资产已失效";
                    return (
                        <div key={binding.nodeId} className="group relative min-w-0">
                            <Tooltip title={`${title} · ${ROLE_LABELS[binding.role]}`}>
                                <button
                                    type="button"
                                    disabled={!node}
                                    className="flex min-w-0 max-w-[88px] items-center gap-1 rounded-md border border-foreground/10 bg-foreground/[0.035] px-0.5 py-0.5 pr-3.5 text-left text-foreground/70 outline-none transition enabled:hover:border-foreground/30 enabled:hover:text-foreground focus-visible:ring-2 focus-visible:ring-[var(--color-primary)] disabled:cursor-not-allowed"
                                    aria-label={`预览${node?.title || "失效资产"}`}
                                    onMouseDown={(event) => event.stopPropagation()}
                                    onPointerDown={(event) => event.stopPropagation()}
                                    onClick={(event) => {
                                        event.stopPropagation();
                                        if (node) setPreviewNode(node);
                                    }}
                                >
                                    <span className="relative grid size-8 shrink-0 place-items-center overflow-hidden rounded-[5px]">
                                        {node ? <AssetThumbnail node={node} /> : <ImageIcon className="size-4" />}
                                        <span className="absolute bottom-0 right-0 rounded bg-black/65 px-0.5 text-[7px] leading-3 text-white">{ROLE_LABELS[binding.role].slice(0, 1)}</span>
                                    </span>
                                    <span className="min-w-0 truncate text-[10px] font-medium leading-3">{node?.title || "已失效"}</span>
                                </button>
                            </Tooltip>
                            {onRemove ? (
                                <button
                                    type="button"
                                    className="absolute -right-0.5 -top-0.5 grid size-4 min-h-4 min-w-4 place-items-center rounded-full bg-foreground text-[10px] leading-none text-background opacity-90 shadow-sm outline-none hover:opacity-100 focus-visible:ring-2 focus-visible:ring-[var(--color-primary)] group-hover:opacity-100"
                                    aria-label={node?.title ? `从本镜移除「${node.title}」` : "移除失效资产"}
                                    onMouseDown={(event) => event.stopPropagation()}
                                    onPointerDown={(event) => event.stopPropagation()}
                                    onClick={(event) => {
                                        event.preventDefault();
                                        event.stopPropagation();
                                        onRemove(binding.nodeId);
                                    }}
                                >
                                    <X className="size-2.5" strokeWidth={2.5} />
                                </button>
                            ) : null}
                        </div>
                    );
                })}
                {hiddenCount ? <span className="shrink-0 text-[var(--fs-caption)] font-medium text-foreground/45">+{hiddenCount}</span> : null}
            </div>
            <AssetPreviewModal node={previewNode} onClose={() => setPreviewNode(null)} />
        </>
    );
}

function AssetThumbnail({ node }: { node: CanvasNodeData }) {
    const videoPreview = canvasNodeVideoPreviewUrl(node);
    const source = useNodeMediaSource(node.type === CanvasNodeType.Video ? null : node);
    if (node.type === CanvasNodeType.Audio) return <Music2 className="size-4" />;
    if (node.metadata?.workflowKind === "character" && !source) return <UserRound className="size-4" />;
    if (node.type === CanvasNodeType.Video) {
        return videoPreview ? (
            <>
                <img src={videoPreview} alt="" loading="lazy" decoding="async" draggable={false} className="size-full object-cover" />
                <span className="absolute inset-0 grid place-items-center bg-black/15"><Play className="size-3.5 fill-white text-white" /></span>
            </>
        ) : <Play className="size-4" />;
    }
    return source ? <img src={source} alt="" loading="lazy" decoding="async" draggable={false} className="size-full object-cover" /> : <ImageIcon className="size-4" />;
}

function AssetPreviewModal({ node, onClose }: { node: CanvasNodeData | null; onClose: () => void }) {
    const source = useNodeMediaSource(node);
    return (
        <Modal title={node?.title || "资产预览"} open={Boolean(node)} onCancel={onClose} footer={null} width={880} centered destroyOnHidden>
            {node ? (
                <div className="grid min-h-56 place-items-center overflow-hidden rounded-lg bg-black/[0.035] p-3 dark:bg-white/[0.035]" data-canvas-no-zoom>
                    {node.type === CanvasNodeType.Video && source ? <video src={source} controls autoPlay playsInline className="max-h-[68vh] max-w-full rounded-md" />
                        : node.type === CanvasNodeType.Audio && source ? <audio src={source} controls autoPlay className="w-full max-w-xl" />
                            : source ? <img src={source} alt={node.title || "资产预览"} className="max-h-[68vh] max-w-full object-contain" />
                                : <span className="text-sm text-foreground/45">当前资产没有可预览的媒体内容</span>}
                </div>
            ) : null}
        </Modal>
    );
}

function useNodeMediaSource(node: CanvasNodeData | null) {
    const fallback = node ? node.metadata?.workflowKind === "character"
        ? node.metadata.characterCoverUrl || ""
        : node.type === CanvasNodeType.Drawing
            ? node.metadata?.drawingPreviewUrl || node.metadata?.content || ""
            : node.metadata?.content || "" : "";
    const storageKey = node?.metadata?.storageKey;
    const [source, setSource] = useState(fallback);
    useEffect(() => {
        let cancelled = false;
        setSource(fallback);
        if (storageKey) void resolveMediaUrl(storageKey, fallback).then((url) => {
            if (!cancelled) setSource(url || fallback);
        }).catch(() => {
            if (!cancelled) setSource(fallback);
        });
        return () => { cancelled = true; };
    }, [fallback, storageKey]);
    return source;
}
