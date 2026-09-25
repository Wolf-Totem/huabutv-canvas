import { normalizeAccessRole, type AccessRole } from "@/lib/user-role";

export const PERMISSIONS = {
    workspaceCreate: "workspace.create",
    workspaceProjects: "workspace.projects",
    workspaceCanvas: "workspace.canvas",
    workspaceAssets: "workspace.assets",
    workspaceSkills: "workspace.skills",
    workspacePlugins: "workspace.plugins",
    workspaceTasks: "workspace.tasks",
    workspaceSettings: "workspace.settings",
    agentConsole: "agent.console",
    adminAccess: "admin.access",
    adminOverview: "admin.overview",
    adminUsers: "admin.users",
    adminRoles: "admin.roles",
    adminStreamers: "admin.streamers",
    adminChannels: "admin.channels",
    adminModels: "admin.models",
    adminPlugins: "admin.plugins",
    adminPrompts: "admin.prompts",
    adminResources: "admin.resources",
    adminAnnouncements: "admin.announcements",
    adminLessons: "admin.agent_lessons",
    adminPayments: "admin.payments",
    adminCredits: "admin.credits",
    adminRedemption: "admin.redemption",
    adminLogs: "admin.logs",
    adminSettings: "admin.settings",
    adminPlaza: "admin.plaza",
} as const;

const WORKSPACE_DEFAULTS = [
    PERMISSIONS.workspaceCreate,
    PERMISSIONS.workspaceProjects,
    PERMISSIONS.workspaceCanvas,
    PERMISSIONS.workspaceAssets,
    PERMISSIONS.workspaceSkills,
    PERMISSIONS.workspacePlugins,
    PERMISSIONS.workspaceTasks,
    PERMISSIONS.workspaceSettings,
];

const AGENT_DEFAULTS = [...WORKSPACE_DEFAULTS, PERMISSIONS.agentConsole];

export function fallbackPermissions(role?: string | null): string[] {
    const value = normalizeAccessRole(role);
    if (value === "admin") return Object.values(PERMISSIONS);
    if (value === "agent") return AGENT_DEFAULTS;
    return WORKSPACE_DEFAULTS;
}

export function effectivePermissions(role?: string | null, permissions?: string[] | null): string[] {
    if (normalizeAccessRole(role) === "admin") return Object.values(PERMISSIONS);
    if (permissions && permissions.length > 0) return permissions;
    return fallbackPermissions(role);
}

export function hasPermission(role: string | null | undefined, permissions: string[] | null | undefined, key: string) {
    if (!key) return false;
    return effectivePermissions(role, permissions).includes(key);
}

export function canAccessAdmin(role?: string | null, permissions?: string[] | null) {
    const value = normalizeAccessRole(role);
    if (value === "admin") return true;
    const keys = effectivePermissions(role, permissions);
    return keys.some((key) => key === PERMISSIONS.adminAccess || key.startsWith("admin."));
}

export function isAgentRole(role?: string | null): role is AccessRole {
    return normalizeAccessRole(role) === "agent";
}
