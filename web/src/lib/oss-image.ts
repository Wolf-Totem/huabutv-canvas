/** 阿里云 OSS/CDN 图片实时压缩。关 OSS 或非阿里云地址时原样返回。 */
export function ossProcessedImage(url: string | undefined, width = 720) {
    const source = String(url || "").trim();
    if (!source) return "";
    if (source.startsWith("data:") || source.startsWith("blob:") || source.startsWith("/api/")) return source;
    if (/x-oss-process=/i.test(source)) return source;
    const host = safeHost(source);
    const aliyun = /aliyuncs\.com$|\.aliyuncs\.com$|cdn\.j11\.net$|liblib\.art$|liblib\.cloud$|oss-cn-/i.test(host);
    if (!aliyun && !host.endsWith("j11.net")) return source;
    const spec = `image/resize,w_${Math.max(32, Math.min(width, 1920))},m_lfit/format,webp/quality,q_80`;
    return `${source}${source.includes("?") ? "&" : "?"}x-oss-process=${spec}`;
}

function safeHost(url: string) {
    try {
        return new URL(url, "https://canvas.j11.net").hostname;
    } catch {
        return "";
    }
}
