export type AccessRole = "admin" | "agent" | "user";

export function normalizeAccessRole(role?: string | null): AccessRole {
    if (role === "admin" || role === "agent") return role;
    return "user";
}

export function roleLabel(role?: string | null) {
    const value = normalizeAccessRole(role);
    if (value === "admin") return "管理员";
    if (value === "agent") return "代理";
    return "普通用户";
}

export const ROLE_SELECT_OPTIONS = [
    { label: "管理员", value: "admin" },
    { label: "代理", value: "agent" },
    { label: "普通用户", value: "user" },
];
