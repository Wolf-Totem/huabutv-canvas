import { AlipayCircleFilled, WechatFilled } from "@ant-design/icons";
import { Callout } from "@/pages/admin/ui/controls";
import { App, Button, DatePicker, Descriptions, Drawer, Form, Input, InputNumber, Select, Tabs, Typography } from "antd";
import { AdminDrawer } from "@/pages/admin/ui/overlays";
import { Switch } from "@/pages/admin/ui/controls";
import type { ColumnsType } from "antd/es/table";
import dayjs, { type Dayjs } from "dayjs";
import { Eye, Plus, RefreshCw, Search, Settings2, Trash2, XCircle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { PaginationBar } from "@/pages/admin/components/admin-ui";
import { formatCredits } from "@/constant/credits";
import {
    closeAdminPaymentOrder,
    createAdminTopupProduct,
    listAdminPaymentOrders,
    listAdminPaymentProviders,
    listAdminPaymentReconciliationItems,
    listAdminPaymentReconciliations,
    listAdminTopupProducts,
    queryAdminPaymentOrder,
    runAdminPaymentReconciliation,
    updateAdminPaymentProvider,
    updateAdminTopupProduct,
    type AdminPaymentProvider,
    type AdminPaymentOrder,
    type PaymentOrder,
    type PaymentReconciliationItem,
    type PaymentReconciliationRun,
    type TopupProduct,
} from "@/services/api/payments";
import { formatStorageDuration, membershipSKUTier } from "@/lib/membership";
import { getAdminCommerceMethods, getAdminMembershipFreeShowcase, listAdminMembershipProducts, updateAdminCommerceMethods, updateAdminMembershipFreeShowcase, updateAdminMembershipProduct, type CommerceMethods, type MembershipProduct, type MembershipShowcase } from "@/services/api/membership";

import { AdminPageFrame } from "../components/admin-shell";
import { AdminDataTable, AdminRowActions, AdminStatusBadge, AdminTableEmpty, configuredSecretText } from "../components/admin-ui";
import { AdminUserDetailDrawer } from "../components/admin-user-detail-drawer";
import "./payments-page.css";

type ProviderFormValues = {
    enabled: boolean;
    closeAfterMinutes: number;
    values: Record<string, string>;
};

type ProductFormValues = {
    name: string;
    description?: string;
    amountYuan: number;
    credits: number;
    kind: "credit_topup" | "storage_topup";
    storageGiB: number;
    durationDays?: number;
    badge?: string;
    enabled: boolean;
    sortOrder: number;
};

type MembershipFeatureFormLine = { text?: string; included?: boolean };

type MembershipFormValues = {
    name: string;
    description?: string;
    amountYuan: number;
    originalAmountYuan?: number;
    credits: number;
    storageGiB: number;
    badge?: string;
    highlighted: boolean;
    enabled: boolean;
    sortOrder: number;
    entryLabel?: string;
    audience?: string;
    addOnLabel?: string;
    featureLines?: MembershipFeatureFormLine[];
    syncShowcaseToTier?: boolean;
};

type FreeShowcaseFormValues = {
    title?: string;
    description?: string;
    entryLabel?: string;
    audience?: string;
    addOnLabel?: string;
    featureLines?: MembershipFeatureFormLine[];
};

const paymentOrderStatus: Record<string, { label: string; tone: "neutral" | "success" | "warning" | "error" | "info" }> = {
    created: { label: "创建中", tone: "info" },
    pending: { label: "待支付", tone: "warning" },
    closing: { label: "关单中", tone: "warning" },
    closed: { label: "已关闭", tone: "neutral" },
    credited: { label: "已入账", tone: "success" },
    create_failed: { label: "下单失败", tone: "error" },
};

const reconciliationResult: Record<string, { label: string; tone: "neutral" | "success" | "warning" | "error" | "info" }> = {
    matched: { label: "一致", tone: "success" },
    recovered: { label: "已自动补发", tone: "info" },
    local_order_not_found: { label: "本地订单缺失", tone: "error" },
    provider_record_missing: { label: "渠道记录缺失", tone: "error" },
    amount_mismatch: { label: "金额不一致", tone: "error" },
    trade_no_mismatch: { label: "交易号不一致", tone: "error" },
    credit_failed: { label: "补发失败", tone: "error" },
};

export default function AdminPaymentsPage() {
    const { message, modal } = App.useApp();
    const [activeTab, setActiveTab] = useState("providers");
    const [providers, setProviders] = useState<AdminPaymentProvider[]>([]);
    const [products, setProducts] = useState<TopupProduct[]>([]);
    const [membershipProducts, setMembershipProducts] = useState<MembershipProduct[]>([]);
    const [commerceMethods, setCommerceMethods] = useState<CommerceMethods>({ onlinePaymentEnabled: true, redeemEnabled: true });
    const [loading, setLoading] = useState(true);

    const [providerDrawer, setProviderDrawer] = useState<AdminPaymentProvider>();
    const [providerSaving, setProviderSaving] = useState(false);
    const [providerForm] = Form.useForm<ProviderFormValues>();

    const [productDrawer, setProductDrawer] = useState<TopupProduct | null | undefined>();
    const [productSaving, setProductSaving] = useState(false);
    const [productForm] = Form.useForm<ProductFormValues>();
    const productKind = Form.useWatch("kind", productForm) || "credit_topup";
    const [membershipDrawer, setMembershipDrawer] = useState<MembershipProduct | null>(null);
    const [membershipSaving, setMembershipSaving] = useState(false);
    const [membershipForm] = Form.useForm<MembershipFormValues>();
    const [freeShowcase, setFreeShowcase] = useState<MembershipShowcase>({});
    const [freeDrawerOpen, setFreeDrawerOpen] = useState(false);
    const [freeSaving, setFreeSaving] = useState(false);
    const [freeForm] = Form.useForm<FreeShowcaseFormValues>();

    const [orders, setOrders] = useState<AdminPaymentOrder[]>([]);
    const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
    const [selectedOrder, setSelectedOrder] = useState<AdminPaymentOrder | null>(null);
    const [orderTotal, setOrderTotal] = useState(0);
    const [orderPage, setOrderPage] = useState(1);
    const [orderPageSize, setOrderPageSize] = useState(30);
    const [orderStatusFilter, setOrderStatusFilter] = useState("all");
    const [orderKeyword, setOrderKeyword] = useState("");
    const [ordersLoading, setOrdersLoading] = useState(false);
    const [orderActionId, setOrderActionId] = useState("");

    const [runs, setRuns] = useState<PaymentReconciliationRun[]>([]);
    const [runTotal, setRunTotal] = useState(0);
    const [runPage, setRunPage] = useState(1);
    const [runPageSize, setRunPageSize] = useState(30);
    const [runsLoading, setRunsLoading] = useState(false);
    const [runProviderFilter, setRunProviderFilter] = useState("all");
    const [runningBill, setRunningBill] = useState(false);
    const [billProviderId, setBillProviderId] = useState("");
    const [billDate, setBillDate] = useState<Dayjs>(dayjs().subtract(1, "day"));
    const [detailRun, setDetailRun] = useState<PaymentReconciliationRun>();
    const [detailItems, setDetailItems] = useState<PaymentReconciliationItem[]>([]);
    const [detailTotal, setDetailTotal] = useState(0);
    const [detailPage, setDetailPage] = useState(1);
    const [detailPageSize, setDetailPageSize] = useState(50);
    const [detailResult, setDetailResult] = useState("all");
    const [detailLoading, setDetailLoading] = useState(false);

    const loadBase = async () => {
        setLoading(true);
        try {
            const [providerResult, productResult, membershipResult, methodsResult, freeResult] = await Promise.all([listAdminPaymentProviders(), listAdminTopupProducts(), listAdminMembershipProducts(), getAdminCommerceMethods(), getAdminMembershipFreeShowcase()]);
            setProviders(providerResult.providers);
            setProducts(productResult.products);
            setMembershipProducts(membershipResult.products);
            setCommerceMethods(methodsResult);
            setFreeShowcase(freeResult);
            setBillProviderId((current) => current || providerResult.providers.find((item) => item.configured)?.id || providerResult.providers[0]?.id || "");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取支付配置失败");
        } finally {
            setLoading(false);
        }
    };

    const loadOrders = async (page = orderPage, pageSize = orderPageSize) => {
        setOrdersLoading(true);
        try {
            const result = await listAdminPaymentOrders({ status: orderStatusFilter === "all" ? undefined : orderStatusFilter, keyword: orderKeyword.trim() || undefined, page, pageSize });
            setOrders(result.orders);
            setOrderTotal(result.total);
            setOrderPage(result.page);
            setOrderPageSize(result.pageSize);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取支付订单失败");
        } finally {
            setOrdersLoading(false);
        }
    };

    const loadRuns = async (page = runPage, pageSize = runPageSize) => {
        setRunsLoading(true);
        try {
            const result = await listAdminPaymentReconciliations({ providerId: runProviderFilter === "all" ? undefined : runProviderFilter, page, pageSize });
            setRuns(result.runs);
            setRunTotal(result.total);
            setRunPage(result.page);
            setRunPageSize(result.pageSize);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取对账记录失败");
        } finally {
            setRunsLoading(false);
        }
    };

    useEffect(() => {
        void Promise.all([loadBase(), loadOrders(1, orderPageSize), loadRuns(1, runPageSize)]);
    }, []);

    const refresh = async () => {
        if (activeTab === "orders") await loadOrders();
        else if (activeTab === "reconciliation") await loadRuns();
        else await loadBase();
    };

    const openProvider = (provider: AdminPaymentProvider) => {
        providerForm.resetFields();
        const configValues = { ...(provider.values || {}) };
        for (const field of provider.configFields) {
            if (!configValues[field.name] && field.default !== undefined && field.default !== null) configValues[field.name] = String(field.default);
        }
        providerForm.setFieldsValue({ enabled: provider.configEnabled, closeAfterMinutes: provider.closeAfterMinutes || 30, values: configValues });
        setProviderDrawer(provider);
    };

    const saveProvider = async () => {
        if (!providerDrawer) return;
        const values = await providerForm.validateFields();
        setProviderSaving(true);
        try {
            await updateAdminPaymentProvider(providerDrawer.id, {
                enabled: values.enabled,
                closeAfterMinutes: values.closeAfterMinutes,
                values: Object.fromEntries(Object.entries(values.values || {}).map(([key, value]) => [key, String(value || "")])),
            });
            message.success(`${providerDrawer.name}配置已保存为新版本`);
            setProviderDrawer(undefined);
            await loadBase();
        } catch (error) {
            message.error(error instanceof Error ? error.message : "保存支付渠道失败");
        } finally {
            setProviderSaving(false);
        }
    };

    const openProduct = (product?: TopupProduct, kind: "credit_topup" | "storage_topup" = "credit_topup") => {
        productForm.resetFields();
        const nextKind = (product?.kind === "storage_topup" ? "storage_topup" : kind) as ProductFormValues["kind"];
        productForm.setFieldsValue(
            product
                ? {
                      name: product.name,
                      description: product.description,
                      amountYuan: product.amountFen / 100,
                      credits: product.creditsMicrocredits / 1_000_000,
                      kind: nextKind,
                      storageGiB: product.storageBytes ? Math.round(product.storageBytes / 1024 ** 3) : 20,
                      durationDays: nextKind === "storage_topup" ? product.durationDays || 365 : 0,
                      badge: product.badge,
                      enabled: product.enabled,
                      sortOrder: product.sortOrder,
                  }
                : { enabled: true, sortOrder: products.length * 10, amountYuan: 10, credits: nextKind === "credit_topup" ? 10 : 0, kind: nextKind, storageGiB: 20, durationDays: nextKind === "storage_topup" ? 365 : 0 },
        );
        setProductDrawer(product || null);
    };

    const saveProduct = async () => {
        if (productDrawer === undefined) return;
        const values = await productForm.validateFields();
        const kind = values.kind || "credit_topup";
        const input = {
            name: values.name.trim(),
            description: values.description?.trim(),
            amountFen: Math.round(values.amountYuan * 100),
            creditsMicrocredits: Math.round((values.credits || 0) * 1_000_000),
            kind,
            storageBytes: kind === "storage_topup" ? Math.round((values.storageGiB || 0) * 1024 ** 3) : 0,
            durationDays: kind === "storage_topup" ? Math.round(Number(values.durationDays || 365)) : 0,
            badge: values.badge?.trim(),
            enabled: values.enabled,
            sortOrder: values.sortOrder || 0,
        };
        setProductSaving(true);
        try {
            if (productDrawer) await updateAdminTopupProduct(productDrawer.id, input);
            else await createAdminTopupProduct(input);
            message.success(productDrawer ? "充值商品已更新" : "充值商品已创建");
            setProductDrawer(undefined);
            await loadBase();
        } catch (error) {
            message.error(error instanceof Error ? error.message : "保存充值商品失败");
        } finally {
            setProductSaving(false);
        }
    };

    const queryOrder = async (order: PaymentOrder) => {
        setOrderActionId(order.id);
        try {
            await queryAdminPaymentOrder(order.id);
            message.success("已向支付渠道查单");
            await loadOrders();
        } catch (error) {
            message.error(error instanceof Error ? error.message : "查单失败");
        } finally {
            setOrderActionId("");
        }
    };

    const closeOrder = (order: PaymentOrder) => {
        modal.confirm({
            title: "关闭未支付订单？",
            content: "系统会先向支付渠道查单；若渠道已支付则立即入账，否则执行关单。",
            okText: "查单并关单",
            cancelText: "取消",
            onOk: async () => {
                setOrderActionId(order.id);
                try {
                    await closeAdminPaymentOrder(order.id);
                    message.success("订单状态已更新");
                    await loadOrders();
                } catch (error) {
                    message.error(error instanceof Error ? error.message : "关单失败");
                    throw error;
                } finally {
                    setOrderActionId("");
                }
            },
        });
    };

    const runReconciliation = async () => {
        if (!billProviderId || !billDate) return;
        setRunningBill(true);
        try {
            const result = await runAdminPaymentReconciliation({ providerId: billProviderId, billDate: billDate.format("YYYY-MM-DD") });
            if (result.run.status === "running") {
                message.info("该渠道与账单日期的对账正在执行，请稍后刷新查看结果");
            } else {
                message.success(result.run.recoveredItems ? `对账完成，自动补发 ${result.run.recoveredItems} 笔` : "对账完成");
            }
            await loadRuns(1, runPageSize);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "执行对账失败");
            await loadRuns(1, runPageSize);
        } finally {
            setRunningBill(false);
        }
    };

    const openRunDetails = async (run: PaymentReconciliationRun, page = 1, pageSize = detailPageSize, result = detailResult) => {
        setDetailRun(run);
        setDetailLoading(true);
        try {
            const response = await listAdminPaymentReconciliationItems(run.id, { result: result === "all" ? undefined : result, page, pageSize });
            setDetailRun(response.run);
            setDetailItems(response.items);
            setDetailTotal(response.total);
            setDetailPage(response.page);
            setDetailPageSize(response.pageSize);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取对账明细失败");
        } finally {
            setDetailLoading(false);
        }
    };

    const providerNames = useMemo(() => Object.fromEntries(providers.map((item) => [item.id, item.name])), [providers]);

    const providerColumns: ColumnsType<AdminPaymentProvider> = [
        {
            title: "支付渠道",
            key: "provider",
            render: (_, provider) => (
                <div className="flex items-center gap-3">
                    <PaymentBrandIcon providerId={provider.id} />
                    <div>
                        <div className="font-medium">{provider.name}</div>
                        <div className="mt-0.5 font-mono text-xs text-foreground/45">{provider.id}</div>
                    </div>
                </div>
            ),
        },
        { title: "支付方式", dataIndex: "checkoutMode", width: 120, align: "center", render: (value) => (value === "qr_code" ? "扫码支付" : "网站跳转") },
        {
            title: "状态",
            key: "status",
            width: 180,
            align: "center",
            render: (_, provider) => (
                <div className="flex flex-col items-center gap-1.5">
                    <AdminStatusBadge label={provider.enabled ? "可用" : "不可用"} tone={provider.enabled ? "success" : "neutral"} />
                    <span className="text-xs text-foreground/45">
                        插件{provider.pluginEnabled ? "已开放" : "已停用"} · 配置{provider.configEnabled ? "已启用" : "已停用"}
                    </span>
                </div>
            ),
        },
        { title: "未支付自动关闭", dataIndex: "closeAfterMinutes", width: 150, align: "center", render: (value) => `${value || 30} 分钟` },
        { title: "配置版本", dataIndex: "version", width: 105, align: "center", render: (value) => (value ? `v${value}` : "未配置") },
        {
            title: "操作",
            key: "actions",
            width: 100,
            align: "center",
            render: (_, provider) => (
                <Button size="small" icon={<Settings2 className="size-3.5" />} onClick={() => openProvider(provider)}>
                    配置
                </Button>
            ),
        },
    ];

    const creditProducts = products.filter((item) => item.kind !== "storage_topup");
    const storageTopupProducts = products.filter((item) => item.kind === "storage_topup");

    const productColumns = (kind: "credit_topup" | "storage_topup"): ColumnsType<TopupProduct> => [
        {
            title: "商品",
            key: "name",
            render: (_, product) => (
                <div>
                    <div className="font-medium">{product.name}</div>
                    <div className="mt-0.5 text-xs text-foreground/45">{product.description || "无说明"}</div>
                </div>
            ),
        },
        { title: "售价", dataIndex: "amountFen", width: 130, align: "right", render: (value) => <span className="font-medium tabular-nums">¥ {(value / 100).toFixed(2)}</span> },
        kind === "storage_topup"
            ? { title: "加购容量", dataIndex: "storageBytes", width: 130, align: "right", render: (value: number) => <span className="tabular-nums">{value ? `${Math.round(value / 1024 ** 3)} GiB` : "--"}</span> }
            : { title: "到账积分", dataIndex: "creditsMicrocredits", width: 150, align: "right", render: (value) => <span className="tabular-nums">{formatCredits(value)}</span> },
        ...(kind === "storage_topup"
            ? [{ title: "有效期", dataIndex: "durationDays" as const, width: 110, align: "center" as const, render: (value: number) => formatStorageDuration(value || 365) }]
            : []),
        { title: "排序", dataIndex: "sortOrder", width: 90, align: "center" },
        { title: "状态", dataIndex: "enabled", width: 100, align: "center", render: (value) => <AdminStatusBadge label={value ? "销售中" : "已停用"} tone={value ? "success" : "neutral"} /> },
        {
            title: "操作",
            key: "actions",
            width: 90,
            align: "center",
            render: (_, product) => (
                <Button size="small" onClick={() => openProduct(product, kind)}>
                    编辑
                </Button>
            ),
        },
    ];

    const orderColumns: ColumnsType<AdminPaymentOrder> = [
        {
            title: "用户",
            key: "user",
            width: 230,
            render: (_, order) => (
                <div className="min-w-0">
                    {order.user ? <>
                        <div className="flex min-w-0 items-baseline gap-2">
                            <button type="button" className="admin-table-primary-link truncate font-medium" title={order.user.displayName || order.user.username} onClick={() => setSelectedUserId(order.user!.id)}>{order.user.displayName || order.user.username}</button>
                            <span className="truncate text-xs text-foreground/45" title={`@${order.user.username}`}>@{order.user.username}</span>
                        </div>
                        <div className="mt-1 truncate text-xs text-foreground/60" title={order.user.email}>{order.user.email || "未填写邮箱"}</div>
                    </> : <>
                        <div className="text-foreground/60">用户不存在</div>
                        <Typography.Text className="text-xs" copyable={order.userId ? { text: order.userId } : false}>{order.userId ? `${order.userId.slice(0, 8)}…${order.userId.slice(-6)}` : "--"}</Typography.Text>
                    </>}
                </div>
            ),
        },
        {
            title: "订单 / 商品",
            key: "order",
            width: 210,
            render: (_, order) => (
                <div>
                    <Typography.Text className="font-mono text-xs" title={order.merchantOrderNo} copyable={{ text: order.merchantOrderNo }}>{order.merchantOrderNo.length > 20 ? `${order.merchantOrderNo.slice(0, 10)}…${order.merchantOrderNo.slice(-6)}` : order.merchantOrderNo}</Typography.Text>
                    <div className="mt-1 truncate text-xs text-foreground/45" title={order.productName}>
                        {order.productName}
                    </div>
                </div>
            ),
        },
        {
            title: "渠道",
            dataIndex: "providerId",
            width: 150,
            render: (value) => (
                <span className="inline-flex items-center gap-2">
                    <PaymentBrandIcon providerId={value} compact />
                    {providerNames[value] || value}
                </span>
            ),
        },
        {
            title: "金额 / 积分",
            key: "amount",
            width: 130,
            align: "right",
            render: (_, order) => (
                <div>
                    <div className="font-medium tabular-nums">¥ {(order.amountFen / 100).toFixed(2)}</div>
                    <div className="text-xs text-foreground/45">{formatCredits(order.creditsMicrocredits)} 积分</div>
                </div>
            ),
        },
        { title: "状态", dataIndex: "status", width: 110, align: "center", render: (value) => <AdminStatusBadge {...(paymentOrderStatus[value] || { label: value, tone: "neutral" as const })} /> },
        { title: "创建时间", dataIndex: "createdAt", width: 130, render: (value) => <span title={formatDateTime(value)}>{dayjs(value).format("MM-DD HH:mm")}</span> },
        {
            title: "操作",
            key: "actions",
            width: 150,
            fixed: "right",
            align: "center",
            render: (_, order) => (
                <AdminRowActions
                    primary={{ label: "详情", icon: <Eye className="size-3.5" />, onClick: () => setSelectedOrder(order) }}
                    visibleActionCount={0}
                    actions={[
                        { key: "sync", label: "同步支付状态", icon: <RefreshCw className="size-3.5" />, disabled: Boolean(orderActionId) || ["credited", "closed"].includes(order.status), onClick: () => queryOrder(order) },
                        { key: "close", label: "关闭订单", icon: <XCircle className="size-3.5" />, danger: true, disabled: Boolean(orderActionId) || !["created", "pending", "create_failed", "closing"].includes(order.status), onClick: () => closeOrder(order) },
                    ]}
                />
            ),
        },
    ];

    const runColumns: ColumnsType<PaymentReconciliationRun> = [
        { title: "账单日期", dataIndex: "billDate", width: 120 },
        {
            title: "渠道",
            dataIndex: "providerId",
            width: 190,
            render: (value) => (
                <span className="inline-flex items-center gap-2">
                    <PaymentBrandIcon providerId={value} compact />
                    {providerNames[value] || value}
                </span>
            ),
        },
        {
            title: "状态",
            dataIndex: "status",
            width: 110,
            align: "center",
            render: (value) => <AdminStatusBadge label={value === "completed" ? "已完成" : value === "running" ? "执行中" : "失败"} tone={value === "completed" ? "success" : value === "running" ? "info" : "error"} />,
        },
        { title: "一致", dataIndex: "matchItems", width: 80, align: "right" },
        { title: "自动补发", dataIndex: "recoveredItems", width: 100, align: "right", render: (value) => <span className={value ? "font-medium text-status-success" : ""}>{value}</span> },
        { title: "异常", dataIndex: "errorItems", width: 80, align: "right", render: (value) => <span className={value ? "font-medium text-status-error" : ""}>{value}</span> },
        { title: "完成时间", dataIndex: "completedAt", width: 170, render: (value) => (value ? formatDateTime(value) : "--") },
        {
            title: "明细",
            key: "actions",
            width: 90,
            align: "center",
            render: (_, run) => (
                <Button type="text" size="small" icon={<Eye className="size-3.5" />} onClick={() => void openRunDetails(run)}>
                    查看
                </Button>
            ),
        },
    ];

    const detailColumns: ColumnsType<PaymentReconciliationItem> = [
        { title: "结果", dataIndex: "result", width: 135, render: (value) => <AdminStatusBadge {...(reconciliationResult[value] || { label: value, tone: "neutral" as const })} /> },
        { title: "商户订单号", dataIndex: "merchantOrderNo", width: 255, render: (value) => <span className="font-mono text-xs">{value}</span> },
        { title: "渠道交易号", dataIndex: "providerTradeNo", width: 220, render: (value) => (value ? <span className="font-mono text-xs">{value}</span> : "--") },
        { title: "金额", dataIndex: "amountFen", width: 110, align: "right", render: (value, item) => `${item.currency} ${(value / 100).toFixed(2)}` },
        { title: "说明", dataIndex: "detail", render: (value) => value || "账单与本地订单一致" },
    ];

    return (
        <AdminPageFrame
            title="支付充值"
            description="管理系统支付适配器、充值商品、支付订单与 T+1 对账"
            actions={
                <Button icon={<RefreshCw className="size-4" />} loading={loading || ordersLoading || runsLoading} onClick={() => void refresh()}>
                    刷新
                </Button>
            }
            scroll
        >
            <Callout className="my-4" tone="info" title="平台不提供支付退款">
                管理端仅提供查单、关单和对账。关单前始终先向渠道查单；对账发现已支付未入账订单时会幂等补发积分。
            </Callout>
            <Tabs
                activeKey={activeTab}
                onChange={setActiveTab}
                items={[
                    {
                        key: "providers",
                        label: "支付渠道",
                        children: <AdminDataTable table={{ rowKey: "id", loading, columns: providerColumns, dataSource: providers, pagination: false, scroll: { x: 980 } }} empty={<AdminTableEmpty title="没有发现支付渠道插件" />} />,
                    },
                    {
                        key: "membership",
                        label: "订阅商品",
                        children: (
                            <div className="space-y-4">
                                <div className="flex flex-wrap items-center gap-4 rounded-md border border-border p-3">
                                    <span className="text-sm">在线支付</span>
                                    <Switch checked={commerceMethods.onlinePaymentEnabled} onChange={(onlinePaymentEnabled) => {
                                        void updateAdminCommerceMethods({ ...commerceMethods, onlinePaymentEnabled }).then(setCommerceMethods).catch((error) => message.error(error instanceof Error ? error.message : "保存失败"));
                                    }} />
                                    <span className="text-sm">兑换码</span>
                                    <Switch checked={commerceMethods.redeemEnabled} onChange={(redeemEnabled) => {
                                        void updateAdminCommerceMethods({ ...commerceMethods, redeemEnabled }).then(setCommerceMethods).catch((error) => message.error(error instanceof Error ? error.message : "保存失败"));
                                    }} />
                                    <Button size="small" onClick={() => {
                                        freeForm.setFieldsValue({
                                            title: freeShowcase.title || "免费使用",
                                            description: freeShowcase.description,
                                            entryLabel: freeShowcase.entryLabel,
                                            audience: freeShowcase.audience,
                                            addOnLabel: freeShowcase.addOnLabel,
                                            featureLines: (freeShowcase.featureLines || []).map((item) => ({ text: item.text, included: item.included })),
                                        });
                                        setFreeDrawerOpen(true);
                                    }}>编辑未开通列</Button>
                                    <span className="text-xs text-foreground/45">未开通不是第七种会员，只改钱包免费列展示。云存储那一行仍读平台默认配额。</span>
                                </div>
                                <AdminDataTable
                                    table={{
                                        rowKey: "id",
                                        loading,
                                        pagination: false,
                                        dataSource: membershipProducts,
                                        columns: [
                                            { title: "SKU", dataIndex: "sku", width: 140 },
                                            { title: "档位", dataIndex: "tier", width: 80, render: (_: string, product: MembershipProduct) => (membershipSKUTier(product.tier || product.sku) || "--").toUpperCase() },
                                            { title: "名称", dataIndex: "name" },
                                            { title: "售价", dataIndex: "amountFen", width: 110, render: (value: number) => value > 0 ? `¥ ${(value / 100).toFixed(2)}` : "未定价" },
                                            { title: "赠送积分", dataIndex: "creditsMicrocredits", width: 110, render: (value: number) => formatCredits(value, 0) },
                                            { title: "套餐容量", dataIndex: "storageQuotaBytes", width: 110, render: (value: number) => value ? `${Math.round(value / 1024 ** 3)} GiB` : "默认" },
                                            { title: "状态", dataIndex: "enabled", width: 90, render: (value: boolean) => <AdminStatusBadge label={value ? "销售中" : "未上架"} tone={value ? "success" : "neutral"} /> },
                                            { title: "操作", key: "actions", width: 90, render: (_: unknown, product: MembershipProduct) => (
                                                <Button size="small" onClick={() => {
                                                    setMembershipDrawer(product);
                                                    membershipForm.setFieldsValue({
                                                        name: product.name,
                                                        description: product.description,
                                                        amountYuan: product.amountFen / 100,
                                                        originalAmountYuan: (product.originalAmountFen || 0) / 100,
                                                        credits: product.creditsMicrocredits / 1_000_000,
                                                        storageGiB: product.storageQuotaBytes ? Math.round(product.storageQuotaBytes / 1024 ** 3) : 0,
                                                        badge: product.badge,
                                                        highlighted: Boolean(product.highlighted),
                                                        enabled: product.enabled,
                                                        sortOrder: product.sortOrder,
                                                        entryLabel: product.entryLabel,
                                                        audience: product.audience,
                                                        addOnLabel: product.addOnLabel,
                                                        featureLines: (product.featureLines || []).map((item) => ({ text: item.text, included: item.included })),
                                                        syncShowcaseToTier: false,
                                                    });
                                                }}>编辑</Button>
                                            ) },
                                        ],
                                    }}
                                    empty={<AdminTableEmpty title="还没有订阅商品" />}
                                />
                            </div>
                        ),
                    },
                    {
                        key: "products",
                        label: "积分商品",
                        children: (
                            <AdminDataTable
                                toolbar={<span />}
                                trailing={
                                    <Button type="primary" className="admin-toolbar-primary-action" icon={<Plus className="size-4" />} onClick={() => openProduct(undefined, "credit_topup")}>
                                        新增积分商品
                                    </Button>
                                }
                                table={{ rowKey: "id", loading, columns: productColumns("credit_topup"), dataSource: creditProducts, pagination: false, scroll: { x: 820 } }}
                                empty={<AdminTableEmpty title="还没有积分商品" />}
                            />
                        ),
                    },
                    {
                        key: "storage",
                        label: "容量商品",
                        children: (
                            <AdminDataTable
                                toolbar={<span />}
                                trailing={
                                    <Button type="primary" className="admin-toolbar-primary-action" icon={<Plus className="size-4" />} onClick={() => openProduct(undefined, "storage_topup")}>
                                        新增容量商品
                                    </Button>
                                }
                                table={{ rowKey: "id", loading, columns: productColumns("storage_topup"), dataSource: storageTopupProducts, pagination: false, scroll: { x: 820 } }}
                                empty={<AdminTableEmpty title="还没有容量商品" />}
                            />
                        ),
                    },
                    {
                        key: "orders",
                        label: "支付订单",
                        children: (
                            <AdminDataTable
                                toolbar={
                                    <Input
                                        className="app-list-search"
                                        allowClear
                                        prefix={<Search className="size-4 text-foreground/40" />}
                                        value={orderKeyword}
                                        placeholder="搜索名称、用户名、邮箱、订单号"
                                        title="支持名称、用户名、邮箱、订单号、渠道交易号和完整用户 ID"
                                        onChange={(event) => setOrderKeyword(event.target.value)}
                                        onPressEnter={() => void loadOrders(1)}
                                    />
                                }
                                toolbarFilters={
                                    <Select
                                        className="w-36"
                                        value={orderStatusFilter}
                                        onChange={setOrderStatusFilter}
                                        options={[{ value: "all", label: "全部状态" }, ...Object.entries(paymentOrderStatus).map(([value, item]) => ({ value, label: item.label }))]}
                                    />
                                }
                                trailing={<Button onClick={() => void loadOrders(1)}>查询</Button>}
                                table={{ rowKey: "id", loading: ordersLoading, columns: orderColumns, dataSource: orders, pagination: false, tableLayout: "fixed", scroll: { x: 1110 } }}
                                empty={<AdminTableEmpty filtered={Boolean(orderKeyword || orderStatusFilter !== "all")} title="没有支付订单" />}
                                footer={<PaginationBar alwaysShow current={orderPage} pageSize={orderPageSize} total={orderTotal} onChange={(page, size) => void loadOrders(size !== orderPageSize ? 1 : page, size)} />}
                            />
                        ),
                    },
                    {
                        key: "reconciliation",
                        label: "支付对账",
                        children: (
                            <div className="space-y-4">
                                <div className="flex flex-wrap items-center gap-2 rounded-lg border border-border/70 bg-card p-3">
                                    <Select
                                        className="min-w-52"
                                        value={billProviderId || undefined}
                                        placeholder="选择支付渠道"
                                        onChange={setBillProviderId}
                                        options={providers.map((provider) => ({ value: provider.id, label: provider.name, disabled: !provider.configured }))}
                                    />
                                    <DatePicker value={billDate} allowClear={false} disabledDate={(date) => !date.isBefore(dayjs(), "day") || date.isBefore(dayjs().subtract(3, "month"), "day")} onChange={(date) => date && setBillDate(date)} />
                                    <Button type="primary" loading={runningBill} disabled={!billProviderId} onClick={() => void runReconciliation()}>
                                        执行对账
                                    </Button>
                                    <span className="text-xs text-foreground/45">系统每天 10:15 后自动对账昨日账单；也可在此手动重跑最近三个月账单。</span>
                                </div>
                                <AdminDataTable
                                    toolbar={
                                        <Select
                                            className="w-52"
                                            value={runProviderFilter}
                                            onChange={(value) => {
                                                setRunProviderFilter(value);
                                                setRunPage(1);
                                            }}
                                            options={[{ value: "all", label: "全部支付渠道" }, ...providers.map((provider) => ({ value: provider.id, label: provider.name }))]}
                                        />
                                    }
                                    trailing={<Button onClick={() => void loadRuns(1)}>筛选</Button>}
                                    table={{ rowKey: "id", loading: runsLoading, columns: runColumns, dataSource: runs, pagination: false, scroll: { x: 1000 } }}
                                    empty={<AdminTableEmpty title="还没有对账记录" />}
                                    footer={<PaginationBar alwaysShow current={runPage} pageSize={runPageSize} total={runTotal} onChange={(page, size) => void loadRuns(size !== runPageSize ? 1 : page, size)} />}
                                />
                            </div>
                        ),
                    },
                ]}
            />

            <AdminDrawer title="支付订单详情" size="min(680px, 100vw)" open={Boolean(selectedOrder)} onClose={() => setSelectedOrder(null)}>
                {selectedOrder && <Descriptions column={1} bordered size="small" items={[
                    { key: "user", label: "用户", children: selectedOrder.user ? <button type="button" className="admin-table-primary-link" onClick={() => { setSelectedUserId(selectedOrder.user!.id); setSelectedOrder(null); }}>{selectedOrder.user.displayName || selectedOrder.user.username} · @{selectedOrder.user.username}</button> : "用户不存在" },
                    { key: "email", label: "邮箱", children: selectedOrder.user?.email || "未填写邮箱" },
                    { key: "userId", label: "用户 ID", children: <Typography.Text copyable className="break-all">{selectedOrder.userId || "--"}</Typography.Text> },
                    { key: "order", label: "订单号", children: <Typography.Text copyable className="break-all">{selectedOrder.merchantOrderNo}</Typography.Text> },
                    { key: "trade", label: "渠道交易号", children: selectedOrder.providerTradeNo ? <Typography.Text copyable className="break-all">{selectedOrder.providerTradeNo}</Typography.Text> : "--" },
                    { key: "product", label: "商品", children: selectedOrder.productName },
                    { key: "channel", label: "支付渠道", children: providerNames[selectedOrder.providerId] || selectedOrder.providerId },
                    { key: "amount", label: "金额 / 积分", children: `¥ ${(selectedOrder.amountFen / 100).toFixed(2)} / ${formatCredits(selectedOrder.creditsMicrocredits)} 积分` },
                    { key: "status", label: "状态", children: paymentOrderStatus[selectedOrder.status]?.label || selectedOrder.status },
                    ...([{ key: "createdAt", label: "创建时间" }, { key: "expiresAt", label: "过期时间" }, { key: "providerPaidAt", label: "支付时间" }, { key: "creditedAt", label: "入账时间" }, { key: "closedAt", label: "关闭时间" }] as const).map(({ key, label }) => ({ key, label, children: selectedOrder[key] ? formatDateTime(selectedOrder[key]!) : "--" })),
                ]} />}
            </AdminDrawer>
            <AdminUserDetailDrawer userId={selectedUserId} onClose={() => setSelectedUserId(null)} />

            <Drawer
                title={providerDrawer ? `配置 ${providerDrawer.name}` : "配置支付渠道"}
                width={620}
                open={Boolean(providerDrawer)}
                destroyOnHidden
                onClose={() => setProviderDrawer(undefined)}
                extra={
                    <Button type="primary" loading={providerSaving} onClick={() => void saveProvider()}>
                        保存新版本
                    </Button>
                }
            >
                {providerDrawer ? (
                    <Form form={providerForm} layout="vertical" requiredMark="optional">
                        <Callout
                            className="mb-4"
                            tone={providerDrawer.pluginEnabled ? "info" : "warning"}
                            title={providerDrawer.pluginEnabled ? "密钥会加密保存，历史订单固定使用创建时的配置版本。" : "该宿主插件当前已在插件管理中停用；保存配置后仍需开放插件才能接受新订单。"}
                        />
                        <Form.Item name="enabled" label="渠道配置启用" valuePropName="checked">
                            <Switch />
                        </Form.Item>
                        <Form.Item name="closeAfterMinutes" label="未支付订单自动关闭时间（分钟）" rules={[{ required: true }, { type: "number", min: 5, max: 1440 }]}>
                            <InputNumber min={5} max={1440} precision={0} className="w-full" />
                        </Form.Item>
                        {providerDrawer.configFields.map((field) => {
                            const secretReady = Boolean(providerDrawer.secretConfigured[field.name]);
                            const rules = field.required && !secretReady ? [{ required: true, message: `请输入${field.label || field.name}` }] : undefined;
                            const placeholder = field.secret && secretReady ? configuredSecretText : field.description || undefined;
                            const input =
                                field.type === "textarea" ? (
                                    <Input.TextArea autoSize={{ minRows: 4, maxRows: 10 }} placeholder={placeholder} autoComplete="off" />
                                ) : field.type === "password" || field.secret ? (
                                    <Input.Password placeholder={placeholder} autoComplete="new-password" />
                                ) : (
                                    <Input placeholder={placeholder} />
                                );
                            return (
                                <Form.Item key={field.name} name={["values", field.name]} label={field.label || field.name} extra={field.description} rules={rules}>
                                    {input}
                                </Form.Item>
                            );
                        })}
                    </Form>
                ) : null}
            </Drawer>

            <Drawer
                title={membershipDrawer ? `编辑订阅 · ${membershipDrawer.sku}` : "编辑订阅"}
                width={640}
                open={Boolean(membershipDrawer)}
                destroyOnHidden
                onClose={() => setMembershipDrawer(null)}
                extra={
                    <Button type="primary" loading={membershipSaving} onClick={() => {
                        if (!membershipDrawer) return;
                        void membershipForm.validateFields().then((values) => {
                            setMembershipSaving(true);
                            return updateAdminMembershipProduct(membershipDrawer.id, {
                                name: values.name,
                                description: values.description,
                                amountFen: Math.round(Number(values.amountYuan) * 100),
                                originalAmountFen: Math.round(Number(values.originalAmountYuan || 0) * 100),
                                creditsMicrocredits: Math.round(Number(values.credits || 0) * 1_000_000),
                                storageQuotaBytes: Math.round(Number(values.storageGiB || 0) * 1024 ** 3),
                                badge: values.badge,
                                highlighted: values.highlighted,
                                enabled: values.enabled,
                                sortOrder: values.sortOrder,
                                entryLabel: values.entryLabel,
                                audience: values.audience,
                                addOnLabel: values.addOnLabel,
                                featureLines: normalizeFeatureLines(values.featureLines),
                                syncShowcaseToTier: Boolean(values.syncShowcaseToTier),
                            }).then(async (result) => {
                                if (values.syncShowcaseToTier) await loadBase();
                                else setMembershipProducts((current) => current.map((item) => item.id === result.product.id ? result.product : item));
                                setMembershipDrawer(null);
                                message.success(values.syncShowcaseToTier ? "已更新订阅商品，并同步到同档其他周期" : "已更新订阅商品");
                            });
                        }).catch((error) => {
                            if (error instanceof Error) message.error(error.message);
                        }).finally(() => setMembershipSaving(false));
                    }}>
                        保存
                    </Button>
                }
            >
                <Callout className="mb-4" tone="info" title="价/积分/容量进新订单快照">
                    改价、赠送积分和套餐容量只影响之后的新订单。功能清单、入口文案、适合谁和加购文案改完立刻出现在钱包，不写入订单。SKU、档位和时长不可改。清单只展示，不会锁定入口。
                </Callout>
                <Form form={membershipForm} layout="vertical">
                    <Form.Item name="name" label="名称" rules={[{ required: true, max: 120 }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name="description" label="说明" rules={[{ max: 500 }]}>
                        <Input.TextArea rows={3} />
                    </Form.Item>
                    <div className="grid grid-cols-2 gap-3">
                        <Form.Item name="amountYuan" label="售价（元，0 表示未定价）" rules={[{ required: true, type: "number", min: 0 }]}>
                            <InputNumber min={0} precision={2} className="w-full" />
                        </Form.Item>
                        <Form.Item name="originalAmountYuan" label="划线价（元，可选）" rules={[{ type: "number", min: 0 }]}>
                            <InputNumber min={0} precision={2} className="w-full" />
                        </Form.Item>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                        <Form.Item name="credits" label="开通赠送积分" rules={[{ required: true, type: "number", min: 0 }]}>
                            <InputNumber min={0} precision={0} className="w-full" />
                        </Form.Item>
                        <Form.Item name="storageGiB" label="套餐容量（GiB）" rules={[{ required: true, type: "number", min: 0, max: 3072 }]}>
                            <InputNumber min={0} max={3072} precision={0} className="w-full" />
                        </Form.Item>
                    </div>
                    <Form.Item name="badge" label="角标" rules={[{ max: 40 }]}>
                        <Input placeholder="例如：最受欢迎" />
                    </Form.Item>
                    <Form.Item name="sortOrder" label="排序" rules={[{ required: true }]}>
                        <InputNumber precision={0} className="w-full" />
                    </Form.Item>
                    <div className="grid grid-cols-2 gap-3">
                        <Form.Item name="highlighted" label="强调展示" valuePropName="checked">
                            <Switch />
                        </Form.Item>
                        <Form.Item name="enabled" label="上架销售" valuePropName="checked">
                            <Switch />
                        </Form.Item>
                    </div>
                    <Form.Item name="entryLabel" label="平台功能入口" extra="对照表那一行，最多 40 字。" rules={[{ max: 40 }]}>
                        <Input placeholder="例如：创作工作台" />
                    </Form.Item>
                    <Form.Item name="audience" label="适合谁" extra="对照表那一行，最多 40 字。" rules={[{ max: 40 }]}>
                        <Input placeholder="例如：个人稳定产出" />
                    </Form.Item>
                    <Form.Item name="addOnLabel" label="容量加购文案" extra="对照表那一行，最多 40 字。" rules={[{ max: 40 }]}>
                        <Input placeholder="例如：可叠加" />
                    </Form.Item>
                    <MembershipFeatureLinesFields />
                    <Form.Item name="syncShowcaseToTier" valuePropName="checked" extra="只拷贝入口文案、适合谁、加购文案和功能清单，不同步价格和积分。">
                        <Switch checkedChildren="同步到同档其他周期" unCheckedChildren="仅保存当前卡" />
                    </Form.Item>
                </Form>
            </Drawer>

            <Drawer
                title="编辑未开通列"
                width={640}
                open={freeDrawerOpen}
                destroyOnHidden
                onClose={() => setFreeDrawerOpen(false)}
                extra={
                    <Button type="primary" loading={freeSaving} onClick={() => {
                        void freeForm.validateFields().then((values) => {
                            setFreeSaving(true);
                            return updateAdminMembershipFreeShowcase({
                                title: values.title,
                                description: values.description,
                                entryLabel: values.entryLabel,
                                audience: values.audience,
                                addOnLabel: values.addOnLabel,
                                featureLines: normalizeFeatureLines(values.featureLines),
                            }).then((result) => {
                                setFreeShowcase(result);
                                setFreeDrawerOpen(false);
                                message.success("未开通列已更新");
                            });
                        }).catch((error) => {
                            if (error instanceof Error) message.error(error.message);
                        }).finally(() => setFreeSaving(false));
                    }}>
                        保存
                    </Button>
                }
            >
                <Callout className="mb-4" tone="info" title="这不是第七种会员">
                    未开通列不能购买。云存储那一行仍读运行策略默认配额，不在这里填 GiB。改完立刻出现在钱包。
                </Callout>
                <Form form={freeForm} layout="vertical">
                    <Form.Item name="title" label="标题" rules={[{ required: true, max: 40 }]}>
                        <Input placeholder="免费使用" />
                    </Form.Item>
                    <Form.Item name="description" label="副文案" rules={[{ max: 120 }]}>
                        <Input.TextArea rows={3} />
                    </Form.Item>
                    <Form.Item name="entryLabel" label="平台功能入口" rules={[{ max: 40 }]}>
                        <Input placeholder="例如：开放（基础）" />
                    </Form.Item>
                    <Form.Item name="audience" label="适合谁" rules={[{ max: 40 }]}>
                        <Input placeholder="例如：试用与轻量创作" />
                    </Form.Item>
                    <Form.Item name="addOnLabel" label="容量加购文案" rules={[{ max: 40 }]}>
                        <Input placeholder="例如：可买，不加会员" />
                    </Form.Item>
                    <MembershipFeatureLinesFields />
                </Form>
            </Drawer>

            <Drawer
                title={productDrawer ? "编辑商品" : "新增商品"}
                width={520}
                open={productDrawer !== undefined}
                destroyOnHidden
                onClose={() => setProductDrawer(undefined)}
                extra={
                    <Button type="primary" loading={productSaving} onClick={() => void saveProduct()}>
                        保存
                    </Button>
                }
            >
                <Form form={productForm} layout="vertical" requiredMark="optional">
                    <Form.Item name="kind" hidden>
                        <Input />
                    </Form.Item>
                    <Form.Item name="name" label="商品名称" rules={[{ required: true, max: 120 }]}>
                        <Input placeholder={productKind === "storage_topup" ? "例如：+50 GB 容量" : "例如：100 积分"} />
                    </Form.Item>
                    <Form.Item name="description" label="商品说明" rules={[{ max: 500 }]}>
                        <Input.TextArea rows={3} />
                    </Form.Item>
                    <div className="grid grid-cols-2 gap-3">
                        <Form.Item name="amountYuan" label="售价（元）" rules={[{ required: true }, { type: "number", min: 0.01, max: 1_000_000 }]}>
                            <InputNumber min={0.01} max={1_000_000} precision={2} className="w-full" />
                        </Form.Item>
                        {productKind === "storage_topup" ? (
                            <Form.Item name="storageGiB" label="加购容量（GiB）" rules={[{ required: true }, { type: "number", min: 1, max: 3072 }]}>
                                <InputNumber min={1} max={3072} precision={0} className="w-full" />
                            </Form.Item>
                        ) : (
                            <Form.Item
                                name="credits"
                                label="到账积分"
                                rules={[
                                    { required: true },
                                    {
                                        validator: (_, value) => {
                                            const credits = Number(value);
                                            const microcredits = Math.round(credits * 1_000_000);
                                            return Number.isFinite(credits) && credits > 0 && credits <= 1_000_000_000 && Number.isSafeInteger(microcredits) ? Promise.resolve() : Promise.reject(new Error("请输入 0.000001 至 10 亿之间且可安全处理的积分"));
                                        },
                                    },
                                ]}
                            >
                                <InputNumber min={0.000001} max={1_000_000_000} precision={6} className="w-full" />
                            </Form.Item>
                        )}
                    </div>
                    {productKind === "storage_topup" ? (
                        <Form.Item
                            name="durationDays"
                            label="有效天数"
                            extra="默认 365 天。到期后该笔加购从配额拿掉，不删文件。范围 1–3650。"
                            rules={[{ required: true }, { type: "number", min: 1, max: 3650 }]}
                        >
                            <InputNumber min={1} max={3650} precision={0} className="w-full" addonAfter="天" />
                        </Form.Item>
                    ) : null}
                    <Form.Item name="badge" label="角标" rules={[{ max: 40 }]}>
                        <Input placeholder="可选" />
                    </Form.Item>
                    <Form.Item name="sortOrder" label="排序" rules={[{ required: true }]}>
                        <InputNumber precision={0} className="w-full" />
                    </Form.Item>
                    <Form.Item name="enabled" label="上架销售" valuePropName="checked">
                        <Switch />
                    </Form.Item>
                </Form>
            </Drawer>

            <Drawer
                title={detailRun ? `${providerNames[detailRun.providerId] || detailRun.providerId} · ${detailRun.billDate} 对账明细` : "对账明细"}
                width="min(1080px, 94vw)"
                open={Boolean(detailRun)}
                destroyOnHidden
                onClose={() => {
                    setDetailRun(undefined);
                    setDetailItems([]);
                }}
            >
                {detailRun?.error ? (
                    <Callout className="mb-4" tone="error" title="对账执行失败">
                        {detailRun.error}
                    </Callout>
                ) : null}
                <AdminDataTable
                    toolbar={
                        <Select
                            className="w-44"
                            value={detailResult}
                            onChange={(value) => {
                                setDetailResult(value);
                                if (detailRun) void openRunDetails(detailRun, 1, detailPageSize, value);
                            }}
                            options={[{ value: "all", label: "全部结果" }, ...Object.entries(reconciliationResult).map(([value, item]) => ({ value, label: item.label }))]}
                        />
                    }
                    table={{ rowKey: "id", loading: detailLoading, columns: detailColumns, dataSource: detailItems, pagination: false, scroll: { x: 950 } }}
                    empty={<AdminTableEmpty title={detailRun?.status === "failed" ? "本次对账未生成明细" : "账单没有交易记录"} />}
                    footer={<PaginationBar alwaysShow current={detailPage} pageSize={detailPageSize} total={detailTotal} onChange={(page, size) => detailRun && void openRunDetails(detailRun, size !== detailPageSize ? 1 : page, size, detailResult)} />}
                />
            </Drawer>
        </AdminPageFrame>
    );
}

function PaymentBrandIcon({ providerId, compact = false }: { providerId: string; compact?: boolean }) {
    const size = compact ? "size-6" : "size-10";
    if (providerId === "wechat-native")
        return (
            <span className={`grid ${size} shrink-0 place-items-center rounded-lg bg-[#07c160]/10 text-[#07c160]`}>
                <WechatFilled className={compact ? "text-sm" : "text-xl"} aria-hidden />
            </span>
        );
    if (providerId === "alipay-page-pay")
        return (
            <span className={`grid ${size} shrink-0 place-items-center rounded-lg bg-[#1677ff]/10 text-[#1677ff]`}>
                <AlipayCircleFilled className={compact ? "text-sm" : "text-xl"} aria-hidden />
            </span>
        );
    return <span className={`grid ${size} shrink-0 place-items-center rounded-lg bg-muted text-xs`}>PAY</span>;
}

function formatDateTime(value: string) {
    return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

function normalizeFeatureLines(lines?: MembershipFeatureFormLine[]) {
    return (lines || [])
        .map((item) => ({ text: (item.text || "").trim(), included: Boolean(item.included) }))
        .filter((item) => item.text)
        .slice(0, 12);
}

function MembershipFeatureLinesFields() {
    return (
        <Form.List name="featureLines">
            {(fields, { add, remove }) => (
                <div className="admin-membership-feature-list">
                    <div className="mb-2 flex items-start justify-between gap-3">
                        <div>
                            <div className="text-sm font-medium">功能清单</div>
                            <p className="mt-0.5 text-xs text-foreground/45">最多 12 条，每条 40 字。含/不含只影响钱包展示，不会锁定入口。</p>
                        </div>
                        <Button type="text" size="small" icon={<Plus className="size-3.5" />} disabled={fields.length >= 12} onClick={() => add({ text: "", included: true })}>
                            添加
                        </Button>
                    </div>
                    {fields.length ? fields.map((field, index) => (
                        <div className="admin-membership-feature-row" key={field.key}>
                            <Form.Item name={[field.name, "included"]} valuePropName="checked">
                                <Switch size="sm" checkedChildren="含" unCheckedChildren="不含" aria-label={`第 ${index + 1} 条是否包含`} />
                            </Form.Item>
                            <Form.Item name={[field.name, "text"]} rules={[{ required: true, message: "请填写清单文案" }, { max: 40, message: "每条不超过 40 字" }]}>
                                <Input maxLength={40} placeholder="例如：短剧工作台" aria-label={`第 ${index + 1} 条文案`} />
                            </Form.Item>
                            <Button type="text" danger className="admin-membership-feature-remove" icon={<Trash2 className="size-4" />} aria-label={`删除第 ${index + 1} 条`} onClick={() => remove(field.name)} />
                        </div>
                    )) : <div className="admin-membership-feature-empty">还没有清单条目。前台该列会显示为空。</div>}
                </div>
            )}
        </Form.List>
    );
}
