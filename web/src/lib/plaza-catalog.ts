import featured from "@/lib/plaza-featured.json";
import type { PlazaWork } from "@/services/api/plaza";
import { CanvasNodeType, type CanvasConnection } from "@/types/canvas";
import type { CanvasProject } from "@/stores/canvas/use-canvas-store";

export const PLAZA_DEMO_AUTHOR = {
    id: "plaza-demo",
    name: "画布TV精选",
    avatarUrl: "/logo.png",
    bio: "演示账号，用于作品广场预览、制作过程参观和复制项目。",
};

export const WORK_TAGS = [
    { slug: "campaign-manhua", name: "AI漫剧崛起计划" },
    { slug: "campaign-director", name: "全民导演请开机" },
    { slug: "featured", name: "精选画布" },
    { slug: "film", name: "专业影视" },
    { slug: "short-drama", name: "短剧漫剧" },
    { slug: "ad", name: "商业广告" },
    { slug: "game", name: "动漫游戏" },
    { slug: "edu", name: "教育生活" },
] as const;

export type WorkTagSlug = (typeof WORK_TAGS)[number]["slug"];

export type FeaturedCanvas = {
    slug: string;
    projectId: string;
    title: string;
    subtitle: string;
    coverUrl: string;
    previewUrl: string;
    watchUrl: string;
    hlsUrl: string;
    authorName: string;
    authorId: string;
    authorAvatar: string;
    tag: WorkTagSlug | string;
    tags: string[];
    mode: "video" | "image" | "text";
    likeCount: number;
    updatedAt: string;
};

type FeaturedRaw = {
    id?: string;
    projectUuid?: string;
    publicationKey?: string;
    name?: string;
    description?: string;
    coverUrl?: string;
    previewUrl?: string;
    watchUrl?: string;
    hlsUrl?: string;
    authorName?: string;
    authorId?: string;
    authorAvatar?: string;
    tag?: string;
    tags?: string[];
    likeCount?: number;
    updatedAt?: string;
};

function inferTag(name: string, description: string, tags: string[]): string {
    const mapped: Record<string, string> = {
        精选画布: "featured",
        短剧漫剧: "short-drama",
        短片剧集: "short-drama",
        专业影视: "film",
        商业广告: "ad",
        电商爆款再推出: "ad",
        动漫游戏: "game",
        教育生活: "edu",
        AI漫剧崛起计划: "campaign-manhua",
        全民导演请开机: "campaign-director",
    };
    for (const tag of tags) {
        if (mapped[tag]) return mapped[tag];
    }
    const text = `${name} ${description}`;
    if (/广告|品牌|饮料|咖啡/.test(text)) return "ad";
    if (/清华|迎新|教育/.test(text)) return "edu";
    if (/游戏|王者|魔兽|机甲/.test(text)) return "game";
    if (/电影|预告|姜文|真人/.test(text)) return "film";
    if (/短剧|喜剧|穿越|仙侠|修仙/.test(text)) return "short-drama";
    return "featured";
}

export function featuredCanvases(): FeaturedCanvas[] {
    return (featured as FeaturedRaw[]).map((item, index) => {
        const slug = String(item.publicationKey || item.id || item.projectUuid || `featured-${index}`);
        const title = String(item.name || "精选画布");
        const subtitle = String(item.description || "").replace(/\\n/g, "\n");
        const tags = (item.tags || []).map(String).filter(Boolean);
        const previewUrl = String(item.previewUrl || "");
        const watchUrl = String(item.watchUrl || item.previewUrl || "");
        return {
            slug,
            projectId: String(item.projectUuid || slug),
            title,
            subtitle,
            coverUrl: String(item.coverUrl || ""),
            previewUrl,
            watchUrl,
            hlsUrl: String(item.hlsUrl || ""),
            authorName: PLAZA_DEMO_AUTHOR.name,
            authorId: PLAZA_DEMO_AUTHOR.id,
            authorAvatar: PLAZA_DEMO_AUTHOR.avatarUrl,
            tag: item.tag || inferTag(title, subtitle, tags),
            tags: tags.length ? tags : ["精选画布"],
            mode: previewUrl || watchUrl ? "video" : "image",
            likeCount: Number(item.likeCount) || 0,
            updatedAt: String(item.updatedAt || ""),
        };
    });
}

export function featuredBySlug(slug: string) {
    const key = String(slug || "").trim();
    return featuredCanvases().find((item) => item.slug === key || item.projectId === key) || null;
}

export function watchFromPlazaWork(work: PlazaWork): FeaturedCanvas {
    const title = work.title;
    const subtitle = work.subtitle;
    const watchUrl = work.watchUrl || "";
    const tags = work.category?.name ? [work.category.name] : [];
    return {
        slug: work.slug,
        projectId: work.id,
        title,
        subtitle,
        coverUrl: work.coverUrl || "",
        previewUrl: watchUrl,
        watchUrl,
        hlsUrl: "",
        authorName: work.author?.displayName || PLAZA_DEMO_AUTHOR.name,
        authorId: work.author?.id || PLAZA_DEMO_AUTHOR.id,
        authorAvatar: work.author?.avatarUrl || PLAZA_DEMO_AUTHOR.avatarUrl,
        tag: inferTag(title, subtitle, tags),
        tags,
        mode: watchUrl ? "video" : "image",
        likeCount: work.likeCount || 0,
        updatedAt: work.listedAt || work.updatedAt || "",
    };
}

export function featuredTourProject(item: FeaturedCanvas): CanvasProject {
    const now = new Date().toISOString();
    const mediaType = item.watchUrl || item.previewUrl ? CanvasNodeType.Video : CanvasNodeType.Image;
    const mediaId = `featured-media-${item.slug}`;
    const noteId = `featured-note-${item.slug}`;
    const connections: CanvasConnection[] = [
        { id: `featured-link-cover-${item.slug}`, fromNodeId: `featured-cover-${item.slug}`, toNodeId: mediaId },
        { id: `featured-link-${item.slug}`, fromNodeId: mediaId, toNodeId: noteId },
    ];
    return {
        id: `featured-tour-${item.slug}`,
        title: item.title,
        createdAt: now,
        updatedAt: now,
        nodes: [
            {
                id: `featured-cover-${item.slug}`,
                type: CanvasNodeType.Image,
                title: "封面",
                position: { x: 80, y: 80 },
                width: 420,
                height: 236,
                metadata: { content: item.coverUrl, previewContent: item.coverUrl },
            },
            {
                id: mediaId,
                type: mediaType,
                title: item.title,
                position: { x: 560, y: 80 },
                width: 720,
                height: 405,
                metadata: {
                    content: item.watchUrl || item.previewUrl || item.coverUrl,
                    previewContent: item.coverUrl,
                },
            },
            {
                id: noteId,
                type: CanvasNodeType.Text,
                title: "作品说明",
                position: { x: 560, y: 520 },
                width: 520,
                height: 220,
                metadata: { content: item.subtitle || item.title },
            },
        ],
        connections,
        chatSessions: [],
        activeChatId: null,
        backgroundMode: "lines",
        showImageInfo: false,
        viewport: { x: 40, y: 20, k: 0.82 },
        directorScenes: [],
    };
}

export function formatPlazaTime(value: string) {
    const date = new Date(value);
    if (!Number.isFinite(date.getTime())) return "";
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}
