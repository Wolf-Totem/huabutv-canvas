import { describe, expect, test } from "bun:test";

import { backendProvider, changesRequireOSSRetest, DEFAULT_OSS_PATH_PREFIX, getS3PresetHints, normalizeOSSConnectionTestInput, storageModeFromSetting } from "../src/lib/oss-settings";

describe("OSS settings helpers", () => {
    test("provides editable S3 endpoint hints for known presets", () => {
        expect(getS3PresetHints("r2")).toMatchObject({ region: "auto" });
        expect(getS3PresetHints("b2").endpoint).toContain("backblazeb2.com");
    });

    test("only connection fields invalidate a previous test", () => {
        expect(changesRequireOSSRetest({ endpoint: "https://s3.example.com" })).toBe(true);
        expect(changesRequireOSSRetest({ enabled: true })).toBe(false);
        expect(changesRequireOSSRetest({ allowUserS3: true })).toBe(false);
    });

    test("uses the product path prefix by default", () => {
        expect(DEFAULT_OSS_PATH_PREFIX).toBe("open-ai-canvas");
    });

    test("maps first-class R2 mode to s3+r2 without emitting provider r2", () => {
        expect(backendProvider("r2")).toBe("s3");
        expect(backendProvider("s3")).toBe("s3");
        expect(storageModeFromSetting({ enabled: true, provider: "s3", s3Preset: "r2" })).toBe("r2");
        expect(storageModeFromSetting({ enabled: true, provider: "s3", s3Preset: "aws" })).toBe("s3");
        expect(storageModeFromSetting({ enabled: false, provider: "s3", s3Preset: "r2" })).toBe("local");
    });

    test("normalizes a Tencent COS test draft when S3-only fields are not mounted", () => {
        const input = normalizeOSSConnectionTestInput({
            provider: "tencent",
            region: " ap-guangzhou ",
            endpoint: " https://cos.ap-guangzhou.myqcloud.com/ ",
            bucket: " example-1250000000 ",
            accessKeyId: " secret-id ",
            accessKeySecret: " secret-key ",
            pathPrefix: " /canvas/ ",
        });

        expect(input).toMatchObject({
            provider: "tencent",
            region: "ap-guangzhou",
            endpoint: "https://cos.ap-guangzhou.myqcloud.com",
            cdnBaseUrl: "",
            bucket: "example-1250000000",
            accessKeyId: "secret-id",
            accessKeySecret: "secret-key",
            sessionToken: "",
            pathPrefix: "canvas",
            s3Preset: "custom",
            pathStyle: false,
        });
    });
});
