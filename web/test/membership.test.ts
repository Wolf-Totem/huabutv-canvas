import { describe, expect, test } from "bun:test";

import { defaultMembership, formatMembershipStorage, formatStorageDuration, groupMembershipProductsByTier, membershipPurchaseBlocked, membershipStatusLabel, periodSavingsPercent, productForPeriod } from "../src/lib/membership";

describe("membership purchase rules", () => {
    test("blocks all SKUs when an open membership order exists", () => {
        const status = { ...defaultMembership, hasOpenMembershipOrder: true, openMembershipOrderId: "order-1" };
        expect(membershipPurchaseBlocked(status, "permanent").disabled).toBe(true);
        expect(membershipPurchaseBlocked(status, "vip_month").disabled).toBe(true);
        expect(membershipPurchaseBlocked(status, "advanced_year").reason).toContain("未完成");
    });

    test("allows vip to svip inside the 30 day window", () => {
        const status = { ...defaultMembership, advancedPlanSku: "vip_year", advancedRemainingSeconds: 20 * 24 * 3600 };
        expect(membershipPurchaseBlocked(status, "vip_month").disabled).toBe(false);
        expect(membershipPurchaseBlocked(status, "svip_year").disabled).toBe(false);
    });

    test("blocks every membership SKU when remaining is over 30 days", () => {
        const status = { ...defaultMembership, advancedPlanSku: "vip_year", advancedRemainingSeconds: 31 * 24 * 3600 };
        expect(membershipPurchaseBlocked(status, "vip_month").disabled).toBe(true);
        expect(membershipPurchaseBlocked(status, "svip_month").disabled).toBe(true);
    });

    test("blocks vip while svip is still active even in the last 30 days", () => {
        const status = { ...defaultMembership, advancedPlanSku: "svip_month", advancedRemainingSeconds: 10 * 24 * 3600 };
        expect(membershipPurchaseBlocked(status, "vip_month").disabled).toBe(true);
        expect(membershipPurchaseBlocked(status, "vip_month").reasonKey).toBe("wallet.block.svipToVip");
        expect(membershipPurchaseBlocked(status, "svip_year").disabled).toBe(false);
        const byTier = { ...defaultMembership, tier: "svip", advancedRemainingSeconds: 10 * 24 * 3600 };
        expect(membershipPurchaseBlocked(byTier, "vip_month").disabled).toBe(true);
    });

    test("formats GiB/TiB without 32-bit shift overflow", () => {
        expect(formatMembershipStorage(1024 ** 3)).toBe("1 GiB");
        expect(formatMembershipStorage(5 * 1024 ** 3)).toBe("5 GiB");
        expect(formatMembershipStorage(3 * 1024 ** 4)).toBe("3 TiB");
    });

    test("groups only catalog vip/svip cards", () => {
        const grouped = groupMembershipProductsByTier([
            { id: "legacy", sku: "advanced_month", name: "旧月卡", amountFen: 1, creditsMicrocredits: 1, storageQuotaBytes: 1, durationDays: 30, enabled: true, sortOrder: 1 },
            { id: "vip", sku: "vip_month", name: "VIP 月卡", amountFen: 3000, creditsMicrocredits: 1, storageQuotaBytes: 1, durationDays: 30, enabled: true, sortOrder: 2, tier: "vip" },
            { id: "svip", sku: "svip_year", name: "SVIP 年卡", amountFen: 68000, creditsMicrocredits: 1, storageQuotaBytes: 1, durationDays: 365, enabled: true, sortOrder: 3, tier: "svip" },
        ]);
        expect(grouped.vip.map((item) => item.sku)).toEqual(["vip_month"]);
        expect(grouped.svip.map((item) => item.sku)).toEqual(["svip_year"]);
    });

    test("computes period savings against the same-tier monthly price", () => {
        expect(periodSavingsPercent(3000, 8000, 90)).toBe(11);
        expect(periodSavingsPercent(3000, 28800, 365)).toBe(20);
        expect(periodSavingsPercent(3000, 9000, 90)).toBe(0);
        expect(periodSavingsPercent(3000, 3000, 30)).toBe(0);
    });

    test("picks the catalog card for the selected period", () => {
        const products = [
            { id: "m", sku: "vip_month", name: "VIP 月卡", amountFen: 3000, creditsMicrocredits: 1, storageQuotaBytes: 1, durationDays: 30, enabled: true, sortOrder: 1 },
            { id: "y", sku: "vip_year", name: "VIP 年卡", amountFen: 28800, creditsMicrocredits: 1, storageQuotaBytes: 1, durationDays: 365, enabled: true, sortOrder: 2 },
        ];
        expect(productForPeriod(products, "year")?.sku).toBe("vip_year");
        expect(productForPeriod(products, "month")?.sku).toBe("vip_month");
        expect(formatStorageDuration(365)).toBe("1 年");
        expect(formatStorageDuration(90)).toBe("90 天");
    });

    test("labels membership status for the sidebar account row", () => {
        expect(membershipStatusLabel(null)).toBe("未开通");
        expect(membershipStatusLabel(defaultMembership)).toBe("未开通");
        expect(membershipStatusLabel({ ...defaultMembership, advancedPlanSku: "advanced_month" })).toBe("未开通");
        expect(membershipStatusLabel({ ...defaultMembership, advancedPlanSku: "advanced_month", advancedRemainingSeconds: 100, tier: "vip" })).toBe("VIP");
        expect(membershipStatusLabel({ ...defaultMembership, advancedPlanSku: "svip_year", advancedRemainingSeconds: 100, tier: "svip" })).toBe("SVIP");
        expect(membershipStatusLabel({ ...defaultMembership, permanentActive: true, advancedPlanSku: "advanced_month" })).toBe("永久会员");
    });
});
