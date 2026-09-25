import {
    defaultBaseUrlForApiFormat,
    JIASU_BASE_URL,
    type ApiCallFormat,
    type ModelChannel,
    type UserChannelConnection,
} from "@/stores/use-config-store";

export type { UserChannelConnection };

export function apiFormatForUserChannelConnection(connection: UserChannelConnection): ApiCallFormat {
    return connection === "gemini" ? "gemini" : "openai";
}

export function defaultBaseUrlForUserChannelConnection(connection: UserChannelConnection): string {
    if (connection === "gemini") return defaultBaseUrlForApiFormat("gemini");
    if (connection === "jiasu") return JIASU_BASE_URL;
    return defaultBaseUrlForApiFormat("openai");
}

export function normalizeChannelBaseUrl(value: string) {
    return value.trim().replace(/\/+$/, "");
}

export function knownDefaultChannelBaseUrls() {
    return [
        defaultBaseUrlForUserChannelConnection("openai"),
        defaultBaseUrlForUserChannelConnection("gemini"),
        defaultBaseUrlForUserChannelConnection("jiasu"),
    ];
}

export function isKnownDefaultBaseUrl(value: string) {
    const normalized = normalizeChannelBaseUrl(value);
    if (!normalized) return true;
    return knownDefaultChannelBaseUrls().some((candidate) => normalizeChannelBaseUrl(candidate) === normalized);
}

export function userChannelConnectionMode(channel: Pick<ModelChannel, "apiFormat" | "baseUrl" | "catalogConnection">): UserChannelConnection {
    if (channel.catalogConnection === "openai" || channel.catalogConnection === "gemini" || channel.catalogConnection === "jiasu") {
        return channel.catalogConnection;
    }
    if (channel.apiFormat === "gemini") return "gemini";
    if (normalizeChannelBaseUrl(channel.baseUrl) === normalizeChannelBaseUrl(defaultBaseUrlForUserChannelConnection("jiasu"))) return "jiasu";
    return "openai";
}

export function userChannelConnectionPatch(channel: Pick<ModelChannel, "baseUrl">, connection: UserChannelConnection): Partial<ModelChannel> {
    const defaultBaseUrl = defaultBaseUrlForUserChannelConnection(connection);
    return {
        apiFormat: apiFormatForUserChannelConnection(connection),
        catalogConnection: connection,
        interfaceType: undefined,
        baseUrl: isKnownDefaultBaseUrl(channel.baseUrl) ? defaultBaseUrl : channel.baseUrl,
    };
}

export function nextUserChannelName(existing: Array<{ name: string }>, base: string) {
    const names = new Set(existing.map((item) => item.name.trim()));
    const label = base.trim() || "佳速";
    if (!names.has(label)) return label;
    for (let index = 2; ; index += 1) {
        const candidate = `${label} ${index}`;
        if (!names.has(candidate)) return candidate;
    }
}
