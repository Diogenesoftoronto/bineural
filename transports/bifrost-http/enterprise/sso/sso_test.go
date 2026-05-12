package sso

import (
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
	assert.False(t, p.IsEnabled())
}

func TestInit_WithProviders(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Providers: []ProviderConfig{
			{
				Name:        "google",
				Type:        "oidc",
				ClientID:    "test-client-id",
				AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:    "https://oauth2.googleapis.com/token",
				UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
				RedirectURL: "http://localhost:8080/sso/callback",
				Scopes:      []string{"profile"},
			},
		},
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
	providers := p.GetProviders()
	assert.Len(t, providers, 1)
	assert.Equal(t, "google", providers[0].Name)
}

func TestInit_DisabledButHasProviders(t *testing.T) {
	cfg := &Config{
		Enabled: false,
		Providers: []ProviderConfig{
			{Name: "google", Type: "oidc"},
		},
	}
	p := Init(cfg, &mockLogger{})
	assert.False(t, p.IsEnabled())
}

func TestGenerateState(t *testing.T) {
	p := Init(&Config{Enabled: true, Providers: []ProviderConfig{{Name: "google"}}}, &mockLogger{})

	state := p.GenerateState("google")
	assert.NotEmpty(t, state)

	p.mu.RLock()
	auth, ok := p.sessions[state]
	p.mu.RUnlock()
	require.True(t, ok)
	assert.Equal(t, "google", auth.Provider)
	assert.False(t, auth.ExpiresAt.Before(time.Now()))
}

func TestVerifyState(t *testing.T) {
	p := Init(&Config{Enabled: true, Providers: []ProviderConfig{{Name: "google"}}}, &mockLogger{})

	state := p.GenerateState("google")

	auth, ok := p.VerifyState(state)
	require.True(t, ok)
	assert.Equal(t, "google", auth.Provider)

	// State should be consumed (one-time use)
	_, ok = p.VerifyState(state)
	assert.False(t, ok)
}

func TestVerifyState_Expired(t *testing.T) {
	p := Init(&Config{Enabled: true, Providers: []ProviderConfig{{Name: "google"}}}, &mockLogger{})

	// Manually insert expired state
	p.mu.Lock()
	p.sessions["expired-state"] = &PendingAuth{
		State:     "expired-state",
		Provider:  "google",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	p.mu.Unlock()

	_, ok := p.VerifyState("expired-state")
	assert.False(t, ok)
}

func TestVerifyState_Unknown(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, ok := p.VerifyState("nonexistent")
	assert.False(t, ok)
}

func TestGetAuthURL(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Providers: []ProviderConfig{
			{
				Name:        "google",
				Type:        "oidc",
				ClientID:    "test-client-id",
				AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:    "https://oauth2.googleapis.com/token",
				UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
				RedirectURL: "http://localhost:8080/sso/callback",
			},
		},
	}
	p := Init(cfg, &mockLogger{})

	authURL, err := p.GetAuthURL("google", "http://localhost:8080/sso/callback")
	require.NoError(t, err)
	assert.Contains(t, authURL, "accounts.google.com")
	assert.Contains(t, authURL, "client_id=test-client-id")
	assert.Contains(t, authURL, "response_type=code")
	assert.Contains(t, authURL, "state=")
	assert.Contains(t, authURL, "scope=")
}

func TestGetAuthURL_UnknownProvider(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, err := p.GetAuthURL("unknown", "http://localhost/callback")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")
}

func TestBuildUserFromToken(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Providers: []ProviderConfig{
			{Name: "google", Type: "oidc", ClientID: "test"},
		},
	}
	p := Init(cfg, &mockLogger{})

	tokenData := map[string]any{
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "Test User",
		"groups": []any{"admin", "dev"},
	}

	claim, err := p.BuildUserFromToken("google", tokenData)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claim.ID)
	assert.Equal(t, "user@example.com", claim.Email)
	assert.Equal(t, "Test User", claim.Name)
	assert.Equal(t, []string{"admin", "dev"}, claim.Groups)
	assert.Equal(t, "google", claim.Provider)
}

func TestBuildUserFromToken_NoSub(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Providers: []ProviderConfig{
			{Name: "google", Type: "oidc", ClientID: "test"},
		},
	}
	p := Init(cfg, &mockLogger{})

	tokenData := map[string]any{
		"email": "user@example.com",
	}

	claim, err := p.BuildUserFromToken("google", tokenData)
	require.NoError(t, err)
	assert.Contains(t, claim.ID, "user@example.com")
}

func TestBuildUserFromToken_UnknownProvider(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, err := p.BuildUserFromToken("unknown", map[string]any{})
	assert.Error(t, err)
}

func TestHTTPTransportPreHook_NoOp(t *testing.T) {
	p := Init(nil, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.HTTPRequest{Method: "GET", Path: "/api/keys"}

	resp, err := p.HTTPTransportPreHook(ctx, req)
	assert.Nil(t, err)
	assert.Nil(t, resp)
}

func TestHTTPTransportPostHook_NoOp(t *testing.T) {
	p := Init(nil, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.HTTPRequest{Method: "GET", Path: "/api/keys"}
	resp := &schemas.HTTPResponse{StatusCode: 200}

	err := p.HTTPTransportPostHook(ctx, req, resp)
	assert.Nil(t, err)
}

func TestHTTPTransportStreamChunkHook_PassThrough(t *testing.T) {
	p := Init(nil, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.HTTPRequest{Method: "POST", Path: "/v1/chat/completions"}
	chunk := &schemas.BifrostStreamChunk{}

	returnedChunk, err := p.HTTPTransportStreamChunkHook(ctx, req, chunk)
	assert.Nil(t, err)
	assert.Equal(t, chunk, returnedChunk)
}
