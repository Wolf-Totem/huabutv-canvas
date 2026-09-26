import { type FormEvent, useEffect, useRef, useState, type ReactNode } from "react";
import { App, Button, Divider, Input } from "antd";
import { ArrowRight, Info, LockKeyhole, Mail, ShieldCheck, Smartphone, TriangleAlert, UserRound } from "lucide-react";
import { useSearchParams } from "react-router";

import { useTranslation } from "react-i18next";

import { canvasWorkspaceURL, isAgentHost, isStreamerMarketingHost } from "@/lib/public-hosts";
import { getAuthSettings, linuxDOLoginURL, register, sendRegistrationEmailCode, sendRegistrationSmsCode } from "@/services/api/auth";
import { LinuxDOIcon } from "./auth-scene";
import { ApiError } from "@/services/api/request";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";

type AuthSettings = Awaited<ReturnType<typeof getAuthSettings>>;

export default function RegisterPage({ embedded = false }: { embedded?: boolean }) {
    const { i18n } = useTranslation();
    const [params] = useSearchParams();
    const { message } = App.useApp();
    const dialogNext = useAuthDialogStore((state) => state.next);
    const [settings, setSettings] = useState<AuthSettings | null>(null);
    const [username, setUsername] = useState("");
    const [channel, setChannel] = useState<"email" | "sms">("email");
    const [email, setEmail] = useState("");
    const [emailCode, setEmailCode] = useState("");
    const [phone, setPhone] = useState("");
    const [smsCode, setSmsCode] = useState("");
    const [displayName, setDisplayName] = useState("");
    const [password, setPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [inviteCode, setInviteCode] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [sendingCode, setSendingCode] = useState(false);
    const [countdown, setCountdown] = useState(0);
    const [registerCountdown, setRegisterCountdown] = useState(0);
    const [formError, setFormError] = useState("");
    const sending = useRef(false),
        registering = useRef(false);
    const next = safeNext(params.get("next") || (embedded ? dialogNext : "") || null);

    useEffect(() => {
        let cancelled = false;
        void getAuthSettings()
            .then((value) => {
                if (cancelled) return;
                setSettings(value);
                if (!value.firstUser && value.smsEnabled && !value.emailEnabled) setChannel("sms");
            })
            .catch((error) => !cancelled && message.error(error instanceof Error ? error.message : "读取注册设置失败"));
        return () => {
            cancelled = true;
        };
    }, [message]);

    useEffect(() => {
        if (countdown <= 0) return;
        const timer = window.setInterval(() => setCountdown((value) => Math.max(0, value - 1)), 1000);
        return () => window.clearInterval(timer);
    }, [countdown]);

    const sendCode = async () => {
        if (sending.current || countdown > 0) return;
        if (activeChannel === "sms") {
            if (!phone.trim()) {
                message.warning("请先输入手机号");
                return;
            }
        } else if (!email.trim()) {
            message.warning("请先输入邮箱");
            return;
        }
        sending.current = true;
        setSendingCode(true);
        try {
            if (activeChannel === "sms") {
                await sendRegistrationSmsCode(phone.trim());
                message.success("验证码已发送，请查看短信");
            } else {
                await sendRegistrationEmailCode(email.trim(), i18n.language);
                message.success("验证码已发送，请检查邮箱");
            }
            setCountdown(60);
        } catch (error) {
            if (error instanceof ApiError && error.status === 429) setCountdown(Math.max(1, Math.ceil((error.retryAfterMs ?? 60000) / 1000)));
            message.error(error instanceof Error ? error.message : "发送验证码失败");
        } finally {
            sending.current = false;
            setSendingCode(false);
        }
    };

    const fail = (text: string) => {
        setFormError(text);
        message.error(text);
    };

    const submit = async (event?: FormEvent<HTMLFormElement>) => {
        event?.preventDefault();
        if (registering.current || registerCountdown > 0 || disabled) return;
        const name = username.trim();
        if (name.length < 3 || name.length > 32) {
            fail("用户名需要 3-32 位");
            return;
        }
        if (password.length < 8) {
            fail("密码至少 8 位");
            return;
        }
        if (password !== confirmPassword) {
            fail("两次输入的密码不一致");
            return;
        }
        if (!settings?.firstUser) {
            if (activeChannel === "sms") {
                if (!/^1\d{10}$/.test(phone.trim())) {
                    fail("请输入中国大陆 11 位手机号");
                    return;
                }
                if (requireCode && smsCode.trim().length !== 6) {
                    fail("请填写 6 位短信验证码");
                    return;
                }
            } else {
                if (!email.trim() || !email.includes("@")) {
                    fail("请填写有效邮箱");
                    return;
                }
                if (requireCode && emailCode.trim().length !== 6) {
                    fail("请填写 6 位邮箱验证码");
                    return;
                }
            }
        }
        registering.current = true;
        setSubmitting(true);
        setFormError("");
        try {
            await register(
                activeChannel === "sms"
                    ? { username: name, phone: phone.trim(), smsCode: smsCode.trim(), channel: "sms", displayName, password, inviteCode: inviteCode.trim() || undefined }
                    : { username: name, email: email.trim(), emailCode: emailCode.trim(), channel: "email", displayName, password, inviteCode: inviteCode.trim() || undefined },
            );
            if (!settings?.firstUser) window.sessionStorage.setItem("infinite-canvas:model-setup-guide", "1");
            message.success(settings?.firstUser ? "管理员账号已创建" : "注册成功");
            useAuthDialogStore.getState().closeAuth();
            if (isAgentHost()) window.location.replace("/agent");
            else if (isStreamerMarketingHost()) window.location.replace(canvasWorkspaceURL());
            else {
                const target = next.startsWith("/") ? next : "/";
                window.location.assign(target === "/" && embedded ? "/create" : target);
            }
        } catch (error) {
            if (error instanceof ApiError && error.status === 429) setRegisterCountdown(Math.max(1, Math.ceil((error.retryAfterMs ?? 60000) / 1000)));
            fail(error instanceof Error ? error.message : "注册失败");
        } finally {
            registering.current = false;
            setSubmitting(false);
        }
    };

    useEffect(() => {
        if (registerCountdown <= 0) return;
        const timer = window.setInterval(() => setRegisterCountdown((value) => Math.max(0, value - 1)), 1000);
        return () => window.clearInterval(timer);
    }, [registerCountdown]);

    const registrationClosed = settings?.registrationEnabled === false;
    const emailAvailable = Boolean(settings?.firstUser || settings?.emailEnabled);
    const smsAvailable = Boolean(settings?.firstUser || settings?.smsEnabled);
    const mailUnavailable = Boolean(settings && !settings.firstUser && settings.emailCodeRequired && !settings.emailEnabled);
    const smsUnavailable = Boolean(settings && !settings.firstUser && settings.smsCodeRequired && !settings.smsEnabled);
    const noChannel = Boolean(settings && !settings.firstUser && !emailAvailable && !smsAvailable);
    const activeChannel = !emailAvailable && smsAvailable ? "sms" : channel === "sms" && smsAvailable ? "sms" : "email";
    const disabled = registrationClosed || (activeChannel === "sms" ? smsUnavailable : mailUnavailable) || noChannel;
    const showChannelTabs = emailAvailable && smsAvailable && !settings?.firstUser;
    const inviteLocked = settings?.inviteLocked === true;
    const requireCode = Boolean(settings && !settings.firstUser && (activeChannel === "sms" ? settings.smsCodeRequired !== false : settings.emailCodeRequired));

    return (
        <form noValidate onSubmit={(event) => void submit(event)} className="space-y-4">
            {settings?.firstUser ? (
                <Notice icon={<Info className="size-3.5" />} tone="blue">
                    首个账号自动成为管理员，验证码暂不要求。
                </Notice>
            ) : null}
            {inviteLocked ? (
                <Notice icon={<Info className="size-3.5" />} tone="blue">
                    {settings?.inviteDisplayName || "主播"}欢迎您的加入
                </Notice>
            ) : null}
            {registrationClosed ? (
                <Notice icon={<TriangleAlert className="size-3.5" />} tone="amber">
                    当前已关闭普通注册，请联系管理员创建账号。
                </Notice>
            ) : null}
            {noChannel ? (
                <Notice icon={<TriangleAlert className="size-3.5" />} tone="amber">
                    管理员尚未配置邮箱或短信验证，暂时无法注册。
                </Notice>
            ) : activeChannel === "sms" && smsUnavailable ? (
                <Notice icon={<TriangleAlert className="size-3.5" />} tone="amber">
                    管理员尚未配置注册短信。
                </Notice>
            ) : mailUnavailable && activeChannel === "email" ? (
                <Notice icon={<TriangleAlert className="size-3.5" />} tone="amber">
                    管理员尚未配置注册邮件，普通邮箱注册暂不可用。
                </Notice>
            ) : null}

            {showChannelTabs ? (
                <div className="grid grid-cols-2 gap-2">
                    <Button htmlType="button" type={activeChannel === "email" ? "primary" : "default"} onClick={() => setChannel("email")}>
                        邮箱注册
                    </Button>
                    <Button htmlType="button" type={activeChannel === "sms" ? "primary" : "default"} onClick={() => setChannel("sms")}>
                        短信注册
                    </Button>
                </div>
            ) : null}
            {formError ? <Notice icon={<TriangleAlert className="size-3.5" />} tone="amber">{formError}</Notice> : null}

            <div className="grid gap-4 sm:grid-cols-2">
                <AuthField label="用户名">
                    <Input size="large" prefix={<UserRound className="size-4 text-white/35" />} value={username} onChange={(event) => setUsername(event.target.value)} placeholder="3-32 位字符" autoComplete="username" required disabled={disabled} />
                </AuthField>
                <AuthField label="显示名称">
                    <Input size="large" value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="不填则使用用户名" disabled={disabled} />
                </AuthField>
            </div>

            {activeChannel === "sms" ? (
                <>
                    <AuthField label="手机号">
                        <Input
                            size="large"
                            prefix={<Smartphone className="size-4 text-white/35" />}
                            value={phone}
                            onChange={(event) => setPhone(event.target.value.replace(/\D/g, "").slice(0, 11))}
                            placeholder="中国大陆 11 位手机号"
                            inputMode="tel"
                            autoComplete="tel"
                            required={!settings?.firstUser}
                            disabled={disabled}
                        />
                    </AuthField>
                    {requireCode ? (
                        <AuthField label="短信验证码">
                            <div className="grid grid-cols-[minmax(0,1fr)_116px] gap-2">
                                <Input
                                    size="large"
                                    prefix={<ShieldCheck className="size-4 text-white/35" />}
                                    value={smsCode}
                                    onChange={(event) => setSmsCode(event.target.value.replace(/\D/g, "").slice(0, 6))}
                                    placeholder="6 位验证码"
                                    inputMode="numeric"
                                    autoComplete="one-time-code"
                                    required
                                    disabled={disabled}
                                />
                                <Button htmlType="button" size="large" loading={sendingCode} disabled={disabled || countdown > 0} onClick={() => void sendCode()}>
                                    {countdown > 0 ? `${countdown}s` : "获取验证码"}
                                </Button>
                            </div>
                        </AuthField>
                    ) : null}
                </>
            ) : (
                <>
                    <AuthField label="邮箱">
                        <Input
                            size="large"
                            prefix={<Mail className="size-4 text-white/35" />}
                            value={email}
                            onChange={(event) => setEmail(event.target.value)}
                            placeholder="用于登录与安全验证"
                            autoComplete="email"
                            required={!settings?.firstUser}
                            disabled={disabled}
                        />
                    </AuthField>
                    {requireCode ? (
                        <AuthField label="邮箱验证码">
                            <div className="grid grid-cols-[minmax(0,1fr)_116px] gap-2">
                                <Input
                                    size="large"
                                    prefix={<ShieldCheck className="size-4 text-white/35" />}
                                    value={emailCode}
                                    onChange={(event) => setEmailCode(event.target.value.replace(/\D/g, "").slice(0, 6))}
                                    placeholder="6 位验证码"
                                    inputMode="numeric"
                                    autoComplete="one-time-code"
                                    required
                                    disabled={disabled}
                                />
                                <Button htmlType="button" size="large" loading={sendingCode} disabled={disabled || countdown > 0} onClick={() => void sendCode()}>
                                    {countdown > 0 ? `${countdown}s` : "获取验证码"}
                                </Button>
                            </div>
                        </AuthField>
                    ) : null}
                </>
            )}

            <AuthField label="邀请码（可选）">
                <Input size="large" value={inviteCode} onChange={(event) => setInviteCode(event.target.value)} placeholder="填写邀请码可接受一对一指导和优先服务" disabled={disabled} />
                <span className="block pt-1 text-[11px] leading-5 text-white/45">填写邀请码可接受一对一指导和优先服务</span>
            </AuthField>

            <div className="grid gap-4 sm:grid-cols-2">
                <AuthField label="密码">
                    <Input.Password
                        size="large"
                        prefix={<LockKeyhole className="size-4 text-white/35" />}
                        value={password}
                        onChange={(event) => setPassword(event.target.value)}
                        placeholder="至少 8 位"
                        autoComplete="new-password"
                        required
                        disabled={disabled}
                    />
                </AuthField>
                <AuthField label="确认密码">
                    <Input.Password
                        size="large"
                        prefix={<LockKeyhole className="size-4 text-white/35" />}
                        value={confirmPassword}
                        onChange={(event) => setConfirmPassword(event.target.value)}
                        placeholder="再次输入密码"
                        autoComplete="new-password"
                        required
                        disabled={disabled}
                    />
                </AuthField>
            </div>

            <Button type="primary" htmlType="button" size="large" block loading={submitting} disabled={disabled || registerCountdown > 0} icon={<ArrowRight className="size-4" />} iconPlacement="end" onClick={() => void submit()}>
                {registerCountdown > 0 ? `${registerCountdown} 秒后可重试` : "创建账号"}
            </Button>
            {settings?.linuxdoEnabled ? (
                <>
                    <Divider plain className="!border-white/10 !text-white/30">
                        或
                    </Divider>
                    <Button size="large" block icon={<LinuxDOIcon />} href={linuxDOLoginURL(next)}>
                        使用 Linux.do 注册 / 登录
                    </Button>
                </>
            ) : null}
        </form>
    );
}

function AuthField({ label, children }: { label: string; children: ReactNode }) {
    return (
        <label className="block space-y-2">
            <span className="text-xs font-medium text-white/62">{label}</span>
            {children}
        </label>
    );
}

function Notice({ icon, tone, children }: { icon: ReactNode; tone: "blue" | "amber"; children: ReactNode }) {
    return (
        <div className={`flex items-start gap-2 rounded-lg border px-3 py-2.5 text-xs leading-5 ${tone === "blue" ? "border-blue-300/15 bg-blue-300/[0.06] text-blue-100/78" : "border-amber-300/15 bg-amber-300/[0.06] text-amber-100/78"}`}>
            <span className="mt-0.5 shrink-0">{icon}</span>
            {children}
        </div>
    );
}

function safeNext(value: string | null) {
    if (!value || !value.startsWith("/") || value.startsWith("//")) return "/";
    return value;
}
