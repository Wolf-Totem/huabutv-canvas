import { expect, test } from "bun:test";

test("portal home is the site root for every visitor and create lives at /create", async () => {
    const [root, home, guest, html, router, main] = await Promise.all([
        Bun.file(new URL("../src/pages/public-home/root-home.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/public-home/manchuang-home.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/public-home/guest-home.tsx", import.meta.url)).text(),
        Bun.file(new URL("../index.html", import.meta.url)).text(),
        Bun.file(new URL("../src/router.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/main.tsx", import.meta.url)).text(),
    ]);
    expect(root).toContain("ManchuangHomePage");
    expect(root).not.toContain("CreatePage");
    expect(guest).toContain("ManchuangHomePage");
    expect(guest).not.toContain("getWelcomeAvailability");
    expect(guest).not.toContain("welcomeReady");
    expect(html).toContain("mc-boot-billboard");
    expect(html).not.toContain("portal-boot");
    expect(home).toContain("mc-hero-showcase");
    expect(home).toContain("mc-hero-create");
    expect(home).toContain("openAuth({ tab: \"login\" })");
    expect(home).toContain("mc-hero-preview");
    expect(home).not.toContain("HeroRail");
    expect(home).toContain("openAuth");
    expect(home.indexOf("mc-nav-start")).toBeLessThan(home.indexOf("LANDING_NAV.map"));
    expect(home).toContain('openWorkspace("/create#plaza")');
    expect(home).toContain("作品");
    expect(router).toContain('path: "/create"');
    expect(router).toContain("deferred(<CreatePage />)");
    expect(router).toContain('to="/?auth=login"');
    expect(main).not.toContain("peekAuthUser");
});

test("home posters are standalone banners not plaza works", async () => {
    const posters = (await import("../src/lib/home-posters.json")).default as Array<{ id: string; imageUrl: string; href: string }>;
    expect(posters.length).toBeGreaterThanOrEqual(6);
    expect(posters.every((item) => item.imageUrl.startsWith("https://") && item.href.startsWith("/"))).toBe(true);
});

test("featured plaza cards ship enough covers to fill the home grid", async () => {
    const cards = (await import("../src/lib/plaza-featured.json")).default as Array<{ coverUrl: string; name: string }>;
    expect(cards.length).toBeGreaterThanOrEqual(60);
    expect(cards.every((item) => Boolean(item.name))).toBe(true);
    expect(cards.filter((item) => Boolean(item.coverUrl)).length).toBeGreaterThanOrEqual(60);
});

test("guest agent start opens the login dialog instead of toasting session not ready", async () => {
    const source = await Bun.file(new URL("../src/pages/create/creation-agent-entry.tsx", import.meta.url)).text();
    expect(source).toContain("openAuth({ tab: \"login\"");
    expect(source).not.toContain("agent.sessionNotReady");
    expect(source).toContain("requireLogin");
});

test("register dialog keeps an optional invite code with one-to-one guidance copy", async () => {
    const source = await Bun.file(new URL("../src/pages/auth/register.tsx", import.meta.url)).text();
    expect(source).toContain("邀请码（可选）");
    expect(source).toContain("填写邀请码可接受一对一指导和优先服务");
    expect(source).toContain("inviteCode: inviteCode.trim() || undefined");
});

test("create page keeps the composer and names the feed 作品广场 with work tags", async () => {
    const [createPage, featured, router] = await Promise.all([
        Bun.file(new URL("../src/pages/create/index.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/create/creation-workspace.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/router.tsx", import.meta.url)).text(),
    ]);
    expect(createPage).toContain("creation-home-heading");
    expect(createPage).toContain("CreationAgentEntry");
    expect(createPage).not.toContain("feed=plaza");
    expect(featured).toContain(">作品广场<");
    expect(featured).toContain("WORK_TAGS");
    expect(featured).toContain("plaza-feed-grid");
    expect(featured).toContain("previewUrl");
    expect(router).toContain('path: "/plaza"');
    expect(router).toContain('path: "/plaza/:slug"');
    expect(router).toContain('path: "/u/:userId"');
});
