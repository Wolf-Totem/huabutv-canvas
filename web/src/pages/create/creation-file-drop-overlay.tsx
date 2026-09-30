import { AnimatePresence, motion } from "motion/react";
import { Upload } from "lucide-react";

import { aceternityMotion } from "@/lib/aceternity-motion";
import type { CreationMode } from "./creation-assets";

function creationDropHint(mode: CreationMode) {
    if (mode === "text") return "图片、视频、音频和常用文档";
    if (mode === "video") return "图片、视频和音频";
    return "仅图片";
}

export function CreationFileDropOverlay({
    active,
    busy,
    atLimit,
    maxReferences,
    mode,
}: {
    active: boolean;
    busy: boolean;
    atLimit: boolean;
    maxReferences: number;
    mode: CreationMode;
}) {
    const title = busy
        ? "生成中暂不能添加参考内容"
        : atLimit
            ? `已达到当前模型的参考内容上限（${maxReferences} 个）`
            : "释放文件，添加为参考内容";
    const subtitle = busy || atLimit ? undefined : creationDropHint(mode);

    return (
        <AnimatePresence>
            {active ? (
                <motion.div
                    aria-live="polite"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: aceternityMotion.duration.state }}
                    className="creation-file-drop-overlay pointer-events-none"
                >
                    <div className="creation-file-drop-overlay-card">
                        <Upload aria-hidden="true" />
                        <strong>{title}</strong>
                        {subtitle ? <span>{subtitle}</span> : null}
                    </div>
                </motion.div>
            ) : null}
        </AnimatePresence>
    );
}
