import { useEffect, useState } from "react";
import { Button } from "antd";
import { useNavigate, useParams } from "react-router";

import { FullScreenLoader } from "@/components/ui/aceternity/full-screen-loader";
import { getAdminPlazaApplicationSnapshot } from "@/services/api/plaza";
import type { CanvasProject } from "@/stores/canvas/use-canvas-store";
import { ReadOnlyCanvasView } from "@/pages/canvas/read-only-canvas";

export default function PlazaApplicationPreviewPage() {
    const { id = "" } = useParams();
    const navigate = useNavigate();
    const [project, setProject] = useState<CanvasProject | null>(null);
    const [error, setError] = useState("");

    useEffect(() => {
        getAdminPlazaApplicationSnapshot(id).then((result) => setProject(result.project)).catch((err) => setError(err instanceof Error ? err.message : "无法预览"));
    }, [id]);

    if (!project && !error) return <FullScreenLoader label="正在打开申请草稿" detail="只读预览，不读取作者当前画布" />;
    if (error || !project) return <div className="grid h-screen place-items-center">{error}<Button onClick={() => navigate("/admin/plaza/applications")}>返回</Button></div>;
    return <ReadOnlyCanvasView title="申请草稿预览" project={project} mode="tour" headerRight={<Button onClick={() => navigate("/admin/plaza/applications")}>返回审核</Button>} />;
}
