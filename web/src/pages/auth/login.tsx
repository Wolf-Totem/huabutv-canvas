import { type FormEvent, useEffect, useState, type ReactNode } from "react";
import { App, Button, Divider, Input } from "antd";
import { ArrowRight, LockKeyhole, UserRound } from "lucide-react";
import { Link, useSearchParams } from "react-router";

import { canvasWorkspaceURL, isAgentHost, isStreamerMarketingHost } from "@/lib/public-hosts";
import { getAuthSession, getAuthSettings, linuxDOLoginURL, login } from "@/services/api/auth";
import { useUserStore } from "@/stores/use-user-store";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import { LinuxDOIcon } from "./auth-scene";
import { useTranslation } from "react-i18next";

export default function LoginPage({ embedded = false }: { embedded?: boolean }) {
    const { t } = useTranslation("common");
    const [params] = useSearchParams();
    const { message } = App.useApp();
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [linuxdoEnabled, setLinuxdoEnabled] = useState(false);
    const dialogNext = useAuthDialogStore((state) => state.next);
    const closeAuth = useAuthDialogStore((state) => state.closeAuth);
    const next = safeNext(params.get("next") || (embedded ? dialogNext : ""));
    const forgotPasswordURL = `/forgot-password?next=${encodeURIComponent(next)}`;
    const user = useUserStore((state) => state.user);
    const hydrated = useUserStore((state) => state.hydrated);

    const afterLogin = (path: string) => {
        if (isAgentHost()) {
            window.location.replace(path.startsWith("/agent") ? path : "/agent");
            return;
        }
        if (isStreamerMarketingHost()) {
            window.location.replace(canvasWorkspaceURL());
            return;
        }
        if (embedded) {
            closeAuth();
            const target = path.startsWith("/") ? path : "/";
            if (target === "/" || target === `${window.location.pathname}${window.location.search}`) return;
            window.location.assign(target);
            return;
        }
        window.location.replace(path.startsWith("/") ? path : "/");
    };

    // 如果已登录，直接跳转
    useEffect(() => {
        if (hydrated && user) afterLogin(next);
    }, [hydrated, user, next]);

    useEffect(() => {
        void getAuthSettings()
            .then((settings) => setLinuxdoEnabled(settings.linuxdoEnabled))
            .catch((error) => {
                // 这是登录页的展示配置读取：失败时明确隐藏第三方入口，
                // 账号密码登录仍可用；不能无痕地把配置读取失败当成成功。
                console.warn("读取登录方式配置失败，已隐藏第三方登录入口", error);
            });
        const oauthError = params.get("oauth_error");
        if (oauthError) message.error(oauthError);
    }, [message, params]);

    const submit = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        setSubmitting(true);
        try {
            await login({ username, password });
            const { applyUserSession } = await import("@/lib/user-session");
            await applyUserSession(await getAuthSession());
            message.success(t("auth.success"));
            afterLogin(next);
        } catch (error) {
            message.error(error instanceof Error ? error.message : t("auth.failed"));
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <form onSubmit={submit} className="space-y-5">
            <AuthField label={t("auth.username")} htmlFor="login-account">
                <Input id="login-account" size="large" prefix={<UserRound className="size-4 text-white/35" />} value={username} onChange={(event) => setUsername(event.target.value)} placeholder={t("auth.usernamePlaceholder")} autoComplete="username" required />
            </AuthField>
            <AuthField
                label={t("auth.password")}
                htmlFor="login-password"
                action={
                    <Link
                        to={forgotPasswordURL}
                        className="-my-2 inline-flex min-h-8 items-center rounded-sm text-xs font-medium text-blue-300/80 transition-colors hover:text-blue-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-300/45"
                    >
                        {t("auth.forgot")}
                    </Link>
                }
            >
                <Input.Password
                    id="login-password"
                    size="large"
                    prefix={<LockKeyhole className="size-4 text-white/35" />}
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    placeholder={t("auth.passwordPlaceholder")}
                    autoComplete="current-password"
                    required
                />
            </AuthField>
            <Button type="primary" htmlType="submit" size="large" block loading={submitting} icon={<ArrowRight className="size-4" />} iconPlacement="end">
                {t("auth.login")}
            </Button>
            {linuxdoEnabled ? (
                <>
                    <Divider plain className="!border-white/10 !text-white/30">
                        {t("auth.or")}
                    </Divider>
                    <Button size="large" block icon={<LinuxDOIcon />} href={linuxDOLoginURL(next)}>
                        {t("auth.linuxdo")}
                    </Button>
                </>
            ) : null}
        </form>
    );
}

function AuthField({ label, htmlFor, action, children }: { label: string; htmlFor: string; action?: ReactNode; children: ReactNode }) {
    return (
        <div className="space-y-2">
            <div className="flex items-center justify-between gap-3">
                <label htmlFor={htmlFor} className="text-xs font-medium text-white/62">
                    {label}
                </label>
                {action}
            </div>
            {children}
        </div>
    );
}

function safeNext(value: string | null) {
    if (!value || !value.startsWith("/") || value.startsWith("//")) {
        if (typeof window !== "undefined") return `${window.location.pathname}${window.location.search}` || "/";
        return "/";
    }
    return value;
}
