import i18n from "@/i18n/config";
import { htmlLang } from "@/i18n/display";
import { LOCALE_STORAGE_KEY, type SupportedLocale } from "@/i18n/languages";

export async function applyLocale(locale: SupportedLocale) {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, locale);
    document.documentElement.lang = htmlLang(locale);
    document.documentElement.setAttribute("data-locale", locale);
    if (i18n.language !== locale) await i18n.changeLanguage(locale);
}
