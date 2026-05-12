package scim

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
	cfg := &Config{Enabled: true, Endpoint: "https://idp.example.com/scim"}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
}

func TestCreateAndGetUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	user := &SCIMUser{UserName: "alice", Email: "alice@example.com", Active: true}
	err := p.CreateUser(user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)

	got, ok := p.GetUser(user.ID)
	require.True(t, ok)
	assert.Equal(t, "alice", got.UserName)
	assert.Equal(t, "alice@example.com", got.Email)
}

func TestGetUser_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, ok := p.GetUser("nonexistent")
	assert.False(t, ok)
}

func TestListUsers_Filter(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateUser(&SCIMUser{UserName: "alice", Email: "alice@co.com", Active: true})
	p.CreateUser(&SCIMUser{UserName: "bob", Email: "bob@co.com", Active: true})

	alice := p.ListUsers("alice", 0, 10)
	assert.Len(t, alice, 1)
	assert.Equal(t, "alice", alice[0].UserName)

	all := p.ListUsers("", 0, 10)
	assert.Len(t, all, 2)
}

func TestListUsers_Pagination(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateUser(&SCIMUser{UserName: "u1", Active: true})
	p.CreateUser(&SCIMUser{UserName: "u2", Active: true})
	p.CreateUser(&SCIMUser{UserName: "u3", Active: true})

	page := p.ListUsers("", 1, 1)
	assert.Len(t, page, 1)
}

func TestUpdateUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	user := &SCIMUser{UserName: "alice", Active: true}
	p.CreateUser(user)

	err := p.UpdateUser(user.ID, map[string]any{"display_name": "Alice Smith", "active": false})
	require.NoError(t, err)

	got, _ := p.GetUser(user.ID)
	assert.Equal(t, "Alice Smith", got.DisplayName)
	assert.False(t, got.Active)
}

func TestUpdateUser_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.UpdateUser("nonexistent", map[string]any{"display_name": "X"})
	assert.Error(t, err)
}

func TestDeleteUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	user := &SCIMUser{UserName: "alice", Active: true}
	p.CreateUser(user)

	err := p.DeleteUser(user.ID)
	require.NoError(t, err)
	_, ok := p.GetUser(user.ID)
	assert.False(t, ok)
}

func TestDeleteUser_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.DeleteUser("nonexistent")
	assert.Error(t, err)
}

func TestGroupCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})
	grp := &SCIMGroup{DisplayName: "engineers"}
	err := p.CreateGroup(grp)
	require.NoError(t, err)
	assert.NotEmpty(t, grp.ID)

	got, ok := p.GetGroup(grp.ID)
	require.True(t, ok)
	assert.Equal(t, "engineers", got.DisplayName)
}

func TestGroupMembership(t *testing.T) {
	p := Init(nil, &mockLogger{})
	grp := &SCIMGroup{DisplayName: "team"}
	p.CreateGroup(grp)

	err := p.AddGroupMember(grp.ID, "user-1")
	require.NoError(t, err)

	got, _ := p.GetGroup(grp.ID)
	assert.Contains(t, got.Members, "user-1")

	err = p.RemoveGroupMember(grp.ID, "user-1")
	require.NoError(t, err)

	got, _ = p.GetGroup(grp.ID)
	assert.NotContains(t, got.Members, "user-1")
}

func TestAddGroupMember_GroupNotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.AddGroupMember("nonexistent", "user-1")
	assert.Error(t, err)
}
