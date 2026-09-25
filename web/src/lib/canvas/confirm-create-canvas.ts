import { listPendingCanvasSyncProjects } from "@/services/user-data-sync";

export type CanvasCreateConfirmModal = {
    confirm: (config: {
        title: string;
        content: string;
        okText: string;
        cancelText: string;
        centered?: boolean;
        maskClosable?: boolean;
        className?: string;
        onOk?: () => void;
        onCancel?: () => void;
    }) => void;
};

export type CreateCanvasWhilePendingDecision = { proceed: true } | { proceed: false; projectId: string };

/** 其它画布还在保存时先问要不要新建；否即回到正在保存的旧画布。 */
export function confirmCreateCanvasIfOthersPending(modal: CanvasCreateConfirmModal): Promise<CreateCanvasWhilePendingDecision> {
    const pending = listPendingCanvasSyncProjects();
    if (!pending.length) return Promise.resolve({ proceed: true });
    const current = pending[0];
    const extra = pending.length > 1 ? `等 ${pending.length} 张画布` : "";
    return new Promise((resolve) => {
        modal.confirm({
            className: "workspace-modal workspace-modal-compact",
            title: "其他画布正在保存中",
            content: `「${current.title || "未命名画布"}」${extra}还在保存或同步。创建新画布后，未完成的会在后台继续上传。`,
            okText: "创建新画布",
            cancelText: "继续编辑旧画布",
            centered: true,
            maskClosable: false,
            onOk: () => resolve({ proceed: true }),
            onCancel: () => resolve({ proceed: false, projectId: current.id }),
        });
    });
}
