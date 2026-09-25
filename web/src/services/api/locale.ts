import { http } from "@/services/api/request";
import type { LocalUser } from "@/services/api/auth";
import type { SupportedLocale } from "@/i18n/languages";

export type LocaleRecommendation = {
    country: string;
    recommended: SupportedLocale;
    supported: SupportedLocale[];
};

export function getLocaleRecommendation() {
    return http.get<LocaleRecommendation>("/locale/recommend");
}

export function updateUserLocale(locale: SupportedLocale) {
    return http.patch<{ user: LocalUser; locale: SupportedLocale }>("/me/locale", { locale });
}
