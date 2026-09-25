import { ossProcessedImage, ossVideoSnapshot } from "@/lib/oss-image";

const VIDEO_FILE_PATTERN = /\.(mp4|webm|mov|m4v|mkv)(\?|#|$)/i;

export function looksLikeVideoUrl(url: string | undefined) {
    const source = String(url || "").trim();
    if (!source || source.startsWith("data:image/") || source.startsWith("blob:")) return false;
    if (source.startsWith("data:video/")) return true;
    return VIDEO_FILE_PATTERN.test(source);
}

export function isUsableImagePreview(url: string | undefined, originalVideoUrl?: string) {
    const source = String(url || "").trim();
    if (!source) return false;
    if (originalVideoUrl && source === originalVideoUrl) return false;
    if (looksLikeVideoUrl(source) && !/video\/snapshot/i.test(source)) return false;
    return true;
}

/** 未激活展示用的压缩预览。视频只返回首帧图或 OSS 截帧，绝不返回可播放地址。 */
export function mediaThumbUrl(options: { kind?: string; coverUrl?: string; originalUrl?: string; width?: number }) {
    const width = options.width ?? 480;
    const cover = String(options.coverUrl || "").trim();
    const original = String(options.originalUrl || "").trim();
    const video = options.kind === "video" || looksLikeVideoUrl(original);
    if (video) {
        if (isUsableImagePreview(cover, original)) return ossProcessedImage(cover, width) || cover;
        return ossVideoSnapshot(original, width);
    }
    const source = cover || original;
    if (!source) return "";
    if (source.startsWith("data:") || source.startsWith("blob:")) return source;
    return ossProcessedImage(source, width) || source;
}
