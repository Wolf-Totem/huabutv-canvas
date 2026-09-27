import { describe, expect, it } from "bun:test";
import { agentOperationCategory, agentOperationFailed, agentOperationLabel, agentOperationSegmentLabel, buildAgentFeedSegments, isAgentCarrierRecord, isAgentOperationRecord } from "@/lib/canvas/agent-operation-feed";

type Row = { id: string; role: string; title?: string; text: string; detail?: unknown; planItems?: { id: string }[]; question?: unknown };

const step = (id: string, title: string, text: string, detail: unknown = { eventType: "tool_completed" }): Row => ({ id, role: "tool", title, text, detail });
const kinds = (rows: Row[]) => buildAgentFeedSegments(rows).map((segment) => segment.kind);
const idsOf = (segment: ReturnType<typeof buildAgentFeedSegments<Row>>[number]) => (segment.kind === "operations" ? segment.items.map((item) => item.id) : [segment.item.id]);

describe("Agent operation feed", () => {
    it("folds consecutive tool records into one segment keyed by the first step", () => {
        const rows: Row[] = [
            { id: "u1", role: "user", text: "读一下画布" },
            step("t1", "canvas_get_state", "已读取当前画布"),
            step("t2", "model_list", "已获取可用模型"),
            { id: "a1", role: "assistant", text: "画布里有 1 个节点" },
            step("t3", "canvas_apply_ops", "画布内容已保存至服务端", { eventType: "canvas_updated" }),
        ];
        const segments = buildAgentFeedSegments(rows);
        expect(kinds(rows)).toEqual(["message", "operations", "message", "operations"]);
        expect(segments[1]).toMatchObject({ kind: "operations", key: "t1" });
        expect(idsOf(segments[1])).toEqual(["t1", "t2"]);
    });

    it("appends later steps to the same segment so its key and expanded state survive", () => {
        const before: Row[] = [step("t1", "canvas_get_state", "已读取当前画布"), step("t2", "model_list", "已获取可用模型")];
        const after = buildAgentFeedSegments([...before, step("t3", "task_get", "已查询任务状态")]);
        expect(after).toHaveLength(1);
        expect(after[0]).toMatchObject({ kind: "operations", key: "t1" });
        expect(idsOf(after[0])).toEqual(["t1", "t2", "t3"]);
    });

    it("keeps the approval card and unrelated roles out of the folded group", () => {
        const pending: Row = { id: "p1", role: "tool", title: "canvas_apply_ops", text: "准备更新画布内容", detail: { status: "pending" } };
        const rows: Row[] = [step("t1", "canvas_get_state", "已读取当前画布"), pending, step("t2", "model_list", "已获取可用模型")];
        expect(kinds(rows)).toEqual(["operations", "message", "operations"]);
        expect(isAgentOperationRecord(pending)).toBe(false);
        expect(isAgentOperationRecord(rows[0])).toBe(true);
    });

    it("drops plan and question carriers that carry no body of their own", () => {
        const plan: Row = { id: "plan-run1", role: "tool", text: "", planItems: [{ id: "a" }] };
        const question: Row = { id: "question-run1", role: "assistant", text: "", question: { question: "选哪个？" } };
        const answered: Row = { id: "a1", role: "assistant", text: "那就用第一个方案", question: { question: "选哪个？" } };
        expect(buildAgentFeedSegments([plan, question, answered]).map((segment) => segment.key)).toEqual(["a1"]);
        expect(isAgentCarrierRecord(plan)).toBe(true);
        expect(isAgentCarrierRecord(question)).toBe(true);
        expect(isAgentCarrierRecord(answered)).toBe(false);
    });

    it("reports the latest step, including an in-flight and a failed one", () => {
        expect(agentOperationLabel(step("t1", "canvas_get_state", "工具执行成功"))).toBe("已读取当前画布");
        expect(agentOperationLabel(step("t2", "canvas_apply_ops", "", {}))).toBe("准备更新画布内容");
        expect(agentOperationLabel(step("t3", "canvas_apply_ops", "", { eventType: "tool_failed" }))).toBe("更新画布内容失败");
        expect(agentOperationFailed(step("t3", "canvas_apply_ops", "", { eventType: "tool_failed" }))).toBe(true);
        expect(agentOperationFailed(step("t1", "canvas_get_state", "工具执行成功"))).toBe(false);
    });
});

describe("Agent operation segment label", () => {
    const viewed = (id: string, title: string, extra: Record<string, unknown> = {}) => step(id, "canvas_inspect_image", "工具执行成功", { eventType: "tool_completed", result: { nodeId: id, title, ...extra } });

    it("aggregates a vision run by the images that really went out", () => {
        const items = [viewed("n1", "剧照1.png"), viewed("n2", "剧照2.png"), viewed("n3", "封面.png")];
        expect(agentOperationSegmentLabel(items)).toBe("查看了 3 张画面 · 最新《封面.png》");
        expect(agentOperationCategory(items[0])).toBe("vision");
    });

    it("does not count repeat receipts without pixels", () => {
        const items = [viewed("n1", "剧照1.png"), viewed("n1", "剧照1.png", { repeat: true })];
        expect(agentOperationSegmentLabel(items)).toBe("查看了 1 张画面 · 最新《剧照1.png》");
    });
});
