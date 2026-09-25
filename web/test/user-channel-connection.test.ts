import { describe, expect, test } from "bun:test";

import {
    apiFormatForUserChannelConnection,
    defaultBaseUrlForUserChannelConnection,
    isKnownDefaultBaseUrl,
    nextUserChannelName,
    userChannelConnectionMode,
    userChannelConnectionPatch,
} from "../src/lib/user-channel-connection";
import { createModelChannel, defaultBaseUrlForApiFormat, JIASU_BASE_URL, normalizeConfigSnapshot } from "../src/stores/use-config-store";

describe("user channel connection presets", () => {
    test("jiasu preset uses openai catalog format and the jiasu base url", () => {
        expect(apiFormatForUserChannelConnection("jiasu")).toBe("openai");
        expect(defaultBaseUrlForUserChannelConnection("jiasu")).toBe("https://ai.jiasuapi.com");
        expect(defaultBaseUrlForUserChannelConnection("jiasu")).toBe(JIASU_BASE_URL);
        expect(isKnownDefaultBaseUrl(JIASU_BASE_URL)).toBe(true);
        expect(isKnownDefaultBaseUrl(`${JIASU_BASE_URL}/`)).toBe(true);
    });

    test("infers jiasu from the default base url when catalogConnection is missing", () => {
        const channel = createModelChannel({ apiFormat: "openai", baseUrl: JIASU_BASE_URL });
        expect(channel.catalogConnection).toBeUndefined();
        expect(userChannelConnectionMode(channel)).toBe("jiasu");
    });

    test("switching from the openai default to jiasu replaces the base url", () => {
        const channel = createModelChannel({ apiFormat: "openai" });
        expect(channel.baseUrl).toBe(defaultBaseUrlForApiFormat("openai"));
        expect(userChannelConnectionPatch(channel, "jiasu")).toEqual({
            apiFormat: "openai",
            catalogConnection: "jiasu",
            interfaceType: undefined,
            baseUrl: JIASU_BASE_URL,
        });
    });

    test("switching presets keeps a custom base url", () => {
        const channel = createModelChannel({ apiFormat: "openai", baseUrl: "https://gateway.example/v1" });
        const patch = userChannelConnectionPatch(channel, "jiasu");
        expect(patch.baseUrl).toBe("https://gateway.example/v1");
        expect(patch.catalogConnection).toBe("jiasu");
        expect(patch.apiFormat).toBe("openai");
    });

    test("persisted catalogConnection wins over a matching jiasu base url", () => {
        const channel = createModelChannel({
            apiFormat: "openai",
            catalogConnection: "openai",
            baseUrl: JIASU_BASE_URL,
        });
        expect(userChannelConnectionMode(channel)).toBe("openai");
    });

    test("createModelChannel keeps jiasu catalogConnection and fills the jiasu url", () => {
        const channel = createModelChannel({ catalogConnection: "jiasu" });
        expect(channel.catalogConnection).toBe("jiasu");
        expect(channel.apiFormat).toBe("openai");
        expect(channel.baseUrl).toBe(JIASU_BASE_URL);
    });

    test("normalizeConfigSnapshot keeps the jiasu catalogConnection", () => {
        const channel = createModelChannel({ name: "佳速", catalogConnection: "jiasu", apiKey: "k" });
        const config = normalizeConfigSnapshot({ config: { channels: [channel] } }).config;
        expect(config.channels[0]?.catalogConnection).toBe("jiasu");
        expect(config.channels[0]?.baseUrl).toBe(JIASU_BASE_URL);
        expect(config.channels[0]?.apiFormat).toBe("openai");
    });

    test("nextUserChannelName increments duplicate jiasu names", () => {
        expect(nextUserChannelName([], "佳速")).toBe("佳速");
        expect(nextUserChannelName([{ name: "佳速" }], "佳速")).toBe("佳速 2");
        expect(nextUserChannelName([{ name: "佳速" }, { name: "佳速 2" }], "佳速")).toBe("佳速 3");
    });
});
