import { useEffect, useState, type ReactNode } from "react";
import { applyLocale } from "@/i18n/apply-locale";
import { FALLBACK_LOCALE, LOCALE_CHOICE_SESSION_KEY, LOCALE_STORAGE_KEY, isSupportedLocale, normalizeLocale, resolveInitialLocale, shouldPromptIpLocaleChoice, type SupportedLocale } from "@/i18n/languages";
import { LocaleChoiceModal } from "@/components/i18n/locale-choice-modal";
import { getLocaleRecommendation } from "@/services/api/locale";
import { useUserStore } from "@/stores/use-user-store";

export function I18nRuntime({ children }: { children: ReactNode }) {
    const user = useUserStore((state) => state.user);
    const hydrated = useUserStore((state) => state.hydrated);
    const ipLocalePromptEnabled = useUserStore((state) => state.features.ipLocalePromptEnabled);
    const [choice, setChoice] = useState<{ saved: SupportedLocale; recommended: SupportedLocale } | null>(null);

    useEffect(() => {
        if (!hydrated) return;
        let cancelled = false;
        void (async () => {
            const recommendation = await getLocaleRecommendation().catch(() => ({ country: "", recommended: FALLBACK_LOCALE as SupportedLocale, supported: [] }));
            const recommended = isSupportedLocale(recommendation.recommended) ? recommendation.recommended : FALLBACK_LOCALE;
            const visitor = normalizeLocale(window.localStorage.getItem(LOCALE_STORAGE_KEY));
            const saved = normalizeLocale(user?.locale);
            if (cancelled) return;
            if (user && shouldPromptIpLocaleChoice(ipLocalePromptEnabled, saved, recommended)) {
                const token = `${user.id}:${saved}:${recommended}`;
                if (sessionStorage.getItem(LOCALE_CHOICE_SESSION_KEY) !== token) {
                    setChoice({ saved, recommended });
                    await applyLocale(saved);
                    return;
                }
            }
            setChoice(null);
            await applyLocale(resolveInitialLocale({ ipLocalePromptEnabled, userLocale: saved, visitorLocale: visitor, recommended }));
        })();
        return () => {
            cancelled = true;
        };
    }, [hydrated, ipLocalePromptEnabled, user?.id, user?.locale]);

    return (
        <>
            {children}
            {choice ? <LocaleChoiceModal open saved={choice.saved} recommended={choice.recommended} onClose={() => setChoice(null)} /> : null}
        </>
    );
}
