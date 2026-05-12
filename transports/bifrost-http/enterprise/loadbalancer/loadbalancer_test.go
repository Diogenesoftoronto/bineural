package loadbalancer

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
	assert.False(t, p.IsEnabled())
}

func TestInit_WithConfig(t *testing.T) {
	cfg := &Config{
		Enabled:              true,
		WeightUpdateInterval: 10,
		ErrorWeight:          0.6,
		LatencyWeight:        0.3,
		UtilWeight:           0.1,
		ExplorationPercent:   0.2,
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
	assert.Equal(t, 0.6, p.errorWeight)
	assert.Equal(t, 0.3, p.latencyWeight)
	assert.Equal(t, 0.1, p.utilWeight)
	assert.Equal(t, 0.2, p.exploration)
	p.Cleanup()
}

func TestRecordResult(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	defer p.Cleanup()

	p.RecordResult("key-1", 100, false)
	p.RecordResult("key-1", 200, false)
	p.RecordResult("key-1", 50, true)

	p.mu.RLock()
	m, ok := p.metrics["key-1"]
	p.mu.RUnlock()

	require.True(t, ok)
	assert.Equal(t, int64(3), m.TotalRequests)
	assert.Equal(t, int64(1), m.ErrorCount)
	assert.Equal(t, int64(350), m.TotalLatency)
}

func TestRecordResult_NewKey(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	defer p.Cleanup()

	p.RecordResult("new-key", 100, false)

	p.mu.RLock()
	m, ok := p.metrics["new-key"]
	p.mu.RUnlock()

	require.True(t, ok)
	assert.Equal(t, 1.0, m.CurrentWeight) // Default weight
}

func TestSelectKey_Empty(t *testing.T) {
	p := Init(nil, &mockLogger{})
	assert.Equal(t, "", p.SelectKey([]string{}))
}

func TestSelectKey_Single(t *testing.T) {
	p := Init(nil, &mockLogger{})
	assert.Equal(t, "key-1", p.SelectKey([]string{"key-1"}))
}

func TestSelectKey_Weighted(t *testing.T) {
	p := Init(&Config{Enabled: true, ExplorationPercent: 0.0}, &mockLogger{})
	defer p.Cleanup()

	p.RecordResult("key-1", 50, false)
	p.RecordResult("key-2", 5000, true)
	p.RecordResult("key-2", 5000, true)
	p.RecordResult("key-2", 5000, true)

	p.recalculateWeights()

	p.mu.RLock()
	w1 := p.metrics["key-1"].CurrentWeight
	w2 := p.metrics["key-2"].CurrentWeight
	p.mu.RUnlock()

	assert.Greater(t, w1, w2, "healthy key should have higher weight")

	selected := p.SelectKey([]string{"key-1", "key-2"})
	assert.Contains(t, []string{"key-1", "key-2"}, selected)
}

func TestSelectKey_Disabled(t *testing.T) {
	p := Init(&Config{Enabled: false}, &mockLogger{})

	selected := p.SelectKey([]string{"a", "b", "c"})
	assert.Contains(t, []string{"a", "b", "c"}, selected)
}

func TestRecalculateWeights(t *testing.T) {
	p := Init(&Config{Enabled: true, ExplorationPercent: 0.0}, &mockLogger{})
	defer p.Cleanup()

	p.RecordResult("healthy-key", 10, false)
	p.RecordResult("healthy-key", 10, false)
	p.RecordResult("healthy-key", 10, false)

	p.RecordResult("error-key", 1000, true)
	p.RecordResult("error-key", 1000, true)
	p.RecordResult("error-key", 1000, true)

	p.recalculateWeights()

	p.mu.RLock()
	healthyWeight := p.metrics["healthy-key"].CurrentWeight
	errorWeight := p.metrics["error-key"].CurrentWeight
	p.mu.RUnlock()

	assert.Greater(t, healthyWeight, errorWeight)
	assert.GreaterOrEqual(t, errorWeight, 0.1) // Minimum weight
}

func TestRecalculateWeights_NoRequests(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	defer p.Cleanup()

	p.mu.Lock()
	p.metrics["idle-key"] = &KeyMetrics{KeyID: "idle-key", CurrentWeight: 0}
	p.mu.Unlock()

	p.recalculateWeights()

	p.mu.RLock()
	w := p.metrics["idle-key"].CurrentWeight
	p.mu.RUnlock()

	assert.Equal(t, 1.0, w)
}

func TestPreLLMHook_Passthrough(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	defer p.Cleanup()

	ctx := schemas.NewBifrostContext(nil, time.Time{})
	req := &schemas.BifrostRequest{}

	returnedReq, shortCircuit, err := p.PreLLMHook(ctx, req)
	assert.Nil(t, err)
	assert.Nil(t, shortCircuit)
	assert.Equal(t, req, returnedReq)
}

func TestPostLLMHook_RecordsResult(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	defer p.Cleanup()

	ctx := schemas.NewBifrostContext(nil, time.Time{})
	ctx.SetValue(schemas.BifrostContextKeySelectedKeyID, "key-1")

	resp := &schemas.BifrostResponse{}
	returnedResp, returnedErr, err := p.PostLLMHook(ctx, resp, nil)
	assert.Nil(t, err)
	assert.Equal(t, resp, returnedResp)
	assert.Nil(t, returnedErr)

	p.mu.RLock()
	_, ok := p.metrics["key-1"]
	p.mu.RUnlock()
	assert.True(t, ok, "should have recorded result for key-1")
}

func TestPostLLMHook_RecordsError(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	defer p.Cleanup()

	ctx := schemas.NewBifrostContext(nil, time.Time{})
	ctx.SetValue(schemas.BifrostContextKeySelectedKeyID, "key-1")

	resp := &schemas.BifrostResponse{}
	berr := &schemas.BifrostError{IsBifrostError: true}
	returnedResp, returnedErr, err := p.PostLLMHook(ctx, resp, berr)
	assert.Nil(t, err)
	assert.Equal(t, resp, returnedResp)
	assert.Equal(t, berr, returnedErr)

	p.mu.RLock()
	m := p.metrics["key-1"]
	p.mu.RUnlock()
	require.NotNil(t, m)
	assert.Equal(t, int64(1), m.ErrorCount)
}

func TestPostLLMHook_Disabled(t *testing.T) {
	p := Init(nil, &mockLogger{})
	ctx := schemas.NewBifrostContext(nil, time.Time{})
	resp := &schemas.BifrostResponse{}

	returnedResp, returnedErr, err := p.PostLLMHook(ctx, resp, nil)
	assert.Nil(t, err)
	assert.Equal(t, resp, returnedResp)
	assert.Nil(t, returnedErr)
}
