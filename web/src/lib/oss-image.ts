/** 阿里云 OSS/CDN 图片实时压缩。关 OSS 或非阿里云地址时原样返回。 */
export function ossProcessedImage(url: string | undefined, width = 720) {
    const source = String(url || "").trim();
    if (!source || !canProcessOssUrl(source)) return source;
    if (/x-oss-process=/i.test(source)) return source;
    const spec = `image/resize,w_${clampOssWidth(width, 1920)},m_lfit/format,webp/quality,q_80`;
    return appendOssProcess(source, spec);
}

/** 对象存储上的视频首帧截图。不能处理的地址返回空，调用方应退回封面图或占位，而不是挂原视频。 */
export function ossVideoSnapshot(url: string | undefined, width = 480) {
    const source = String(url || "").trim();
    if (!source || !canProcessOssUrl(source)) return "";
    if (/video\/snapshot/i.test(source)) return source;
    if (/x-oss-process=/i.test(source)) return "";
    const spec = `video/snapshot,t_0,f_jpg,w_${clampOssWidth(width, 1280)},m_fast,ar_auto`;
    return appendOssProcess(source, spec);
}

export function canProcessOssUrl(url: string) {
    const source = String(url || "").trim();
    if (!source || source.startsWith("data:") || source.startsWith("blob:") || source.startsWith("/api/")) return false;
    const host = safeHost(source);
    return /aliyuncs\.com$|\.aliyuncs\.com$|cdn\.j11\.net$|liblib\.art$|liblib\.cloud$|oss-cn-/i.test(host) || host.endsWith("j11.net");
}

function clampOssWidth(width: number, max: number) {
    return Math.max(32, Math.min(width, max));
}

function appendOssProcess(source: string, spec: string) {
    return `${source}${source.includes("?") ? "&" : "?"}x-oss-process=${spec}`;
}

function safeHost(url: string) {
    try {
        return new URL(url, "https://canvas.j11.net").hostname;
    } catch {
        return "";
    }
}
