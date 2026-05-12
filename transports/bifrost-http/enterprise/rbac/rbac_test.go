package rbac

import (
	"testing"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger implements schemas.Logger for testing
type mockLogger struct{}

func (m *mockLogger) SetLevel(level schemas.LogLevel)                        {}
func (m *mockLogger) SetOutputType(outputType schemas.LoggerOutputType)      {}
func (m *mockLogger) Debug(msg string, args ...any)                          {}
func (m *mockLogger) Info(msg string, args ...any)                           {}
func (m *mockLogger) Warn(msg string, args ...any)                           {}
func (m *mockLogger) Error(msg string, args ...any)                          {}
func (m *mockLogger) Fatal(msg string, args ...any)                          {}
func (m *mockLogger) LogHTTPRequest(level schemas.LogLevel, msg string) schemas.LogEventBuilder {
	return schemas.NoopLogEvent
}

func TestInit(t *testing.T) {
	p := Init(nil, &mockLogger{})
	require.NotNil(t, p)
	assert.Equal(t, PluginName, p.GetName())
	assert.Nil(t, p.Cleanup())

	// System roles should be loaded
	assert.Equal(t, 3, len(p.GetRoles()))
}

func TestInit_WithConfig(t *testing.T) {
	cfg := &Config{
		Roles: []Role{
			{ID: "custom:editor", Name: "Editor", IsSystem: false, Permissions: map[Resource][]Permission{
				ResourcePrompts: {PermissionCreate, PermissionRead},
			}},
		},
		Assignments: []UserRoleAssignment{
			{UserID: "user-1", RoleIDs: []string{"system:viewer"}},
		},
	}

	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)

	roles := p.GetRoles()
	assert.GreaterOrEqual(t, len(roles), 4) // 3 system + 1 custom

	// Assignment should work
	assert.True(t, p.Can("user-1", ResourcePrompts, PermissionRead))
}

func TestSystemRoles(t *testing.T) {
	p := Init(nil, &mockLogger{})

	// Admin can do everything
	p.AssignRoles("admin-user", []string{SystemRoleAdmin.ID})
	assert.True(t, p.Can("admin-user", ResourceVirtualKeys, PermissionDelete))
	assert.True(t, p.Can("admin-user", ResourceSystemConfig, PermissionCreate))
	assert.True(t, p.IsAdmin("admin-user"))

	// Viewer can only read
	p.AssignRoles("viewer-user", []string{SystemRoleViewer.ID})
	assert.True(t, p.Can("viewer-user", ResourcePrompts, PermissionRead))
	assert.False(t, p.Can("viewer-user", ResourcePrompts, PermissionCreate))
	assert.False(t, p.IsAdmin("viewer-user"))

	// Developer can read most + manage prompts
	p.AssignRoles("dev-user", []string{SystemRoleDeveloper.ID})
	assert.True(t, p.Can("dev-user", ResourcePrompts, PermissionCreate))
	assert.False(t, p.Can("dev-user", ResourceVirtualKeys, PermissionCreate))
}

func TestCan_NoUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	assert.False(t, p.Can("", ResourceVirtualKeys, PermissionRead))
	assert.False(t, p.Can("unknown", ResourceVirtualKeys, PermissionRead))
}

func TestAddRole(t *testing.T) {
	p := Init(nil, &mockLogger{})

	err := p.AddRole(&Role{
		ID:   "custom:test",
		Name: "Test Role",
		Permissions: map[Resource][]Permission{
			ResourceAuditLogs: {PermissionRead},
		},
	})
	require.NoError(t, err)

	p.AssignRoles("test-user", []string{"custom:test"})
	assert.True(t, p.Can("test-user", ResourceAuditLogs, PermissionRead))
	assert.False(t, p.Can("test-user", ResourceAuditLogs, PermissionDelete))
}

func TestAddRole_SystemRoleBlocked(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.AddRole(&Role{ID: "system:admin", Name: "Fake Admin"})
	assert.Error(t, err)
}

func TestAddRole_NoID(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.AddRole(&Role{Name: "No ID"})
	assert.Error(t, err)
}

func TestHTTPTransportPreHook_Allowed(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.AssignRoles("admin", []string{SystemRoleAdmin.ID})

	ctx := schemas.NewBifrostContext(nil, time.Time{})
	ctx.SetValue(schemas.BifrostContextKeyUserID, "admin")

	req := &schemas.HTTPRequest{Method: "GET", Path: "/api/virtual-keys"}
	resp, err := p.HTTPTransportPreHook(ctx, req)
	assert.Nil(t, resp)
	assert.Nil(t, err)
}

func TestHTTPTransportPreHook_Forbidden(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.AssignRoles("viewer", []string{SystemRoleViewer.ID})

	ctx := schemas.NewBifrostContext(nil, time.Time{})
	ctx.SetValue(schemas.BifrostContextKeyUserID, "viewer")

	req := &schemas.HTTPRequest{Method: "POST", Path: "/api/virtual-keys"}
	resp, err := p.HTTPTransportPreHook(ctx, req)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestHTTPTransportPreHook_NoUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.HTTPRequest{Method: "GET", Path: "/api/virtual-keys"}
	resp, err := p.HTTPTransportPreHook(ctx, req)
	assert.Nil(t, resp)
	assert.Nil(t, err)
}

func TestPathToResource(t *testing.T) {
	p := Init(nil, &mockLogger{})

	tests := []struct {
		path     string
		method   string
		wantRes  Resource
		wantPerm Permission
	}{
		{"/api/virtual-keys", "GET", ResourceVirtualKeys, PermissionRead},
		{"/api/virtual-keys", "POST", ResourceVirtualKeys, PermissionCreate},
		{"/api/teams/123", "PUT", ResourceTeams, PermissionUpdate},
		{"/api/budgets/1", "DELETE", ResourceBudgets, PermissionDelete},
		{"/api/prompts", "GET", ResourcePrompts, PermissionRead},
		{"/api/config", "PATCH", ResourceSystemConfig, PermissionUpdate},
		{"/health", "GET", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.method, func(t *testing.T) {
			res, perm := p.pathToResource(tt.path, tt.method)
			assert.Equal(t, tt.wantRes, res)
			assert.Equal(t, tt.wantPerm, perm)
		})
	}
}
