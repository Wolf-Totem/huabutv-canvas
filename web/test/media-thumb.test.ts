import { describe, expect, test } from "bun:test";

import { isUsableImagePreview, looksLikeVideoUrl, mediaThumbUrl } from "@/lib/media-thumb";
import { ossProcessedImage, ossVideoSnapshot } from "@/lib/oss-image";

describe("mediaThumbUrl", () => {
    test("视频优先使用独立封面并做图片压缩", () => {
        const cover = "https://cdn.j11.net/covers/a.jpg";
        const original = "https://cdn.j11.net/videos/a.mp4";
        expect(mediaThumbUrl({ kind: "video", coverUrl: cover, originalUrl: original, width: 480 })).toBe(ossProcessedImage(cover, 480));
        expect(looksLikeVideoUrl(original)).toBe(true);
        expect(isUsableImagePreview(original, original)).toBe(false);
    });

    test("没有封面时对对象存储视频出首帧截图，而不是返回可播放地址", () => {
        const original = "https://cdn.j11.net/videos/a.mp4";
        const thumb = mediaThumbUrl({ kind: "video", originalUrl: original, width: 480 });
        expect(thumb).toBe(ossVideoSnapshot(original, 480));
        expect(thumb).toContain("video/snapshot");
        expect(thumb).not.toBe(original);
    });

    test("本地 blob 视频没有封面时不返回可播放地址", () => {
        expect(mediaThumbUrl({ kind: "video", originalUrl: "blob:http://localhost/1", width: 480 })).toBe("");
    });

    test("图片走压缩预览", () => {
        const original = "https://cdn.j11.net/images/a.png";
        expect(mediaThumbUrl({ kind: "image", originalUrl: original, width: 480 })).toBe(ossProcessedImage(original, 480));
    });
});
