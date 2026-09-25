import { describe, expect, test } from "bun:test";

import { generationUsesUserChannel, userChannelDisconnectNotice } from "../src/lib/generation-user-channel";
import { defaultConfig, type AiConfig } from "../src/stores/use-config-store";

function userConfig(): AiConfig {
    return {
        ...defaultConfig,
        channels: [
            {
                id: "user-1",
                name: "我的渠道",
                baseUrl: "https://example.com/v1",
                apiKey: "sk-test",
                apiFormat: "openai",
                models: ["demo"],
                scope: "user",
                enabled: true,
            },
        ],
        model: "user-1::demo",
    };
}

describe("user channel generation notices", () => {
    test("detects user-owned channels", () => {
        expect(generationUsesUserChannel(userConfig())).toBe(true);
        expect(generationUsesUserChannel(defaultConfig)).toBe(false);
    });

    test("warns not to close the browser for user-channel video", () => {
        const notice = userChannelDisconnectNotice(userConfig(), "user-1::demo", "video");
        expect(notice).toContain("请勿关闭浏览器");
        expect(notice).toContain("同步请求中断不会自动重试");
    });
});
