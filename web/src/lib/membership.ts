export const MEMBERSHIP_RENEWAL_BLOCK_SECONDS = 30 * 24 * 3600;

export type MembershipTier = "vip" | "svip" | "";

export type MembershipStatus = {
    permanentActive: boolean;
    advancedPlanSku: string;
    advancedExpiresAt?: string;
    advancedRemainingSeconds: number;
    tier?: MembershipTier | string;
    canPurchaseAdvanced: boolean;
    canPurchasePermanent: boolean;
    canPurchaseVip?: boolean;
    canPurchaseSvip?: boolean;
    minPurchasableAdvancedSku?: string;
    hasOpenMembershipOrder: boolean;
    openMembershipOrderId?: string;
    personalStorageAllowed: boolean;
    personalBucketEnabled?: boolean;
    storageDisplay?: "personal" | "platform" | string;
    effectiveStoredFileBytes: number;
    quotaSource: string;
    storageOverrideBytes?: number | null;
    storageBonusBytes?: number;
    storageExpansionMessage?: string;
    supportTicketUrl?: string;
    supportQq?: string;
    redeemEnabled?: boolean;
    onlinePaymentEnabled?: boolean;
    featureShowcase?: MembershipFeatureLine[];
    freeShowcase?: MembershipShowcase;
    defaultStoredFileBytes?: number;
    storageBonusExpiresAt?: string;
};

export type MembershipFeatureLine = {
    text: string;
    included: boolean;
};

export type MembershipShowcase = {
    title?: string;
    description?: string;
    entryLabel?: string;
    audience?: string;
    addOnLabel?: string;
    featureLines?: MembershipFeatureLine[];
};

export const defaultMembership: MembershipStatus = {
    permanentActive: false,
    advancedPlanSku: "",
    advancedRemainingSeconds: 0,
    canPurchaseAdvanced: true,
    canPurchasePermanent: true,
    canPurchaseVip: true,
    canPurchaseSvip: true,
    hasOpenMembershipOrder: false,
    personalStorageAllowed: false,
    personalBucketEnabled: false,
    storageDisplay: "platform",
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
    originalAmountFen?: number;
    creditsMicrocredits: number;
    storageQuotaBytes: number;
    durationDays: number;
    tier?: string;
    badge?: string;
    highlighted?: boolean;
    enabled: boolean;
    sortOrder: number;
    entryLabel?: string;
    audience?: string;
    addOnLabel?: string;
    featureLines?: MembershipFeatureLine[];
};

export type MembershipPeriod = "month" | "quarter" | "year";

export const MEMBERSHIP_PERIODS: { id: MembershipPeriod; days: number }[] = [
    { id: "month", days: 30 },
    { id: "quarter", days: 90 },
    { id: "year", days: 365 },
];

export const CATALOG_MEMBERSHIP_SKUS = [
    { value: "vip_month", label: "VIP 月卡", tier: "vip" as const },
    { value: "vip_quarter", label: "VIP 季卡", tier: "vip" as const },
    { value: "vip_year", label: "VIP 年卡", tier: "vip" as const },
    { value: "svip_month", label: "SVIP 月卡", tier: "svip" as const },
    { value: "svip_quarter", label: "SVIP 季卡", tier: "svip" as const },
    { value: "svip_year", label: "SVIP 年卡", tier: "svip" as const },
];

export const ADMIN_MEMBERSHIP_SKUS = [
    ...CATALOG_MEMBERSHIP_SKUS,
    { value: "permanent", label: "永久订阅", tier: "" as const },
    { value: "advanced_month", label: "旧月卡", tier: "vip" as const },
    { value: "advanced_quarter", label: "旧季卡", tier: "vip" as const },
    { value: "advanced_year", label: "旧年卡", tier: "vip" as const },
];

export function membershipSKUTier(sku: string | null | undefined): MembershipTier {
    const value = (sku || "").trim().toLowerCase();
    if (value === "svip" || value.startsWith("svip_")) return "svip";
    if (value === "vip" || value.startsWith("vip_") || value.startsWith("advanced_")) return "vip";
    return "";
}

export function membershipSKULabel(sku: string | null | undefined) {
    const value = (sku || "").trim();
    return ADMIN_MEMBERSHIP_SKUS.find((item) => item.value === value)?.label || value;
}

export function isCatalogMembershipSKU(sku: string | null | undefined) {
    return CATALOG_MEMBERSHIP_SKUS.some((item) => item.value === (sku || "").trim());
}

export function membershipPurchaseBlocked(status: MembershipStatus | null | undefined, sku: string) {
    if (!status) return { disabled: true, reason: "", reasonKey: "" };
    if (status.hasOpenMembershipOrder) return { disabled: true, reason: "你有一笔未完成的订阅订单", reasonKey: "wallet.block.openOrder" };
    if (sku === "permanent" && status.permanentActive) return { disabled: true, reason: "已拥有永久订阅", reasonKey: "wallet.block.permanentOwned" };
    if (status.advancedRemainingSeconds > MEMBERSHIP_RENEWAL_BLOCK_SECONDS) {
        return { disabled: true, reason: "当前订阅剩余超过 30 天，暂不可购买其他会员", reasonKey: "wallet.block.renewalWindow" };
    }
    const currentTier = membershipSKUTier(status.tier || status.advancedPlanSku);
    const targetTier = membershipSKUTier(sku);
    if (currentTier === "svip" && status.advancedRemainingSeconds > 0 && targetTier === "vip") {
        return { disabled: true, reason: "SVIP 有效期内不能改买 VIP，到期后再选", reasonKey: "wallet.block.svipToVip" };
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
    const activeSku = (status?.advancedRemainingSeconds || 0) > 0 ? (status?.tier || status?.advancedPlanSku) : "";
    const tier = membershipSKUTier(activeSku);
    if (tier === "svip") return "svip";
    if (tier === "vip") return "vip";
    if (status?.permanentActive) return "permanent";
    return "none";
}

export function membershipStatusLabel(status: MembershipStatus | null | undefined) {
    const key = membershipStatusKey(status);
    if (key === "svip") return "SVIP";
    if (key === "vip") return "VIP";
    if (key === "permanent") return "永久会员";
    return "未开通";
}

export function groupMembershipProductsByTier(products: MembershipProduct[]) {
    const catalog = products.filter((item) => isCatalogMembershipSKU(item.sku));
    const vip = catalog.filter((item) => membershipSKUTier(item.tier || item.sku) === "vip");
    const svip = catalog.filter((item) => membershipSKUTier(item.tier || item.sku) === "svip");
    return { vip, svip };
}

export function membershipSKUDurationDays(sku: string | null | undefined) {
    const value = (sku || "").trim();
    if (value.endsWith("_year")) return 365;
    if (value.endsWith("_quarter")) return 90;
    if (value.endsWith("_month")) return 30;
    return 0;
}

export function periodFromDurationDays(days: number): MembershipPeriod {
    if (days === 90) return "quarter";
    if (days === 30) return "month";
    return "year";
}

export function productForPeriod(products: MembershipProduct[], period: MembershipPeriod) {
    const days = MEMBERSHIP_PERIODS.find((item) => item.id === period)?.days;
    return products.find((item) => item.durationDays === days) || products[0];
}

export function periodSavingsPercent(monthFen: number, periodFen: number, days: number) {
    if (!(monthFen > 0) || !(periodFen > 0) || days <= 30) return 0;
    const months = days === 90 ? 3 : days === 365 ? 12 : 1;
    const full = monthFen * months;
    if (periodFen >= full) return 0;
    return Math.round((1 - periodFen / full) * 100);
}

export function dailyPriceYuan(amountFen: number, days: number) {
    if (!(amountFen > 0) || !(days > 0)) return "";
    const value = amountFen / 100 / days;
    return value >= 10 ? value.toFixed(0) : value.toFixed(value >= 1 ? 1 : 2);
}

export function formatStorageDuration(days: number) {
    if (days === 365) return "1 年";
    if (days === 30) return "30 天";
    if (days === 90) return "90 天";
    if (days > 0) return `${days} 天`;
    return "1 年";
}
