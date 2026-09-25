import { expect, test } from "bun:test";

import { appearanceLogoURL, normalizePublicAppearance } from "../src/stores/use-appearance-store";

test("initial HTML stays brand neutral until the public appearance is resolved", async () => {
    const [html, mainSource] = await Promise.all([Bun.file(new URL("../index.html", import.meta.url)).text(), Bun.file(new URL("../src/main.tsx", import.meta.url)).text()]);

    expect(html).not.toContain("影策");
    expect(html).not.toContain("fonts.googleapis.com");
    expect(html).toContain("/favicon.png");
    expect(html).toContain("<title>画布TV</title>");
    expect(html).toContain('property="og:title" content="画布TV"');
    expect(html).toContain('property="og:image" content="https://canvas.j11.net/og-image.png?v=1.5.39"');
    expect(mainSource).toContain("restoreCachedAppearance()");
    expect(mainSource.indexOf("restoreCachedAppearance()")).toBeLessThan(mainSource.indexOf("bootstrapAppearance()"));
    expect(mainSource.indexOf("bootstrapAppearance()")).toBeLessThan(mainSource.indexOf('import("./application")'));
});

test("public appearance is cached locally so the next visit can paint without waiting for the network", async () => {
    const source = await Bun.file(new URL("../src/services/appearance-bootstrap.ts", import.meta.url)).text();
    expect(source).toContain("APPEARANCE_CACHE_KEY");
    expect(source).toContain("localStorage.setItem(APPEARANCE_CACHE_KEY");
    expect(source).toContain("restoreCachedAppearance");
});

test("login hydrates local workspace before waiting for the model catalog", async () => {
    const source = await Bun.file(new URL("../src/lib/user-session.ts", import.meta.url)).text();
    const hydrateIndex = source.indexOf("setHydrated(true)");
    const catalogIndex = source.indexOf("void refreshLocalModelCatalog");
    expect(catalogIndex).toBeGreaterThan(0);
    expect(hydrateIndex).toBeGreaterThan(catalogIndex);
    expect(source).toContain("已先使用本地配置");
});

test("a custom login video never falls back to the built-in poster", () => {
    const appearance = normalizePublicAppearance({
        brandName: "HIMA Studio",
        brandSlug: "hima-studio",
        authHeroTitle: "把灵感，\n变成可见的故事。",
        authHeroDescription: "从同一个创作空间持续推进。",
        authVideoConfigured: true,
        authVideoUrl: "/api/public/appearance/assets/video?v=next",
        authVideoPosterConfigured: false,
        authVideoPosterUrl: "",
    });

    expect(appearance.brandName).toBe("HIMA Studio");
    expect(appearance.brandSlug).toBe("hima-studio");
    expect(appearance.authHeroTitle).toBe("把灵感，\n变成可见的故事。");
    expect(appearance.authHeroDescription).toBe("从同一个创作空间持续推进。");
    expect(appearance.authVideoPosterUrl).toBe("");
    expect(appearance.authVideoAutoplay).toBe(true);
});

test("login video autoplay defaults on and can be disabled explicitly", () => {
    expect(normalizePublicAppearance({}).authVideoAutoplay).toBe(true);
    expect(normalizePublicAppearance({ authVideoAutoplay: false }).authVideoAutoplay).toBe(false);
});

test("appearance URLs reject executable and insecure remote schemes", () => {
    const appearance = normalizePublicAppearance({
        logoConfigured: true,
        logoUrl: "javascript:alert(1)",
        authVideoConfigured: true,
        authVideoUrl: "http://example.com/brand.mp4",
    });

    expect(appearance.logoUrl).toBe("/logo.png");
    expect(appearance.authVideoUrl).not.toContain("example.com");
});

test("appearance selects theme logos and falls back to the single configured logo", () => {
    const dual = normalizePublicAppearance({
        logoConfigured: true,
        darkLogoConfigured: true,
        logoUrl: "/api/public/appearance/assets/logo?v=dual",
        darkLogoUrl: "/api/public/appearance/assets/logo-dark?v=dual",
        logoFrameEnabled: false,
    });
    expect(appearanceLogoURL(dual, "light")).toContain("/logo?");
    expect(appearanceLogoURL(dual, "dark")).toContain("/logo-dark?");
    expect(dual.logoFrameEnabled).toBe(false);

    const single = normalizePublicAppearance({ logoConfigured: true, logoUrl: "/api/public/appearance/assets/logo?v=single" });
    expect(appearanceLogoURL(single, "light")).toBe(single.logoUrl);
    expect(appearanceLogoURL(single, "dark")).toBe(single.logoUrl);
    expect(single.logoFrameEnabled).toBe(true);
});

test("public homepage defaults to the original welcome page and can switch to manchuang", async () => {
    expect(normalizePublicAppearance({}).publicHomepage).toBe("welcome");
    expect(normalizePublicAppearance({ publicHomepage: "manchuang" }).publicHomepage).toBe("manchuang");
    expect(normalizePublicAppearance({ homeCtaLabel: "立即开始", homeNavItems: [{ label: "定价", href: "#pricing" }] }).homeCtaLabel).toBe("立即开始");
    expect(normalizePublicAppearance({ homeNavItems: [{ label: "定价", href: "#pricing" }] }).homeNavItems).toEqual([{ label: "定价", href: "#pricing", openInNewTab: false }]);
    expect(normalizePublicAppearance({ publicHomepage: "unknown" as "welcome" }).publicHomepage).toBe("welcome");
    const [routerSource, rootSource, pageSource] = await Promise.all([
        Bun.file(new URL("../src/router.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/public-home/root-home.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/public-home/manchuang-home.tsx", import.meta.url)).text(),
    ]);
    expect(routerSource).toContain("RootHome");
    expect(rootSource).toContain("ManchuangHomePage");
    expect(rootSource).toContain('window.location.replace("/welcome")');
    expect(rootSource).not.toContain('Navigate to="/login"');
    expect(pageSource).toContain("mc-hero");
    expect(pageSource).toContain("绘无限");
    expect(pageSource).toContain("mc-public-home");
    expect(pageSource).toContain("mc-hero-showcase");
    expect(pageSource).toContain("HeroRail");
    expect(pageSource).toContain("is-lit");
    expect(pageSource).not.toContain("最近项目");
    const landingCss = await Bun.file(new URL("../src/pages/public-home/manchuang-home.css", import.meta.url)).text();
    expect(landingCss).toContain("overflow-y: auto");
    expect(landingCss).toContain("Zhi Mang Xing");
    expect(landingCss).toContain("--flow-fill");
    expect(landingCss).not.toContain("STKaiti");
});

test("auth scene consumes resolved appearance instead of hardcoded media constants", async () => {
    const source = await Bun.file(new URL("../src/pages/auth/auth-scene.tsx", import.meta.url)).text();

    expect(source).toContain("appearance.authVideoUrl");
    expect(source).toContain("appearance.authVideoAutoplay");
    expect(source).toContain("appearance.authVideoPosterUrl || undefined");
    expect(source).toContain("appearance.brandName");
    expect(source).toContain("appearance.authHeroTitle");
    expect(source).toContain("appearance.authHeroDescription");
    expect(source).toContain('theme="dark"');
    expect(source).toContain("handlePublicHomeClick");
    expect(source).toContain("PUBLIC_HOME_HREF");
    expect(source).toContain("auth.backHome");
    expect(source).not.toContain("<Link");
    expect(source).not.toContain("让一个故事，");
    expect(source).not.toContain("AUTH_VIDEO_URL");
    expect(source).not.toContain("AUTH_VIDEO_POSTER");
});

test("appearance management exposes light and dark logo uploads plus the frame switch", async () => {
    const [pageSource, brandSource, adminStyles, globalStyles] = await Promise.all([
        Bun.file(new URL("../src/pages/admin/settings/appearance-settings-page.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/components/brand/brand-logo.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/styles/admin-ui.css", import.meta.url)).text(),
        Bun.file(new URL("../src/styles/globals.css", import.meta.url)).text(),
    ]);

    expect(pageSource).toContain('slot="logo-dark"');
    expect(pageSource).toContain("浅色模式 Logo");
    expect(pageSource).toContain("深色模式 Logo（可选）");
    expect(pageSource).toContain("禁用 Logo 后面的圆角矩形外框");
    expect(pageSource).toContain("<Switch");
    expect(pageSource).toContain("checked={!logoFrameEnabled}");
    expect(pageSource).toContain("setLogoFrameEnabled(!checked)");
    expect(pageSource).not.toContain("<Checkbox");
    expect(pageSource).toContain("深浅模式 Logo 预览");
    expect(pageSource).toContain("登录页视频自动播放");
    expect(pageSource).toContain("authVideoAutoplay");
    expect(brandSource).toContain("use-theme-store");
    expect(brandSource).toContain("data-logo-frame-enabled");
    expect(brandSource).toContain("failedSource === source");
    expect(brandSource).toContain('aria-hidden="true"');
    expect(brandSource).toContain('style.visibility = "hidden"');
    expect(brandSource).toContain("setFailedSource(source)");
    expect(adminStyles).toContain(".admin-appearance-logo-preview-mark.is-unframed img");
    expect(globalStyles).toContain('.brand-logo-frame[data-logo-frame-enabled="false"] > :is(img, svg)');
});

test("object storage can adopt the configured English brand identifier without replacing saved prefixes automatically", async () => {
    const source = await Bun.file(new URL("../src/pages/admin/settings/storage-settings-page.tsx", import.meta.url)).text();

    expect(source).toContain("state.appearance.brandSlug");
    expect(source).toContain("使用品牌标识");
    expect(source).toContain('form.setFieldValue("pathPrefix", brandSlug)');
    expect(source).toContain("setting.pathPrefix || DEFAULT_OSS_PATH_PREFIX");
});

test("appearance management exposes a server-side reset to the built-in Yingce brand", async () => {
    const [pageSource, apiSource] = await Promise.all([Bun.file(new URL("../src/pages/admin/settings/appearance-settings-page.tsx", import.meta.url)).text(), Bun.file(new URL("../src/services/api/appearance.ts", import.meta.url)).text()]);

    expect(pageSource).toContain("恢复影策默认");
    expect(pageSource).toContain("resetAdminAppearance()");
    expect(pageSource).toContain("已上传文件仍保留在存储资源中");
    expect(apiSource).toContain('http.delete<{ setting: AdminAppearance }>("/admin/settings/appearance")');
});
