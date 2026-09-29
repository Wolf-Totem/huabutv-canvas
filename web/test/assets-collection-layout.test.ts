import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

describe("asset library category sidebar", () => {
    test("keeps type, business and folder filters in a left-hand nav", () => {
        const page = readFileSync(resolve(import.meta.dir, "../src/pages/assets/index.tsx"), "utf8");
        const css = readFileSync(resolve(import.meta.dir, "../src/styles/workspace-product.css"), "utf8");
        expect(page).toContain('className="assets-collection-layout"');
        expect(page).toContain('aria-label="素材分类"');
        expect(page).toContain('title="素材类型"');
        expect(page).toContain('title="业务分类"');
        expect(page).toContain("我的分类");
        expect(page).not.toContain("全部自定义分类");
        expect(css).toMatch(/\.assets-collection-layout\s*\{[^}]*grid-template-columns:\s*220px minmax\(0, 1fr\)/s);
    });
});

describe("wallet history pagination", () => {
    test("pins ledger pagination to the history panel footer", () => {
        const modal = readFileSync(resolve(import.meta.dir, "../src/components/layout/workspace-wallet-modal.tsx"), "utf8");
        const css = readFileSync(resolve(import.meta.dir, "../src/styles/globals.css"), "utf8");
        expect(modal).toContain("workspace-wallet-history-scroll");
        expect(modal).toContain("workspace-wallet-pagination");
        expect(modal).not.toContain("wallet.total > 20");
        expect(css).toMatch(/\.workspace-wallet-content\.is-history\s*\{[^}]*overflow:\s*hidden/s);
        expect(css).toMatch(/\.workspace-wallet-pagination\s*\{[^}]*margin-top:\s*auto/s);
    });
});

describe("wallet scan checkout", () => {
    test("opens a brand picker before creating a scan order, and keeps chips only in the pay modal", () => {
        const modal = readFileSync(resolve(import.meta.dir, "../src/components/layout/workspace-wallet-modal.tsx"), "utf8");
        const css = readFileSync(resolve(import.meta.dir, "../src/styles/globals.css"), "utf8");
        expect(modal).toContain("beginBrandCheckout");
        expect(modal).toContain("pickCheckoutBrand");
        expect(modal).toContain("solePaymentScanBrand");
        expect(modal).toContain("workspace-wallet-brand-picker");
        expect(modal).toContain("wallet.choosePayLead");
        expect(modal).toContain("workspace-wallet-scan-channels");
        expect(modal).toContain("wallet.payWechat");
        expect(modal).toContain("wallet.payAlipay");
        expect(modal).toContain("selectScanBrand");
        expect(modal).toContain("resumeOpenMembershipOrder");
        expect(modal).toContain("cancelOpenMembershipOrder");
        expect(modal).toContain("wallet.continuePay");
        expect(modal).toContain("wallet.closeOrder");
        expect(modal).toContain('footer={null}');
        expect(modal).not.toContain("openScanCheckout");
        expect(modal).not.toContain("wallet.iPaid");
        expect(modal).not.toContain("paymentFooter");
        expect(modal).not.toContain("selectedProviderId");
        expect(modal).not.toContain("我已完成支付");
        expect(modal).not.toContain("{provider.name}");
        expect(modal).not.toContain("workspace-wallet-providers");
        expect(modal).not.toContain("斗拱");
        expect(css).toMatch(/\.workspace-wallet-scan-channels button\.is-selected\s*\{[^}]*color:\s*#16161a/s);
        expect(css).toMatch(/\.workspace-wallet-brand-picker-actions button\s*\{[^}]*color:\s*#16161a/s);
    });
});

describe("wallet subscription center", () => {
    test("renders a 1080px three-column subscribe grid bound to API showcase fields", () => {
        const modal = readFileSync(resolve(import.meta.dir, "../src/components/layout/workspace-wallet-modal.tsx"), "utf8");
        const css = readFileSync(resolve(import.meta.dir, "../src/styles/globals.css"), "utf8");
        expect(modal).toContain('width="min(1080px, calc(100vw - 28px))"');
        expect(modal).toContain("workspace-wallet-plan-grid");
        expect(modal).toContain("freeShowcase");
        expect(modal).toContain("featureLines");
        expect(modal).toContain("wallet.displayOnly");
        expect(modal).not.toContain("MEMBERSHIP_ADVANCED_FEATURE_KEYS");
        expect(css).toMatch(/\.workspace-wallet-plan-grid\s*\{[^}]*grid-template-columns:\s*repeat\(3, minmax\(0, 1fr\)\)/s);
        expect(css).toMatch(/\.workspace-wallet-plan-card\.is-vip::before/s);
        expect(css).toMatch(/\.workspace-wallet-plan-card\.is-svip::before/s);
    });
});

describe("admin membership showcase editor", () => {
    test("lets operators edit feature lists, sync a tier, and set storage duration days", () => {
        const page = readFileSync(resolve(import.meta.dir, "../src/pages/admin/payments/payments-page.tsx"), "utf8");
        const redeem = readFileSync(resolve(import.meta.dir, "../src/pages/admin/components/redemption-codes-panel.tsx"), "utf8");
        expect(page).toContain("syncShowcaseToTier");
        expect(page).toContain("featureLines");
        expect(page).toContain("编辑未开通列");
        expect(page).toContain("updateAdminMembershipFreeShowcase");
        expect(page).toContain("durationDays");
        expect(page).not.toContain("权益清单只读：全部平台功能");
        expect(redeem).toContain("durationDays");
        expect(redeem).toContain("容量有效天数");
    });
});
