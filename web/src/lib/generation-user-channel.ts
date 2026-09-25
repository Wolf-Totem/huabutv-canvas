import { resolveModelChannel, type AiConfig } from "@/stores/use-config-store";

export function generationUsesUserChannel(config: AiConfig, model = config.model) {
    const channel = resolveModelChannel(config, model);
    if (channel.scope === "system") return false;
    return Boolean(channel.baseUrl.trim() && (channel.apiKey.trim() || channel.hasApiKey));
}

export function userChannelDisconnectNotice(config: AiConfig, model: string, kind: "video" | "image" | "generic") {
    if (!generationUsesUserChannel(config, model)) return "";
    if (kind === "video") {
        return "当前使用你自己的 API。出视频请勿关闭浏览器；异步任务号写入后，掉线可由服务器继续查询。同步请求中断不会自动重试。";
    }
    if (kind === "image") {
        return "当前使用你自己的 API。异步出图的任务号会写入服务器，关掉页面后仍可续查。同步请求中断不会自动重试。";
    }
    return "当前使用你自己的 API。同步请求中断不会自动重试。";
}
