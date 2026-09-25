import { useEffect, useState } from "react";
import { Button } from "antd";
import { LogIn } from "lucide-react";
import { Link, useParams } from "react-router";

import { FullScreenLoader } from "@/components/ui/aceternity/full-screen-loader";
import { WorkspaceState } from "@/components/layout/workspace-state";
import { getPublicCanvasShare } from "@/services/api/canvas-share";
import type { CanvasProject } from "@/stores/canvas/use-canvas-store";
import { ReadOnlyCanvasView } from "./read-only-canvas";

export default function SharedCanvasPage() {
    const { token = "" } = useParams();
    const [project, setProject] = useState<CanvasProject | null>(null);
    const [loading, setLoading] = useState(true);
    const [loadError, setLoadError] = useState("");

    useEffect(() => {
        let active = true;
        setLoading(true);
        getPublicCanvasShare(token).then(({ project: next }) => {
            if (active) setProject(next);
        }).catch((error) => {
            if (active) setLoadError(error instanceof Error ? error.message : "分享链接无效或已失效");
        }).finally(() => {
            if (active) setLoading(false);
        });
        return () => { active = false; };
    }, [token]);

    if (loading) return <FullScreenLoader label="正在打开共享画布" detail="读取节点、连线和视图状态" />;
    if (loadError || !project) return <div className="grid h-screen place-items-center px-5"><WorkspaceState icon="error" title="分享链接不可用" description={loadError} action={<Link to="/"><Button>返回首页</Button></Link>} /></div>;

    return (
        <ReadOnlyCanvasView
            title={project.title || "共享画布"}
            project={project}
            mode="share"
            headerRight={<Link to="/login"><Button type="text" icon={<LogIn className="size-4" />}>登录</Button></Link>}
        />
    );
}
