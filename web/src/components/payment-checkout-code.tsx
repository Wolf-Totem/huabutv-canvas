import { QRCode } from "antd";

import { PaymentScanBrandMark } from "@/components/payment-scan-brand";
import type { PaymentScanBrand } from "@/lib/payment-brands";
import { isPaymentQrImageURL } from "@/lib/payment-checkout-code";

import "./payment-checkout-code.css";

const QR_SIZE = 208;
const QR_DARK = "#111111";
const QR_LIGHT = "#ffffff";

export function PaymentCheckoutCode({ value, brand }: { value: string; brand?: PaymentScanBrand }) {
    return (
        <div className="payment-checkout-code">
            {isPaymentQrImageURL(value) ? (
                <img src={value} alt="支付二维码" width={QR_SIZE} height={QR_SIZE} referrerPolicy="no-referrer" />
            ) : (
                <QRCode value={value} size={QR_SIZE} bordered={false} color={QR_DARK} bgColor={QR_LIGHT} />
            )}
            {brand ? (
                <span className={`payment-checkout-code-mark is-${brand}`}>
                    <PaymentScanBrandMark brand={brand} />
                </span>
            ) : null}
        </div>
    );
}
