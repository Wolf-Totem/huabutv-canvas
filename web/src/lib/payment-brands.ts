export type PaymentScanBrand = "wechat" | "alipay";

export type PaymentScanProvider = {
    id: string;
    checkoutMode: string;
};

const WECHAT_QR_IDS = ["huifu-wechat-native", "wechat-native"] as const;
const ALIPAY_QR_IDS = ["huifu-aggregate-native"] as const;

function isQrCode(provider: PaymentScanProvider) {
    return provider.checkoutMode === "qr_code";
}

function pickProvider(providers: PaymentScanProvider[], ids: readonly string[]) {
    for (const id of ids) {
        const match = providers.find((item) => item.id === id && isQrCode(item));
        if (match) return match;
    }
    return undefined;
}

export function paymentScanBrand(providerId: string): PaymentScanBrand | null {
    const id = providerId.trim();
    if ((WECHAT_QR_IDS as readonly string[]).includes(id)) return "wechat";
    if ((ALIPAY_QR_IDS as readonly string[]).includes(id) || id === "alipay-page-pay") return "alipay";
    return null;
}

export function qrScanChannels(providers: PaymentScanProvider[]): Partial<Record<PaymentScanBrand, PaymentScanProvider>> {
    return {
        wechat: pickProvider(providers, WECHAT_QR_IDS),
        alipay: pickProvider(providers, ALIPAY_QR_IDS),
    };
}

export function defaultPaymentScanBrand(channels: Partial<Record<PaymentScanBrand, PaymentScanProvider>>): PaymentScanBrand | null {
    if (channels.wechat) return "wechat";
    if (channels.alipay) return "alipay";
    return null;
}

export function hasPaymentScanChannel(providers: PaymentScanProvider[]) {
    return defaultPaymentScanBrand(qrScanChannels(providers)) !== null;
}
