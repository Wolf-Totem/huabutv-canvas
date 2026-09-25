package app

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const rolePermissionsSettingKey = "role_permissions"

type PermissionDefinition struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
}

type RoleAccessView struct {
	Role        string   `json:"role"`
	Label       string   `json:"label"`
	Locked      bool     `json:"locked"`
	Permissions []string `json:"permissions"`
}

type RoleAccessCatalog struct {
	Roles   []RoleAccessView       `json:"roles"`
	Catalog []PermissionDefinition `json:"catalog"`
}

type UpdateRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func PermissionCatalog() []PermissionDefinition {
	return []PermissionDefinition{
		{Key: model.PermWorkspaceCreate, Label: "创作台", Group: "工作台"},
		{Key: model.PermWorkspaceProjects, Label: "短剧项目", Group: "工作台"},
		{Key: model.PermWorkspaceCanvas, Label: "无限画布", Group: "工作台"},
		{Key: model.PermWorkspaceAssets, Label: "素材库", Group: "工作台"},
		{Key: model.PermWorkspaceSkills, Label: "Skills", Group: "工作台"},
		{Key: model.PermWorkspacePlugins, Label: "插件中心", Group: "工作台"},
		{Key: model.PermWorkspaceTasks, Label: "任务中心", Group: "工作台"},
		{Key: model.PermWorkspaceSettings, Label: "个人设置", Group: "工作台"},
		{Key: model.PermAgentConsole, Label: "代理后台", Group: "代理"},
		{Key: model.PermAdminAccess, Label: "进入管理后台", Group: "管理后台"},
		{Key: model.PermAdminOverview, Label: "数据概览", Group: "管理后台"},
		{Key: model.PermAdminUsers, Label: "用户管理", Group: "管理后台"},
		{Key: model.PermAdminRoles, Label: "角色管理", Group: "管理后台"},
		{Key: model.PermAdminStreamers, Label: "主播代理", Group: "管理后台"},
		{Key: model.PermAdminChannels, Label: "系统渠道", Group: "管理后台"},
		{Key: model.PermAdminModels, Label: "前台模型", Group: "管理后台"},
		{Key: model.PermAdminPlugins, Label: "插件管理", Group: "管理后台"},
		{Key: model.PermAdminPrompts, Label: "提示词模板", Group: "管理后台"},
		{Key: model.PermAdminResources, Label: "存储资源", Group: "管理后台"},
		{Key: model.PermAdminAnnouncements, Label: "系统公告", Group: "管理后台"},
		{Key: model.PermAdminLessons, Label: "Agent 记忆", Group: "管理后台"},
		{Key: model.PermAdminPayments, Label: "支付充值", Group: "管理后台"},
		{Key: model.PermAdminCredits, Label: "积分运营", Group: "管理后台"},
		{Key: model.PermAdminRedemption, Label: "兑换码", Group: "管理后台"},
		{Key: model.PermAdminLogs, Label: "请求明细", Group: "管理后台"},
		{Key: model.PermAdminSettings, Label: "系统配置", Group: "管理后台"},
		{Key: model.PermAdminPlaza, Label: "作品广场", Group: "管理后台"},
	}
}

func AllPermissionKeys() []string {
	catalog := PermissionCatalog()
	out := make([]string, 0, len(catalog))
	for _, item := range catalog {
		out = append(out, item.Key)
	}
	return out
}

func DefaultRolePermissions(role model.UserRole) []string {
	workspace := []string{
		model.PermWorkspaceCreate,
		model.PermWorkspaceProjects,
		model.PermWorkspaceCanvas,
		model.PermWorkspaceAssets,
		model.PermWorkspaceSkills,
		model.PermWorkspacePlugins,
		model.PermWorkspaceTasks,
		model.PermWorkspaceSettings,
	}
	switch role {
	case model.UserRoleAdmin:
		return AllPermissionKeys()
	case model.UserRoleAgent:
		return append(append([]string{}, workspace...), model.PermAgentConsole)
	default:
		return append([]string{}, workspace...)
	}
}

func knownPermissionSet() map[string]struct{} {
	out := make(map[string]struct{}, len(PermissionCatalog()))
	for _, item := range PermissionCatalog() {
		out[item.Key] = struct{}{}
	}
	return out
}

func normalizePermissionList(role model.UserRole, keys []string) []string {
	if role == model.UserRoleAdmin {
		return AllPermissionKeys()
	}
	known := knownPermissionSet()
	seen := map[string]struct{}{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if _, ok := known[key]; !ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	hasAdminPage := false
	for key := range seen {
		if strings.HasPrefix(key, "admin.") && key != model.PermAdminAccess {
			hasAdminPage = true
			break
		}
	}
	if hasAdminPage {
		if _, ok := seen[model.PermAdminAccess]; !ok {
			out = append(out, model.PermAdminAccess)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Service) rolePermissionMap() (map[string][]string, error) {
	out := map[string][]string{
		string(model.UserRoleUser):  DefaultRolePermissions(model.UserRoleUser),
		string(model.UserRoleAgent): DefaultRolePermissions(model.UserRoleAgent),
		string(model.UserRoleAdmin): DefaultRolePermissions(model.UserRoleAdmin),
	}
	if s == nil || s.repo == nil {
		return out, nil
	}
	setting, err := s.repo.SystemSetting(rolePermissionsSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	stored := map[string][]string{}
	if strings.TrimSpace(setting.ValueJSON) != "" {
		if err := json.Unmarshal([]byte(setting.ValueJSON), &stored); err != nil {
			return nil, err
		}
	}
	if keys, ok := stored[string(model.UserRoleUser)]; ok {
		out[string(model.UserRoleUser)] = normalizePermissionList(model.UserRoleUser, keys)
	}
	if keys, ok := stored[string(model.UserRoleAgent)]; ok {
		out[string(model.UserRoleAgent)] = normalizePermissionList(model.UserRoleAgent, keys)
	}
	out[string(model.UserRoleAdmin)] = DefaultRolePermissions(model.UserRoleAdmin)
	return out, nil
}

func (s *Service) PermissionsForRole(role model.UserRole) ([]string, error) {
	if role == model.UserRoleAdmin {
		return DefaultRolePermissions(model.UserRoleAdmin), nil
	}
	m, err := s.rolePermissionMap()
	if err != nil {
		return nil, err
	}
	if keys, ok := m[string(role)]; ok {
		return append([]string{}, keys...), nil
	}
	return DefaultRolePermissions(model.UserRoleUser), nil
}

func (s *Service) PermissionsForUser(user *model.User) ([]string, error) {
	if user == nil {
		return []string{}, nil
	}
	return s.PermissionsForRole(user.Role)
}

func (s *Service) UserHasPermission(user *model.User, key string) bool {
	if user == nil || strings.TrimSpace(key) == "" {
		return false
	}
	if user.Role == model.UserRoleAdmin {
		return true
	}
	keys, err := s.PermissionsForRole(user.Role)
	if err != nil {
		return false
	}
	for _, item := range keys {
		if item == key {
			return true
		}
	}
	return false
}

func (s *Service) RequirePermission(user *model.User, keys ...string) error {
	if user == nil {
		return Unauthorized("请先登录")
	}
	if user.Role == model.UserRoleAdmin {
		return nil
	}
	for _, key := range keys {
		if s.UserHasPermission(user, key) {
			return nil
		}
	}
	return Forbidden("当前角色无权访问该内容")
}

func (s *Service) AdminRoleCatalog(actor *model.User) (*RoleAccessCatalog, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	m, err := s.rolePermissionMap()
	if err != nil {
		return nil, err
	}
	return &RoleAccessCatalog{
		Catalog: PermissionCatalog(),
		Roles: []RoleAccessView{
			{Role: string(model.UserRoleAdmin), Label: model.RoleDisplayName(model.UserRoleAdmin), Locked: true, Permissions: m[string(model.UserRoleAdmin)]},
			{Role: string(model.UserRoleAgent), Label: model.RoleDisplayName(model.UserRoleAgent), Locked: false, Permissions: m[string(model.UserRoleAgent)]},
			{Role: string(model.UserRoleUser), Label: model.RoleDisplayName(model.UserRoleUser), Locked: false, Permissions: m[string(model.UserRoleUser)]},
		},
	}, nil
}

func (s *Service) AdminUpdateRolePermissions(actor *model.User, roleName string, req UpdateRolePermissionsRequest) (*RoleAccessCatalog, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	role := model.UserRole(strings.TrimSpace(roleName))
	if role == model.UserRoleAdmin {
		return nil, BadAuthRequest("管理员权限不可修改")
	}
	if role != model.UserRoleAgent && role != model.UserRoleUser {
		return nil, BadAuthRequest("未知角色")
	}
	normalized := normalizePermissionList(role, req.Permissions)
	m, err := s.rolePermissionMap()
	if err != nil {
		return nil, err
	}
	m[string(role)] = normalized
	m[string(model.UserRoleAdmin)] = DefaultRolePermissions(model.UserRoleAdmin)
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	setting := model.SystemSetting{Key: rolePermissionsSettingKey, ValueJSON: string(encoded), UpdatedBy: actor.ID}
	current, err := s.repo.SystemSetting(rolePermissionsSettingKey)
	if err == nil {
		setting.CreatedAt = current.CreatedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := s.repo.SaveSystemSetting(&setting); err != nil {
		return nil, err
	}
	if err := s.appendAdminAudit(actor, "role_permissions.update", "system_setting", rolePermissionsSettingKey, "更新角色权限", map[string]any{"role": role, "permissions": normalized}); err != nil {
		return nil, err
	}
	return s.AdminRoleCatalog(actor)
}
