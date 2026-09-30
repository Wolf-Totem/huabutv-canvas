import { describe, expect, test } from "bun:test";

import { isUsableImagePreview, looksLikeVideoUrl, mediaThumbUrl } from "@/lib/media-thumb";
import { canProcessOssUrl, ossProcessedImage, ossVideoSnapshot, rewriteLegacyCdnHost } from "@/lib/oss-image";

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
        expect(thumb).toContain("cdn.huabutv.com");
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

describe("oss process allowlist", () => {
    test("cdn.huabutv.com 可以追加处理参数", () => {
        const source = "https://cdn.huabutv.com/images/a.png";
        const processed = ossProcessedImage(source, 480);
        expect(canProcessOssUrl(source)).toBe(true);
        expect(processed).toContain("cdn.huabutv.com");
        expect(processed).toContain("x-oss-process=");
        expect(processed).toContain("w_480");
    });

    test("旧 CDN host 先改写再处理", () => {
        const source = "https://cdn.j11.net/images/a.png";
        expect(rewriteLegacyCdnHost(source)).toBe("https://cdn.huabutv.com/images/a.png");
        expect(rewriteLegacyCdnHost("https://oss.j11.net/videos/a.mp4")).toBe("https://cdn.huabutv.com/videos/a.mp4");
        const processed = ossProcessedImage(source, 480);
        expect(processed).toContain("https://cdn.huabutv.com/images/a.png");
        expect(processed).toContain("x-oss-process=");
        expect(processed).not.toContain("cdn.j11.net");
        expect(rewriteLegacyCdnHost("/manchuang/hero.webp")).toBe("/manchuang/hero.webp");
        expect(canProcessOssUrl("/manchuang/hero.webp")).toBe(false);
        expect(canProcessOssUrl("https://www.huabutv.com/og-image.png")).toBe(false);
    });

    test("第三方对象 host 仍可 process", () => {
        expect(canProcessOssUrl("https://liblib.art/covers/a.jpg")).toBe(true);
        expect(canProcessOssUrl("https://cdn.liblib.cloud/covers/a.jpg")).toBe(true);
        expect(ossProcessedImage("https://liblib.art/covers/a.jpg", 480)).toContain("x-oss-process=");
        expect(ossProcessedImage("https://liblib.art/covers/a.jpg", 480)).toContain("liblib.art");
    });
});
