export const PROMPT_EDITOR_LINE_HEIGHT = 20;
export const PROMPT_EDITOR_EXPANDED_LINE_HEIGHT = 24;
export const PROMPT_EDITOR_VERTICAL_PADDING = 12;
export const PROMPT_EDITOR_EXPANDED_VERTICAL_PADDING = 20;
export const PROMPT_EDITOR_MIN_HEIGHT = 72;
export const PROMPT_EDITOR_EXPANDED_MIN_HEIGHT = 76;
export const PROMPT_EDITOR_MAX_LINES = 8;
export const PROMPT_REFERENCE_SHELF_HEIGHT = 58;
export const PROMPT_EDITOR_COMFORT_CAP = 280;
export const PROMPT_EDITOR_COMFORT_RATIO = 0.4;
export const PROMPT_MODAL_MARGIN_Y = 32;
export const PROMPT_MODAL_PAD_Y = 24;
export const PROMPT_MODAL_HEADER_FALLBACK = 36;
export const PROMPT_MODAL_FOOTER_FALLBACK = 44;
export const PROMPT_RESIZE_HANDLE_HEIGHT = 12;
export const PROMPT_STACK_GAP = 10;

function shelfHeight(hasReferences: boolean) {
    return hasReferences ? PROMPT_REFERENCE_SHELF_HEIGHT : 0;
}

function minBody(expanded: boolean) {
    return expanded ? PROMPT_EDITOR_EXPANDED_MIN_HEIGHT : PROMPT_EDITOR_MIN_HEIGHT;
}

export function expandedComfortHeight(availableViewport: number) {
    const comfort = Math.min(PROMPT_EDITOR_COMFORT_CAP, Math.round(PROMPT_EDITOR_COMFORT_RATIO * Math.max(0, availableViewport)));
    return Math.max(PROMPT_EDITOR_EXPANDED_MIN_HEIGHT, comfort);
}

export function expandedChromeHeight(args: {
    hasVideoPromptTools: boolean;
    headerHeight?: number;
    footerHeight?: number;
    toolsHeight?: number;
}) {
    const stackCount = args.hasVideoPromptTools ? 4 : 3;
    const toolsHeight = args.hasVideoPromptTools ? Math.max(0, args.toolsHeight ?? 0) : 0;
    return PROMPT_MODAL_MARGIN_Y
        + PROMPT_MODAL_PAD_Y
        + (args.headerHeight ?? PROMPT_MODAL_HEADER_FALLBACK)
        + (args.footerHeight ?? PROMPT_MODAL_FOOTER_FALLBACK)
        + PROMPT_RESIZE_HANDLE_HEIGHT
        + PROMPT_STACK_GAP * stackCount
        + toolsHeight;
}

export function promptEditorBounds(args: {
    expanded: boolean;
    hasReferences: boolean;
    viewportHeight?: number;
    chromeHeight?: number;
}): { min: number; max: number } {
    const shelf = shelfHeight(args.hasReferences);
    const min = minBody(args.expanded) + shelf;
    if (!args.expanded) {
        return {
            min,
            max: PROMPT_EDITOR_LINE_HEIGHT * PROMPT_EDITOR_MAX_LINES + PROMPT_EDITOR_VERTICAL_PADDING + shelf,
        };
    }
    const viewportMax = Math.max(min, (args.viewportHeight ?? 0) - (args.chromeHeight ?? 0));
    return { min, max: viewportMax };
}

export function estimatePromptContentHeight(value: string, expanded: boolean) {
    const min = minBody(expanded);
    if (!value.trim()) return min;
    const charsPerLine = expanded ? 34 : 38;
    const lineCount = value.split("\n").reduce((total, line) => total + Math.max(1, Math.ceil(Array.from(line).length / charsPerLine)), 0);
    const lineHeight = expanded ? PROMPT_EDITOR_EXPANDED_LINE_HEIGHT : PROMPT_EDITOR_LINE_HEIGHT;
    const verticalPadding = expanded ? PROMPT_EDITOR_EXPANDED_VERTICAL_PADDING : PROMPT_EDITOR_VERTICAL_PADDING;
    return Math.max(min, lineCount * lineHeight + verticalPadding);
}

export function clampPromptHeight(height: number, bounds: { min: number; max: number }) {
    return Math.min(bounds.max, Math.max(bounds.min, height));
}

export function autoExpandedEditorHeight(args: {
    contentHeight: number;
    hasReferences: boolean;
    viewportHeight: number;
    chromeHeight: number;
    manualHeight: number | null;
}) {
    const bounds = promptEditorBounds({
        expanded: true,
        hasReferences: args.hasReferences,
        viewportHeight: args.viewportHeight,
        chromeHeight: args.chromeHeight,
    });
    const shelf = shelfHeight(args.hasReferences);
    if (args.manualHeight != null) return clampPromptHeight(args.manualHeight, bounds);
    const comfort = expandedComfortHeight(args.viewportHeight - args.chromeHeight);
    return clampPromptHeight(Math.max(args.contentHeight + shelf, comfort + shelf), bounds);
}
