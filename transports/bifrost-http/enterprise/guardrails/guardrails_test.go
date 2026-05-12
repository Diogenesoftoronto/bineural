package guardrails

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
	assert.False(t, p.enabled)
}

func TestInit_WithConfig(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{ID: "r1", Name: "Block Secrets", Type: RuleTypeRegex, Pattern: `(?i)password\s*[:=]\s*\S+`, Action: ActionBlock, Enabled: true, ContentType: "input"},
		},
		Profiles: []Profile{
			{ID: "p1", Name: "Default", RuleIDs: []string{"r1"}, Enabled: true},
		},
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.enabled)
	assert.Len(t, p.rules, 1)
	assert.Len(t, p.profiles, 1)
}

func TestEvaluate_Disabled(t *testing.T) {
	p := Init(nil, &mockLogger{})
	result := p.Evaluate("some content", "input")
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Violations)
}

func TestEvaluate_Blocklist(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "bl1",
				Name:        "Bad Words",
				Type:        RuleTypeBlocklist,
				Patterns:    []string{"forbidden", "blocked"},
				Action:      ActionBlock,
				Enabled:     true,
				ContentType: "both",
			},
		},
	}
	p := Init(cfg, &mockLogger{})

	result := p.Evaluate("this is fine", "input")
	assert.True(t, result.Allowed)

	result = p.Evaluate("this is forbidden content", "input")
	assert.False(t, result.Allowed)
	assert.Len(t, result.Violations, 1)
	assert.Equal(t, "bl1", result.Violations[0].RuleID)
	assert.Equal(t, ActionBlock, result.Violations[0].Action)
}

func TestEvaluate_Regex(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "re1",
				Name:        "SSN Detection",
				Type:        RuleTypeRegex,
				Pattern:     `\b\d{3}-\d{2}-\d{4}\b`,
				Action:      ActionFlag,
				Enabled:     true,
				ContentType: "both",
			},
		},
	}
	p := Init(cfg, &mockLogger{})

	result := p.Evaluate("my SSN is 123-45-6789", "input")
	assert.True(t, result.Allowed) // Flag doesn't block
	assert.Len(t, result.Violations, 1)
	assert.Equal(t, RuleTypeRegex, result.Violations[0].Type)

	result = p.Evaluate("no SSN here", "input")
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Violations)
}

func TestEvaluate_PII(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "pii1",
				Name:        "PII Detection",
				Type:        RuleTypePII,
				Action:      ActionMask,
				Enabled:     true,
				ContentType: "both",
			},
		},
	}
	p := Init(cfg, &mockLogger{})

	result := p.Evaluate("contact user@example.com for info", "input")
	assert.True(t, result.Allowed) // Mask doesn't block
	assert.Len(t, result.Violations, 1)
	assert.Equal(t, "user@example.com", result.Violations[0].Matched)
	assert.Contains(t, result.MaskedText, "[REDACTED]")
}

func TestEvaluate_ContentType(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "input-only",
				Name:        "Input Rule",
				Type:        RuleTypeBlocklist,
				Patterns:    []string{"secret"},
				Action:      ActionBlock,
				Enabled:     true,
				ContentType: "input",
			},
		},
	}
	p := Init(cfg, &mockLogger{})

	result := p.Evaluate("this is a secret", "input")
	assert.False(t, result.Allowed)

	result = p.Evaluate("this is a secret", "output")
	assert.True(t, result.Allowed) // Rule only applies to input
}

func TestAddRule(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})

	err := p.AddRule(&Rule{
		ID:      "new-rule",
		Name:    "Test",
		Type:    RuleTypeBlocklist,
		Patterns: []string{"test"},
		Action:  ActionBlock,
		Enabled: true,
	})
	require.NoError(t, err)
	assert.Len(t, p.rules, 1)

	result := p.Evaluate("this is a test", "input")
	assert.False(t, result.Allowed)
}

func TestAddRule_NoID(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.AddRule(&Rule{Name: "No ID"})
	assert.Error(t, err)
}

func TestAddRule_InvalidRegex(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.AddRule(&Rule{
		ID:      "bad-regex",
		Type:    RuleTypeRegex,
		Pattern: "[invalid",
	})
	assert.Error(t, err)
}

func strPtr(s string) *string { return &s }

func TestPreLLMHook_BlocksContent(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "bl1",
				Name:        "Block Bad",
				Type:        RuleTypeBlocklist,
				Patterns:    []string{"dangerous"},
				Action:      ActionBlock,
				Enabled:     true,
				ContentType: "input",
			},
		},
	}
	p := Init(cfg, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})

	dangerous := "dangerous"
	req := &schemas.BifrostRequest{
		ChatRequest: &schemas.BifrostChatRequest{
			Input: []schemas.ChatMessage{
				{Content: &schemas.ChatMessageContent{ContentStr: &dangerous}},
			},
		},
	}

	returnedReq, shortCircuit, err := p.PreLLMHook(ctx, req)
	assert.Nil(t, err)
	assert.Nil(t, returnedReq)
	require.NotNil(t, shortCircuit)
	require.NotNil(t, shortCircuit.Error)
	assert.True(t, shortCircuit.Error.IsBifrostError)
}

func TestPreLLMHook_AllowsContent(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "bl1",
				Name:        "Block Bad",
				Type:        RuleTypeBlocklist,
				Patterns:    []string{"dangerous"},
				Action:      ActionBlock,
				Enabled:     true,
				ContentType: "input",
			},
		},
	}
	p := Init(cfg, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})

	safe := "this is safe"
	req := &schemas.BifrostRequest{
		ChatRequest: &schemas.BifrostChatRequest{
			Input: []schemas.ChatMessage{
				{Content: &schemas.ChatMessageContent{ContentStr: &safe}},
			},
		},
	}

	returnedReq, shortCircuit, err := p.PreLLMHook(ctx, req)
	assert.Nil(t, err)
	assert.Nil(t, shortCircuit)
	assert.Equal(t, req, returnedReq)
}

func TestPreLLMHook_Disabled(t *testing.T) {
	p := Init(nil, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.BifrostRequest{}

	returnedReq, shortCircuit, err := p.PreLLMHook(ctx, req)
	assert.Nil(t, err)
	assert.Nil(t, shortCircuit)
	assert.Equal(t, req, returnedReq)
}

func TestPreLLMHook_TextCompletion(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "bl1",
				Name:        "Block Bad",
				Type:        RuleTypeBlocklist,
				Patterns:    []string{"dangerous"},
				Action:      ActionBlock,
				Enabled:     true,
				ContentType: "input",
			},
		},
	}
	p := Init(cfg, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})

	dangerous := "dangerous content"
	req := &schemas.BifrostRequest{
		TextCompletionRequest: &schemas.BifrostTextCompletionRequest{
			Input: &schemas.TextCompletionInput{PromptStr: &dangerous},
		},
	}

	returnedReq, shortCircuit, err := p.PreLLMHook(ctx, req)
	assert.Nil(t, err)
	assert.Nil(t, returnedReq)
	require.NotNil(t, shortCircuit)
}

func TestPostLLMHook_BlocksResponse(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "bl1",
				Name:        "Block Bad Output",
				Type:        RuleTypeBlocklist,
				Patterns:    []string{"harmful"},
				Action:      ActionBlock,
				Enabled:     true,
				ContentType: "output",
			},
		},
	}
	p := Init(cfg, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})

	harmful := "harmful response"
	resp := &schemas.BifrostResponse{
		ChatResponse: &schemas.BifrostChatResponse{
			Choices: []schemas.BifrostResponseChoice{
				{
					ChatNonStreamResponseChoice: &schemas.ChatNonStreamResponseChoice{
						Message: &schemas.ChatMessage{
							Content: &schemas.ChatMessageContent{ContentStr: &harmful},
						},
					},
				},
			},
		},
	}

	returnedResp, returnedErr, err := p.PostLLMHook(ctx, resp, nil)
	assert.Nil(t, err)
	assert.Nil(t, returnedResp)
	require.NotNil(t, returnedErr)
	assert.True(t, returnedErr.IsBifrostError)
}

func TestPostLLMHook_AllowsResponse(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "bl1",
				Name:        "Block Bad Output",
				Type:        RuleTypeBlocklist,
				Patterns:    []string{"harmful"},
				Action:      ActionBlock,
				Enabled:     true,
				ContentType: "output",
			},
		},
	}
	p := Init(cfg, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})

	safe := "safe response"
	resp := &schemas.BifrostResponse{
		ChatResponse: &schemas.BifrostChatResponse{
			Choices: []schemas.BifrostResponseChoice{
				{
					ChatNonStreamResponseChoice: &schemas.ChatNonStreamResponseChoice{
						Message: &schemas.ChatMessage{
							Content: &schemas.ChatMessageContent{ContentStr: &safe},
						},
					},
				},
			},
		},
	}

	returnedResp, returnedErr, err := p.PostLLMHook(ctx, resp, nil)
	assert.Nil(t, err)
	assert.Equal(t, resp, returnedResp)
	assert.Nil(t, returnedErr)
}

func TestPostLLMHook_WithBifrostError(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	resp := &schemas.BifrostResponse{}
	berr := &schemas.BifrostError{IsBifrostError: true}

	returnedResp, returnedErr, err := p.PostLLMHook(ctx, resp, berr)
	assert.Nil(t, err)
	assert.Equal(t, resp, returnedResp)
	assert.Equal(t, berr, returnedErr)
}
