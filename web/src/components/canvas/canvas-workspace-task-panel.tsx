import { ListChecks, RefreshCw } from "lucide-react";

import { WorkspaceState } from "@/components/layout/workspace-state";
import type { GenerationTask } from "@/services/api/task-center";
import { TaskListItem } from "./canvas-workspace-task-list-item";

export function CanvasWorkspaceTaskPanel({ tasks, refreshing, onRefresh, onCancelTask }: { tasks: GenerationTask[]; refreshing?: boolean; onRefresh?: () => void; onCancelTask?: (task: GenerationTask) => void }) {
    return (
        <>
            <header className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-2.5">
                <ListChecks className="size-3.5 shrink-0" />
                <span className="truncate text-xs font-semibold">任务</span>
                <span className="tabular-nums text-foreground/32">{tasks.length.toLocaleString("zh-CN")}</span>
                <span className="ml-auto">
                    <button type="button" className="grid size-7 place-items-center rounded-md text-foreground/45 hover:bg-surface-hover hover:text-foreground" aria-label="刷新" onClick={onRefresh}>
                        <RefreshCw className={`size-3.5 ${refreshing ? "animate-spin" : ""}`} />
                    </button>
                </span>
            </header>
            <div className="thin-scrollbar min-h-0 flex-1 overflow-y-auto overscroll-contain py-1">
                {tasks.length ? (
                    <div className="space-y-0.5">
                        {tasks.map((task) => (
                            <TaskListItem key={task.id} task={task} onCancelTask={onCancelTask} />
                        ))}
                    </div>
                ) : (
                    <WorkspaceState icon="canvas" compact title="暂无任务" description="点击画布上的生成按钮开始创作。" />
                )}
            </div>
        </>
    );
}
