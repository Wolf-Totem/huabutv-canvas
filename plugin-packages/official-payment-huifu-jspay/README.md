# 斗拱支付宝正扫

官方 `yingce.payment/v1` 支付插件。对接汇付斗拱「聚合正扫」接口，交易类型固定 `A_NATIVE`（支付宝正扫），**不上送 `alipay_data`**。下单成功后返回 `qr_code` 支付链接，由宿主编码成二维码，用户用支付宝扫码完成支付。加签与官方 Go SDK（`bspay-go-sdk`）一致：对 `data` 对象 JSON 做 SHA256WithRSA。构建脚本交叉编译 macOS、Linux、Windows 的 amd64/arm64，入口仍声明为 `backend/provider`。
