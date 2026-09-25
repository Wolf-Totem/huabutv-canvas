import { useEffect, useState } from "react";

import ManchuangHomePage from "@/pages/public-home/manchuang-home";
import { canvasWorkspaceURL, isAgentHost, isStreamerMarketingHost } from "@/lib/public-hosts";
import { getPublicSiteSkin, type PublicSiteSkin } from "@/services/api/streamer";

export default function GuestHomePage() {
    const [siteSkin, setSiteSkin] = useState<PublicSiteSkin | null>(null);

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

    if (isAgentHost()) {
        return <div className="mc-boot" aria-hidden="true" />;
    }
    return <ManchuangHomePage heroVideoUrl={siteSkin?.heroVideoUrl} heroPosterUrl={siteSkin?.heroPosterUrl} />;
}

export function redirectLoggedInFromPublicHost() {
    if (isStreamerMarketingHost()) {
        window.location.replace(canvasWorkspaceURL());
        return true;
    }
    return false;
}
