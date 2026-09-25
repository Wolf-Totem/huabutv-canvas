import { describe, expect, test } from "bun:test";

import { defaultMembership, formatMembershipStorage, membershipPurchaseBlocked, membershipStatusLabel } from "../src/lib/membership";

describe("membership purchase rules", () => {
    test("blocks all SKUs when an open membership order exists", () => {
        const status = { ...defaultMembership, hasOpenMembershipOrder: true, openMembershipOrderId: "order-1" };
        expect(membershipPurchaseBlocked(status, "permanent").disabled).toBe(true);
        expect(membershipPurchaseBlocked(status, "advanced_month").disabled).toBe(true);
        expect(membershipPurchaseBlocked(status, "advanced_year").reason).toContain("未完成");
    });

    test("allows year-to-month inside the 30 day window", () => {
        const status = { ...defaultMembership, advancedPlanSku: "advanced_year", advancedRemainingSeconds: 20 * 24 * 3600 };
        expect(membershipPurchaseBlocked(status, "advanced_month").disabled).toBe(false);
        expect(membershipPurchaseBlocked(status, "advanced_year").disabled).toBe(false);
    });

    test("blocks every advanced SKU when remaining is over 30 days", () => {
        const status = { ...defaultMembership, advancedPlanSku: "advanced_year", advancedRemainingSeconds: 31 * 24 * 3600 };
        expect(membershipPurchaseBlocked(status, "advanced_month").disabled).toBe(true);
        expect(membershipPurchaseBlocked(status, "advanced_year").disabled).toBe(true);
    });

    test("formats GiB/TiB without 32-bit shift overflow", () => {
        expect(formatMembershipStorage(1024 ** 3)).toBe("1 GiB");
        expect(formatMembershipStorage(5 * 1024 ** 3)).toBe("5 GiB");
        expect(formatMembershipStorage(3 * 1024 ** 4)).toBe("3 TiB");
    });

    test("labels membership status for the sidebar account row", () => {
        expect(membershipStatusLabel(null)).toBe("平台会员");
        expect(membershipStatusLabel(defaultMembership)).toBe("平台会员");
        expect(membershipStatusLabel({ ...defaultMembership, advancedPlanSku: "advanced_month" })).toBe("高级会员");
        expect(membershipStatusLabel({ ...defaultMembership, permanentActive: true })).toBe("永久会员");
    });
});
