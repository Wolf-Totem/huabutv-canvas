import { useEffect, useMemo, useRef, useState } from "react";
import { App, Button } from "antd";
import { ChevronLeft, Clapperboard, Heart, Play, Share2, X } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router";

import { ossProcessedImage } from "@/lib/oss-image";
import { featuredBySlug, featuredCanvases, featuredTourProject, formatPlazaTime, watchFromPlazaWork, type FeaturedCanvas } from "@/lib/plaza-catalog";
import { copyPlazaWork, getPlazaSnapshot, getPlazaWork, likePlazaWork, recordPlazaEvent, unlikePlazaWork, type PlazaWork } from "@/services/api/plaza";
import { createCanvasProjectWithRemoteSync, hasRemoteUserDataSyncSession } from "@/services/user-data-sync";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import type { CanvasProject } from "@/stores/canvas/use-canvas-store";
import { useUserStore } from "@/stores/use-user-store";
import { ReadOnlyCanvasView } from "@/pages/canvas/read-only-canvas";
import "./plaza-watch.css";

const LIKE_KEY = "plaza:liked-slugs";

function readLocalLikes(): string[] {
    try {
        const raw = window.localStorage.getItem(LIKE_KEY);
        const parsed = raw ? JSON.parse(raw) : [];
        return Array.isArray(parsed) ? parsed.filter((item) => typeof item === "string") : [];
    } catch {
        return [];
    }
}

function writeLocalLikes(slugs: string[]) {
    try {
        window.localStorage.setItem(LIKE_KEY, JSON.stringify(slugs));
    } catch {
        // ignore
    }
}

export default function PlazaWorkPage() {
    const { slug = "" } = useParams();
    const navigate = useNavigate();
    const { message } = App.useApp();
    const user = useUserStore((state) => state.user);
    const previewRef = useRef<HTMLVideoElement>(null);
    const watchRef = useRef<HTMLVideoElement>(null);
    const stripRef = useRef<HTMLDivElement>(null);
    const [apiWork, setApiWork] = useState<PlazaWork | null>(null);
    const [liked, setLiked] = useState(false);
    const [watching, setWatching] = useState(false);
    const [tourOpen, setTourOpen] = useState(false);
    const [tourProject, setTourProject] = useState<CanvasProject | null>(null);
    const [hoverStrip, setHoverStrip] = useState(false);

    const item = useMemo(() => {
        const catalog = featuredBySlug(slug);
        if (!apiWork) return catalog;
        const fromApi = watchFromPlazaWork(apiWork);
        if (!catalog) return fromApi;
        return {
            ...fromApi,
            coverUrl: fromApi.coverUrl || catalog.coverUrl,
            previewUrl: fromApi.previewUrl || catalog.previewUrl,
            watchUrl: fromApi.watchUrl || catalog.watchUrl,
            hlsUrl: fromApi.hlsUrl || catalog.hlsUrl,
        };
    }, [apiWork, slug]);
    const others = useMemo(() => featuredCanvases(), []);

    useEffect(() => {
        let active = true;
        setWatching(false);
        setTourOpen(false);
        setLiked(readLocalLikes().includes(slug));
        getPlazaWork(slug).then(({ work }) => {
            if (!active) return;
            setApiWork(work);
            setLiked(Boolean(work.liked) || readLocalLikes().includes(slug));
            void recordPlazaEvent(work.id, "view");
        }).catch(() => {
            if (active) setApiWork(null);
        });
        return () => { active = false; };
    }, [slug]);

    useEffect(() => {
        const node = previewRef.current;
        if (!node) return;
        if (watching) {
            node.pause();
            return;
        }
        node.muted = true;
        void node.play().catch(() => undefined);
    }, [slug, watching, item?.previewUrl]);

    useEffect(() => {
        const root = stripRef.current;
        if (!root || !others.length) return;
        let offset = root.scrollLeft;
        let frame = 0;
        const tick = () => {
            if (!hoverStrip && !watching && !tourOpen) {
                offset += 0.45;
                const max = Math.max(0, root.scrollWidth - root.clientWidth);
                if (offset >= max) offset = 0;
                root.scrollLeft = offset;
            } else {
                offset = root.scrollLeft;
            }
            frame = window.requestAnimationFrame(tick);
        };
        frame = window.requestAnimationFrame(tick);
        return () => window.cancelAnimationFrame(frame);
    }, [others.length, hoverStrip, watching, tourOpen]);

    if (!item) return <div className="grid min-h-screen place-items-center bg-black text-white">作品不存在</div>;

    const requireLogin = (next: string) => useAuthDialogStore.getState().openAuth({ tab: "login", next });
    const previewSrc = item.previewUrl || item.watchUrl;
    const watchSrc = item.watchUrl || item.previewUrl;
    const likes = item.likeCount + (liked ? 1 : 0);

    const playLoud = () => {
        if (apiWork) void recordPlazaEvent(apiWork.id, "watch");
        setWatching(true);
        window.setTimeout(() => {
            const node = watchRef.current;
            if (!node) return;
            node.muted = false;
            void node.play().catch(() => undefined);
        }, 30);
    };

    const share = async () => {
        await navigator.clipboard.writeText(`${window.location.origin}/plaza/${encodeURIComponent(item.slug)}`);
        message.success("链接已复制");
    };

    const like = async () => {
        if (!user) {
            requireLogin(`/plaza/${item.slug}`);
            return;
        }
        const next = !liked;
        setLiked(next);
        const local = new Set(readLocalLikes());
        if (next) local.add(item.slug);
        else local.delete(item.slug);
        writeLocalLikes([...local]);
        if (!apiWork) return;
        try {
            const result = liked ? await unlikePlazaWork(apiWork.id) : await likePlazaWork(apiWork.id);
            setApiWork(result.work);
            setLiked(Boolean(result.work.liked));
        } catch (error) {
            message.error(error instanceof Error ? error.message : "点赞失败");
        }
    };

    const openTour = async () => {
        try {
            const { work } = apiWork ? { work: apiWork } : await getPlazaWork(item.slug);
            const snapshot = await getPlazaSnapshot(work.id);
            setApiWork(work);
            setTourProject(snapshot.project);
            setTourOpen(true);
            void recordPlazaEvent(work.id, "tour");
        } catch {
            message.warning("该作品暂未开放制作过程");
        }
    };

    const copyProject = async () => {
        if (!user || !hasRemoteUserDataSyncSession()) {
            requireLogin(`/plaza/${item.slug}`);
            return;
        }
        if (apiWork) {
            try {
                const result = await copyPlazaWork(apiWork.id);
                message.success("已复制到你的画布");
                navigate(`/canvas/${result.projectId}`);
                return;
            } catch {
                // featured fallback
            }
        }
        const snapshot = featuredTourProject(item);
        const result = await createCanvasProjectWithRemoteSync(`${item.title} 副本`, undefined, { nodes: snapshot.nodes, connections: snapshot.connections });
        if (result.syncError) message.warning("已复制到本机画布，云端稍后同步");
        else message.success("已复制到你的画布");
        navigate(`/canvas/${result.id}`);
    };

    return (
        <div className="plaza-watch">
            {previewSrc && !watching ? (
                <video ref={previewRef} className="plaza-watch-preview" src={previewSrc} poster={ossProcessedImage(item.coverUrl, 1600) || item.coverUrl} autoPlay muted loop playsInline preload="auto" />
            ) : (
                <img className="plaza-watch-preview" src={ossProcessedImage(item.coverUrl, 1600) || item.coverUrl} alt="" />
            )}
            <div className="plaza-watch-scrim" />

            <div className={`plaza-watch-chrome ${watching ? "is-hidden" : ""}`}>
                <div className="plaza-watch-top">
                    <button type="button" className="plaza-watch-back" onClick={() => navigate(-1)} aria-label="返回"><ChevronLeft /></button>
                    <Link className="plaza-watch-author" to={`/u/${encodeURIComponent(item.authorId)}`}>
                        <img className="plaza-watch-avatar" src={item.authorAvatar || "/logo.png"} alt="" />
                        <span>{item.authorName}</span>
                    </Link>
                    <span className="plaza-watch-dot" />
                    <span className="plaza-watch-title">{item.title}</span>
                    <div className="plaza-watch-meta">
                        {item.updatedAt ? <div>更新时间: {formatPlazaTime(item.updatedAt)}</div> : null}
                        <div>含 AI 生成内容</div>
                    </div>
                </div>

                <div className="plaza-watch-bar">
                    <div className="plaza-watch-actions">
                        <button type="button" className="is-play" onClick={playLoud}><Play size={16} /> 立即观看</button>
                        <button type="button" className="is-ghost" onClick={() => void openTour()}><Clapperboard size={16} /> 查看制作过程</button>
                        <button type="button" className={`is-icon ${liked ? "is-liked" : ""}`} aria-label={`点赞，当前 ${likes}`} onClick={() => void like()}><Heart size={16} fill={liked ? "currentColor" : "none"} /></button>
                        <button type="button" className="is-icon" aria-label="复制链接" onClick={() => void share()}><Share2 size={16} /></button>
                    </div>
                </div>

                <div className="plaza-watch-strip-wrap" onMouseEnter={() => setHoverStrip(true)} onMouseLeave={() => setHoverStrip(false)}>
                    <div ref={stripRef} className="plaza-watch-strip">
                        {others.map((row) => (
                            <button key={row.slug} type="button" data-detail-active={row.slug === item.slug ? "true" : undefined} className={`plaza-watch-thumb ${row.slug === item.slug ? "is-active" : ""}`} onClick={() => navigate(`/plaza/${row.slug}`)}>
                                <img src={ossProcessedImage(row.coverUrl, 400) || row.coverUrl} alt="" />
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            {watching ? (
                <div className="plaza-watch-player">
                    <button type="button" className="plaza-watch-player-close" aria-label="关闭" onClick={() => setWatching(false)}><X size={18} /></button>
                    {watchSrc ? (
                        <video ref={watchRef} className="plaza-watch-player-video" src={watchSrc} poster={ossProcessedImage(item.coverUrl, 1600) || item.coverUrl} controls autoPlay playsInline />
                    ) : (
                        <img src={ossProcessedImage(item.coverUrl, 1600) || item.coverUrl} alt="" />
                    )}
                </div>
            ) : null}

            {tourOpen ? (
                <div className="plaza-watch-tour">
                    {tourProject ? (
                        <ReadOnlyCanvasView title={item.title} project={tourProject} mode="tour" onCopy={() => void copyProject()} headerRight={<Button onClick={() => setTourOpen(false)}>关闭</Button>} />
                    ) : (
                        <div className="grid h-full place-items-center text-white">正在打开制作过程</div>
                    )}
                </div>
            ) : null}
        </div>
    );
}


