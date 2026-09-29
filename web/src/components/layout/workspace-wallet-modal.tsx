import { App, Button, Input, Skeleton, type InputRef } from "antd";
import { Check, ChevronLeft, ChevronRight, CircleAlert, Coins, CreditCard, HardDrive, History, RefreshCw, TicketCheck, WalletCards } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { PaymentCheckoutCode } from "@/components/payment-checkout-code";
import { AppModal } from "@/components/ui/product/app-modal";
import { formatCredits } from "@/constant/credits";
import { closePaymentOrder, createPaymentOrder, getPaymentOrder, listPaymentProviders, listTopupProducts, queryPaymentOrder, refreshPaymentCheckout, type PaymentOrder, type PaymentProvider, type TopupProduct } from "@/services/api/payments";
import { getWallet, redeemCredits, type CreditLedgerEntry, type WalletSummary } from "@/services/api/wallet";
import { getMembership, listMembershipProducts } from "@/services/api/membership";
import { invalidateAuthSessionCache } from "@/services/api/auth";
import { useAccountFileStorageUsage } from "@/hooks/use-account-file-storage-usage";
import { accountStorageMeter } from "@/lib/account-storage-usage";
import { cn } from "@/lib/utils";
import {
    defaultMembership,
    formatMembershipStorage,
    groupMembershipProductsByTier,
    isCatalogMembershipSKU,
    MEMBERSHIP_ADVANCED_FEATURE_KEYS,
    membershipPurchaseBlocked,
    membershipSKUTier,
    membershipStatusLabel,
    type MembershipProduct,
    type MembershipStatus,
} from "@/lib/membership";
import { openWorkspaceWallet, resolveWalletTab, WORKSPACE_WALLET_OPEN_EVENT, type WalletModalTab, type WorkspaceWalletOpenDetail } from "@/lib/workspace-wallet";
import { useTranslation } from "react-i18next";
import { useUserStore } from "@/stores/use-user-store";

export function WorkspaceWalletHost() {
    const { pathname, search } = useLocation();
    const navigate = useNavigate();
    const [open, setOpen] = useState(false);
    const [pendingPaymentOrderId, setPendingPaymentOrderId] = useState("");
    const [paymentInvalid, setPaymentInvalid] = useState(false);
    const [initialTab, setInitialTab] = useState<WalletModalTab>("subscribe");
    const [focusRedeem, setFocusRedeem] = useState(false);

    const applyOpen = (detail: WorkspaceWalletOpenDetail = {}) => {
        setPendingPaymentOrderId(detail.paymentOrderId || "");
        setPaymentInvalid(Boolean(detail.paymentInvalid));
        setInitialTab(resolveWalletTab(detail.tab));
        setFocusRedeem(Boolean(detail.focusRedeem));
        setOpen(true);
    };

    useEffect(() => {
        const handleOpen = (raw: Event) => {
            applyOpen((raw as CustomEvent<WorkspaceWalletOpenDetail>).detail || {});
        };
        window.addEventListener(WORKSPACE_WALLET_OPEN_EVENT, handleOpen);
        return () => window.removeEventListener(WORKSPACE_WALLET_OPEN_EVENT, handleOpen);
    }, []);

    useEffect(() => {
        if (pathname !== "/wallet") return;
        const params = new URLSearchParams(search);
        openWorkspaceWallet({
            paymentOrderId: params.get("paymentOrder") || undefined,
            paymentInvalid: params.get("payment") === "invalid",
        });
        navigate("/", { replace: true });
    }, [navigate, pathname, search]);

    return (
        <WorkspaceWalletModal
            open={open}
            pendingPaymentOrderId={pendingPaymentOrderId}
            paymentInvalid={paymentInvalid}
            initialTab={initialTab}
            focusRedeem={focusRedeem}
            onClose={() => {
                setOpen(false);
                setPendingPaymentOrderId("");
                setPaymentInvalid(false);
                setFocusRedeem(false);
            }}
        />
    );
}

export function WorkspaceWalletModal({
    open,
    onClose,
    pendingPaymentOrderId,
    paymentInvalid,
    initialTab = "subscribe",
    focusRedeem = false,
}: {
    open: boolean;
    onClose: () => void;
    pendingPaymentOrderId?: string;
    paymentInvalid?: boolean;
    initialTab?: WalletModalTab;
    focusRedeem?: boolean;
}) {
    const { message, modal } = App.useApp();
    const { t } = useTranslation("setting");
    const navigate = useNavigate();
    const creditsEnabled = useUserStore((state) => state.features.creditsEnabled);
    const storageUsage = useAccountFileStorageUsage(open);
    const storageMeter = accountStorageMeter(storageUsage.data);
    const [tab, setTab] = useState<WalletModalTab>(initialTab);
    const redeemSectionRef = useRef<HTMLElement | null>(null);
    const redeemInputRef = useRef<InputRef>(null);
    const [membership, setMembership] = useState<MembershipStatus>(defaultMembership);
    const [membershipProducts, setMembershipProducts] = useState<MembershipProduct[]>([]);
    const [selectedMembershipId, setSelectedMembershipId] = useState("");
    const membershipIdempotencyKey = useRef("");
    const [wallet, setWallet] = useState<WalletSummary | null>(null);
    const [walletLoading, setWalletLoading] = useState(false);
    const [walletError, setWalletError] = useState("");
    const [page, setPage] = useState(1);
    const [creditProducts, setCreditProducts] = useState<TopupProduct[]>([]);
    const [storageProducts, setStorageProducts] = useState<TopupProduct[]>([]);
    const [providers, setProviders] = useState<PaymentProvider[]>([]);
    const [paymentsLoading, setPaymentsLoading] = useState(false);
    const [selectedCreditId, setSelectedCreditId] = useState("");
    const [selectedStorageId, setSelectedStorageId] = useState("");
    const [selectedProviderId, setSelectedProviderId] = useState("");
    const [code, setCode] = useState("");
    const [redeeming, setRedeeming] = useState(false);
    const [paymentCreating, setPaymentCreating] = useState(false);
    const [paymentQuerying, setPaymentQuerying] = useState(false);
    const [paymentOrder, setPaymentOrder] = useState<PaymentOrder | null>(null);
    const [paymentOpen, setPaymentOpen] = useState(false);
    const [clock, setClock] = useState(Date.now());
    const idempotencyKey = useRef("");
    const completedOrderId = useRef("");
    const requestSequence = useRef(0);

    const selectedCredit = useMemo(() => creditProducts.find((item) => item.id === selectedCreditId), [creditProducts, selectedCreditId]);
    const selectedStorage = useMemo(() => storageProducts.find((item) => item.id === selectedStorageId), [storageProducts, selectedStorageId]);
    const selectedProvider = useMemo(() => providers.find((item) => item.id === selectedProviderId), [providers, selectedProviderId]);
    const groupedPlans = useMemo(() => groupMembershipProductsByTier(membershipProducts), [membershipProducts]);
    const personalStorage = membership.personalBucketEnabled || membership.storageDisplay === "personal";

    const reloadWallet = async (targetPage = page) => {
        const sequence = ++requestSequence.current;
        setWalletLoading(true);
        setWalletError("");
        try {
            const result = await getWallet(targetPage, 20, "all");
            if (sequence === requestSequence.current) setWallet(result);
        } catch (error) {
            if (sequence === requestSequence.current) setWalletError(error instanceof Error ? error.message : "读取积分账户失败");
        } finally {
            if (sequence === requestSequence.current) setWalletLoading(false);
        }
    };

    useEffect(() => {
        if (!open) return;
        setTab(initialTab);
        setPage(1);
        if (creditsEnabled) void reloadWallet(1);
        setPaymentsLoading(true);
        void (async () => {
            const [creditResult, storageResult, providerResult, membershipResult, membershipProductResult] = await Promise.allSettled([
                creditsEnabled ? listTopupProducts("credit_topup") : Promise.resolve({ products: [] as TopupProduct[] }),
                listTopupProducts("storage_topup"),
                listPaymentProviders(),
                getMembership(),
                listMembershipProducts(),
            ]);
            const fail = (result: PromiseSettledResult<unknown>, fallback: string) => {
                if (result.status === "rejected") message.error(result.reason instanceof Error ? result.reason.message : fallback);
            };
            fail(creditResult, "读取积分商品失败");
            fail(storageResult, "读取容量商品失败");
            fail(providerResult, "读取支付渠道失败");
            fail(membershipResult, "读取订阅状态失败");
            fail(membershipProductResult, "读取订阅商品失败");
            if (creditResult.status === "fulfilled") {
                setCreditProducts(creditResult.value.products.filter((item) => item.enabled));
                setSelectedCreditId((current) => current || creditResult.value.products.find((item) => item.enabled)?.id || "");
            }
            if (storageResult.status === "fulfilled") {
                setStorageProducts(storageResult.value.products.filter((item) => item.enabled));
                setSelectedStorageId((current) => current || storageResult.value.products.find((item) => item.enabled)?.id || "");
            }
            if (providerResult.status === "fulfilled") {
                setProviders(providerResult.value.providers.filter((item) => item.enabled && item.pluginEnabled && item.configured));
                setSelectedProviderId((current) => current || providerResult.value.providers.find((item) => item.enabled && item.pluginEnabled && item.configured)?.id || "");
            }
            if (membershipResult.status === "fulfilled") {
                setMembership({ ...defaultMembership, ...membershipResult.value });
                useUserStore.getState().setMembership(membershipResult.value);
            }
            if (membershipProductResult.status === "fulfilled") {
                const listed = membershipProductResult.value.products.filter((item) => item.enabled && isCatalogMembershipSKU(item.sku));
                setMembershipProducts(listed);
                setSelectedMembershipId((current) => current || listed.find((item) => item.highlighted)?.id || listed[0]?.id || "");
            }
            setPaymentsLoading(false);
        })();
    }, [open, creditsEnabled, initialTab]);

    useEffect(() => {
        if (!open || !focusRedeem) return;
        const timer = window.setTimeout(() => {
            redeemSectionRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
            redeemInputRef.current?.focus();
        }, 80);
        return () => window.clearTimeout(timer);
    }, [open, focusRedeem, tab]);

    useEffect(() => {
        if (!open || !paymentInvalid) return;
        message.error("支付结果无效，未产生入账");
    }, [open, paymentInvalid]);

    useEffect(() => {
        idempotencyKey.current = "";
    }, [selectedCreditId, selectedStorageId, selectedProviderId]);

    useEffect(() => {
        if (!paymentOpen || !paymentOrder || !["created", "pending", "closing"].includes(paymentOrder.status)) return;
        const interval = window.setInterval(() => {
            void refreshPaymentStatus(paymentOrder.id, true);
        }, 4_000);
        return () => window.clearInterval(interval);
    }, [paymentOpen, paymentOrder?.id, paymentOrder?.status]);

    useEffect(() => {
        if (!paymentOpen) return;
        const interval = window.setInterval(() => setClock(Date.now()), 1_000);
        return () => window.clearInterval(interval);
    }, [paymentOpen]);

    const announceWalletUpdated = async (orderId?: string) => {
        if (orderId && completedOrderId.current === orderId) return;
        if (orderId) completedOrderId.current = orderId;
        setPage(1);
        if (creditsEnabled) await reloadWallet(1);
        invalidateAuthSessionCache();
        try {
            const next = await getMembership();
            setMembership({ ...defaultMembership, ...next });
            useUserStore.getState().setMembership(next);
        } catch {
            /* session refresh is best-effort */
        }
        window.dispatchEvent(new CustomEvent("wallet:updated"));
    };

    useEffect(() => {
        if (!open || !pendingPaymentOrderId) return;
        getPaymentOrder(pendingPaymentOrderId)
            .then(async ({ order }) => {
                setPaymentOrder(order);
                setPaymentOpen(true);
                if (order.status === "credited") await announceWalletUpdated(order.id);
            })
            .catch((error) => message.error(error instanceof Error ? error.message : "读取支付结果失败"));
    }, [open, pendingPaymentOrderId]);

    const redeem = async () => {
        const normalized = code.trim().toLowerCase();
        if (normalized.length !== 32) {
            message.error("请输入完整的 32 位兑换码");
            return;
        }
        setRedeeming(true);
        try {
            const outcome = await redeemCredits(normalized);
            setCode("");
            await announceWalletUpdated();
            if (outcome.granted?.kind === "membership" && outcome.granted.planSku === "permanent") message.success("已开通永久订阅");
            else if (outcome.granted?.kind === "membership") message.success("订阅已开通");
            else if (outcome.granted?.kind === "storage") message.success(`容量已增加 ${formatMembershipStorage(outcome.granted.storageQuotaBytes || 0)}`);
            else message.success("兑换成功，积分已到账");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "兑换失败");
        } finally {
            setRedeeming(false);
        }
    };

    const startMembershipPayment = async (product: MembershipProduct) => {
        if (!selectedProvider) {
            message.error("请选择支付方式");
            return;
        }
        const blocked = membershipPurchaseBlocked(membership, product.sku);
        if (blocked.disabled) {
            message.error(blocked.reason);
            return;
        }
        const pay = async () => {
            setPaymentCreating(true);
            try {
                if (!membershipIdempotencyKey.current) membershipIdempotencyKey.current = crypto.randomUUID();
                const result = await createPaymentOrder({ productId: product.id, providerId: selectedProvider.id, idempotencyKey: membershipIdempotencyKey.current, productKind: "membership" });
                membershipIdempotencyKey.current = "";
                setPaymentOrder(result.order);
                if (result.order.status === "credited") await announceWalletUpdated(result.order.id);
                if (result.order.checkout.mode === "redirect" && result.order.checkout.url) {
                    window.location.assign(result.order.checkout.url);
                    return;
                }
                setPaymentOpen(true);
            } catch (error) {
                message.error(error instanceof Error ? error.message : "创建订阅订单失败");
            } finally {
                setPaymentCreating(false);
            }
        };
        if (membership.advancedRemainingSeconds > 0) {
            modal.confirm({
                title: "确认开通",
                content: "支付成功后立即按新套餐的容量和赠送积分生效，时长叠加到当前到期日。",
                onOk: () => pay(),
            });
            return;
        }
        await pay();
    };

    const startTopupPayment = async (product: TopupProduct, kind: "credit_topup" | "storage_topup") => {
        if (!selectedProvider) {
            message.error("请选择支付方式");
            return;
        }
        setPaymentCreating(true);
        try {
            if (!idempotencyKey.current) idempotencyKey.current = crypto.randomUUID();
            const result = await createPaymentOrder({ productId: product.id, providerId: selectedProvider.id, idempotencyKey: idempotencyKey.current, productKind: kind });
            idempotencyKey.current = "";
            setPaymentOrder(result.order);
            if (result.order.status === "credited") await announceWalletUpdated(result.order.id);
            if (result.order.checkout.mode === "redirect" && result.order.checkout.url) {
                window.location.assign(result.order.checkout.url);
                return;
            }
            setPaymentOpen(true);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "创建支付订单失败");
        } finally {
            setPaymentCreating(false);
        }
    };

    async function refreshPaymentStatus(orderId = paymentOrder?.id, silent = false) {
        if (!orderId || paymentQuerying) return;
        setPaymentQuerying(true);
        try {
            const result = await queryPaymentOrder(orderId);
            setPaymentOrder(result.order);
            if (result.order.status === "credited") {
                await announceWalletUpdated(result.order.id);
                if (!silent) message.success(creditedMessage(result.order));
            } else if (!silent && result.order.status === "closed") message.warning("订单已关闭，未产生入账");
            else if (!silent) message.info("渠道尚未确认支付，请稍后再试");
        } catch (error) {
            if (!silent) message.error(error instanceof Error ? error.message : "查询支付结果失败");
        } finally {
            setPaymentQuerying(false);
        }
    }

    const cancelPayment = async () => {
        if (!paymentOrder) return;
        setPaymentQuerying(true);
        try {
            const result = await closePaymentOrder(paymentOrder.id);
            setPaymentOrder(result.order);
            if (result.order.status === "credited") await announceWalletUpdated(result.order.id);
            else message.success("未支付订单已关闭");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "关闭订单失败");
        } finally {
            setPaymentQuerying(false);
        }
    };

    const retryCheckout = async () => {
        if (!paymentOrder) return;
        setPaymentQuerying(true);
        try {
            const result = await refreshPaymentCheckout(paymentOrder.id);
            setPaymentOrder(result.order);
            if (result.order.checkout.mode === "redirect" && result.order.checkout.url) window.location.assign(result.order.checkout.url);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "刷新支付入口失败");
        } finally {
            setPaymentQuerying(false);
        }
    };

    const available = wallet?.account.availableMicrocredits ?? 0;
    const totalPages = Math.max(1, Math.ceil((wallet?.total || 0) / 20));
    const identity = membershipStatusLabel(membership);
    const expiresLabel = membership.advancedExpiresAt ? new Date(membership.advancedExpiresAt).toLocaleDateString() : "";

    const redeemBlock = membership.redeemEnabled !== false ? (
        <section ref={redeemSectionRef} className="workspace-wallet-section is-redeem">
            <div className="workspace-wallet-section-heading"><div><h3>{t("wallet.redeem")}</h3><p>{t("wallet.redeemLead")}</p></div><TicketCheck /></div>
            <div className="workspace-wallet-redeem-row">
                <Input ref={redeemInputRef} size="large" value={code} maxLength={32} placeholder={t("wallet.redeemPlaceholder")} onChange={(event) => setCode(event.target.value.replace(/\s/g, ""))} onPressEnter={() => void redeem()} />
                <Button size="large" loading={redeeming} disabled={code.trim().length !== 32} onClick={() => void redeem()}>{t("wallet.redeemConfirm")}</Button>
            </div>
        </section>
    ) : null;

    const providerBar = (onPay: () => void, disabled: boolean, label: string) => (
        <div className="workspace-wallet-provider-row">
            <div className="workspace-wallet-providers" role="radiogroup" aria-label={t("wallet.payMethod")}>
                {providers.map((provider) => <button key={provider.id} type="button" role="radio" aria-checked={selectedProviderId === provider.id} className={selectedProviderId === provider.id ? "is-selected" : ""} onClick={() => setSelectedProviderId(provider.id)}><CreditCard />{provider.name}</button>)}
            </div>
            <Button type="primary" size="large" loading={paymentCreating} disabled={disabled || !selectedProvider || !membership.onlinePaymentEnabled} onClick={onPay}>{label}</Button>
        </div>
    );

    return (
        <>
            <AppModal
                flush
                open={open}
                title={null}
                footer={null}
                centered
                width="min(880px, calc(100vw - 28px))"
                onCancel={onClose}
                rootClassName="workspace-wallet-modal"
                classNames={{ root: "workspace-wallet-modal", wrapper: "workspace-wallet-modal", container: "workspace-wallet-modal", body: "workspace-wallet-modal-body" }}
                styles={{
                    root: { background: "var(--user-page-bg, #f8f8fa)", backdropFilter: "none", opacity: 1 },
                    container: { background: "var(--user-page-bg, #f8f8fa)", backdropFilter: "none", opacity: 1 },
                    body: { background: "var(--user-page-bg, #f8f8fa)" },
                }}
            >
                <div className="workspace-wallet-shell">
                    <header className="workspace-wallet-header">
                        <div>
                            <span className="workspace-wallet-kicker"><WalletCards />{t("wallet.subscribeCenter")}</span>
                            <h2>{identity}{expiresLabel ? ` · ${expiresLabel}` : ""}</h2>
                            <p>{personalStorage ? t("wallet.headerPersonal") : t("wallet.headerPlatform", { size: formatMembershipStorage(membership.effectiveStoredFileBytes) })}</p>
                        </div>
                        <div className="workspace-wallet-balance">
                            {creditsEnabled ? <>
                                <span>{t("wallet.available")}</span>
                                <strong>{wallet ? formatCredits(available, 6) : "--"}</strong>
                                <small>{personalStorage ? `${t("wallet.personalUsed")} ${storageMeter.usedLabel}` : t("wallet.cloudQuota")}</small>
                            </> : <>
                                <span>{t("wallet.status")}</span>
                                <strong>{identity}</strong>
                                <small>{membership.personalStorageAllowed ? t("wallet.personalOn") : t("wallet.personalOff")}</small>
                            </>}
                        </div>
                    </header>

                    <div className="workspace-wallet-tabs" role="tablist" aria-label={t("wallet.tabs")}>
                        <button type="button" role="tab" aria-selected={tab === "subscribe"} onClick={() => setTab("subscribe")}><WalletCards />{t("wallet.tabSubscribe")}</button>
                        <button type="button" role="tab" aria-selected={tab === "storage"} onClick={() => setTab("storage")}><HardDrive />{t("wallet.tabStorage")}</button>
                        {creditsEnabled ? <button type="button" role="tab" aria-selected={tab === "credits"} onClick={() => setTab("credits")}><Coins />{t("wallet.tabCredits")}</button> : null}
                        {creditsEnabled ? <button type="button" role="tab" aria-selected={tab === "history"} onClick={() => setTab("history")}><History />{t("wallet.tabHistory")}</button> : null}
                    </div>

                    {tab === "subscribe" ? (
                        <div className="workspace-wallet-content is-topup">
                            {membershipStatusLabel(membership) === "未开通" ? (
                                <section className="workspace-wallet-section">
                                    <div className="workspace-wallet-section-heading"><div><h3>{t("wallet.compareTitle")}</h3><p>{t("wallet.compareLead")}</p></div></div>
                                    <ul className="workspace-wallet-feature-list">
                                        {MEMBERSHIP_ADVANCED_FEATURE_KEYS.map((key) => (
                                            <li key={key}><span aria-hidden="true">×</span>{t(`wallet.feature.${key}`)}</li>
                                        ))}
                                    </ul>
                                </section>
                            ) : null}
                            <section className="workspace-wallet-section">
                                <div className="workspace-wallet-section-heading"><div><h3>{t("wallet.plans")}</h3><p>{t("wallet.plansLead")}</p></div><WalletCards /></div>
                                {paymentsLoading ? <Skeleton active paragraph={{ rows: 3 }} /> : membershipProducts.length ? <>
                                    {membership.hasOpenMembershipOrder ? <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.openOrder")}</strong><span>{t("wallet.openOrderLead")}</span></div><Button onClick={() => { if (membership.openMembershipOrderId) { void getPaymentOrder(membership.openMembershipOrderId).then(({ order }) => { setPaymentOrder(order); setPaymentOpen(true); }); } }}>{t("wallet.continuePay")}</Button></div> : null}
                                    <div className="workspace-wallet-plan-scroller">
                                        {(["vip", "svip"] as const).map((tier) => {
                                            const items = groupedPlans[tier];
                                            if (!items.length) return null;
                                            const selected = items.find((item) => item.id === selectedMembershipId) || items[0];
                                            const blocked = membershipPurchaseBlocked(membership, selected.sku);
                                            const unpriced = selected.amountFen <= 0;
                                            return (
                                                <article key={tier} className={cn("workspace-wallet-plan-card", selected.highlighted && "is-highlighted", membershipSKUTier(membership.tier || membership.advancedPlanSku) === tier && "is-current")}>
                                                    <header>
                                                        <strong>{tier === "svip" ? "SVIP" : "VIP"}</strong>
                                                        {selected.badge ? <em>{selected.badge}</em> : null}
                                                    </header>
                                                    <div className="workspace-wallet-plan-periods" role="tablist">
                                                        {items.map((item) => (
                                                            <button key={item.id} type="button" className={item.id === selected.id ? "is-selected" : ""} onClick={() => setSelectedMembershipId(item.id)}>
                                                                {t(`sku.period.${item.durationDays}`, { defaultValue: `${item.durationDays} 天` })}
                                                            </button>
                                                        ))}
                                                    </div>
                                                    <p className="workspace-wallet-plan-price">
                                                        <b>{selected.amountFen > 0 ? `¥ ${(selected.amountFen / 100).toFixed(0)}` : t("wallet.unpriced")}</b>
                                                        {selected.originalAmountFen && selected.originalAmountFen > selected.amountFen ? <s>¥ {(selected.originalAmountFen / 100).toFixed(0)}</s> : null}
                                                    </p>
                                                    <ul>
                                                        <li>{t("wallet.planAllFeatures")}</li>
                                                        <li>{t("wallet.planGift", { credits: formatCredits(selected.creditsMicrocredits, 0) })}</li>
                                                        <li>{t("wallet.planStorage", { size: formatMembershipStorage(selected.storageQuotaBytes) })}</li>
                                                    </ul>
                                                    {blocked.disabled ? <small>{blocked.reasonKey ? t(blocked.reasonKey) : blocked.reason}</small> : null}
                                                    <Button type="primary" loading={paymentCreating} disabled={blocked.disabled || unpriced || !membership.onlinePaymentEnabled || !selectedProvider} onClick={() => void startMembershipPayment(selected)}>{t("wallet.buy")}</Button>
                                                </article>
                                            );
                                        })}
                                    </div>
                                    {providerBar(() => {
                                        const product = membershipProducts.find((item) => item.id === selectedMembershipId);
                                        if (product) void startMembershipPayment(product);
                                    }, !selectedMembershipId || membershipPurchaseBlocked(membership, membershipProducts.find((item) => item.id === selectedMembershipId)?.sku || "").disabled || (membershipProducts.find((item) => item.id === selectedMembershipId)?.amountFen || 0) <= 0, t("wallet.buy"))}
                                </> : <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.unlisted")}</strong><span>{t("wallet.unlistedLead")}</span></div></div>}
                            </section>
                            {redeemBlock}
                        </div>
                    ) : null}

                    {tab === "storage" ? (
                        <div className="workspace-wallet-content is-topup">
                            <section className="workspace-wallet-section">
                                <div className="workspace-wallet-section-heading"><div><h3>{t("wallet.storage")}</h3><p>{personalStorage ? t("wallet.storagePersonalLead") : t("wallet.storageLead", { size: formatMembershipStorage(membership.effectiveStoredFileBytes) })}{!personalStorage && membership.storageBonusBytes ? t("wallet.storageBonus", { size: formatMembershipStorage(membership.storageBonusBytes) }) : ""}</p></div><HardDrive /></div>
                                {personalStorage ? <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.personalActive")}</strong><span>{t("wallet.storagePersonalHint")}</span></div><Button onClick={() => { onClose(); navigate("/settings"); }}>{t("wallet.manageStorage")}</Button></div> : null}
                                {paymentsLoading ? <Skeleton active paragraph={{ rows: 4 }} /> : storageProducts.length && providers.length ? <>
                                    <div className="workspace-wallet-products">
                                        {storageProducts.map((product) => (
                                            <button key={product.id} type="button" className={cn("workspace-wallet-product", selectedStorageId === product.id && "is-selected")} aria-pressed={selectedStorageId === product.id} onClick={() => setSelectedStorageId(product.id)}>
                                                <span>{product.name}</span>
                                                <strong>{formatMembershipStorage(product.storageBytes || 0)}</strong>
                                                <small>¥ {(product.amountFen / 100).toFixed(2)}{product.description ? ` · ${product.description}` : ""}</small>
                                                {selectedStorageId === product.id ? <Check /> : null}
                                            </button>
                                        ))}
                                    </div>
                                    {providerBar(() => { if (selectedStorage) void startTopupPayment(selectedStorage, "storage_topup"); }, !selectedStorage, t("wallet.buyStorage"))}
                                </> : <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.storageUnlisted")}</strong><span>{t("wallet.storageUnlistedLead")}</span></div></div>}
                            </section>
                            {redeemBlock}
                        </div>
                    ) : null}

                    {tab === "credits" && creditsEnabled ? (
                        <div className="workspace-wallet-content is-topup">
                            <section className="workspace-wallet-section">
                                <div className="workspace-wallet-section-heading"><div><h3>{t("wallet.online")}</h3><p>{t("wallet.onlineLead")}</p></div><CreditCard /></div>
                                {paymentsLoading ? <Skeleton active paragraph={{ rows: 4 }} /> : creditProducts.length && providers.length ? <>
                                    <div className="workspace-wallet-products">
                                        {creditProducts.map((product) => (
                                            <button key={product.id} type="button" className={cn("workspace-wallet-product", selectedCreditId === product.id && "is-selected")} aria-pressed={selectedCreditId === product.id} onClick={() => setSelectedCreditId(product.id)}>
                                                <span>{product.name}</span>
                                                <strong>{formatCredits(product.creditsMicrocredits, 6)} {t("wallet.creditsUnit")}</strong>
                                                <small>¥ {(product.amountFen / 100).toFixed(2)}{product.description ? ` · ${product.description}` : ""}</small>
                                                {selectedCreditId === product.id ? <Check /> : null}
                                            </button>
                                        ))}
                                    </div>
                                    {providerBar(() => { if (selectedCredit) void startTopupPayment(selectedCredit, "credit_topup"); }, !selectedCredit, t("wallet.payNow"))}
                                </> : <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.onlineOff")}</strong><span>{t("wallet.onlineOffLead")}</span></div></div>}
                            </section>
                            {redeemBlock}
                        </div>
                    ) : null}

                    {tab === "history" ? (
                        <div className="workspace-wallet-content is-history">
                            <div className="workspace-wallet-history-toolbar"><div><h3>{t("wallet.tabHistory")}</h3><p>{t("wallet.historyLead")}</p></div><Button type="text" icon={<RefreshCw />} loading={walletLoading} onClick={() => void reloadWallet(page)}>{t("wallet.refresh")}</Button></div>
                            <div className="workspace-wallet-history-scroll">
                                {walletError ? <div className="workspace-wallet-inline-state is-error"><CircleAlert /><div><strong>{t("wallet.loadFailed")}</strong><span>{walletError}</span></div><Button onClick={() => void reloadWallet(page)}>{t("wallet.retry")}</Button></div> : walletLoading && !wallet ? <Skeleton active paragraph={{ rows: 6 }} /> : wallet?.entries.length ? <div className="workspace-wallet-ledger">
                                    {wallet.entries.map((entry) => <WalletLedgerRow key={entry.id} entry={entry} />)}
                                </div> : <div className="workspace-wallet-empty"><History /><strong>{t("wallet.emptyHistory")}</strong><span>{t("wallet.emptyHistoryLead")}</span></div>}
                            </div>
                            <div className="workspace-wallet-pagination">
                                <span>{t("wallet.page", { page, total: totalPages })}</span>
                                <button type="button" disabled={page <= 1 || walletLoading} aria-label="上一页" onClick={() => { const next = page - 1; setPage(next); void reloadWallet(next); }}><ChevronLeft /></button>
                                <button type="button" disabled={page >= totalPages || walletLoading} aria-label="下一页" onClick={() => { const next = page + 1; setPage(next); void reloadWallet(next); }}><ChevronRight /></button>
                            </div>
                        </div>
                    ) : null}
                </div>
            </AppModal>

            <AppModal open={paymentOpen} title={paymentOrder?.status === "credited" ? creditedTitle(paymentOrder) : paymentOrder?.checkout.mode === "qr_code" ? "扫码支付" : "确认支付结果"} centered width={430} onCancel={() => setPaymentOpen(false)} footer={paymentFooter(paymentOrder, paymentQuerying, () => setPaymentOpen(false), cancelPayment, refreshPaymentStatus, retryCheckout)}>
                {paymentOrder ? <div className="workspace-wallet-payment">
                    <span className="workspace-wallet-payment-icon"><CreditCard /></span>
                    <strong>¥ {(paymentOrder.amountFen / 100).toFixed(2)}</strong>
                    <p>{paymentOrderCaption(paymentOrder)}</p>
                    {paymentOrder.status === "pending" && paymentOrder.checkout.mode === "qr_code" && paymentOrder.checkout.value ? <><PaymentCheckoutCode value={paymentOrder.checkout.value} /><span>请使用支付应用扫码完成支付</span></> : null}
                    <PaymentStatus order={paymentOrder} now={clock} />
                </div> : null}
            </AppModal>
        </>
    );
}

function creditedMessage(order: PaymentOrder) {
    if (order.productKind === "membership" && order.planSku === "permanent") return "已开通永久订阅";
    if (order.productKind === "membership") return "订阅已开通";
    if (order.productKind === "storage_topup") return "容量已到账";
    return "支付已确认，积分已到账";
}

function creditedTitle(order: PaymentOrder) {
    if (order.productKind === "membership") return "订阅完成";
    if (order.productKind === "storage_topup") return "容量到账";
    return "充值完成";
}

function paymentOrderCaption(order: PaymentOrder) {
    if (order.productKind === "membership" && order.planSku === "permanent") return `${order.productName} · 永久订阅`;
    if (order.productKind === "membership") return order.productName;
    if (order.productKind === "storage_topup") return `${order.productName} · ${formatMembershipStorage(order.storageQuotaBytes || 0)}`;
    return `${order.productName} · ${formatCredits(order.creditsMicrocredits, 6)} 积分`;
}

function WalletLedgerRow({ entry }: { entry: CreditLedgerEntry }) {
    const { t, i18n } = useTranslation("setting");
    const positive = entry.amountMicrocredits > 0;
    const title = entry.type === "consume" ? t("wallet.ledger.consume") : entry.type === "refund" ? t("wallet.ledger.refund") : entry.type === "payment_topup" ? t("wallet.ledger.topup") : entry.type === "redeem" ? t("wallet.ledger.redeem") : entry.note || t("wallet.ledger.adjust");
    return <article className="workspace-wallet-ledger-row"><span className={cn("workspace-wallet-ledger-icon", positive ? "is-income" : "is-consume")}>{positive ? <Coins /> : <CreditCard />}</span><div><strong>{title}</strong><span>{[entry.scene, entry.model, entry.note].filter(Boolean).join(" · ") || t("wallet.ledger.change")}</span></div><time>{new Date(entry.createdAt).toLocaleString(i18n.language === "zh" ? "zh-CN" : i18n.language, { hour12: false })}</time><b className={positive ? "is-income" : "is-consume"}>{positive ? "+" : ""}{formatCredits(entry.amountMicrocredits, 6)}</b></article>;
}

function PaymentStatus({ order, now }: { order: PaymentOrder; now: number }) {
    if (order.status === "credited") return <div className="workspace-wallet-payment-status is-success">{creditedMessage(order)}</div>;
    if (order.status === "closed") return <div className="workspace-wallet-payment-status">订单已关闭，未产生入账</div>;
    if (order.status === "create_failed") return <div className="workspace-wallet-payment-status is-error">支付入口创建失败，请重新生成支付入口。</div>;
    const remaining = Math.max(0, Math.floor((new Date(order.expiresAt).getTime() - now) / 1000));
    const hours = Math.floor(remaining / 3600);
    const minutes = Math.floor((remaining % 3600) / 60);
    const seconds = remaining % 60;
    return <div className="workspace-wallet-payment-status">订单剩余 {String(hours).padStart(2, "0")}:{String(minutes).padStart(2, "0")}:{String(seconds).padStart(2, "0")}，将自动确认支付结果</div>;
}

function paymentFooter(order: PaymentOrder | null, loading: boolean, close: () => void, cancel: () => Promise<void>, query: (id?: string, silent?: boolean) => Promise<void>, retry: () => Promise<void>) {
    if (order?.status === "pending") return [<Button key="cancel" danger disabled={loading} onClick={() => void cancel()}>关闭订单</Button>, <Button key="query" type="primary" loading={loading} onClick={() => void query()}>我已完成支付</Button>];
    if (order?.status === "create_failed") return [<Button key="close" onClick={close}>稍后处理</Button>, <Button key="retry" type="primary" loading={loading} onClick={() => void retry()}>重新生成支付入口</Button>];
    return [<Button key="done" type="primary" onClick={close}>完成</Button>];
}
