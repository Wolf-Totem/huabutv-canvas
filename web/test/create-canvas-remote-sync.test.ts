import { afterEach, expect, spyOn, test } from "bun:test";

import { confirmCreateCanvasIfOthersPending } from "@/lib/canvas/confirm-create-canvas";
import { http } from "@/services/api/request";
import { createCanvasProjectWithRemoteSync, initializeRemoteUserDataSession, listPendingCanvasSyncProjects, resetRemoteUserDataSync } from "@/services/user-data-sync";
import { useCanvasStore, type CanvasProject } from "@/stores/canvas/use-canvas-store";
import { useSyncProgressStore } from "@/stores/use-sync-progress-store";

const oldProject: CanvasProject = {
    id: "old-canvas",
    title: "旧画布",
    createdAt: "2026-01-01T00:00:00.000Z",
    updatedAt: "2026-01-02T00:00:00.000Z",
    nodes: [],
    connections: [],
    chatSessions: [],
    activeChatId: null,
    backgroundMode: "dots",
    showImageInfo: false,
    viewport: { x: 0, y: 0, k: 1 },
    directorScenes: [],
};

let restore = () => {};

afterEach(() => {
    resetRemoteUserDataSync();
    restore();
    restore = () => {};
    useSyncProgressStore.getState().clearAll();
    useCanvasStore.setState({ projects: [] });
});

test("未同步或正在上传的旧画布会出现在待保存列表", async () => {
    useCanvasStore.setState({ projects: [oldProject] });
    await initializeRemoteUserDataSession("user-1");
    expect(listPendingCanvasSyncProjects()).toEqual([]);

    useCanvasStore.setState({ projects: [{ ...oldProject, title: "旧画布已改", updatedAt: "2026-01-03T00:00:00.000Z" }] });
    expect(listPendingCanvasSyncProjects().map((item) => item.id)).toEqual(["old-canvas"]);

    useCanvasStore.setState({ projects: [oldProject] });
    useSyncProgressStore.getState().setProjectProgress("old-canvas", { projectId: "old-canvas", phase: "uploading", total: 1, completed: 0 });
    expect(listPendingCanvasSyncProjects().map((item) => item.id)).toEqual(["old-canvas"]);
});

test("新建画布只上传这一张，不把未同步的旧画布绑进同一次请求", async () => {
    useCanvasStore.setState({ projects: [oldProject] });
    await initializeRemoteUserDataSession("user-1");
    useCanvasStore.setState({ projects: [{ ...oldProject, title: "旧画布已改", updatedAt: "2026-01-03T00:00:00.000Z" }] });

    const put = spyOn(http, "put").mockImplementation(async (url: string) => {
        if (String(url).includes("old-canvas")) throw new Error("旧画布不应随新画布一起上传");
        return { project: { id: "created", title: "Agent 创作", createdAt: "", updatedAt: "" } };
    });
    restore = () => put.mockRestore();

    const { id, syncError } = await createCanvasProjectWithRemoteSync("Agent 创作");
    expect(id).toBeTruthy();
    expect(syncError).toBeUndefined();
    expect(put.mock.calls.some((call) => String(call[0]).includes(id))).toBe(true);
    expect(put.mock.calls.some((call) => String(call[0]).includes("old-canvas"))).toBe(false);
    expect(listPendingCanvasSyncProjects().map((item) => item.id)).toEqual(["old-canvas"]);
});

test("其它画布待保存时确认框：是则新建，否则回到旧画布", async () => {
    useCanvasStore.setState({ projects: [oldProject] });
    await initializeRemoteUserDataSession("user-1");
    useCanvasStore.setState({ projects: [{ ...oldProject, title: "旧画布已改", updatedAt: "2026-01-03T00:00:00.000Z" }] });

    let captured: { onOk?: () => void; onCancel?: () => void } = {};
    const modal = {
        confirm: (config: { onOk?: () => void; onCancel?: () => void }) => {
            captured = config;
        },
    };

    const pending = confirmCreateCanvasIfOthersPending(modal);
    expect(captured.onOk).toBeTypeOf("function");
    captured.onOk?.();
    await expect(pending).resolves.toEqual({ proceed: true });

    const cancelled = confirmCreateCanvasIfOthersPending(modal);
    captured.onCancel?.();
    await expect(cancelled).resolves.toEqual({ proceed: false, projectId: "old-canvas" });
});
