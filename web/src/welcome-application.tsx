import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { bootstrapAppearance } from "@/services/appearance-bootstrap";
import { getWelcomeAvailability } from "@/services/api/welcome";
import WelcomePage from "@/pages/welcome";

async function renderWelcome() {
    try {
        const [{ welcomeEnabled }] = await Promise.all([getWelcomeAvailability(), bootstrapAppearance()]);
        if (welcomeEnabled !== true) {
            window.location.replace("/");
            return;
        }
        createRoot(document.getElementById("root")!).render(<StrictMode><WelcomePage /></StrictMode>);
    } catch (error) {
        console.error("Welcome page initialization failed", error);
        createRoot(document.getElementById("root")!).render(
            <main role="alert">
                <p>暂时无法打开欢迎页，请稍后重试。</p>
                <button onClick={() => window.location.reload()}>重试</button>
                <a href="/">返回首页</a>
            </main>,
        );
    }
}

void renderWelcome();
