import { http } from "@/services/api/request";
import type { CanvasProject } from "@/stores/canvas/use-canvas-store";

export type PlazaSettings = {
    enabled: boolean;
    applyEnabled: boolean;
    copyEnabled: boolean;
    publicWatch: boolean;
    applyDailyLimit: number;
};

export type PlazaCategory = {
    id: string;
    slug: string;
    name: string;
    kind: string;
    enabled?: boolean;
};

export type PlazaAuthor = {
    id: string;
    displayName: string;
    avatarUrl?: string;
};

export type PlazaWork = {
    id: string;
    slug: string;
    title: string;
    subtitle: string;
    status: string;
    allowWatch: boolean;
    allowProcessView: boolean;
    allowCopy: boolean;
    badges: string[];
    score: number;
    viewCount: number;
    watchCount: number;
    tourCount: number;
    likeCount: number;
    copyCount: number;
    liked?: boolean;
    coverUrl?: string;
    watchUrl?: string;
    author: PlazaAuthor;
    category: { id: string; slug: string; name: string; kind: string };
    listedAt?: string;
    updatedAt: string;
};

export type PlazaApplication = {
    id: string;
    userId: string;
    projectId: string;
    workId?: string;
    title: string;
    subtitle: string;
    categoryId: string;
    campaignIds: string[];
    coverNodeId?: string;
    watchNodeId?: string;
    allowWatch: boolean;
    allowProcessView: boolean;
    allowCopy: boolean;
    status: string;
    reviewNote?: string;
    nodeCount: number;
    mediaCount: number;
    submittedAt: string;
    reviewedAt?: string;
    author: PlazaAuthor;
};

export type PlazaApplyRequest = {
    title: string;
    subtitle: string;
    categoryId: string;
    campaignIds?: string[];
    coverNodeId?: string;
    watchNodeId?: string;
    allowWatch: boolean;
    allowProcessView: boolean;
    allowCopy: boolean;
    originalityAck: boolean;
};

export function getPlazaSettings() {
    return http.get<{ settings: PlazaSettings }>("/plaza/settings");
}

export function getPlazaCategories() {
    return http.get<{ categories: PlazaCategory[] }>("/plaza/categories");
}

export function listPlazaWorks(params: { category?: string; sort?: string; page?: number; pageSize?: number }) {
    return http.get<{ works: PlazaWork[]; hasMore: boolean }>("/plaza/works", { params });
}

export function getPlazaWork(slug: string) {
    return http.get<{ work: PlazaWork }>(`/plaza/works/${encodeURIComponent(slug)}`);
}

export function getPlazaSnapshot(id: string) {
    return http.get<{ project: CanvasProject }>(`/public/plaza-works/${encodeURIComponent(id)}/snapshot`);
}

export function recordPlazaEvent(id: string, kind: "view" | "watch" | "tour") {
    return http.post<{ ok: boolean }>(`/plaza/works/${encodeURIComponent(id)}/events`, { kind });
}

export function likePlazaWork(id: string) {
    return http.post<{ work: PlazaWork }>(`/plaza/works/${encodeURIComponent(id)}/like`);
}

export function unlikePlazaWork(id: string) {
    return http.delete<{ work: PlazaWork }>(`/plaza/works/${encodeURIComponent(id)}/like`);
}

export function copyPlazaWork(id: string) {
    return http.post<{ projectId: string; title: string }>(`/plaza/works/${encodeURIComponent(id)}/copy`);
}

export function applyPlazaWork(projectId: string, body: PlazaApplyRequest) {
    return http.post<{ application: PlazaApplication }>(`/canvas-projects/${encodeURIComponent(projectId)}/plaza-applications`, body);
}

export function listMyPlazaApplications() {
    return http.get<{ applications: PlazaApplication[] }>("/me/plaza-applications");
}

export function getAdminPlazaSettings() {
    return http.get<{ settings: PlazaSettings }>("/admin/plaza/settings");
}

export function updateAdminPlazaSettings(settings: PlazaSettings) {
    return http.patch<{ settings: PlazaSettings }>("/admin/plaza/settings", settings);
}

export function listAdminPlazaApplications(params: { status?: string; page?: number; pageSize?: number }) {
    return http.get<{ applications: PlazaApplication[]; total: number }>("/admin/plaza/applications", { params });
}

export function getAdminPlazaApplicationSnapshot(id: string) {
    return http.get<{ project: CanvasProject }>(`/admin/plaza/applications/${encodeURIComponent(id)}/snapshot`);
}

export function approvePlazaApplication(id: string) {
    return http.post<{ work: PlazaWork }>(`/admin/plaza/applications/${encodeURIComponent(id)}/approve`);
}

export function rejectPlazaApplication(id: string, note: string) {
    return http.post<{ application: PlazaApplication }>(`/admin/plaza/applications/${encodeURIComponent(id)}/reject`, { note });
}

export function listAdminPlazaWorks(params: { status?: string; page?: number; pageSize?: number }) {
    return http.get<{ works: PlazaWork[]; total: number }>("/admin/plaza/works", { params });
}

export function takeDownPlazaWork(id: string, note: string) {
    return http.post<{ work: PlazaWork }>(`/admin/plaza/works/${encodeURIComponent(id)}/take-down`, { note });
}

export function deletePlazaWork(id: string) {
    return http.delete<{ ok: true }>(`/admin/plaza/works/${encodeURIComponent(id)}`);
}
