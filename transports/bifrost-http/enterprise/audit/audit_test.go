package audit

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
	assert.NotEmpty(t, p.secretKey)
}

func TestInit_WithConfig(t *testing.T) {
	cfg := &Config{
		SecretKey:  "my-secret",
		BufferSize: 500,
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.Equal(t, 500, p.bufferSize)
}

func TestLog_AndQuery(t *testing.T) {
	p := Init(nil, &mockLogger{})

	p.Log(AuditLogEntry{
		EventType: EventAuth,
		Action:    "login",
		UserID:    "user-1",
		Status:    "success",
	})
	p.Log(AuditLogEntry{
		EventType: EventAuthz,
		Action:    "GET /api/keys",
		UserID:    "user-1",
		Status:    "success",
	})
	p.Log(AuditLogEntry{
		EventType: EventSecurity,
		Action:    "failed_login",
		UserID:    "user-2",
		Status:    "failure",
	})

	// Query all
	all := p.Query(nil, nil, nil, 10)
	assert.Len(t, all, 3)

	// Query by event type
	authEvents := p.Query(&[]EventType{EventAuth}[0], nil, nil, 10)
	assert.Len(t, authEvents, 1)
	assert.Equal(t, EventAuth, authEvents[0].EventType)

	// Query by user
	user1Events := p.Query(nil, &[]string{"user-1"}[0], nil, 10)
	assert.Len(t, user1Events, 2)
}

func TestHMAC_Verify(t *testing.T) {
	p := Init(&Config{SecretKey: "test-key"}, &mockLogger{})

	entry := AuditLogEntry{
		EventType: EventConfigChange,
		Action:    "update-provider",
		UserID:    "admin",
		Status:    "success",
	}
	p.Log(entry)

	// Get the logged entry
	entries := p.Query(nil, nil, nil, 1)
	require.Len(t, entries, 1)

	// Verify HMAC
	assert.True(t, p.Verify(entries[0]))

	// Tamper and verify fails
	entries[0].Status = "failure"
	assert.False(t, p.Verify(entries[0]))
}

func TestBufferSize(t *testing.T) {
	p := Init(&Config{BufferSize: 3}, &mockLogger{})

	for i := 0; i < 5; i++ {
		p.Log(AuditLogEntry{EventType: EventInference, Action: "request"})
	}

	entries := p.Query(nil, nil, nil, 10)
	assert.Len(t, entries, 3)
}

func TestHTTPTransportPostHook(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ctx := schemas.NewBifrostContext(nil, time.Time{})
	ctx.SetValue(schemas.BifrostContextKeyUserID, "user-1")
	ctx.SetValue(schemas.BifrostContextKeyUserName, "Alice")

	req := &schemas.HTTPRequest{Method: "GET", Path: "/api/keys"}
	resp := &schemas.HTTPResponse{StatusCode: 200}

	err := p.HTTPTransportPostHook(ctx, req, resp)
	assert.Nil(t, err)

	entries := p.Query(nil, nil, nil, 10)
	require.Len(t, entries, 1)
	assert.Equal(t, "user-1", entries[0].UserID)
	assert.Equal(t, "Alice", entries[0].UserName)
	assert.Equal(t, EventAuthz, entries[0].EventType)
}

func TestHTTPTransportPostHook_ErrorStatus(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.HTTPRequest{Method: "POST", Path: "/api/keys"}
	resp := &schemas.HTTPResponse{StatusCode: 500}

	err := p.HTTPTransportPostHook(ctx, req, resp)
	assert.Nil(t, err)

	entries := p.Query(nil, nil, nil, 10)
	require.Len(t, entries, 1)
	assert.Equal(t, "failure", entries[0].Status)
}
