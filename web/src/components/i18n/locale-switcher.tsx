import { Select } from "antd";
import { Languages } from "lucide-react";
import { useTranslation } from "react-i18next";

import { applyLocale } from "@/i18n/apply-locale";
import { SUPPORTED_LOCALES, type SupportedLocale } from "@/i18n/languages";
import { updateUserLocale } from "@/services/api/locale";
import { useUserStore } from "@/stores/use-user-store";
import { cn } from "@/lib/utils";

export function LocaleSwitcher({ className, compact = false, variant = "default" }: { className?: string; compact?: boolean; variant?: "default" | "auth" }) {
    const { t, i18n } = useTranslation("common");
    const user = useUserStore((state) => state.user);
    const current = (SUPPORTED_LOCALES.includes(i18n.language as SupportedLocale) ? i18n.language : "zh") as SupportedLocale;

    const change = async (value: SupportedLocale) => {
        await applyLocale(value);
        if (user) {
            const result = await updateUserLocale(value);
            useUserStore.getState().setUser({ ...user, ...result.user, locale: result.locale });
        }
    };

    if (variant === "auth") {
        return (
            <select
                className={cn("auth-locale-select", className)}
                aria-label={t("language.label")}
                value={current}
                onChange={(event) => void change(event.target.value as SupportedLocale)}
            >
                {SUPPORTED_LOCALES.map((code) => (
                    <option key={code} value={code}>
                        {t(`language.${code}`)}
                    </option>
                ))}
            </select>
        );
    }

    const select = (
        <Select
            size="small"
            aria-label={t("language.label")}
            value={current}
            popupMatchSelectWidth={false}
            className={compact ? "app-workspace-topbar-locale-select" : undefined}
            style={{ minWidth: compact ? 108 : 148 }}
            onChange={(value) => void change(value)}
            options={SUPPORTED_LOCALES.map((code) => ({ value: code, label: t(`language.${code}`) }))}
        />
    );

    if (compact) {
        return (
            <div className={cn("app-workspace-topbar-locale", className)} title={t("language.label")}>
                <Languages className="size-3.5 shrink-0 opacity-70" aria-hidden />
                {select}
            </div>
        );
    }

    return (
        <label className={className} style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span className="text-xs text-foreground/65">{t("language.label")}</span>
            {select}
        </label>
    );
}
