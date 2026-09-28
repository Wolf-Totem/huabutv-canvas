import { canvasConnectionPath } from "@/components/canvas/canvas-connections";
import { canvasSelectionHitsBounds, type CanvasSpatialBounds } from "@/lib/canvas/canvas-selection";
import { removeStoryboardRowBinding } from "@/lib/canvas/canvas-storyboard-assets";
import type { CanvasConnection, CanvasDisplayConnection, CanvasNodeData, CanvasSelectionHitMode } from "@/types/canvas";

const CONNECTION_SAMPLE_COUNT = 16;

type Point = { x: number; y: number };

export function sampleCanvasConnectionPoints(entry: CanvasDisplayConnection, fromScrollTop = 0, toScrollTop = 0): Point[] {
    const { startX, startY, endX, endY } = canvasConnectionPath(entry.connection, entry.from, entry.to, fromScrollTop, toScrollTop);
    const dx = Math.abs(endX - startX);
    const curvature = Math.max(dx * 0.5, 50);
    const p0 = { x: startX, y: startY };
    const p1 = { x: startX + curvature, y: startY };
    const p2 = { x: endX - curvature, y: endY };
    const p3 = { x: endX, y: endY };
    const points: Point[] = [];
    for (let index = 0; index <= CONNECTION_SAMPLE_COUNT; index++) {
        const t = index / CONNECTION_SAMPLE_COUNT;
        const u = 1 - t;
        points.push({
            x: u * u * u * p0.x + 3 * u * u * t * p1.x + 3 * u * t * t * p2.x + t * t * t * p3.x,
            y: u * u * u * p0.y + 3 * u * u * t * p1.y + 3 * u * t * t * p2.y + t * t * t * p3.y,
        });
    }
    return points;
}

export function canvasConnectionSelectionBounds(entry: CanvasDisplayConnection, fromScrollTop = 0, toScrollTop = 0): CanvasSpatialBounds {
    const points = sampleCanvasConnectionPoints(entry, fromScrollTop, toScrollTop);
    let left = points[0].x;
    let right = points[0].x;
    let top = points[0].y;
    let bottom = points[0].y;
    for (const point of points) {
        if (point.x < left) left = point.x;
        if (point.x > right) right = point.x;
        if (point.y < top) top = point.y;
        if (point.y > bottom) bottom = point.y;
    }
    return { left, top, right, bottom };
}

function pointInBounds(x: number, y: number, bounds: CanvasSpatialBounds) {
    return x >= bounds.left && x <= bounds.right && y >= bounds.top && y <= bounds.bottom;
}

function segmentIntersectsBounds(ax: number, ay: number, bx: number, by: number, bounds: CanvasSpatialBounds) {
    if (pointInBounds(ax, ay, bounds) || pointInBounds(bx, by, bounds)) return true;
    return clipSegment(ax, ay, bx, by, bounds.left, bounds.top, bounds.right, bounds.bottom);
}

function clipSegment(x0: number, y0: number, x1: number, y1: number, left: number, top: number, right: number, bottom: number) {
    let t0 = 0;
    let t1 = 1;
    const dx = x1 - x0;
    const dy = y1 - y0;
    const clips: Array<[number, number]> = [
        [-dx, x0 - left],
        [dx, right - x0],
        [-dy, y0 - top],
        [dy, bottom - y0],
    ];
    for (const [p, q] of clips) {
        if (p === 0) {
            if (q < 0) return false;
            continue;
        }
        const r = q / p;
        if (p < 0) {
            if (r > t1) return false;
            if (r > t0) t0 = r;
        } else {
            if (r < t0) return false;
            if (r < t1) t1 = r;
        }
    }
    return t0 <= t1;
}

export function canvasConnectionHitsSelection(
    entry: CanvasDisplayConnection,
    selectionBounds: CanvasSpatialBounds,
    hitMode: CanvasSelectionHitMode,
    fromScrollTop = 0,
    toScrollTop = 0,
) {
    const aabb = canvasConnectionSelectionBounds(entry, fromScrollTop, toScrollTop);
    if (hitMode === "contain") return canvasSelectionHitsBounds(selectionBounds, aabb, "contain");
    if (!canvasSelectionHitsBounds(selectionBounds, aabb, "intersect")) return false;
    const points = sampleCanvasConnectionPoints(entry, fromScrollTop, toScrollTop);
    for (let index = 1; index < points.length; index++) {
        if (segmentIntersectsBounds(points[index - 1].x, points[index - 1].y, points[index].x, points[index].y, selectionBounds)) return true;
    }
    return false;
}

export function hitCanvasConnectionsInSelection(
    connections: CanvasDisplayConnection[],
    selectionBounds: CanvasSpatialBounds,
    hitMode: CanvasSelectionHitMode,
    scrollTopById: Record<string, number> = {},
) {
    return new Set(connections
        .filter((entry) => canvasConnectionHitsSelection(entry, selectionBounds, hitMode, scrollTopById[entry.from.id] || 0, scrollTopById[entry.to.id] || 0))
        .map((entry) => entry.connection.id));
}

export function unbindStoryboardAssetsForDeletedConnections(nodes: CanvasNodeData[], deleted: CanvasConnection[]): CanvasNodeData[] {
    const removals = new Map<string, Array<{ rowId: string; nodeId: string }>>();
    for (const connection of deleted) {
        if (connection.relation !== "storyboard-asset-reference") continue;
        const rowId = connection.storyboardRowId || (connection.toHandleId?.startsWith("row:") ? connection.toHandleId.slice(4) : "");
        if (!rowId || !connection.fromNodeId) continue;
        const list = removals.get(connection.toNodeId) || [];
        list.push({ rowId, nodeId: connection.fromNodeId });
        removals.set(connection.toNodeId, list);
    }
    if (!removals.size) return nodes;
    let changed = false;
    const next = nodes.map((node) => {
        const items = removals.get(node.id);
        const rows = node.metadata?.storyboard?.rows;
        if (!items || !rows) return node;
        let nextRows = rows;
        for (const item of items) nextRows = removeStoryboardRowBinding(nextRows, item.rowId, item.nodeId);
        if (nextRows === rows) return node;
        changed = true;
        return { ...node, metadata: { ...node.metadata, storyboard: { ...node.metadata.storyboard, rows: nextRows } } };
    });
    return changed ? next : nodes;
}
