import { describe, expect, test } from "bun:test";

import {
    autoExpandedEditorHeight,
    clampPromptHeight,
    estimatePromptContentHeight,
    expandedChromeHeight,
    expandedComfortHeight,
    promptEditorBounds,
    PROMPT_EDITOR_EXPANDED_MIN_HEIGHT,
    PROMPT_EDITOR_LINE_HEIGHT,
    PROMPT_EDITOR_MAX_LINES,
    PROMPT_EDITOR_MIN_HEIGHT,
    PROMPT_EDITOR_VERTICAL_PADDING,
    PROMPT_REFERENCE_SHELF_HEIGHT,
} from "@/lib/canvas/prompt-editor-height";

describe("prompt editor height", () => {
    test("keeps the compact editor capped at eight lines", () => {
        const plain = promptEditorBounds({ expanded: false, hasReferences: false });
        expect(plain.min).toBe(PROMPT_EDITOR_MIN_HEIGHT);
        expect(plain.max).toBe(PROMPT_EDITOR_LINE_HEIGHT * PROMPT_EDITOR_MAX_LINES + PROMPT_EDITOR_VERTICAL_PADDING);

        const withShelf = promptEditorBounds({ expanded: false, hasReferences: true });
        expect(withShelf.min).toBe(PROMPT_EDITOR_MIN_HEIGHT + PROMPT_REFERENCE_SHELF_HEIGHT);
        expect(withShelf.max).toBe(plain.max + PROMPT_REFERENCE_SHELF_HEIGHT);
    });

    test("lets the expanded editor grow with remaining viewport instead of eight lines", () => {
        const bounds = promptEditorBounds({
            expanded: true,
            hasReferences: false,
            viewportHeight: 900,
            chromeHeight: 200,
        });
        expect(bounds.min).toBe(PROMPT_EDITOR_EXPANDED_MIN_HEIGHT);
        expect(bounds.max).toBe(700);
        expect(bounds.max).toBeGreaterThan(PROMPT_EDITOR_LINE_HEIGHT * PROMPT_EDITOR_MAX_LINES + PROMPT_EDITOR_VERTICAL_PADDING);
    });

    test("never lets expanded max fall below min plus the reference shelf", () => {
        const bounds = promptEditorBounds({
            expanded: true,
            hasReferences: true,
            viewportHeight: 120,
            chromeHeight: 100,
        });
        expect(bounds.min).toBe(PROMPT_EDITOR_EXPANDED_MIN_HEIGHT + PROMPT_REFERENCE_SHELF_HEIGHT);
        expect(bounds.max).toBe(bounds.min);
    });

    test("uses a comfort floor for short expanded prompts and grows with content", () => {
        expect(expandedComfortHeight(800)).toBe(280);
        expect(expandedComfortHeight(400)).toBe(160);
        expect(expandedComfortHeight(0)).toBe(PROMPT_EDITOR_EXPANDED_MIN_HEIGHT);

        const short = autoExpandedEditorHeight({
            contentHeight: 76,
            hasReferences: false,
            viewportHeight: 1000,
            chromeHeight: 200,
            manualHeight: null,
        });
        expect(short).toBe(280);

        const long = autoExpandedEditorHeight({
            contentHeight: 420,
            hasReferences: false,
            viewportHeight: 1000,
            chromeHeight: 200,
            manualHeight: null,
        });
        expect(long).toBe(420);
    });

    test("keeps a manual expanded height until the viewport clamps it", () => {
        const kept = autoExpandedEditorHeight({
            contentHeight: 76,
            hasReferences: false,
            viewportHeight: 1000,
            chromeHeight: 200,
            manualHeight: 240,
        });
        expect(kept).toBe(240);

        const clamped = autoExpandedEditorHeight({
            contentHeight: 76,
            hasReferences: false,
            viewportHeight: 500,
            chromeHeight: 200,
            manualHeight: 900,
        });
        expect(clamped).toBe(300);
        expect(clampPromptHeight(10, { min: 76, max: 300 })).toBe(76);
    });

    test("adds a stack gap when video tools occupy chrome", () => {
        const withoutTools = expandedChromeHeight({ hasVideoPromptTools: false });
        const withTools = expandedChromeHeight({ hasVideoPromptTools: true, toolsHeight: 48 });
        expect(withTools - withoutTools).toBe(10 + 48);
        expect(estimatePromptContentHeight("", false)).toBe(PROMPT_EDITOR_MIN_HEIGHT);
        expect(estimatePromptContentHeight("一行提示词", true)).toBeGreaterThan(PROMPT_EDITOR_EXPANDED_MIN_HEIGHT - 1);
    });
});
