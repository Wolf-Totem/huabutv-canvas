import i18n from "i18next";
import HttpBackend from "i18next-http-backend";
import { initReactI18next } from "react-i18next";

import { apiBaseURL } from "@/services/api/request";
import { FALLBACK_LOCALE, I18N_NAMESPACES, LOCALE_STORAGE_KEY, SUPPORTED_LOCALES, normalizeLocale } from "@/i18n/languages";

const loadPath = `${String(apiBaseURL).replace(/\/+$/, "")}/locales/{{lng}}/{{ns}}`;

void i18n
    .use(HttpBackend)
    .use(initReactI18next)
    .init({
        lng: normalizeLocale(window.localStorage.getItem(LOCALE_STORAGE_KEY)) || FALLBACK_LOCALE,
        fallbackLng: { zh: ["zh"], default: ["en"] },
        supportedLngs: [...SUPPORTED_LOCALES],
        ns: [...I18N_NAMESPACES],
        defaultNS: "common",
        interpolation: { escapeValue: false },
        backend: { loadPath },
        react: { useSuspense: false },
        load: "languageOnly",
        nonExplicitSupportedLngs: true,
    });

export default i18n;
