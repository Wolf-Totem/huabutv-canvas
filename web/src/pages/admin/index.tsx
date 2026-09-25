import { canAccessAdmin } from "@/lib/access";
import { useUserStore } from "@/stores/use-user-store";
import { AdminProvider } from "./admin-context";
import { AdminShell } from "./components/admin-shell";
import "./theme/admin-tokens.css";
import "./theme/admin-chrome.css";

export default function AdminPage() {
    const actor = useUserStore((state) => state.user);
    const permissions = useUserStore((state) => state.permissions);
    const hydrated = useUserStore((state) => state.hydrated);

    if (!hydrated) return null;
    if (!canAccessAdmin(actor?.role, permissions)) {
        return (
            <main data-admin-root className="admin-unauthorized">
                <div className="admin-unauthorized-card">
                    <h1>无权限</h1>
                    <p>当前账号没有管理后台权限。</p>
                </div>
            </main>
        );
    }

    return (
        <AdminProvider>
            <AdminShell />
        </AdminProvider>
    );
}
