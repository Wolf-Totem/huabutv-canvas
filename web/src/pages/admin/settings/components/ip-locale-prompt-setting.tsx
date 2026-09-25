import { useEffect, useState } from "react";
import { App, Button } from "antd";
import { Switch } from "@/pages/admin/ui/controls";
import { getAdminFeatureAvailability, updateAdminFeatureAvailability } from "@/services/api/auth";
import { useUserStore } from "@/stores/use-user-store";

export function IpLocalePromptSetting() {
    const { message } = App.useApp();
    const [enabled, setEnabled] = useState<boolean | null>(null);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");
    const [revision, setRevision] = useState(0);
    const setFeatures = useUserStore((state) => state.setFeatures);

    useEffect(() => {
        let active = true;
        setError("");
        getAdminFeatureAvailability()
            .then(({ features }) => {
                if (active) setEnabled(features.ipLocalePromptEnabled === true);
            })
            .catch(() => {
                if (active) setError("读取 IP 语言提示失败，请重试。");
            });
        return () => {
            active = false;
        };
    }, [revision]);

    async function change(value: boolean) {
        if (saving) return;
        setSaving(true);
        try {
            const { features } = await updateAdminFeatureAvailability({ ipLocalePromptEnabled: value });
            if (features.ipLocalePromptEnabled !== value) throw new Error("IP 语言提示未保存，请重试");
            setEnabled(features.ipLocalePromptEnabled);
            setFeatures(features);
            message.success(value ? "已开启 IP 语言提示" : "已关闭 IP 语言提示，登录后不再弹出切换语言");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "保存 IP 语言提示失败");
        } finally {
            setSaving(false);
        }
    }

    return (
        <div className="admin-appearance-logo-frame-option">
            <div className="admin-appearance-logo-frame-copy">
                <strong>根据 IP 提示切换语言</strong>
                <p>默认关闭。开启后，会按访问 IP 推荐语言：未选过语言的访客自动切换，登录后若账号语言与推荐不一致则弹出提示。关闭后不再按 IP 改语言、也不再弹窗。切换后立即保存。</p>
                {error ? (
                    <div role="alert">
                        {error}
                        <Button onClick={() => setRevision((value) => value + 1)}>重试</Button>
                    </div>
                ) : null}
            </div>
            <div className="admin-appearance-logo-frame-control">
                <span>{enabled === null ? "读取中" : saving ? "保存中" : enabled ? "已开启" : "已关闭"}</span>
                <Switch aria-label="根据 IP 提示切换语言" checked={enabled === true} disabled={enabled === null || saving || Boolean(error)} onChange={(value) => void change(value)} />
            </div>
        </div>
    );
}
