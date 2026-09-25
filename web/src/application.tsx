import React from "react";
import { createRoot } from "react-dom/client";
import "@fontsource-variable/inter";
import "@fontsource-variable/jetbrains-mono";
import "antd/dist/reset.css";
import "./styles/globals.css";
// 全局自举内置插件注册（editor-shell 等预设以模块副作用注册编辑器插槽）：
// 冷启动直达编辑器时素材/时间线等插槽不再为空。
import "@/lib/plugins/builtin";
import { RouterProvider } from "react-router";

import { AppProviders } from "@/components/layout/app-providers";
import { clearChunkReloadGuard, reloadForStaleChunk } from "@/lib/chunk-load";
import { router } from "@/router";

clearChunkReloadGuard();
window.addEventListener("unhandledrejection", (event) => {
    if (reloadForStaleChunk(event.reason)) event.preventDefault();
});
window.addEventListener("error", (event) => {
    reloadForStaleChunk(event.error || event.message);
});

document.body.style.fontFamily = '"Inter Variable","PingFang SC","Microsoft YaHei","Noto Sans SC","Segoe UI","Helvetica Neue",sans-serif';

createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
        <AppProviders>
            <RouterProvider router={router} />
        </AppProviders>
    </React.StrictMode>,
);
