## yingce.payment/v1

支持 `validate_config`、`create_order`、`query_order`、`close_order`、`verify_notification` 和 `download_trade_bill`，统一返回 JSON 响应。

斗拱聚合正扫微信渠道协议封装在本插件内，字段与官方接口页一致：

- 下单 `POST https://api.huifu.com/v3/trade/payment/jspay`（[聚合正扫](https://paas.huifu.com/partners/api/doc/smzf/api_jhzs.md)），`trade_type=T_NATIVE`（官方异步 `trans_type` 枚举为「微信正扫」），**不传 `wx_data` / `alipay_data` / `hosting_data`**
- 查询 `POST https://api.huifu.com/v3/trade/payment/scanpay/query`（[扫码交易查询](https://paas.huifu.com/partners/api/doc/smzf/api_qrpay_cx.md)）
- 关单 `POST https://api.huifu.com/v2/trade/payment/scanpay/close`（[扫码交易关单](https://paas.huifu.com/partners/api/doc/smzf/api_qrpay_jygd.md)）
- 异步通知按 [异步消息规范](https://paas.huifu.com/partners/start/ybxx/jiekouguifan_ybxx.md)：POST 表单、`resp_data` + `sign`、汇付公钥对 `resp_data` 原文验签、HTTP 200，正文为 `RECV_ORD_ID_` 加上 `req_seq_id`

`create_order` 固定 `trade_type=T_NATIVE`。收银 `mode=qr_code`，`value` 为官方返回的 `qr_code`（微信 Native 常见 `weixin://wxpay/bizpayurl?...`，也可能是 https 支付链接；宿主编码成二维码，微信扫码拉起支付）。下单同步成功以返回非空 `qr_code` 为准，此时 `trans_stat` 通常为 `P`（处理中），不能当成已支付。金额按官方「单位元、两位小数」与宿主分互转。`req_seq_id` 使用宿主商户订单号。查询需要官方 `org_req_date`，宿主只给订单号，故按请求日与前一日各查一次。已支付订单不再关单。扫码关单返回的 `trans_stat` 是关单状态，不是支付状态，关单成功不得记为已支付。聚合正扫页未定义账单下载，`download_trade_bill` 返回 not found。

配置字段：`publicBaseUrl`、`sysId`、`productId`、`huifuId`、`merchantPrivateKey`、`huifuPublicKey`、`gateway`。不需要统一收银台的 `projectId` / `projectTitle`。`gateway` 必须为 https，生产示例为 `https://api.huifu.com`。加签与官方 Go SDK `FormatSignSrcText` 一致：对 `data` 对象 JSON 做 SHA256WithRSA，响应验签使用返回报文里 `data` 字段的原始 JSON。

<!-- YINGCE_MANIFEST_CONTRACT_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "yingce.plugin/v1",
  "id": "official-payment-huifu-wechat-jspay",
  "name": "斗拱微信正扫",
  "version": "1.0.0",
  "author": "汇付斗拱",
  "description": "斗拱聚合正扫适配器。微信正扫 T_NATIVE 返回 qr_code 支付链接，由宿主展示二维码。",
  "enabled": true,
  "installable": true,
  "runtime": {
    "backend": "rpc",
    "backendEntry": "backend/provider"
  },
  "surfaces": [
    "wallet",
    "settings"
  ],
  "permissions": [
    "payment.create",
    "payment.query",
    "payment.close",
    "payment.reconcile"
  ],
  "configuration": {
    "fields": [
      {
        "name": "publicBaseUrl",
        "type": "url",
        "label": "服务器公网地址",
        "required": true,
        "description": "用于确认异步通知可达。实际 notify_url 由宿主生成，须为 http/https 且不能带查询参数。"
      },
      {
        "name": "sysId",
        "type": "string",
        "label": "系统号 (sys_id)",
        "required": true,
        "description": "渠道商填渠道商 huifu_id；直连商户填商户 huifu_id。"
      },
      {
        "name": "productId",
        "type": "string",
        "label": "产品号 (product_id)",
        "required": true,
        "description": "汇付分配的产品号，例如 YYZY。"
      },
      {
        "name": "huifuId",
        "type": "string",
        "label": "商户号 (huifu_id)",
        "required": true
      },
      {
        "name": "merchantPrivateKey",
        "type": "textarea",
        "label": "商户 RSA 私钥",
        "required": true,
        "secret": true,
        "description": "请求加签。PEM 或裸 Base64。"
      },
      {
        "name": "huifuPublicKey",
        "type": "textarea",
        "label": "汇付 RSA 公钥",
        "required": true,
        "description": "同步响应与异步通知验签。PEM 或裸 Base64。"
      },
      {
        "name": "gateway",
        "type": "url",
        "label": "支付网关",
        "required": true,
        "default": "https://api.huifu.com",
        "description": "官方生产地址 https://api.huifu.com。联调环境填写汇付提供的测试网关，必须是 https。"
      }
    ]
  },
  "contributes": {
    "paymentProviders": [
      {
        "id": "huifu-wechat-native",
        "label": "斗拱微信正扫",
        "icon": "assets/icon.svg",
        "checkoutMode": "qr_code",
        "identityFields": [
          "sysId",
          "huifuId"
        ],
        "expiryPolicy": {
          "defaultMinutes": 120,
          "minMinutes": 5,
          "maxMinutes": 1440
        },
        "notificationSuccess": {
          "status": 200,
          "contentType": "text/plain; charset=utf-8",
          "body": "RECV_ORD_ID_"
        },
        "notificationFailure": {
          "status": 400,
          "contentType": "text/plain; charset=utf-8",
          "body": "fail"
        }
      }
    ]
  },
  "documentation": "<当前插件的完整 documentation，由 README.md 与 docs/interface.md 拼接而成；为避免 JSON 递归，此处不重复展开正文。>"
}
```
<!-- YINGCE_MANIFEST_CONTRACT_END -->
