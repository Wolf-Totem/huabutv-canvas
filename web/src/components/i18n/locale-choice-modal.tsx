import { Modal } from "antd";
import { useTranslation } from "react-i18next";

import { applyLocale } from "@/i18n/apply-locale";
import { LOCALE_CHOICE_SESSION_KEY, type SupportedLocale } from "@/i18n/languages";
import { updateUserLocale } from "@/services/api/locale";
import { useUserStore } from "@/stores/use-user-store";

export function LocaleChoiceModal({
    open,
    saved,
    recommended,
    onClose,
}: {
    open: boolean;
    saved: SupportedLocale;
    recommended: SupportedLocale;
    onClose: () => void;
}) {
    const { t } = useTranslation("common");
    const user = useUserStore((state) => state.user);

    const choose = async (locale: SupportedLocale) => {
        sessionStorage.setItem(LOCALE_CHOICE_SESSION_KEY, `${user?.id || ""}:${saved}:${recommended}`);
        await applyLocale(locale);
        if (user) {
            const result = await updateUserLocale(locale);
            useUserStore.getState().setUser({ ...user, ...result.user, locale: result.locale });
        }
        onClose();
    };

    return (
        <Modal
            open={open}
            title={t("language.label")}
            centered
            maskClosable={false}
            footer={null}
            onCancel={onClose}
        >
            <p className="mb-4 text-sm text-foreground/75">{t("language.prompt", { saved: t(`language.${saved}`), recommended: t(`language.${recommended}`) })}</p>
            <div className="flex flex-col gap-2">
                <button type="button" className="h-9 rounded-lg bg-[var(--user-accent,#695df0)] px-3 text-sm text-white" onClick={() => void choose(saved)}>
                    {t("language.useSaved")} · {t(`language.${saved}`)}
                </button>
                <button type="button" className="h-9 rounded-lg bg-foreground/6 px-3 text-sm" onClick={() => void choose(recommended)}>
                    {t("language.useRecommended")} · {t(`language.${recommended}`)}
                </button>
            </div>
        </Modal>
    );
}
