package largepayload

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

func TestInit_Defaults(t *testing.T) {
	p := Init(nil, &mockLogger{})
	require.NotNil(t, p)
	assert.Equal(t, PluginName, p.GetName())
	assert.Nil(t, p.Cleanup())
	assert.False(t, p.IsEnabled())

	assert.Equal(t, 10*1024*1024, p.GetMaxNormalBytes())
	assert.Equal(t, 100*1024*1024, p.GetMaxLargeBytes())
	assert.Equal(t, 1024*1024, p.GetChunkSize())
	assert.Equal(t, 5*1024*1024, p.GetStreamThreshold())
}

func TestInit_WithConfig(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		MaxNormalBytes:  20 * 1024 * 1024,
		MaxLargeBytes:   200 * 1024 * 1024,
		ChunkSize:       2 * 1024 * 1024,
		StreamThreshold: 10 * 1024 * 1024,
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
	assert.Equal(t, 20*1024*1024, p.GetMaxNormalBytes())
	assert.Equal(t, 200*1024*1024, p.GetMaxLargeBytes())
	assert.Equal(t, 2*1024*1024, p.GetChunkSize())
	assert.Equal(t, 10*1024*1024, p.GetStreamThreshold())
}

func TestShouldUseLargeMode(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})

	assert.False(t, p.ShouldUseLargeMode(1*1024*1024))    // 1MB - below threshold
	assert.False(t, p.ShouldUseLargeMode(5*1024*1024))    // 5MB - at threshold
	assert.True(t, p.ShouldUseLargeMode(10*1024*1024))    // 10MB - above threshold
}

func TestShouldUseLargeMode_Disabled(t *testing.T) {
	p := Init(&Config{Enabled: false}, &mockLogger{})
	assert.False(t, p.ShouldUseLargeMode(100*1024*1024))
}

func TestIsWithinLimit(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})

	assert.True(t, p.IsWithinLimit(50*1024*1024))         // Below 100MB limit
	assert.False(t, p.IsWithinLimit(200*1024*1024))       // Above 100MB limit
}

func TestIsWithinLimit_Disabled(t *testing.T) {
	p := Init(&Config{Enabled: false}, &mockLogger{})
	// When disabled, uses maxNormalBytes (10MB) as limit
	assert.True(t, p.IsWithinLimit(5*1024*1024))
	assert.False(t, p.IsWithinLimit(20*1024*1024))
}

func TestSetConfig(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})

	p.SetConfig(50*1024*1024, 500*1024*1024, 5*1024*1024, 20*1024*1024)

	assert.Equal(t, 50*1024*1024, p.GetMaxNormalBytes())
	assert.Equal(t, 500*1024*1024, p.GetMaxLargeBytes())
	assert.Equal(t, 5*1024*1024, p.GetChunkSize())
	assert.Equal(t, 20*1024*1024, p.GetStreamThreshold())
}

func TestSetConfig_ZeroValues(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	origNormal := p.GetMaxNormalBytes()

	p.SetConfig(0, 0, 0, 0)

	assert.Equal(t, origNormal, p.GetMaxNormalBytes())
}
