import { App, Button, Input, Skeleton, type InputRef } from "antd";
import { Check, ChevronLeft, ChevronRight, CircleAlert, Coins, CreditCard, HardDrive, History, Minus, RefreshCw, TicketCheck, WalletCards } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { PaymentCheckoutCode } from "@/components/payment-checkout-code";
import { PaymentScanBrandMark } from "@/components/payment-scan-brand";
import { AppModal } from "@/components/ui/product/app-modal";
import { formatCredits } from "@/constant/credits";
import { defaultPaymentScanBrand, hasPaymentScanChannel, paymentScanBrand, qrScanChannels, type PaymentScanBrand } from "@/lib/payment-brands";
import { closePaymentOrder, createPaymentOrder, getPaymentOrder, listPaymentProviders, listTopupProducts, queryPaymentOrder, type PaymentOrder, type PaymentProvider, type TopupProduct } from "@/services/api/payments";
import { getWallet, redeemCredits, type CreditLedgerEntry, type WalletSummary } from "@/services/api/wallet";
import { getMembership, listMembershipProducts } from "@/services/api/membership";
import { invalidateAuthSessionCache } from "@/services/api/auth";
import { useAccountFileStorageUsage } from "@/hooks/use-account-file-storage-usage";
import { accountStorageMeter } from "@/lib/account-storage-usage";
import { cn } from "@/lib/utils";
import {
    dailyPriceYuan,
    defaultMembership,
    formatMembershipStorage,
    groupMembershipProductsByTier,
    isCatalogMembershipSKU,
    membershipPurchaseBlocked,
    MEMBERSHIP_PERIODS,
    membershipSKUDurationDays,
    membershipSKUTier,
    membershipStatusKey,
    membershipStatusLabel,
    periodFromDurationDays,
    periodSavingsPercent,
    productForPeriod,
    type MembershipPeriod,
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
    const [period, setPeriod] = useState<MembershipPeriod>("year");
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
    const [code, setCode] = useState("");
    const [redeeming, setRedeeming] = useState(false);
    const [paymentCreating, setPaymentCreating] = useState(false);
    const [paymentOrder, setPaymentOrder] = useState<PaymentOrder | null>(null);
    const [paymentOpen, setPaymentOpen] = useState(false);
    const [orderResuming, setOrderResuming] = useState(false);
    const [orderCancelling, setOrderCancelling] = useState(false);
    const [scanBrand, setScanBrand] = useState<PaymentScanBrand>("wechat");
    const [checkoutIntent, setCheckoutIntent] = useState<{ productId: string; productKind: "membership" | "credit_topup" | "storage_topup" } | null>(null);
    const [clock, setClock] = useState(Date.now());
    const completedOrderId = useRef("");
    const requestSequence = useRef(0);
    const scanGeneration = useRef(0);
    const silentQuerying = useRef(false);

    const selectedCredit = useMemo(() => creditProducts.find((item) => item.id === selectedCreditId), [creditProducts, selectedCreditId]);
    const selectedStorage = useMemo(() => storageProducts.find((item) => item.id === selectedStorageId), [storageProducts, selectedStorageId]);
    const scanChannels = useMemo(() => qrScanChannels(providers), [providers]);
    const canScanPay = hasPaymentScanChannel(providers);
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
            }
            if (membershipResult.status === "fulfilled") {
                setMembership({ ...defaultMembership, ...membershipResult.value });
                useUserStore.getState().setMembership(membershipResult.value);
            }
            if (membershipProductResult.status === "fulfilled") {
                const listed = membershipProductResult.value.products.filter((item) => item.enabled && isCatalogMembershipSKU(item.sku));
                setMembershipProducts(listed);
                const membershipValue = membershipResult.status === "fulfilled" ? membershipResult.value : undefined;
                const preferredDays = membershipSKUDurationDays(membershipValue?.advancedPlanSku);
                const nextPeriod = preferredDays ? periodFromDurationDays(preferredDays) : listed.some((item) => item.durationDays === 365) ? "year" : listed.some((item) => item.durationDays === 90) ? "quarter" : "month";
                setPeriod(nextPeriod);
                const days = MEMBERSHIP_PERIODS.find((item) => item.id === nextPeriod)?.days;
                setSelectedMembershipId((current) => current || listed.find((item) => item.durationDays === days && item.highlighted)?.id || listed.find((item) => item.durationDays === days)?.id || listed.find((item) => item.highlighted)?.id || listed[0]?.id || "");
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
                adoptPaymentOrder(order);
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

    const adoptPaymentOrder = (order: PaymentOrder) => {
        const kind = order.productKind === "membership" || order.productKind === "storage_topup" || order.productKind === "credit_topup" ? order.productKind : "credit_topup";
        setCheckoutIntent({ productId: order.productId, productKind: kind });
        setScanBrand(paymentScanBrand(order.providerId) || defaultPaymentScanBrand(qrScanChannels(providers)) || "wechat");
        setPaymentOrder(order);
        setPaymentOpen(true);
    };

    const startMembershipPayment = async (product: MembershipProduct) => {
        const brand = defaultPaymentScanBrand(scanChannels);
        if (!brand) {
            message.error(t("wallet.noScanChannel"));
            return;
        }
        const blocked = membershipPurchaseBlocked(membership, product.sku);
        if (blocked.disabled) {
            message.error(blocked.reason);
            return;
        }
        const pay = () => {
            setSelectedMembershipId(product.id);
            void openScanCheckout({ productId: product.id, productKind: "membership" }, brand);
        };
        if (membership.advancedRemainingSeconds > 0) {
            modal.confirm({
                title: "确认开通",
                content: "支付成功后立即按新套餐的容量和赠送积分生效，时长叠加到当前到期日。",
                onOk: () => pay(),
            });
            return;
        }
        pay();
    };

    const startTopupPayment = async (product: TopupProduct, kind: "credit_topup" | "storage_topup") => {
        const brand = defaultPaymentScanBrand(scanChannels);
        if (!brand) {
            message.error(t("wallet.noScanChannel"));
            return;
        }
        void openScanCheckout({ productId: product.id, productKind: kind }, brand);
    };

    const openScanCheckout = async (intent: { productId: string; productKind: "membership" | "credit_topup" | "storage_topup" }, brand: PaymentScanBrand) => {
        setCheckoutIntent(intent);
        setScanBrand(brand);
        setPaymentOpen(true);
        const previous = isActivePaymentOrder(paymentOrder) ? paymentOrder : null;
        if (
            previous
            && previous.productId === intent.productId
            && previous.productKind === intent.productKind
            && paymentScanBrand(previous.providerId) === brand
            && previous.checkout.mode === "qr_code"
            && previous.checkout.value
        ) {
            setPaymentOrder(previous);
            return;
        }
        if (!previous) setPaymentOrder(null);
        await createScanOrder(intent, brand, previous);
    };

    const createScanOrder = async (
        intent: { productId: string; productKind: "membership" | "credit_topup" | "storage_topup" },
        brand: PaymentScanBrand,
        previous: PaymentOrder | null,
    ) => {
        const provider = qrScanChannels(providers)[brand];
        if (!provider) {
            message.error(t("wallet.noScanChannel"));
            return;
        }
        const generation = ++scanGeneration.current;
        setPaymentCreating(true);
        try {
            if (previous && isActivePaymentOrder(previous)) {
                try {
                    const closed = await closePaymentOrder(previous.id);
                    if (generation !== scanGeneration.current) return;
                    if (closed.order.status === "credited") {
                        setPaymentOrder(closed.order);
                        setScanBrand(paymentScanBrand(closed.order.providerId) || brand);
                        await announceWalletUpdated(closed.order.id);
                        return;
                    }
                } catch (error) {
                    if (generation !== scanGeneration.current) return;
                    setScanBrand(paymentScanBrand(previous.providerId) || brand);
                    message.error(error instanceof Error ? error.message : "关闭订单失败");
                    return;
                }
            }
            const result = await createPaymentOrder({
                productId: intent.productId,
                providerId: provider.id,
                idempotencyKey: crypto.randomUUID(),
                productKind: intent.productKind,
            });
            if (generation !== scanGeneration.current) {
                if (isActivePaymentOrder(result.order)) {
                    void closePaymentOrder(result.order.id).catch(() => undefined);
                }
                return;
            }
            setPaymentOrder(result.order);
            if (result.order.status === "credited") await announceWalletUpdated(result.order.id);
            if (result.order.checkout.mode === "redirect" && result.order.checkout.url) {
                window.location.assign(result.order.checkout.url);
            }
        } catch (error) {
            if (generation !== scanGeneration.current) return;
            message.error(error instanceof Error ? error.message : intent.productKind === "membership" ? "创建订阅订单失败" : "创建支付订单失败");
        } finally {
            if (generation === scanGeneration.current) setPaymentCreating(false);
        }
    };

    const selectScanBrand = (brand: PaymentScanBrand) => {
        if (!checkoutIntent || !scanChannels[brand]) return;
        if (brand === scanBrand && paymentOrder && paymentOrder.status !== "create_failed") return;
        setScanBrand(brand);
        void createScanOrder(checkoutIntent, brand, paymentOrder);
    };

    const resumeOpenMembershipOrder = async () => {
        const id = membership.openMembershipOrderId;
        if (!id) {
            message.error("找不到未完成的订单，请关闭后重试");
            return;
        }
        setOrderResuming(true);
        try {
            const { order } = await getPaymentOrder(id);
            const kind = order.productKind === "membership" || order.productKind === "storage_topup" || order.productKind === "credit_topup" ? order.productKind : "membership";
            const brand = paymentScanBrand(order.providerId) || defaultPaymentScanBrand(scanChannels) || "wechat";
            const intent = { productId: order.productId, productKind: kind };
            setCheckoutIntent(intent);
            setScanBrand(brand);
            setPaymentOpen(true);
            if (order.status === "pending" && order.checkout.mode === "qr_code" && order.checkout.value) {
                setPaymentOrder(order);
                return;
            }
            if (order.status === "credited") {
                setPaymentOrder(order);
                await announceWalletUpdated(order.id);
                return;
            }
            setPaymentOrder(order);
            if (isActivePaymentOrder(order)) {
                await createScanOrder(intent, brand, order);
            }
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取支付订单失败");
        } finally {
            setOrderResuming(false);
        }
    };

    const cancelOpenMembershipOrder = async () => {
        const id = membership.openMembershipOrderId || (paymentOrder?.productKind === "membership" ? paymentOrder.id : "");
        if (!id) {
            message.error("没有可关闭的订单");
            return;
        }
        setOrderCancelling(true);
        try {
            const closed = await closePaymentOrder(id);
            if (closed.order.status === "credited") {
                setPaymentOrder(closed.order);
                setPaymentOpen(true);
                await announceWalletUpdated(closed.order.id);
                message.success(creditedMessage(closed.order));
                return;
            }
            setPaymentOpen(false);
            setPaymentOrder(null);
            setCheckoutIntent(null);
            await announceWalletUpdated();
            message.success("订单已关闭，未产生入账");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "关闭订单失败");
        } finally {
            setOrderCancelling(false);
        }
    };

    async function refreshPaymentStatus(orderId = paymentOrder?.id, silent = false) {
        if (!orderId) return;
        if (silent) {
            if (silentQuerying.current) return;
            silentQuerying.current = true;
        }
        const generation = scanGeneration.current;
        try {
            const result = await queryPaymentOrder(orderId);
            if (generation !== scanGeneration.current) return;
            setPaymentOrder(result.order);
            if (result.order.status === "credited") {
                await announceWalletUpdated(result.order.id);
                if (!silent) message.success(creditedMessage(result.order));
            }
        } catch (error) {
            if (!silent) message.error(error instanceof Error ? error.message : "查询支付结果失败");
        } finally {
            if (silent) silentQuerying.current = false;
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

    const payButton = (onPay: () => void, disabled: boolean, label: string) => (
        <div className="workspace-wallet-provider-row">
            <Button type="primary" size="large" disabled={disabled || !canScanPay || !membership.onlinePaymentEnabled} onClick={onPay}>{label}</Button>
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
                width="min(1080px, calc(100vw - 28px))"
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
                            <section className="workspace-wallet-section">
                                {paymentsLoading ? <Skeleton active paragraph={{ rows: 6 }} /> : membershipProducts.length ? <>
                                    {membership.hasOpenMembershipOrder ? (
                                        <div className="workspace-wallet-inline-state">
                                            <CircleAlert />
                                            <div><strong>{t("wallet.openOrder")}</strong><span>{t("wallet.openOrderLead")}</span></div>
                                            <div className="workspace-wallet-open-order-actions">
                                                <Button loading={orderResuming} disabled={orderCancelling} onClick={() => void resumeOpenMembershipOrder()}>{t("wallet.continuePay")}</Button>
                                                <Button danger loading={orderCancelling} disabled={orderResuming} onClick={() => void cancelOpenMembershipOrder()}>{t("wallet.closeOrder")}</Button>
                                            </div>
                                        </div>
                                    ) : null}
                                    <div className="workspace-wallet-period">
                                        <div className="workspace-wallet-period-switch" role="tablist" aria-label={t("wallet.period")}>
                                            {MEMBERSHIP_PERIODS.map((item) => {
                                                const vipMonth = groupedPlans.vip.find((product) => product.durationDays === 30);
                                                const svipMonth = groupedPlans.svip.find((product) => product.durationDays === 30);
                                                const vipPeriod = groupedPlans.vip.find((product) => product.durationDays === item.days);
                                                const svipPeriod = groupedPlans.svip.find((product) => product.durationDays === item.days);
                                                const vipSave = vipMonth && vipPeriod ? periodSavingsPercent(vipMonth.amountFen, vipPeriod.amountFen, item.days) : 0;
                                                const svipSave = svipMonth && svipPeriod ? periodSavingsPercent(svipMonth.amountFen, svipPeriod.amountFen, item.days) : 0;
                                                const shownSave = Math.max(vipSave, svipSave);
                                                return (
                                                    <button key={item.id} type="button" role="tab" aria-selected={period === item.id} className={period === item.id ? "is-selected" : ""} onClick={() => {
                                                        setPeriod(item.id);
                                                        const next = productForPeriod(groupedPlans.svip, item.id) || productForPeriod(groupedPlans.vip, item.id);
                                                        if (next) setSelectedMembershipId(next.id);
                                                    }}>
                                                        {t(`sku.period.${item.days}`)}
                                                        {shownSave > 0 ? <small>{t("wallet.savePercent", { value: shownSave })}</small> : null}
                                                    </button>
                                                );
                                            })}
                                        </div>
                                    </div>
                                    <div className="workspace-wallet-plan-grid">
                                        <WalletPlanCard
                                            tier="free"
                                            title={membership.freeShowcase?.title || t("wallet.freeTitle")}
                                            price="¥0"
                                            unit={t("wallet.freeUnit")}
                                            gift={membership.freeShowcase?.description || t("wallet.freeGift", { size: formatMembershipStorage(membership.defaultStoredFileBytes || 0) })}
                                            specs={[
                                                { label: t("wallet.specEntry"), value: membership.freeShowcase?.entryLabel || t("wallet.specDash") },
                                                { label: t("wallet.specStorage"), value: formatMembershipStorage(membership.defaultStoredFileBytes || 0) },
                                                { label: t("wallet.specGift"), value: t("wallet.specDash") },
                                                { label: t("wallet.specAddOn"), value: membership.freeShowcase?.addOnLabel || t("wallet.specDash") },
                                                { label: t("wallet.specAudience"), value: membership.freeShowcase?.audience || t("wallet.specDash") },
                                            ]}
                                            features={membership.freeShowcase?.featureLines || []}
                                            action={membershipStatusKey(membership) === "none" ? t("wallet.currentPlan") : t("wallet.freeTitle")}
                                            disabled
                                            current={membershipStatusKey(membership) === "none"}
                                        />
                                        {(["vip", "svip"] as const).map((tier) => {
                                            const items = groupedPlans[tier];
                                            const selected = productForPeriod(items, period);
                                            if (!selected) return null;
                                            const blocked = membershipPurchaseBlocked(membership, selected.sku);
                                            const unpriced = selected.amountFen <= 0;
                                            const isCurrent = (membership.tier || membershipSKUTier(membership.advancedPlanSku)) === tier && membershipSKUDurationDays(membership.advancedPlanSku) === selected.durationDays && membership.advancedRemainingSeconds > 0;
                                            const daily = dailyPriceYuan(selected.amountFen, selected.durationDays);
                                            const action = isCurrent ? (blocked.disabled ? t("wallet.currentPlan") : t("wallet.renew")) : t(tier === "svip" ? "wallet.buySvip" : "wallet.buyVip");
                                            return (
                                                <WalletPlanCard
                                                    key={tier}
                                                    tier={tier}
                                                    title={selected.name}
                                                    badge={selected.badge}
                                                    price={selected.amountFen > 0 ? `¥${Math.round(selected.amountFen / 100)}` : t("wallet.unpriced")}
                                                    origin={selected.originalAmountFen && selected.originalAmountFen > selected.amountFen ? `¥${Math.round(selected.originalAmountFen / 100)}` : undefined}
                                                    unit={`${t(`sku.period.${selected.durationDays}`)}${daily ? ` · ${t("wallet.perDay", { value: daily })}` : ""}`}
                                                    gift={t("wallet.giftBar", { credits: formatCredits(selected.creditsMicrocredits, 0), size: formatMembershipStorage(selected.storageQuotaBytes) })}
                                                    specs={[
                                                        { label: t("wallet.specEntry"), value: selected.entryLabel || t("wallet.specDash") },
                                                        { label: t("wallet.specStorage"), value: formatMembershipStorage(selected.storageQuotaBytes) },
                                                        { label: t("wallet.specGift"), value: formatCredits(selected.creditsMicrocredits, 0) },
                                                        { label: t("wallet.specAddOn"), value: selected.addOnLabel || t("wallet.specDash") },
                                                        { label: t("wallet.specAudience"), value: selected.audience || t("wallet.specDash") },
                                                    ]}
                                                    features={selected.featureLines || []}
                                                    action={action}
                                                    disabled={blocked.disabled || unpriced || !membership.onlinePaymentEnabled || !canScanPay}
                                                    reason={blocked.disabled ? (blocked.reasonKey ? t(blocked.reasonKey) : blocked.reason) : ""}
                                                    current={isCurrent}
                                                    highlighted={Boolean(selected.highlighted)}
                                                    onSelect={() => setSelectedMembershipId(selected.id)}
                                                    onBuy={() => { setSelectedMembershipId(selected.id); void startMembershipPayment(selected); }}
                                                />
                                            );
                                        })}
                                    </div>
                                    <p className="workspace-wallet-display-only">{t("wallet.displayOnly")}</p>
                                </> : <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.unlisted")}</strong><span>{t("wallet.unlistedLead")}</span></div></div>}
                            </section>
                        </div>
                    ) : null}

                    {tab === "storage" ? (
                        <div className="workspace-wallet-content is-topup">
                            <section className="workspace-wallet-section">
                                <div className="workspace-wallet-section-heading"><div><h3>{t("wallet.storage")}</h3><p>{personalStorage ? t("wallet.storagePersonalLead") : t("wallet.storageLead", { size: formatMembershipStorage(membership.effectiveStoredFileBytes) })}{!personalStorage && membership.storageBonusBytes ? t("wallet.storageBonus", { size: formatMembershipStorage(membership.storageBonusBytes) }) : ""}{!personalStorage && membership.storageBonusExpiresAt ? t("wallet.storageBonusExpiry", { date: new Date(membership.storageBonusExpiresAt).toLocaleDateString() }) : ""}</p></div><HardDrive /></div>
                                {personalStorage ? <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.personalActive")}</strong><span>{t("wallet.storagePersonalHint")}</span></div><Button onClick={() => { onClose(); navigate("/settings"); }}>{t("wallet.manageStorage")}</Button></div> : null}
                                {paymentsLoading ? <Skeleton active paragraph={{ rows: 4 }} /> : storageProducts.length ? <>
                                    <div className="workspace-wallet-products">
                                        {storageProducts.map((product) => (
                                            <button key={product.id} type="button" className={cn("workspace-wallet-product", selectedStorageId === product.id && "is-selected")} aria-pressed={selectedStorageId === product.id} onClick={() => setSelectedStorageId(product.id)}>
                                                <span>{product.name}</span>
                                                <strong>{formatMembershipStorage(product.storageBytes || 0)}</strong>
                                                <small>¥ {(product.amountFen / 100).toFixed(2)} · {t("wallet.storageExpiry", { duration: (product.durationDays || 365) === 365 ? t("wallet.durationYear") : t("wallet.durationDays", { count: product.durationDays || 365 }) })}{product.badge ? ` · ${product.badge}` : ""}</small>
                                                {selectedStorageId === product.id ? <Check /> : null}
                                            </button>
                                        ))}
                                    </div>
                                    {payButton(() => { if (selectedStorage) void startTopupPayment(selectedStorage, "storage_topup"); }, !selectedStorage, t("wallet.buyStorage"))}
                                </> : <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.storageUnlisted")}</strong><span>{t("wallet.storageUnlistedLead")}</span></div></div>}
                            </section>
                            {redeemBlock}
                        </div>
                    ) : null}

                    {tab === "credits" && creditsEnabled ? (
                        <div className="workspace-wallet-content is-topup">
                            <section className="workspace-wallet-section">
                                <div className="workspace-wallet-section-heading"><div><h3>{t("wallet.online")}</h3><p>{t("wallet.onlineLead")}</p></div><CreditCard /></div>
                                {paymentsLoading ? <Skeleton active paragraph={{ rows: 4 }} /> : creditProducts.length ? <>
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
                                    {payButton(() => { if (selectedCredit) void startTopupPayment(selectedCredit, "credit_topup"); }, !selectedCredit, t("wallet.payNow"))}
                                </> : <div className="workspace-wallet-inline-state"><CircleAlert /><div><strong>{t("wallet.onlineOff")}</strong><span>{t("wallet.onlineOffLead")}</span></div></div>}
                            </section>
                            {redeemBlock}
                        </div>
                    ) : null}

                    {tab === "history" ? (
                        <div className="workspace-wallet-content is-history">
                            <div className="workspace-wallet-history-toolbar"><div><h3>{t("wallet.historyTitle")}</h3><p>{t("wallet.historyLead")}</p></div><Button type="text" icon={<RefreshCw />} loading={walletLoading} onClick={() => void reloadWallet(page)}>{t("wallet.refresh")}</Button></div>
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

            <AppModal
                open={paymentOpen}
                title={paymentOrder?.status === "credited" ? creditedTitle(paymentOrder) : (checkoutIntent?.productKind === "membership" || paymentOrder?.productKind === "membership") ? t("wallet.scanMembership") : t("wallet.scanPay")}
                centered
                width={430}
                zIndex={1200}
                footer={null}
                onCancel={() => setPaymentOpen(false)}
            >
                <div className="workspace-wallet-payment">
                    {paymentOrder ? <p className="workspace-wallet-payment-product">{paymentOrderCaption(paymentOrder)}</p> : null}
                    {paymentOrder && paymentOrder.status === "pending" ? (
                        <p className="workspace-wallet-payment-countdown">
                            <PaymentScanBrandMark brand={scanBrand} />
                            {t("wallet.scanWithin", { time: formatScanCountdown(paymentOrder.expiresAt, clock) })}
                        </p>
                    ) : null}
                    {paymentOrder ? <strong>¥ {(paymentOrder.amountFen / 100).toFixed(2)}</strong> : paymentCreating ? <strong>¥ --</strong> : null}
                    {paymentCreating ? (
                        <div className="workspace-wallet-scan-qr-loading" aria-hidden />
                    ) : paymentOrder?.status === "pending" && paymentOrder.checkout.mode === "qr_code" && paymentOrder.checkout.value ? (
                        <PaymentCheckoutCode value={paymentOrder.checkout.value} brand={scanBrand} />
                    ) : paymentOrder?.status === "credited" || paymentOrder?.status === "closed" || paymentOrder?.status === "create_failed" ? (
                        <PaymentStatus order={paymentOrder} />
                    ) : (
                        <div className="workspace-wallet-scan-qr-loading" aria-hidden />
                    )}
                    {scanBrandPillsVisible(paymentOrder) && (scanChannels.wechat || scanChannels.alipay) ? (
                        <div className="workspace-wallet-scan-channels" role="radiogroup" aria-label={t("wallet.payMethod")}>
                            {scanChannels.wechat ? (
                                <button type="button" role="radio" aria-checked={scanBrand === "wechat"} className={scanBrand === "wechat" ? "is-selected" : ""} onClick={() => selectScanBrand("wechat")}>
                                    <PaymentScanBrandMark brand="wechat" />
                                    {t("wallet.payWechat")}
                                </button>
                            ) : null}
                            {scanChannels.alipay ? (
                                <button type="button" role="radio" aria-checked={scanBrand === "alipay"} className={scanBrand === "alipay" ? "is-selected" : ""} onClick={() => selectScanBrand("alipay")}>
                                    <PaymentScanBrandMark brand="alipay" />
                                    {t("wallet.payAlipay")}
                                </button>
                            ) : null}
                        </div>
                    ) : null}
                </div>
            </AppModal>
        </>
    );
}

function WalletPlanCard({
    tier,
    title,
    badge,
    price,
    origin,
    unit,
    gift,
    specs,
    features,
    action,
    disabled,
    loading,
    reason,
    current,
    highlighted,
    onSelect,
    onBuy,
}: {
    tier: "free" | "vip" | "svip";
    title: string;
    badge?: string;
    price: string;
    origin?: string;
    unit: string;
    gift: string;
    specs: { label: string; value: string }[];
    features: { text: string; included: boolean }[];
    action: string;
    disabled?: boolean;
    loading?: boolean;
    reason?: string;
    current?: boolean;
    highlighted?: boolean;
    onSelect?: () => void;
    onBuy?: () => void;
}) {
    return (
        <article className={cn("workspace-wallet-plan-card", `is-${tier}`, highlighted && "is-highlighted", current && "is-current")} onClick={onSelect}>
            <header>
                <strong>{title}</strong>
                {badge ? <em>{badge}</em> : null}
            </header>
            <p className="workspace-wallet-plan-price">
                <b>{price}</b>
                {origin ? <s>{origin}</s> : null}
            </p>
            <p className="workspace-wallet-plan-unit">{unit}</p>
            <p className="workspace-wallet-plan-gift">{gift}</p>
            <div className="workspace-wallet-plan-specs">
                {specs.map((row) => (
                    <div key={row.label} className="workspace-wallet-plan-spec">
                        <span>{row.label}</span>
                        <b>{row.value}</b>
                    </div>
                ))}
            </div>
            <ul className="workspace-wallet-plan-features">
                {features.map((item) => (
                    <li key={item.text} className={item.included ? undefined : "is-off"}>
                        {item.included ? <Check aria-hidden="true" /> : <Minus aria-hidden="true" />}
                        {item.text}
                    </li>
                ))}
            </ul>
            {reason ? <small>{reason}</small> : null}
            <Button type={tier === "free" ? "default" : "primary"} className={cn(tier === "svip" && "is-svip", tier === "vip" && "is-vip")} loading={loading} disabled={disabled} onClick={(event) => { event.stopPropagation(); onBuy?.(); }}>{action}</Button>
        </article>
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

function PaymentStatus({ order }: { order: PaymentOrder }) {
    if (order.status === "credited") return <div className="workspace-wallet-payment-status is-success">{creditedMessage(order)}</div>;
    if (order.status === "closed") return <div className="workspace-wallet-payment-status">订单已关闭，未产生入账</div>;
    if (order.status === "create_failed") return <div className="workspace-wallet-payment-status is-error">支付入口创建失败，请改用其他方式或关闭窗口后重试。</div>;
    return null;
}

function formatScanCountdown(expiresAt: string, now: number) {
    const remaining = Math.max(0, Math.floor((new Date(expiresAt).getTime() - now) / 1000));
    const hours = Math.floor(remaining / 3600);
    const minutes = Math.floor((remaining % 3600) / 60);
    const seconds = remaining % 60;
    if (hours > 0) return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
    return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function scanBrandPillsVisible(order: PaymentOrder | null) {
    return !order || order.status === "pending" || order.status === "created" || order.status === "create_failed";
}

function isActivePaymentOrder(order: PaymentOrder | null | undefined): order is PaymentOrder {
    return Boolean(order && ["created", "pending", "closing", "create_failed"].includes(order.status));
}
