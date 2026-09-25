package app

import (
	"testing"

	"infinite-canvas/backend/internal/model"
)

func TestDefaultRolePermissions(t *testing.T) {
	userKeys := DefaultRolePermissions(model.UserRoleUser)
	if containsPermission(userKeys, model.PermAgentConsole) || containsPermission(userKeys, model.PermAdminAccess) {
		t.Fatalf("user should not get agent/admin: %v", userKeys)
	}
	agentKeys := DefaultRolePermissions(model.UserRoleAgent)
	if !containsPermission(agentKeys, model.PermAgentConsole) || containsPermission(agentKeys, model.PermAdminUsers) {
		t.Fatalf("agent defaults = %v", agentKeys)
	}
	adminKeys := DefaultRolePermissions(model.UserRoleAdmin)
	if !containsPermission(adminKeys, model.PermAdminRoles) || len(adminKeys) != len(AllPermissionKeys()) {
		t.Fatalf("admin should have all keys, got %d", len(adminKeys))
	}
}

func TestAdminUpdateRolePermissions(t *testing.T) {
	svc := newStreamerTestService(t)
	admin := seedStreamerAdmin(t, svc)
	updated, err := svc.AdminUpdateRolePermissions(admin, "agent", UpdateRolePermissionsRequest{
		Permissions: []string{model.PermWorkspaceCreate, model.PermAgentConsole, model.PermAdminUsers},
	})
	if err != nil {
		t.Fatal(err)
	}
	var agent RoleAccessView
	for _, role := range updated.Roles {
		if role.Role == "agent" {
			agent = role
		}
	}
	if !containsPermission(agent.Permissions, model.PermAdminAccess) {
		t.Fatalf("granting an admin page should also grant admin.access: %v", agent.Permissions)
	}
	if _, err := svc.AdminUpdateRolePermissions(admin, "admin", UpdateRolePermissionsRequest{Permissions: []string{model.PermWorkspaceCreate}}); err == nil {
		t.Fatal("admin role must stay locked")
	}
	owner := seedStreamerUser(t, svc, "user-1", "plain")
	if err := svc.RequirePermission(owner, model.PermAgentConsole); err == nil {
		t.Fatal("plain user should not access agent console")
	}
}

func containsPermission(keys []string, want string) bool {
	for _, key := range keys {
		if key == want {
			return true
		}
	}
	return false
}
