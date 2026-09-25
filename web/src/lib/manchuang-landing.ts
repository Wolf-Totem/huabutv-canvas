import featuredCanvases from "@/lib/plaza-featured.json";

export type LandingTextCard = { title: string; text: string; imageUrl?: string };
export type LandingStep = { step: string; phase: string; title: string; tagline: string; description: string; imageUrl: string; tags: string[] };
export type LandingRailCard = { id: string; imageUrl: string; previewUrl?: string; label?: string };
export type LandingPricingTier = { id: string; name: string; tagline: string; price: string; unit: string; note: string; cta: string; featured?: boolean; badge?: string; features: string[] };
export type LandingHeroBanner = { id: string; title: string; imageUrl: string; href: string; openInNewTab?: boolean; workId?: string };
export type LandingHeroTile = { id: string; title: string; subtitle: string; badge?: string; href: string; icon?: string };
export type LandingHeroShowcase = {
    banners: LandingHeroBanner[];
    create: { title: string; subtitle: string; href: string };
    tiles: LandingHeroTile[];
};

export type ManchuangLanding = {
    heroKicker: string;
    heroLead: string;
    heroVideoUrl: string;
    heroPosterUrl: string;
    productTitleTop: string;
    productTitleAccent: string;
    productLead: string;
    productMoreLabel: string;
    productStageUrl: string;
    capabilities: LandingTextCard[];
    workflowKicker: string;
    workflowTitle: string;
    workflowLead: string;
    workflowSteps: LandingStep[];
    enterpriseTitle: string;
    enterpriseLead: string;
    enterpriseCards: LandingTextCard[];
    resourcesTitle: string;
    resourcesLead: string;
    resourceCards: LandingTextCard[];
    pricingTitle: string;
    pricingLead: string;
    pricingTiers: LandingPricingTier[];
    rail: LandingRailCard[];
    heroShowcase: LandingHeroShowcase;
};

const img = (name: string) => `/manchuang/${name}`;

const featuredRail = (): LandingRailCard[] =>
    (featuredCanvases as Array<{ id?: string; publicationKey?: string; projectUuid?: string; name?: string; coverUrl?: string; previewUrl?: string }>).map((item, index) => ({
        id: String(item.publicationKey || item.id || item.projectUuid || `featured-${index}`),
        imageUrl: String(item.coverUrl || img(`carousel-${(index % 7) + 1}.webp`)),
        previewUrl: String(item.previewUrl || ""),
        label: String(item.name || ""),
    }));

const defaultHeroShowcase = (): LandingHeroShowcase => {
    const banners = featuredRail().slice(0, 8).map((item) => ({
        id: item.id,
        title: item.label || "精选画布",
        imageUrl: item.imageUrl,
        href: `/plaza/${encodeURIComponent(item.id)}`,
        workId: item.id,
    }));
    return {
        banners: banners.length ? banners : [{ id: "banner-1", title: "精选画布", imageUrl: img("carousel-1.webp"), href: "/create" }],
        create: { title: "开始创作", subtitle: "打开画布，组织图片与视频创作", href: "/create" },
        tiles: [
            { id: "tile-model", title: "新模型", subtitle: "全新增模与视频能力", badge: "全新上线", href: "/create" },
            { id: "tile-agent", title: "Agent 助手", subtitle: "一句话开始，自动规划并执行创作", badge: "智能创作", href: "/agent" },
            { id: "tile-director", title: "导演台", subtitle: "虚拟现场、三维场面与镜头控制", href: "/create" },
            { id: "tile-review", title: "逐帧拉片", subtitle: "上传参考视频，逐帧拉片快建参考", badge: "独家", href: "/create" },
        ],
    };
};

export const DEFAULT_MANCHUANG_LANDING: ManchuangLanding = {
    heroKicker: "企业级 AI 在线画布协作平台",
    heroLead: "从灵感到成片，让创意、计划与团队在同一张画布上无限连接",
    heroVideoUrl: img("home-hero.mp4"),
    heroPosterUrl: img("hero-vortex.webp"),
    productTitleTop: "一张画布",
    productTitleAccent: "所有可能",
    productLead: "集创意表达、思维模型、项目管理于一体，打破工具边界，让团队协作更流畅。",
    productMoreLabel: "了解更多",
    productStageUrl: img("canvas-product.webp"),
    capabilities: [
        { title: "思维可视化", text: "把复杂灵感拆成清晰节点，创意路径一眼可见。" },
        { title: "实时协作", text: "多人同屏推进项目，反馈、分工和产出同步发生。" },
        { title: "资源整合", text: "文档、图片、视频、模型结果和历史资产集中管理。" },
        { title: "安全可控", text: "企业权限、数据边界和团队资产按角色管理。" },
    ],
    workflowKicker: "Workflow",
    workflowTitle: "一张画布，串起完整创作流",
    workflowLead: "从灵感迸发到落地执行，四个阶段在同一空间无缝衔接",
    workflowSteps: [
        { step: "01", phase: "灵感", title: "头脑风暴", tagline: "激发创意，汇聚灵感", description: "在无限画布上自由捕捉每一个灵光乍现，把零散的想法记录、连接、发散与重组，让每一个念头都有无限可能。", imageUrl: img("feature-brainstorm.webp"), tags: ["无限画布", "自由发散", "关联整合"] },
        { step: "02", phase: "梳理", title: "流程图 & 架构图", tagline: "梳理流程，清晰高效", description: "用专业的图形化工具理清思路、组织逻辑、规划路径，无论产品设计还是项目管理，都能让复杂变得清晰可见。", imageUrl: img("feature-flowchart.webp"), tags: ["快速构建", "结构清晰", "高效协作"] },
        { step: "03", phase: "沉淀", title: "个人知识库", tagline: "沉淀内容，构建第二大脑", description: "收集灵感、剪藏资料、记录想法，用标签与链接把碎片化信息串联成网，在专属空间中持续沉淀与生长。", imageUrl: img("feature-knowledge.webp"), tags: ["一键剪藏", "智能标签", "知识图谱"] },
        { step: "04", phase: "落地", title: "个人计划", tagline: "规划目标，管理日程与任务", description: "设定长期目标与里程碑，用日历、时间线与待办清单把大目标拆解成可执行的每一步，在一张画布中掌控成长。", imageUrl: img("feature-plan.webp"), tags: ["目标清晰", "日程可视", "任务可执行"] },
    ],
    enterpriseTitle: "企业服务",
    enterpriseLead: "为团队提供权限、模型、资产与专属配置，把创作能力接到真实业务里。",
    enterpriseCards: [
        { title: "团队权限", text: "角色、项目、资产权限分层管理。" },
        { title: "模型配置", text: "支持多模型能力接入和统一入口管理。" },
        { title: "资产中心", text: "团队素材、生成结果、历史文件集中沉淀。" },
        { title: "数据安全", text: "面向企业使用的访问边界和审计意识。" },
        { title: "企业服务", text: "团队开通、专属配置、部署咨询和培训支持。" },
    ],
    resourcesTitle: "资源中心",
    resourcesLead: "从入门到模板，把可复用的创作方法沉淀下来。",
    resourceCards: [
        { title: "新手指南", text: "快速了解画布、节点和协作流程。" },
        { title: "创作模板", text: "从营销、分镜、项目计划模板开始创作。" },
        { title: "企业方案", text: "为团队权限、模型和资产管理配置服务。" },
        { title: "常见问题", text: "了解账号、资产、团队和使用边界。" },
    ],
    pricingTitle: "选择适合你的创作套餐",
    pricingLead: "VIP 与 SVIP 均可按月、按季或按年开通；微信扫码支付后，积分与存储空间自动到账。",
    pricingTiers: [
        { id: "vip", name: "VIP", tagline: "适合刚开始的 AI 创作者", price: "¥30", unit: "/ 月卡", note: "2,888 积分 / 整期", cta: "微信购买 VIP", features: ["30 GB 云端存储空间", "解锁全部模型与生成能力", "生成、上传、导出全功能开放", "额度用完可继续叠加积分"] },
        { id: "svip", name: "SVIP", tagline: "为重度创作与商用项目而生", price: "¥68", unit: "/ 月卡", note: "6,888 积分 / 整期", cta: "微信购买 SVIP", featured: true, badge: "最受欢迎", features: ["80 GB 云端存储空间", "解锁全部模型与生成能力", "生成、上传、导出全功能开放", "额度用完可继续叠加积分"] },
    ],
    rail: featuredRail(),
    heroShowcase: defaultHeroShowcase(),
};

export function mergeManchuangLanding(value?: Partial<ManchuangLanding> | null): ManchuangLanding {
    const base = DEFAULT_MANCHUANG_LANDING;
    if (!value) return base;
    return {
        ...base,
        ...value,
        capabilities: value.capabilities?.length ? value.capabilities : base.capabilities,
        workflowSteps: value.workflowSteps?.length ? value.workflowSteps : base.workflowSteps,
        enterpriseCards: value.enterpriseCards?.length ? value.enterpriseCards : base.enterpriseCards,
        resourceCards: value.resourceCards?.length ? value.resourceCards : base.resourceCards,
        pricingTiers: value.pricingTiers?.length ? value.pricingTiers : base.pricingTiers,
        rail: value.rail?.length ? value.rail : base.rail,
        heroShowcase: {
            banners: value.heroShowcase?.banners?.length ? value.heroShowcase.banners : base.heroShowcase.banners,
            create: {
                title: value.heroShowcase?.create?.title?.trim() || base.heroShowcase.create.title,
                subtitle: value.heroShowcase?.create?.subtitle?.trim() || base.heroShowcase.create.subtitle,
                href: value.heroShowcase?.create?.href?.trim() || base.heroShowcase.create.href,
            },
            tiles: value.heroShowcase?.tiles?.length ? value.heroShowcase.tiles : base.heroShowcase.tiles,
        },
        heroVideoUrl: value.heroVideoUrl?.trim() || base.heroVideoUrl,
        heroPosterUrl: value.heroPosterUrl?.trim() || base.heroPosterUrl,
        productStageUrl: value.productStageUrl?.trim() || base.productStageUrl,
    };
}

export const LANDING_NAV = [
    { id: "product", label: "产品" },
    { id: "solutions", label: "解决方案" },
    { id: "enterprise", label: "企业服务" },
    { id: "resources", label: "资源中心" },
    { id: "pricing", label: "定价" },
];
