import { useEffect } from "react";
import { Navigate } from "react-router";
import { FullScreenLoader } from "@/components/ui/aceternity/full-screen-loader";
import ManchuangHomePage from "@/pages/public-home/manchuang-home";
import { canvasWorkspaceURL, isAgentHost, isStreamerMarketingHost } from "@/lib/public-hosts";
import { useUserStore } from "@/stores/use-user-store";

/** 主应用里如果被路由到 /welcome，必须整页重载，才能挂上独立欢迎页 bundle。 */
export function WelcomeHardLoad() {
    useEffect(() => {
        window.location.replace("/welcome");
    }, []);
    return <FullScreenLoader label="正在打开首页" detail="前往欢迎页" />;
}

export default function RootHome() {
    const user = useUserStore((state) => state.user);

    if (isAgentHost()) {
        if (user) return <Navigate to="/agent" replace />;
        window.location.replace(`/login?next=${encodeURIComponent("/agent")}`);
        return <FullScreenLoader label="正在打开代理后台" detail="请先登录" />;
    }
    if (user && isStreamerMarketingHost()) {
        window.location.replace(canvasWorkspaceURL());
        return <FullScreenLoader label="正在打开画布" detail="登录后进入同一套工作台" />;
    }
    return <ManchuangHomePage />;
}
