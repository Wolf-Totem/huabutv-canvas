import { describe, expect, test } from "bun:test";

import { filterCanvasDisplayConnections } from "@/lib/canvas/canvas-connection-visibility";
import { CanvasNodeType, type CanvasDisplayConnection, type CanvasNodeData } from "@/types/canvas";

function node(id: string): CanvasNodeData {
    return { id, type: CanvasNodeType.Text, title: id, x: 0, y: 0, width: 120, height: 80 } as CanvasNodeData;
}

function entry(id: string, from: string, to: string): CanvasDisplayConnection {
    return { connection: { id, fromNodeId: from, toNodeId: to }, from: node(from), to: node(to) } as CanvasDisplayConnection;
}

describe("filterCanvasDisplayConnections", () => {
    const connections = [entry("a", "n1", "n2"), entry("b", "n3", "n4")];

    test("keeps every connection when hide mode is off", () => {
        expect(filterCanvasDisplayConnections(connections, { enabled: false }).map((item) => item.connection.id)).toEqual(["a", "b"]);
    });

    test("keeps only the focused node's connections", () => {
        const visible = filterCanvasDisplayConnections(connections, { enabled: true, hoveredNodeId: "n1" });
        expect(visible.map((item) => item.connection.id)).toEqual(["a"]);
    });

    test("keeps a selected connection even if its nodes are not focused", () => {
        const visible = filterCanvasDisplayConnections(connections, { enabled: true, selectedConnectionId: "b" });
        expect(visible.map((item) => item.connection.id)).toEqual(["b"]);
    });
});
