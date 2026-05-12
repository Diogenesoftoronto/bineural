package piiredactor

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
		Rules: []RedactionRule{
			{ID: "r1", Name: "email-rule", Category: PIIEmail, Action: "mask", Enabled: true},
		},
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
}

func TestAddRule_Builtin(t *testing.T) {
	p := Init(nil, &mockLogger{})

	err := p.AddRule(&RedactionRule{ID: "r1", Name: "email", Category: PIIEmail, Action: "mask", Enabled: true})
	require.NoError(t, err)
}

func TestAddRule_CustomPattern(t *testing.T) {
	p := Init(nil, &mockLogger{})

	err := p.AddRule(&RedactionRule{ID: "r2", Name: "custom", Category: PIICustom, Pattern: `customer_\d+`, Action: "replace", Replace: "[ID]", Enabled: true})
	require.NoError(t, err)
}

func TestAddRule_InvalidPattern(t *testing.T) {
	p := Init(nil, &mockLogger{})

	err := p.AddRule(&RedactionRule{ID: "r3", Name: "bad", Category: PIICustom, Pattern: `[invalid`, Action: "mask", Enabled: true})
	assert.Error(t, err)
}

func TestRedact_Email(t *testing.T) {
	p := Init(&Config{Enabled: true, Rules: []RedactionRule{
		{ID: "r1", Category: PIIEmail, Action: "mask", Enabled: true},
	}}, &mockLogger{})

	result := p.Redact("Contact user@example.com for info", "")
	assert.True(t, result.HasRedactions)
	assert.Equal(t, "Contact [REDACTED] for info", result.Redacted)
	assert.Len(t, result.Findings, 1)
	assert.Equal(t, PIIEmail, result.Findings[0].Category)
}

func TestRedact_SSN(t *testing.T) {
	p := Init(&Config{Enabled: true, Rules: []RedactionRule{
		{ID: "r2", Category: PIISSN, Action: "mask", Enabled: true},
	}}, &mockLogger{})

	result := p.Redact("SSN: 123-45-6789", "")
	assert.True(t, result.HasRedactions)
	assert.Contains(t, result.Redacted, "[REDACTED]")
}

func TestRedact_ReplaceAction(t *testing.T) {
	p := Init(&Config{Enabled: true, Rules: []RedactionRule{
		{ID: "r3", Category: PIIEmail, Action: "replace", Replace: "[EMAIL]", Enabled: true},
	}}, &mockLogger{})

	result := p.Redact("Email: test@domain.com", "")
	assert.Equal(t, "Email: [EMAIL]", result.Redacted)
}

func TestRedact_RemoveAction(t *testing.T) {
	p := Init(&Config{Enabled: true, Rules: []RedactionRule{
		{ID: "r4", Category: PIIIPAddress, Action: "remove", Enabled: true},
	}}, &mockLogger{})

	result := p.Redact("Server at 192.168.1.1 is down", "")
	assert.NotContains(t, result.Redacted, "192.168.1.1")
}

func TestRedact_ProfileFilter(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []RedactionRule{
			{ID: "r1", Category: PIIEmail, Action: "mask", Enabled: true},
			{ID: "r2", Category: PIISSN, Action: "mask", Enabled: true},
		},
		Profiles: []RedactionProfile{
			{ID: "p1", Name: "email-only", RuleIDs: []string{"r1"}, Enabled: true},
		},
	}
	p := Init(cfg, &mockLogger{})

	result := p.Redact("Email: a@b.com SSN: 111-22-3333", "p1")
	assert.True(t, result.HasRedactions)
	assert.Contains(t, result.Redacted, "[REDACTED]")
	assert.Contains(t, result.Redacted, "111-22-3333") // SSN NOT redacted with profile (only email rule active)
}

func TestRedact_Disabled(t *testing.T) {
	p := Init(&Config{Enabled: false}, &mockLogger{})
	result := p.Redact("user@example.com", "")
	assert.False(t, result.HasRedactions)
	assert.Equal(t, "user@example.com", result.Redacted)
}

func TestRedact_PhoneAndCreditCard(t *testing.T) {
	p := Init(&Config{Enabled: true, Rules: []RedactionRule{
		{ID: "r5", Category: PIIPhone, Action: "mask", Enabled: true},
		{ID: "r6", Category: PIICreditCard, Action: "mask", Enabled: true},
	}}, &mockLogger{})

	result := p.Redact("Call 555-123-4567, card 4111-1111-1111-1111", "")
	assert.True(t, result.HasRedactions)
}

func TestPreLLMHook_Disabled(t *testing.T) {
	p := Init(&Config{Enabled: false}, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.BifrostRequest{}
	result, shortCircuit, err := p.PreLLMHook(ctx, req)
	require.NoError(t, err)
	assert.Nil(t, shortCircuit)
	assert.Equal(t, req, result)
}

func TestPostLLMHook_WithError(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	resp := &schemas.BifrostResponse{}
	code := 500
	bifrostErr := &schemas.BifrostError{StatusCode: &code}

	result, resultErr, err := p.PostLLMHook(ctx, resp, bifrostErr)
	require.NoError(t, err)
	assert.Equal(t, resp, result)
	assert.Equal(t, bifrostErr, resultErr)
}

func TestAddRule_NoID(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.AddRule(&RedactionRule{Name: "no-id", Category: PIIEmail, Action: "mask", Enabled: true})
	assert.NoError(t, err)
}
