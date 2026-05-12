package scopedkeys

import (
	"strings"
	"testing"
	"time"

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
	p := Init(&Config{Enabled: true}, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
}

func TestCreateKey(t *testing.T) {
	p := Init(nil, &mockLogger{})

	key := &ScopedAPIKey{
		Name:   "test-key",
		UserID: "user-1",
		Scopes: []KeyScope{
			{Resource: "prompts", Actions: []string{"read", "inference"}},
		},
	}
	err := p.CreateKey(key)
	require.NoError(t, err)
	assert.NotEmpty(t, key.ID)
	assert.True(t, key.Active)
	assert.True(t, strings.HasPrefix(key.Key, "bfk_"))
	assert.NotEmpty(t, key.Prefix)
}

func TestGetKey(t *testing.T) {
	p := Init(nil, &mockLogger{})
	key := &ScopedAPIKey{Name: "test"}
	p.CreateKey(key)

	got, ok := p.GetKey(key.ID)
	require.True(t, ok)
	assert.Equal(t, "test", got.Name)
}

func TestGetKey_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, ok := p.GetKey("nonexistent")
	assert.False(t, ok)
}

func TestListKeys_FilterByTeam(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateKey(&ScopedAPIKey{Name: "k1", TeamID: "team-1"})
	p.CreateKey(&ScopedAPIKey{Name: "k2", TeamID: "team-2"})
	p.CreateKey(&ScopedAPIKey{Name: "k3", TeamID: "team-1"})

	team1 := p.ListKeys("team-1")
	assert.Len(t, team1, 2)

	all := p.ListKeys("")
	assert.Len(t, all, 3)
}

func TestValidateKey(t *testing.T) {
	p := Init(nil, &mockLogger{})
	key := &ScopedAPIKey{Name: "test", Scopes: []KeyScope{}}
	p.CreateKey(key)

	got, ok := p.ValidateKey(key.Key)
	require.True(t, ok)
	assert.Equal(t, key.ID, got.ID)
	assert.NotNil(t, got.LastUsedAt)
}

func TestValidateKey_WrongKey(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateKey(&ScopedAPIKey{Name: "test"})

	_, ok := p.ValidateKey("bfk_wrongkey")
	assert.False(t, ok)
}

func TestValidateKey_Expired(t *testing.T) {
	p := Init(nil, &mockLogger{})
	past := time.Now().Add(-1 * time.Hour)
	key := &ScopedAPIKey{Name: "expired", ExpiresAt: &past}
	p.CreateKey(key)

	_, ok := p.ValidateKey(key.Key)
	assert.False(t, ok)
}

func TestValidateKey_Revoked(t *testing.T) {
	p := Init(nil, &mockLogger{})
	key := &ScopedAPIKey{Name: "test"}
	p.CreateKey(key)
	p.RevokeKey(key.ID)

	_, ok := p.ValidateKey(key.Key)
	assert.False(t, ok)
}

func TestRevokeKey(t *testing.T) {
	p := Init(nil, &mockLogger{})
	key := &ScopedAPIKey{Name: "test"}
	p.CreateKey(key)

	err := p.RevokeKey(key.ID)
	require.NoError(t, err)

	got, _ := p.GetKey(key.ID)
	assert.False(t, got.Active)
}

func TestRevokeKey_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.RevokeKey("nonexistent")
	assert.Error(t, err)
}

func TestCheckScope(t *testing.T) {
	p := Init(nil, &mockLogger{})
	key := &ScopedAPIKey{
		Name: "scoped",
		Scopes: []KeyScope{
			{Resource: "prompts", Actions: []string{"read", "inference"}},
			{Resource: "keys", Actions: []string{"read"}},
		},
	}
	p.CreateKey(key)

	assert.True(t, p.CheckScope(key.Key, "prompts", "read"))
	assert.True(t, p.CheckScope(key.Key, "prompts", "inference"))
	assert.False(t, p.CheckScope(key.Key, "prompts", "admin"))
	assert.True(t, p.CheckScope(key.Key, "keys", "read"))
	assert.False(t, p.CheckScope(key.Key, "keys", "admin"))
}

func TestCheckScope_Wildcard(t *testing.T) {
	p := Init(nil, &mockLogger{})
	key := &ScopedAPIKey{
		Name: "wildcard",
		Scopes: []KeyScope{
			{Resource: "*", Actions: []string{"*"}},
		},
	}
	p.CreateKey(key)

	assert.True(t, p.CheckScope(key.Key, "anything", "anyaction"))
}

func TestGenerateKey_Format(t *testing.T) {
	key := generateKey()
	assert.True(t, strings.HasPrefix(key, "bfk_"))
	assert.Equal(t, 4+64, len(key)) // "bfk_" + 32 bytes hex = 64 chars
}
