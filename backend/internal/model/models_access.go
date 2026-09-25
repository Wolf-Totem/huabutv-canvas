package model

// 内置角色可访问的内容键。管理员始终拥有全部键；代理/普通用户由角色管理覆盖。
const (
	PermWorkspaceCreate    = "workspace.create"
	PermWorkspaceProjects  = "workspace.projects"
	PermWorkspaceCanvas    = "workspace.canvas"
	PermWorkspaceAssets    = "workspace.assets"
	PermWorkspaceSkills    = "workspace.skills"
	PermWorkspacePlugins   = "workspace.plugins"
	PermWorkspaceTasks     = "workspace.tasks"
	PermWorkspaceSettings  = "workspace.settings"
	PermAgentConsole       = "agent.console"
	PermAdminAccess        = "admin.access"
	PermAdminOverview      = "admin.overview"
	PermAdminUsers         = "admin.users"
	PermAdminRoles         = "admin.roles"
	PermAdminStreamers     = "admin.streamers"
	PermAdminChannels      = "admin.channels"
	PermAdminModels        = "admin.models"
	PermAdminPlugins       = "admin.plugins"
	PermAdminPrompts       = "admin.prompts"
	PermAdminResources     = "admin.resources"
	PermAdminAnnouncements = "admin.announcements"
	PermAdminLessons       = "admin.agent_lessons"
	PermAdminPayments      = "admin.payments"
	PermAdminCredits       = "admin.credits"
	PermAdminRedemption    = "admin.redemption"
	PermAdminLogs          = "admin.logs"
	PermAdminSettings      = "admin.settings"
	PermAdminPlaza         = "admin.plaza"
)

func ValidUserRole(role UserRole) bool {
	return role == UserRoleAdmin || role == UserRoleAgent || role == UserRoleUser
}

func RoleDisplayName(role UserRole) string {
	switch role {
	case UserRoleAdmin:
		return "管理员"
	case UserRoleAgent:
		return "代理"
	default:
		return "普通用户"
	}
}
