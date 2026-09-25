export type PublicHomepage = "welcome" | "manchuang";

export const PUBLIC_HOME_HREF = "/";

export function normalizePublicHomepage(value: string | undefined | null): PublicHomepage {
    return value === "manchuang" ? "manchuang" : "welcome";
}

/** 登录页「返回首页」必须整页跳到根路径，交给 RootHome 再分发；不能走 React Router，否则会停在 /login。 */
export function publicHomeHref(_homepage?: string | null) {
    return PUBLIC_HOME_HREF;
}

export function goPublicHome(_homepage?: string | null) {
    window.location.assign(PUBLIC_HOME_HREF);
}

export function handlePublicHomeClick(event: { preventDefault(): void }, homepage?: string | null) {
    event.preventDefault();
    goPublicHome(homepage);
}
