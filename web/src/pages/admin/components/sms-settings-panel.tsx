import { App, Button, Form, Input, Skeleton } from "antd";
import { Switch } from "@/pages/admin/ui/controls";
import { AlertTriangle, BadgeCheck, KeyRound, RefreshCw, Save, Send, Smartphone } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { getAdminSmsSetting, sendAdminSmsTest, updateAdminSmsSetting, type SmsSetting } from "@/services/api/wallet";
import { AdminStatusBadge, configuredSecretText, SettingsSectionCard } from "./admin-ui";

export default function SmsSettingsPanel() {
    const { message } = App.useApp();
    const [setting, setSetting] = useState<SmsSetting | null>(null);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [testing, setTesting] = useState(false);
    const [testPhone, setTestPhone] = useState("");
    const [form] = Form.useForm();

    const load = useCallback(async () => {
        setLoading(true);
        try {
            const result = await getAdminSmsSetting();
            setSetting(result.setting);
            form.setFieldsValue({
                enabled: result.setting.enabled,
                accessKeyId: result.setting.accessKeyId,
                accessKeySecret: "",
                signName: result.setting.signName,
                templateCode: result.setting.templateCode,
            });
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取短信配置失败");
        } finally {
            setLoading(false);
        }
    }, [form, message]);

    useEffect(() => {
        void load();
    }, [load]);

    const save = async () => {
        const values = await form.validateFields();
        setSaving(true);
        try {
            const result = await updateAdminSmsSetting({
                enabled: Boolean(values.enabled),
                accessKeyId: values.accessKeyId,
                accessKeySecret: values.accessKeySecret,
                signName: values.signName,
                templateCode: values.templateCode,
                region: "cn-hangzhou",
            });
            setSetting(result.setting);
            form.setFieldValue("accessKeySecret", "");
            message.success("短信服务配置已保存");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "保存失败");
        } finally {
            setSaving(false);
        }
    };

    const sendTest = async () => {
        const phone = testPhone.trim();
        if (!phone) {
            message.error("请填写测试手机号");
            return;
        }
        if (!setting?.enabled) {
            message.error("请先保存并启用短信服务");
            return;
        }
        setTesting(true);
        try {
            const result = await sendAdminSmsTest(phone);
            message.success(`测试短信已发到 ${result.to || phone}`);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "测试短信发送失败");
        } finally {
            setTesting(false);
        }
    };

    if (loading && !setting) {
        return (
            <div className="admin-settings-stack">
                <Skeleton active paragraph={{ rows: 8 }} />
            </div>
        );
    }

    if (!setting) {
        return (
            <div className="admin-settings-stack">
                <div className="admin-email-load-error" role="alert">
                    <AlertTriangle className="size-5" />
                    <div>
                        <h2>无法读取短信服务配置</h2>
                    </div>
                    <Button icon={<RefreshCw className="size-4" />} onClick={() => void load()}>
                        重新读取
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className="admin-settings-stack">
            <SettingsSectionCard
                icon={<Smartphone className="size-4" />}
                title="阿里云短信服务"
                description="用于注册验证码。签名和模板需在阿里云短信控制台审核通过。服务器仍在青岛，只调用国内 dysmsapi。"
                status={<AdminStatusBadge label={setting.enabled ? "已启用" : "未启用"} tone={setting.enabled ? "success" : "neutral"} />}
                footer={
                    <div className="flex flex-wrap items-center gap-2">
                        <Input value={testPhone} onChange={(event) => setTestPhone(event.target.value)} placeholder="测试手机号" style={{ width: 180 }} />
                        <Button icon={<Send className="size-4" />} loading={testing} disabled={!setting.enabled || saving} onClick={() => void sendTest()}>
                            发送测试短信
                        </Button>
                        <Button type="primary" icon={<Save className="size-4" />} loading={saving} onClick={() => void save()}>
                            保存设置
                        </Button>
                    </div>
                }
            >
                <Form form={form} layout="vertical">
                    <div className="mb-4 flex items-center justify-between gap-4">
                        <div>
                            <strong>发送注册短信验证码</strong>
                            <p className="mt-1 text-sm text-foreground/60">启用后，注册页可以选择短信验证。模板变量名必须是 code。</p>
                        </div>
                        <Form.Item name="enabled" valuePropName="checked" className="!mb-0">
                            <Switch />
                        </Form.Item>
                    </div>
                    <div className="grid gap-4 md:grid-cols-2">
                        <Form.Item name="accessKeyId" label="AccessKey ID">
                            <Input autoComplete="off" placeholder="LTAI..." />
                        </Form.Item>
                        <Form.Item name="accessKeySecret" label={setting.hasAccessKeySecret ? `AccessKey Secret（${configuredSecretText}）` : "AccessKey Secret"}>
                            <Input.Password autoComplete="new-password" placeholder={setting.hasAccessKeySecret ? "留空保留原密钥" : "AccessKey Secret"} />
                        </Form.Item>
                        <Form.Item name="signName" label="短信签名">
                            <Input placeholder="已审核的签名名称" />
                        </Form.Item>
                        <Form.Item name="templateCode" label="模板 CODE">
                            <Input placeholder="SMS_********" />
                        </Form.Item>
                    </div>
                    <p className="flex items-center gap-2 text-xs text-foreground/50">
                        <BadgeCheck className="size-3.5" />
                        Region 固定 cn-hangzhou。密钥加密保存，页面不回显明文。
                    </p>
                    <p className="mt-2 flex items-center gap-2 text-xs text-foreground/50">
                        <KeyRound className="size-3.5" />
                        RAM 子账号需要短信发送权限；签名和验证码模板要先在阿里云控制台过审。
                    </p>
                </Form>
            </SettingsSectionCard>
        </div>
    );
}
