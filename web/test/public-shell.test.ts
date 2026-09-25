import { expect, test } from "bun:test";

import { isAuthPath, isRootPath, shouldUsePublicShell } from "../src/lib/public-shell";

test("guest marketing and auth paths use the public shell", () => {
    expect(isRootPath("/")).toBe(true);
    expect(isAuthPath("/login")).toBe(true);
    expect(isAuthPath("/register")).toBe(true);
    expect(shouldUsePublicShell(false, "/")).toBe(true);
    expect(shouldUsePublicShell(false, "/login")).toBe(true);
    expect(shouldUsePublicShell(false, "/projects")).toBe(false);
});

test("homepage stays on the public shell even after sign-in", () => {
    expect(shouldUsePublicShell(true, "/")).toBe(true);
    expect(shouldUsePublicShell(true, "/login")).toBe(true);
    expect(shouldUsePublicShell(true, "/create")).toBe(false);
});

test("bootstrap chooses a public entry before the workspace application", async () => {
    const mainSource = await Bun.file(new URL("../src/main.tsx", import.meta.url)).text();
    expect(mainSource).toContain("isRootPath");
    expect(mainSource).toContain('import("./public-application")');
    expect(mainSource.indexOf('import("./public-application")')).toBeLessThan(mainSource.indexOf('import("./application")'));
});

test("nginx caches manchuang media instead of no-store", async () => {
    const nginx = await Bun.file(new URL("../../nginx.conf", import.meta.url)).text();
    expect(nginx).toContain("location ^~ /manchuang/");
    expect(nginx).toContain("location ^~ /fonts/");
    expect(nginx).toContain("location ^~ /welcome/");
});

test("public application does not boot canvas plugins or workspace layout", async () => {
    const source = await Bun.file(new URL("../src/public-application.tsx", import.meta.url)).text();
    expect(source).not.toContain("plugins/builtin");
    expect(source).not.toContain("user-layout");
    expect(source).not.toContain("AuthSessionHydrator");
});
