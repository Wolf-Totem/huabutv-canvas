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
    expect(source).not.toContain("AppProviders");
});

test("homepage login and register dialogs render inside the public router", async () => {
    const [shell, login, register, workspaceRouter] = await Promise.all([
        Bun.file(new URL("../src/public-application.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/auth/login.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/pages/auth/register.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/router.tsx", import.meta.url)).text(),
    ]);
    expect(login).toContain("useSearchParams");
    expect(register).toContain("useSearchParams");
    expect(shell).toContain("function PublicFrame");
    expect(shell).toContain("<AuthDialogHost />");
    expect(shell).toContain("<Outlet />");
    expect(shell.indexOf("function PublicFrame")).toBeLessThan(shell.indexOf("createBrowserRouter(["));
    expect(shell).toContain("element: <PublicFrame />");
    expect(workspaceRouter).toContain("function AppFrame");
    expect(workspaceRouter).toContain("<AuthDialogHost />");
    expect(workspaceRouter).toContain("<Outlet />");
});

test("register form submits by click with visible errors above the auth overlay", async () => {
    const [register, css] = await Promise.all([
        Bun.file(new URL("../src/pages/auth/register.tsx", import.meta.url)).text(),
        Bun.file(new URL("../src/components/auth/auth-dialog.css", import.meta.url)).text(),
    ]);
    expect(register).toContain("noValidate");
    expect(register).toContain('htmlType="button"');
    expect(register).toContain("onClick={() => void submit()}");
    expect(register).toContain("setFormError");
    expect(register).toContain("请填写 6 位邮箱验证码");
    expect(register).toContain("请填写 6 位短信验证码");
    expect(css).toContain("html.mc-auth-open .ant-message");
    expect(css).toContain("z-index: 11050");
});

test("public shell wraps homepage account menu Query hooks with the shared QueryClient", async () => {
    const { existsSync } = await import("node:fs");
    const { dirname, resolve } = await import("node:path");
    const srcRoot = resolve(import.meta.dir, "../src");
    const entry = resolve(srcRoot, "public-application.tsx");
    const publicApp = await Bun.file(entry).text();
    expect(publicApp).toContain("<QueryClientProvider client={appQueryClient}>");

    const resolveImport = (fromFile: string, spec: string) => {
        if (spec.startsWith("@/")) return resolve(srcRoot, spec.slice(2));
        if (spec.startsWith(".")) return resolve(dirname(fromFile), spec);
        return "";
    };
    const withExt = (base: string) => {
        for (const suffix of ["", ".tsx", ".ts", "/index.tsx", "/index.ts"]) {
            const candidate = base + suffix;
            if (existsSync(candidate) && !candidate.endsWith("/") && (candidate.endsWith(".ts") || candidate.endsWith(".tsx"))) return candidate;
        }
        return "";
    };

    const seen = new Set<string>();
    const queue = [entry];
    const queryFiles: string[] = [];
    while (queue.length) {
        const file = queue.pop()!;
        if (seen.has(file)) continue;
        seen.add(file);
        const source = await Bun.file(file).text();
        if (source.includes("@tanstack/react-query")) queryFiles.push(file.replaceAll("\\", "/"));
        for (const match of source.matchAll(/from ["']([^"']+)["']/g)) {
            const resolved = withExt(resolveImport(file, match[1]));
            if (resolved && resolved.startsWith(srcRoot) && !seen.has(resolved)) queue.push(resolved);
        }
    }

    expect(queryFiles.some((file) => file.endsWith("/hooks/use-wallet-balance.ts"))).toBe(true);
    expect(queryFiles.some((file) => file.endsWith("/public-application.tsx"))).toBe(true);
    expect(queryFiles.some((file) => file.endsWith("/components/layout/app-providers.tsx"))).toBe(false);
});
