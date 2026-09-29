import assert from "node:assert/strict";
import test from "node:test";

import { defaultPaymentScanBrand, hasPaymentScanChannel, paymentScanBrand, qrScanChannels } from "./payment-brands";

const wechatHuifu = { id: "huifu-wechat-native", checkoutMode: "qr_code" };
const wechatDirect = { id: "wechat-native", checkoutMode: "qr_code" };
const alipayHuifu = { id: "huifu-aggregate-native", checkoutMode: "qr_code" };
const alipayPage = { id: "alipay-page-pay", checkoutMode: "redirect" };
const huifuH5 = { id: "huifu-h5-cashier", checkoutMode: "redirect" };

test("paymentScanBrand 把斗拱正扫和直连微信映射成用户品牌", () => {
    assert.equal(paymentScanBrand("huifu-wechat-native"), "wechat");
    assert.equal(paymentScanBrand("wechat-native"), "wechat");
    assert.equal(paymentScanBrand("huifu-aggregate-native"), "alipay");
    assert.equal(paymentScanBrand("alipay-page-pay"), "alipay");
    assert.equal(paymentScanBrand("huifu-h5-cashier"), null);
});

test("qrScanChannels 只收正扫二维码，优先斗拱，忽略跳转收银台", () => {
    const channels = qrScanChannels([huifuH5, alipayPage, wechatDirect, wechatHuifu, alipayHuifu]);
    assert.equal(channels.wechat?.id, "huifu-wechat-native");
    assert.equal(channels.alipay?.id, "huifu-aggregate-native");
});

test("没有斗拱微信时退到直连微信扫码", () => {
    const channels = qrScanChannels([wechatDirect, alipayPage]);
    assert.equal(channels.wechat?.id, "wechat-native");
    assert.equal(channels.alipay, undefined);
});

test("默认先微信，没有微信再用支付宝", () => {
    assert.equal(defaultPaymentScanBrand(qrScanChannels([alipayHuifu, wechatHuifu])), "wechat");
    assert.equal(defaultPaymentScanBrand(qrScanChannels([alipayHuifu])), "alipay");
    assert.equal(defaultPaymentScanBrand(qrScanChannels([huifuH5, alipayPage])), null);
    assert.equal(hasPaymentScanChannel([huifuH5, alipayPage]), false);
    assert.equal(hasPaymentScanChannel([alipayHuifu]), true);
});
