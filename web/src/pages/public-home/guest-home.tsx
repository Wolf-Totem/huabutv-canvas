import { useEffect, useState } from "react";

import ManchuangHomePage from "@/pages/public-home/manchuang-home";
import { canvasWorkspaceURL, isAgentHost, isStreamerMarketingHost } from "@/lib/public-hosts";
import { getWelcomeAvailability } from "@/services/api/welcome";
import { getPublicSiteSkin, type PublicSiteSkin } from "@/services/api/streamer";
import { useAppearanceStore } from "@/stores/use-appearance-store";

export default function GuestHomePage() {
    const homepage = useAppearanceStore((state) => state.appearance.publicHomepage);
    const [siteSkin, setSiteSkin] = useState<PublicSiteSkin | null>(null);
    const [welcomeReady, setWelcomeReady] = useState(homepage === "manchuang" || isStreamerMarketingHost());

    useEffect(() => {
        if (isAgentHost()) {
            window.location.replace(`/login?next=${encodeURIComponent("/agent")}`);
            return;
        }
        let active = true;
        void getPublicSiteSkin()
            .then((skin) => {
                if (active) setSiteSkin(skin);
            })
            .catch(() => undefined);
        return () => {
            active = false;
        };
    }, []);

    useEffect(() => {
        if (isStreamerMarketingHost() || homepage === "manchuang") {
            setWelcomeReady(true);
            return;
        }
        let active = true;
        void getWelcomeAvailability()
            .then((result) => {
                if (!active) return;
                if (result.welcomeEnabled === true) {
                    window.location.replace("/welcome");
                    return;
                }
                setWelcomeReady(true);
            })
            .catch(() => {
                if (active) setWelcomeReady(true);
            });
        return () => {
            active = false;
        };
    }, [homepage]);

    if (isAgentHost() || !welcomeReady) return null;
    return <ManchuangHomePage heroVideoUrl={siteSkin?.heroVideoUrl} heroPosterUrl={siteSkin?.heroPosterUrl} />;
}

export function redirectLoggedInFromPublicHost() {
    if (isStreamerMarketingHost()) {
        window.location.replace(canvasWorkspaceURL());
        return true;
    }
    return false;
}
