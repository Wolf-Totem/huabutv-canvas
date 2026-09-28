import { describe, expect, test } from "bun:test";
import { backendProviderConfig } from "@/services/api/generation-task";
import { defaultConfig, type AiConfig, type ModelChannel, resolveModelRequestConfig, upstreamModelKey } from "@/stores/use-config-store";

function userChannel(patch: Partial<ModelChannel> = {}): ModelChannel {
    return {
        id: "user-jiasu",
        name: "佳速",
        baseUrl: "https://ai.jiasuapi.com",
        apiKey: "sk-test",
        apiFormat: "openai",
        scope: "user",
        models: ["gpt-image-2-1k"],
        modelCosts: [{
            model: "gpt-image-2-1k",
            capability: "image",
            protocol: "jiasu-image",
            billingMode: "fixed_request",
            unitPriceMicrocredits: 0,
            providerModelKey: "gpt-image-2.5-1k",
        }],
        ...patch,
    };
}

function configFor(channel: ModelChannel): AiConfig {
    return { ...defaultConfig, channels: [channel], model: `${channel.id}::gpt-image-2-1k`, imageModel: `${channel.id}::gpt-image-2-1k` };
}

describe("personal channel upstream model id", () => {
    test("sends providerModelKey to the vendor and keeps the catalog name for lookup", () => {
        const channel = userChannel();
        expect(upstreamModelKey(channel, "gpt-image-2-1k")).toBe("gpt-image-2.5-1k");
        const request = resolveModelRequestConfig(configFor(channel), `${channel.id}::gpt-image-2-1k`);
        expect(request.model).toBe("gpt-image-2-1k");
        expect(request.providerModelKey).toBe("gpt-image-2.5-1k");
        expect(request.channelId).toBe("");
        const payload = backendProviderConfig(configFor(channel), "image") as { model?: string; providerModelKey?: string };
        expect(payload.model).toBe("gpt-image-2.5-1k");
        expect(payload.providerModelKey).toBe("gpt-image-2.5-1k");
    });

    test("falls back to the catalog name when upstream id is empty", () => {
        const channel = userChannel({ modelCosts: [{ model: "gpt-image-2-1k", capability: "image", protocol: "jiasu-image", billingMode: "fixed_request", unitPriceMicrocredits: 0 }] });
        expect(upstreamModelKey(channel, "gpt-image-2-1k")).toBe("gpt-image-2-1k");
        const payload = backendProviderConfig(configFor(channel), "image") as { model?: string; providerModelKey?: string };
        expect(payload.model).toBe("gpt-image-2-1k");
        expect(payload.providerModelKey).toBeUndefined();
    });
});
