import { describe, expect, test } from "bun:test";

import { isChunkLoadError } from "../src/lib/chunk-load";

describe("stale chunk recovery", () => {
    test("detects dynamic import failures after a deploy", () => {
        expect(isChunkLoadError(new TypeError("Failed to fetch dynamically imported module: https://canvas.jiasuapi.com/assets/require-feature-C3_QlGU4.js"))).toBe(true);
        expect(isChunkLoadError(new Error("network down"))).toBe(false);
    });
});
