export const MEMBERSHIP_RENEWAL_BLOCK_SECONDS = 30 * 24 * 3600;

export type MembershipStatus = {
    permanentActive: boolean;
    advancedPlanSku: string;
    advancedExpiresAt?: string;
    advancedRemainingSeconds: number;
    canPurchaseAdvanced: boolean;
    canPurchasePermanent: boolean;
    minPurchasableAdvancedSku?: string;
    hasOpenMembershipOrder: boolean;
    openMembershipOrderId?: string;
    personalStorageAllowed: boolean;
    effectiveStoredFileBytes: number;
    quotaSource: string;
    storageOverrideBytes?: number | null;
    storageBonusBytes?: number;
    storageExpansionMessage?: string;
    supportTicketUrl?: string;
    supportQq?: string;
    redeemEnabled?: boolean;
    onlinePaymentEnabled?: boolean;
};

export const defaultMembership: MembershipStatus = {
    permanentActive: false,
    advancedPlanSku: "",
    advancedRemainingSeconds: 0,
    canPurchaseAdvanced: true,
    canPurchasePermanent: true,
    hasOpenMembershipOrder: false,
    personalStorageAllowed: false,
    effectiveStoredFileBytes: 0,
    quotaSource: "global_default",
    redeemEnabled: true,
    onlinePaymentEnabled: true,
};

export type MembershipProduct = {
    id: string;
    sku: string;
    name: string;
    description?: string;
    amountFen: number;
    creditsMicrocredits: number;
    storageQuotaBytes: number;
    durationDays: number;
    enabled: boolean;
    sortOrder: number;
};

export function membershipPurchaseBlocked(status: MembershipStatus | null | undefined, sku: string) {
    if (!status) return { disabled: true, reason: "", reasonKey: "" };
    if (status.hasOpenMembershipOrder) return { disabled: true, reason: "你有一笔未完成的订阅订单", reasonKey: "wallet.block.openOrder" };
    if (sku === "permanent") {
        if (status.permanentActive) return { disabled: true, reason: "已拥有永久订阅", reasonKey: "wallet.block.permanentOwned" };
        return { disabled: false, reason: "", reasonKey: "" };
    }
    if (status.advancedRemainingSeconds > MEMBERSHIP_RENEWAL_BLOCK_SECONDS) {
        return { disabled: true, reason: "当前订阅剩余超过 30 天，暂不可新购", reasonKey: "wallet.block.renewalWindow" };
    }
    return { disabled: false, reason: "", reasonKey: "" };
}

const KIB = 1024;
const MIB = 1024 ** 2;
const GIB = 1024 ** 3;
const TIB = 1024 ** 4;

export function formatMembershipStorage(bytes: number) {
    if (!Number.isFinite(bytes) || bytes <= 0) return "沿用平台默认";
    if (bytes >= TIB && bytes % TIB === 0) return `${bytes / TIB} TiB`;
    if (bytes >= TIB) return `${(bytes / TIB).toFixed(2)} TiB`;
    if (bytes >= GIB && bytes % GIB === 0) return `${bytes / GIB} GiB`;
    if (bytes >= GIB) return `${(bytes / GIB).toFixed(2)} GiB`;
    if (bytes >= MIB) return `${Math.round(bytes / MIB)} MB`;
    return `${Math.max(1, Math.round(bytes / KIB))} KB`;
}

export function membershipStatusKey(status: MembershipStatus | null | undefined) {
    if (status?.permanentActive) return "permanent";
    if (status?.advancedPlanSku) return "premium";
    return "platform";
}

export function membershipStatusLabel(status: MembershipStatus | null | undefined) {
    if (status?.permanentActive) return "永久会员";
    if (status?.advancedPlanSku) return "高级会员";
    return "平台会员";
}
