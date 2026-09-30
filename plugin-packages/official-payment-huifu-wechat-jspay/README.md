# 斗拱微信小程序

官方 `yingce.payment/v1` 支付插件。对接汇付斗拱「聚合正扫」接口，交易类型固定 `T_MINIAPP`（[微信小程序支付](https://paas.huifu.com/help/dev_guide/zf/wx/xcx.md)）。下单必填 `wx_data`：`sub_appid`（插件配置的已上线小程序 AppID）与 `sub_openid`（宿主从微信小程序 `wx.login` → `jscode2session` 取得后传入）。下单成功后返回 `pay_info`，收银 `mode=jsapi`，由小程序 `wx.requestPayment` 调起。网站钱包扫码不走此渠道。加签与官方 Go SDK（`bspay-go-sdk`）一致：对 `data` 对象 JSON 做 SHA256WithRSA。构建脚本交叉编译 macOS、Linux、Windows 的 amd64/arm64，入口仍声明为 `backend/provider`。
