import { http } from "@/services/api/request";

export type PublicSiteSkin = {
    slug?: string;
    displayName?: string;
    title?: string;
    logoUrl?: string;
    tagline?: string;
    theme?: Record<string, string>;
    inviteLocked?: boolean;
    registrationEnabled?: boolean;
    customHomeEnabled?: boolean;
    home?: StreamerHomePayload | null;
    streamerActive?: boolean;
    parentDomain?: string;
    agentHost?: string;
    canvasHost?: string;
    landingHost?: string;
    heroVideoUrl?: string;
    heroPosterUrl?: string;
};

export type StreamerHomePayload = {
    templateId?: string;
    hero?: { title?: string; subtitle?: string; imageUrl?: string };
    features?: Array<{ title?: string; text?: string }>;
    cta?: { label?: string; href?: string };
    footer?: { text?: string };
};

export type StreamerAdmin = {
    id: string;
    userId: string;
    slug: string;
    inviteCode: string;
    status: "active" | "disabled";
    displayName: string;
    customHomeEnabled: boolean;
    serialNo: number;
    note: string;
    rebateRateBps: number;
    textRebateBps: number;
    imageRebateBps: number;
    videoRebateBps: number;
    customChannelsEnabled: boolean;
    rebateTotalCredits: number;
    rebatePendingCredits: number;
    rebateWithdrawnCredits: number;
    rebateWithdrawableCredits: number;
    alipayAccount?: string;
    alipayRealName?: string;
    createdAt: string;
};

export type StreamerConsoleMe = {
    slug: string;
    inviteCode: string;
    host: string;
    landingUrl: string;
    agentUrl?: string;
    canvasUrl?: string;
    status: string;
    serialNo?: number;
    displayName?: string;
    rebateRateBps?: number;
    textRebateBps?: number;
    imageRebateBps?: number;
    videoRebateBps?: number;
    customChannelsEnabled?: boolean;
    alipayAccount?: string;
    alipayRealName?: string;
};

export type StreamerConsoleSummary = {
    referredUserCount: number;
    consumedCredits: number;
    remainingCreditsSum: number;
    rebateRateBps: number;
    textRebateBps?: number;
    imageRebateBps?: number;
    videoRebateBps?: number;
    expectedRebateCredits: number;
    rebateCredits: number;
    rebateTotalCredits?: number;
    rebatePendingCredits?: number;
    rebateWithdrawnCredits?: number;
    rebateWithdrawableCredits?: number;
    rebateRequestedCredits?: number;
    period: string;
};

export type StreamerPayout = {
    id: string;
    streamerId: string;
    displayName?: string;
    amountCredits: number;
    alipayAccount: string;
    alipayRealName: string;
    status: "pending" | "approved" | "rejected";
    rejectReason?: string;
    reviewedAt?: string;
    createdAt: string;
};

export type AgentShareModel = {
    id: string;
    name: string;
    code: string;
    capability: string;
    enabled: boolean;
    agentShareEnabled: boolean;
    agentShareBps: number;
};

export type StreamerConsoleUser = {
    userIdMasked: string;
    displayNameMasked: string;
    registeredAt: string;
    consumedCredits: number;
    rebateCredits: number;
    remainingCredits: number;
    lastActiveAt?: string;
};

export type StreamerConsoleRebate = {
    id: string;
    sourceUserMasked: string;
    capability: string;
    consumedCredits: number;
    modelShareBps: number;
    agentRebateBps: number;
    rebateCredits: number;
    createdAt: string;
};

export function getPublicSiteSkin() {
    return http.get<PublicSiteSkin>("/public/site-skin");
}

export function listAdminStreamers() {
    return http.get<{ items: StreamerAdmin[] }>("/admin/streamers");
}

export function createAdminStreamer(input: Partial<StreamerAdmin> & { userId: string; slug: string }) {
    return http.post<{ streamer: StreamerAdmin }>("/admin/streamers", input);
}

export function updateAdminStreamer(id: string, input: Partial<{
    displayName: string;
    slug: string;
    inviteCode: string;
    serialNo: number;
    note: string;
    rebateRateBps: number;
    textRebateBps: number;
    imageRebateBps: number;
    videoRebateBps: number;
    customChannelsEnabled: boolean;
}>) {
    return http.patch<{ streamer: StreamerAdmin }>(`/admin/streamers/${id}`, input);
}

export function listAdminPayouts(page = 1, pageSize = 20) {
    return http.get<{ items: StreamerPayout[]; total: number }>("/admin/payouts", { params: { page, pageSize } });
}

export function approveAdminPayout(id: string) {
    return http.post<{ payout: StreamerPayout }>(`/admin/payouts/${id}/approve`);
}

export function rejectAdminPayout(id: string, reason: string) {
    return http.post<{ payout: StreamerPayout }>(`/admin/payouts/${id}/reject`, { reason });
}

export function listAdminAgentShareModels() {
    return http.get<{ items: AgentShareModel[] }>("/admin/agent-share-models");
}

export function updateAdminAgentShareModel(id: string, input: { enabled?: boolean; shareBps?: number }) {
    return http.patch<{ model: AgentShareModel }>(`/admin/agent-share-models/${id}`, input);
}

export function listAgentPayouts(page = 1, pageSize = 20) {
    return http.get<{ items: StreamerPayout[]; total: number }>("/agent/payouts", { params: { page, pageSize } });
}

export function createAgentPayout(input: { amountCredits: number; alipayAccount: string; alipayRealName: string }) {
    return http.post<{ payout: StreamerPayout }>("/agent/payouts", input);
}

export function disableAdminStreamer(id: string) {
    return http.post<{ streamer: StreamerAdmin }>(`/admin/streamers/${id}/disable`);
}

export function enableAdminStreamer(id: string) {
    return http.post<{ streamer: StreamerAdmin }>(`/admin/streamers/${id}/enable`);
}

export function rotateAdminStreamerCode(id: string) {
    return http.post<{ streamer: StreamerAdmin }>(`/admin/streamers/${id}/rotate-code`);
}

export type StreamerSkin = {
    logoUrl: string;
    title: string;
    tagline: string;
    themeJson: string;
    homeTemplateId: string;
    homePayloadJson: string;
    heroVideoUrl?: string;
    heroPosterUrl?: string;
    heroVideoResourceId?: string;
    heroPosterResourceId?: string;
    updatedAt?: string;
};

export function getAdminStreamerSkin(id: string) {
    return http.get<{ skin: StreamerSkin; streamer: StreamerAdmin }>(`/admin/streamers/${id}/skin`);
}

export function updateAdminStreamerSkin(id: string, input: Record<string, unknown>) {
    return http.put<{ skin: StreamerSkin }>(`/admin/streamers/${id}/skin`, input);
}

export function uploadAdminStreamerHero(id: string, kind: "video" | "poster", file: File) {
    const body = new FormData();
    body.append("file", file);
    return http.post<{ skin: StreamerSkin }>(`/admin/streamers/${id}/hero-${kind}`, body);
}

export function getStreamerMe() {
    return http.get<StreamerConsoleMe>("/agent/me");
}

export function getStreamerSummary(params?: { from?: string; to?: string }) {
    return http.get<StreamerConsoleSummary>("/agent/summary", { params });
}

export function getStreamerUsers(page = 1, size = 20) {
    return http.get<{ items: StreamerConsoleUser[]; total: number }>("/agent/users", { params: { page, pageSize: size } });
}

export function getStreamerRebates(page = 1, size = 20, capability = "") {
    return http.get<{ items: StreamerConsoleRebate[]; total: number }>("/agent/rebates", { params: { page, pageSize: size, capability: capability || undefined } });
}
