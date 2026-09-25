import { http } from "@/services/api/request";
import type { HomeNavItem } from "@/lib/home-navigation";
import type { ManchuangLanding } from "@/lib/manchuang-landing";
import type { SkinDefinition } from "@/lib/skin-themes";

export type PublicAppearance = {
    schemaVersion: number;
    brandName: string;
    brandSlug: string;
    authHeroTitle: string;
    authHeroDescription: string;
    logoUrl: string;
    darkLogoUrl: string;
    logoFrameEnabled: boolean;
    authVideoUrl: string;
    authVideoPosterUrl: string;
    authVideoAutoplay: boolean;
    skinId: string;
    activeSkin: SkinDefinition;
    enabledSkins?: SkinDefinition[];
    workspaceSkins?: SkinDefinition[];
    defaultMode?: "light" | "dark";
    seoTitle: string;
    seoDescription: string;
    seoKeywords: string;
    footerCopyright: string;
    icpFilingEnabled: boolean;
    icpFilingNumber: string;
    publicHomepage?: "welcome" | "manchuang";
    homeNavItems?: HomeNavItem[];
    homeCtaLabel?: string;
    homeCtaHref?: string;
    landing?: Partial<ManchuangLanding>;
    landingVideoUrl?: string;
    canvas?: {
        agentName: string;
        launcherLabel: string;
        panelTitle: string;
        welcomeTitle: string;
        welcomeDescription: string;
        inputPlaceholder: string;
        avatarType: "orb" | "live2d";
        live2dResourceId: string;
        live2dEntry: string;
        avatarHeight: number;
    };
    logoConfigured: boolean;
    darkLogoConfigured: boolean;
    authVideoConfigured: boolean;
    authVideoPosterConfigured: boolean;
    configured: boolean;
    revision: string;
    updatedAt?: string;
};

export type AdminAppearance = {
    schemaVersion: number;
    brandName: string;
    brandSlug: string;
    authHeroTitle: string;
    authHeroDescription: string;
    logoResourceId: string;
    darkLogoResourceId: string;
    logoFrameEnabled: boolean;
    authVideoResourceId: string;
    authVideoPosterResourceId: string;
    authVideoAutoplay: boolean;
    skinId: string;
    skinThemes: SkinDefinition[];
    enabledSkins?: string[];
    defaultMode?: "light" | "dark";
    seoTitle: string;
    seoDescription: string;
    seoKeywords: string;
    footerCopyright: string;
    icpFilingEnabled: boolean;
    icpFilingNumber: string;
    publicHomepage?: "welcome" | "manchuang";
    homeNavItems?: HomeNavItem[];
    homeCtaLabel?: string;
    homeCtaHref?: string;
    landing?: Partial<ManchuangLanding>;
    landingVideoResourceId?: string;
    canvas?: PublicAppearance["canvas"];
    public: PublicAppearance;
    configured: boolean;
    updatedBy?: string;
    createdAt?: string;
    updatedAt?: string;
};

export type AppearanceAssetSlot = "logo" | "logo-dark" | "video" | "poster" | "landing-video";

export type AppearanceResource = {
    id: string;
    kind: string;
    status: string;
    mimeType: string;
    size: number;
};

export async function getPublicAppearance(signal?: AbortSignal) {
    const result = await http.get<{ appearance: PublicAppearance }>("/public/appearance", { signal });
    return result.appearance;
}

export async function getAdminAppearance(signal?: AbortSignal) {
    const result = await http.get<{ setting: AdminAppearance }>("/admin/settings/appearance", { signal });
    return result.setting;
}

export async function updateAdminAppearance(
    input: Pick<
        AdminAppearance,
        | "brandName"
        | "brandSlug"
        | "authHeroTitle"
        | "authHeroDescription"
        | "logoResourceId"
        | "darkLogoResourceId"
        | "logoFrameEnabled"
        | "authVideoResourceId"
        | "authVideoPosterResourceId"
        | "authVideoAutoplay"
        | "skinId"
        | "skinThemes"
        | "enabledSkins"
        | "defaultMode"
        | "seoTitle"
        | "seoDescription"
        | "seoKeywords"
        | "footerCopyright"
        | "icpFilingEnabled"
        | "icpFilingNumber"
        | "publicHomepage"
        | "homeNavItems"
        | "homeCtaLabel"
        | "homeCtaHref"
        | "landing"
        | "landingVideoResourceId"
        | "canvas"
    >,
) {
    const result = await http.patch<{ setting: AdminAppearance }>("/admin/settings/appearance", input);
    return result.setting;
}

export async function uploadLive2DModel(file: File) {
    const body = new FormData();
    body.append("file", file);
    const result = await http.post<{ model: { resourceId: string; entry: string } }>("/admin/settings/appearance/live2d", body);
    return result.model;
}

export async function resetAdminAppearance() {
    const result = await http.delete<{ setting: AdminAppearance }>("/admin/settings/appearance");
    return result.setting;
}

export async function uploadAppearanceAsset(slot: AppearanceAssetSlot, file: File) {
    const body = new FormData();
    body.append("file", file);
    const result = await http.post<{ resource: AppearanceResource }>(`/admin/settings/appearance/assets/${slot}`, body);
    return result.resource;
}

export async function uploadAppearanceMedia(file: File) {
    const body = new FormData();
    body.append("file", file);
    return http.post<{
        resource: AppearanceResource;
        displayUrl: string;
        originalUrl: string;
        compression: string;
    }>("/admin/settings/appearance/media", body);
}
