import { App, Button, Checkbox, Spin } from "antd";
import { useEffect, useMemo, useState } from "react";

import { getAdminRoleCatalog, updateAdminRolePermissions, type RoleAccessCatalog, type RoleAccessView } from "@/services/api/auth";

export default function RolesPanel() {
    const { message } = App.useApp();
    const [catalog, setCatalog] = useState<RoleAccessCatalog | null>(null);
    const [draft, setDraft] = useState<Record<string, string[]>>({});
    const [savingRole, setSavingRole] = useState("");

    const reload = async () => {
        const data = await getAdminRoleCatalog();
        setCatalog(data);
        const next: Record<string, string[]> = {};
        for (const role of data.roles || []) next[role.role] = [...(role.permissions || [])];
        setDraft(next);
    };

    useEffect(() => {
        void reload().catch((error) => message.error(error instanceof Error ? error.message : "读取角色失败"));
    }, [message]);

    const groups = useMemo(() => {
        const map = new Map<string, RoleAccessCatalog["catalog"]>();
        for (const item of catalog?.catalog || []) {
            const list = map.get(item.group) || [];
            list.push(item);
            map.set(item.group, list);
        }
        return [...map.entries()];
    }, [catalog]);

    if (!catalog) {
        return (
            <div className="flex justify-center py-16">
                <Spin />
            </div>
        );
    }

    const save = async (role: RoleAccessView) => {
        setSavingRole(role.role);
        try {
            const data = await updateAdminRolePermissions(role.role, draft[role.role] || []);
            setCatalog(data);
            const next: Record<string, string[]> = {};
            for (const item of data.roles || []) next[item.role] = [...(item.permissions || [])];
            setDraft(next);
            message.success(`${role.label}权限已保存`);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "保存失败");
        } finally {
            setSavingRole("");
        }
    };

    const toggle = (role: string, key: string, checked: boolean) => {
        setDraft((current) => {
            const set = new Set(current[role] || []);
            if (checked) set.add(key);
            else set.delete(key);
            return { ...current, [role]: [...set] };
        });
    };

    return (
        <div className="grid gap-8">
            <p className="text-sm text-foreground/65">
                三个内置角色：管理员、代理、普通用户。管理员始终拥有全部权限。代理和普通用户可以勾选能进入的工作台、代理后台和管理页面。
            </p>
            <div className="grid gap-6 xl:grid-cols-3">
                {catalog.roles.map((role) => (
                    <section key={role.role} className="rounded-2xl border border-border p-5">
                        <header className="mb-4 flex items-center justify-between gap-3">
                            <div>
                                <h3 className="text-lg font-semibold">{role.label}</h3>
                                <p className="text-xs text-foreground/45">{role.locked ? "系统锁定，不能改" : "保存后立即对这个角色的账号生效"}</p>
                            </div>
                            {role.locked ? null : (
                                <Button type="primary" size="small" loading={savingRole === role.role} onClick={() => void save(role)}>
                                    保存
                                </Button>
                            )}
                        </header>
                        <div className="grid gap-4">
                            {groups.map(([group, items]) => (
                                <div key={group}>
                                    <div className="mb-2 text-xs font-medium text-foreground/50">{group}</div>
                                    <div className="grid gap-2">
                                        {items.map((item) => (
                                            <label key={item.key} className="flex items-center gap-2 text-sm">
                                                <Checkbox
                                                    checked={(draft[role.role] || []).includes(item.key)}
                                                    disabled={role.locked}
                                                    onChange={(event) => toggle(role.role, item.key, event.target.checked)}
                                                />
                                                <span>{item.label}</span>
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </section>
                ))}
            </div>
        </div>
    );
}
