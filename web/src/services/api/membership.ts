import { http } from "@/services/api/request";
import type { MembershipProduct, MembershipStatus } from "@/lib/membership";

export type { MembershipProduct, MembershipStatus };

export type CommerceMethods = {
    onlinePaymentEnabled: boolean;
    redeemEnabled: boolean;
};

export type SupportContactSetting = {
    ticketUrl?: string;
    qq?: string;
};

export function getMembership() {
    return http.get<MembershipStatus>("/membership");
}

export function listMembershipProducts() {
    return http.get<{ products: MembershipProduct[] }>("/membership/products");
}

export function listAdminMembershipProducts() {
    return http.get<{ products: MembershipProduct[] }>("/admin/membership/products");
}

export function updateAdminMembershipProduct(id: string, input: { name: string; description?: string; amountFen: number; enabled: boolean; sortOrder: number }) {
    return http.patch<{ product: MembershipProduct }>(`/admin/membership/products/${encodeURIComponent(id)}`, input);
}

export function getAdminCommerceMethods() {
    return http.get<CommerceMethods>("/admin/settings/commerce-methods");
}

export function updateAdminCommerceMethods(input: CommerceMethods) {
    return http.patch<CommerceMethods>("/admin/settings/commerce-methods", input);
}

export function getAdminSupportContact() {
    return http.get<SupportContactSetting>("/admin/settings/support-contact");
}

export function updateAdminSupportContact(input: SupportContactSetting) {
    return http.patch<SupportContactSetting>("/admin/settings/support-contact", input);
}

export function updateAdminUserStorageQuota(userId: string, storageOverrideBytes: number | null) {
    return http.patch<{ ok: boolean }>(`/admin/users/${encodeURIComponent(userId)}/storage-quota`, { storageOverrideBytes });
}

export function grantAdminMembership(userId: string, input: { sku: string; idempotencyKey: string; note?: string }) {
    return http.post<{ membership: MembershipStatus }>(`/admin/users/${encodeURIComponent(userId)}/membership/grant`, input);
}
