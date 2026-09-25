import { expect, test } from "bun:test";

import { normalizePublicHomepage, publicHomeHref, PUBLIC_HOME_HREF } from "../src/lib/public-home";

test("public homepage helper always sends guests to the root dispatcher", () => {
    expect(normalizePublicHomepage("manchuang")).toBe("manchuang");
    expect(normalizePublicHomepage("welcome")).toBe("welcome");
    expect(normalizePublicHomepage("unknown")).toBe("welcome");
    expect(publicHomeHref("welcome")).toBe("/");
    expect(publicHomeHref("manchuang")).toBe("/");
    expect(PUBLIC_HOME_HREF).toBe("/");
});

test("login back-home and root dispatcher never bounce guests to /login", async () => {
    const [authSource, rootSource, routerSource, i18nSource, helperSource] = await Promise.all([
        Bun.file(new URL("../src/pages/auth/auth-scene.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/public-home/root-home.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/router.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/components/i18n/i18n-runtime.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/lib/public-home.ts", import.meta.url)).text(),
    ]);

    expect(authSource).toContain("handlePublicHomeClick");
    expect(authSource).toContain("PUBLIC_HOME_HREF");
    expect(authSource).not.toContain('to="/"');
    expect(rootSource).toContain('window.location.replace("/welcome")');
    expect(rootSource).toContain("WelcomeHardLoad");
    expect(rootSource).toContain("ManchuangHomePage");
    expect(rootSource).toContain("isStreamerMarketingHost");
    expect(routerSource).toContain('path: "/welcome"');
    expect(i18nSource).toContain("resolveInitialLocale");
    expect(i18nSource).toContain("ipLocalePromptEnabled");
    expect(helperSource).toContain("window.location.assign(PUBLIC_HOME_HREF)");
});

test("j11.net host helpers bind agent portal and streamer landing", async () => {
    const { isAgentHost, isStreamerMarketingHost, publicAgentHost, publicCanvasHost } = await import("../src/lib/public-hosts");
    expect(publicAgentHost()).toBe("agent.j11.net");
    expect(publicCanvasHost()).toBe("canvas.j11.net");
    expect(isAgentHost("agent.j11.net")).toBe(true);
    expect(isAgentHost("a.j11.net")).toBe(false);
    expect(isStreamerMarketingHost("a.j11.net")).toBe(true);
    expect(isStreamerMarketingHost("agent.j11.net")).toBe(false);
    expect(isStreamerMarketingHost("canvas.j11.net")).toBe(false);
});

test("custom streamer home is gated and agent console is read-only", async () => {
    const [rootSource, routerSource, consoleSource] = await Promise.all([
        Bun.file(new URL("../src/pages/public-home/root-home.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/router.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/agent/agent-console.tsx", import.meta.url)).text(),
    ]);
    expect(rootSource).toContain("isStreamerMarketingHost");
    expect(rootSource).toContain("ManchuangHomePage");
    expect(routerSource).toContain('path: "/agent"');
    expect(consoleSource).toContain("只读");
    expect(consoleSource).not.toContain("/credits/adjust");
    expect(consoleSource).not.toContain("http.post");
});

test("auth scene keeps cinematic dark tokens even in light workspace theme", async () => {
    const css = await Bun.file(new URL("../src/styles/globals.css", import.meta.url)).text();
    expect(css).toContain(".auth-scene {");
    expect(css).toContain("color-scheme: dark;");
    expect(css).toContain("--auth-card-bg: rgba(18, 19, 24, 0.94);");
});
