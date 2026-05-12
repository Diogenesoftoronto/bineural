package accessprofiles

import (
	"testing"

	schemas "github.com/maximhq/bifrost/core/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogger struct{}

func (m *mockLogger) SetLevel(level schemas.LogLevel)                        {}
func (m *mockLogger) SetOutputType(outputType schemas.LoggerOutputType)      {}
func (m *mockLogger) Debug(msg string, args ...any)                          {}
func (m *mockLogger) Info(msg string, args ...any)                           {}
func (m *mockLogger) Warn(msg string, args ...any)                          {}
func (m *mockLogger) Error(msg string, args ...any)                          {}
func (m *mockLogger) Fatal(msg string, args ...any)                          {}
func (m *mockLogger) LogHTTPRequest(level schemas.LogLevel, msg string) schemas.LogEventBuilder {
	return schemas.NoopLogEvent
}

func TestInit_NilConfig(t *testing.T) {
	p := Init(nil, &mockLogger{})
	require.NotNil(t, p)
	assert.Equal(t, PluginName, p.GetName())
	assert.Nil(t, p.Cleanup())
	assert.False(t, p.IsEnabled())
}

func TestInit_WithConfig(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Profiles: []AccessProfile{
			{ID: "ap-1", Name: "admin-profile", Permissions: []PermissionSet{{Resource: "keys", Actions: []string{"*"}}}, UserIDs: []string{"user-1"}},
		},
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
}

func TestProfileCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	prof := &AccessProfile{
		Name:        "dev-profile",
		Description: "Developers",
		Permissions: []PermissionSet{
			{Resource: "prompts", Actions: []string{"read", "create"}},
			{Resource: "keys", Actions: []string{"read"}},
		},
		UserIDs: []string{"user-1", "user-2"},
		TeamIDs: []string{"team-1"},
	}
	err := p.CreateProfile(prof)
	require.NoError(t, err)
	assert.NotEmpty(t, prof.ID)

	got, ok := p.GetProfile(prof.ID)
	require.True(t, ok)
	assert.Equal(t, "dev-profile", got.Name)

	profiles := p.ListProfiles()
	assert.Len(t, profiles, 1)
}

func TestUpdateProfile(t *testing.T) {
	p := Init(nil, &mockLogger{})
	prof := &AccessProfile{Name: "test"}
	p.CreateProfile(prof)

	err := p.UpdateProfile(prof.ID, map[string]any{"name": "updated", "description": "new desc"})
	require.NoError(t, err)

	got, _ := p.GetProfile(prof.ID)
	assert.Equal(t, "updated", got.Name)
	assert.Equal(t, "new desc", got.Description)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.UpdateProfile("nonexistent", map[string]any{"name": "x"})
	assert.Error(t, err)
}

func TestDeleteProfile(t *testing.T) {
	p := Init(nil, &mockLogger{})
	prof := &AccessProfile{Name: "test"}
	p.CreateProfile(prof)

	err := p.DeleteProfile(prof.ID)
	require.NoError(t, err)
	_, ok := p.GetProfile(prof.ID)
	assert.False(t, ok)
}

func TestDeleteProfile_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.DeleteProfile("nonexistent")
	assert.Error(t, err)
}

func TestCheckAccess(t *testing.T) {
	p := Init(nil, &mockLogger{})

	prof := &AccessProfile{
		Name: "dev-access",
		Permissions: []PermissionSet{
			{Resource: "prompts", Actions: []string{"read", "create"}},
			{Resource: "keys", Actions: []string{"read"}},
		},
		UserIDs: []string{"user-1"},
	}
	p.CreateProfile(prof)

	assert.True(t, p.CheckAccess("user-1", "prompts", "read"))
	assert.True(t, p.CheckAccess("user-1", "prompts", "create"))
	assert.False(t, p.CheckAccess("user-1", "prompts", "delete"))
	assert.False(t, p.CheckAccess("user-1", "keys", "create"))
	assert.False(t, p.CheckAccess("user-2", "prompts", "read"))
}

func TestCheckAccess_WildcardAction(t *testing.T) {
	p := Init(nil, &mockLogger{})

	prof := &AccessProfile{
		Name: "admin-access",
		Permissions: []PermissionSet{
			{Resource: "keys", Actions: []string{"*"}},
		},
		UserIDs: []string{"admin-1"},
	}
	p.CreateProfile(prof)

	assert.True(t, p.CheckAccess("admin-1", "keys", "read"))
	assert.True(t, p.CheckAccess("admin-1", "keys", "create"))
	assert.True(t, p.CheckAccess("admin-1", "keys", "delete"))
}

func TestGetProfile_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, ok := p.GetProfile("nonexistent")
	assert.False(t, ok)
}
