package alertchannels

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
	p := Init(&Config{Enabled: true}, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
}

func TestChannelCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ch := &AlertChannel{Name: "ops-slack", Type: ChannelTypeSlack, Config: map[string]string{"webhook": "https://hooks.slack.com/xxx"}, Enabled: true}
	err := p.CreateChannel(ch)
	require.NoError(t, err)
	assert.NotEmpty(t, ch.ID)

	got, ok := p.GetChannel(ch.ID)
	require.True(t, ok)
	assert.Equal(t, "ops-slack", got.Name)
	assert.Equal(t, ChannelTypeSlack, got.Type)

	channels := p.ListChannels()
	assert.Len(t, channels, 1)

	err = p.DeleteChannel(ch.ID)
	require.NoError(t, err)
	channels = p.ListChannels()
	assert.Len(t, channels, 0)
}

func TestDeleteChannel_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.DeleteChannel("nonexistent")
	assert.Error(t, err)
}

func TestRuleCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	rule := &AlertRule{Name: "high-latency", Condition: "latency > threshold", Threshold: 5000, ChannelIDs: []string{"ch-1"}, Enabled: true}
	err := p.CreateRule(rule)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID)

	got, ok := p.GetRule(rule.ID)
	require.True(t, ok)
	assert.Equal(t, "high-latency", got.Name)

	rules := p.ListRules()
	assert.Len(t, rules, 1)
}

func TestFireAlert(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ch := &AlertChannel{Name: "pd", Type: ChannelTypePagerDuty, Enabled: true}
	p.CreateChannel(ch)

	rule := &AlertRule{Name: "errors", Condition: "error_rate > 0.1", ChannelIDs: []string{ch.ID}, Enabled: true}
	p.CreateRule(rule)

	p.FireAlert(rule.ID, "Error rate spike detected", "critical")

	history := p.GetHistory(10)
	assert.Len(t, history, 1)
	assert.Equal(t, "Error rate spike detected", history[0].Message)
	assert.Equal(t, "critical", history[0].Severity)
}

func TestFireAlert_DisabledRule(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ch := &AlertChannel{Name: "slack", Type: ChannelTypeSlack, Enabled: true}
	p.CreateChannel(ch)

	rule := &AlertRule{Name: "disabled-rule", Condition: "test", ChannelIDs: []string{ch.ID}, Enabled: false}
	p.CreateRule(rule)

	p.FireAlert(rule.ID, "Should not fire", "warning")
	history := p.GetHistory(10)
	assert.Len(t, history, 0)
}

func TestFireAlert_DisabledChannel(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ch := &AlertChannel{Name: "disabled-ch", Type: ChannelTypeWebhook, Enabled: false}
	p.CreateChannel(ch)

	rule := &AlertRule{Name: "active-rule", Condition: "test", ChannelIDs: []string{ch.ID}, Enabled: true}
	p.CreateRule(rule)

	p.FireAlert(rule.ID, "Should not fire", "info")
	history := p.GetHistory(10)
	assert.Len(t, history, 0)
}

func TestFireAlert_UnknownRule(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.FireAlert("nonexistent", "msg", "info")
	history := p.GetHistory(10)
	assert.Len(t, history, 0)
}

func TestGetHistory_Limit(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ch := &AlertChannel{Name: "ch", Type: ChannelTypeWebhook, Enabled: true}
	p.CreateChannel(ch)
	rule := &AlertRule{Name: "r", Condition: "test", ChannelIDs: []string{ch.ID}, Enabled: true}
	p.CreateRule(rule)

	for i := 0; i < 5; i++ {
		p.FireAlert(rule.ID, "msg", "info")
	}

	history := p.GetHistory(3)
	assert.Len(t, history, 3)
}
