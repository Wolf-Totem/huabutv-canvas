import { isAgentHost, isStreamerMarketingHost } from "@/lib/public-hosts";

const AUTH_PATHS = new Set(["/login", "/register", "/forgot-password"]);

export function isAuthPath(pathname = window.location.pathname) {
    return AUTH_PATHS.has(pathname.replace(/\/+$/, "") || "/");
}

export function isRootPath(pathname = window.location.pathname) {
    return pathname === "/" || pathname === "";
}

/** 首页和登录注册走轻量壳，不启动画布工作台。已登录访客同样先看首页。 */
export function shouldUsePublicShell(hasUser: boolean, pathname = window.location.pathname, host?: string) {
    if (isAgentHost(host) && hasUser) return false;
    if (isStreamerMarketingHost(host) && hasUser) return false;
    return isRootPath(pathname) || isAuthPath(pathname);
}
