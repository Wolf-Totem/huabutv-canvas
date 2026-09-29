# 斗拱微信正扫

官方 `yingce.payment/v1` 支付插件。对接汇付斗拱「聚合正扫」接口，交易类型固定 `T_NATIVE`（微信正扫，官方异步通知枚举），**不上送 `wx_data`**。下单成功后返回 `qr_code` 支付链接（微信原生 `weixin://` 或 https 链接），由宿主编码成二维码，用户用微信扫码完成支付。加签与官方 Go SDK（`bspay-go-sdk`）一致：对 `data` 对象 JSON 做 SHA256WithRSA。构建脚本交叉编译 macOS、Linux、Windows 的 amd64/arm64，入口仍声明为 `backend/provider`。
