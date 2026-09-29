import { AlipayCircleFilled, WechatFilled } from "@ant-design/icons";

import type { PaymentScanBrand } from "@/lib/payment-brands";
import { cn } from "@/lib/utils";

export function PaymentScanBrandMark({ brand, className }: { brand: PaymentScanBrand; className?: string }) {
    const Icon = brand === "wechat" ? WechatFilled : AlipayCircleFilled;
    return <Icon className={cn(brand === "wechat" ? "text-[#07c160]" : "text-[#1677ff]", className)} aria-hidden />;
}
