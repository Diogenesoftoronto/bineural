package evals

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

func TestDatasetCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	ds := &EvalDataset{Name: "qa-benchmark", Description: "Quality assurance benchmark", NumEntries: 100}
	err := p.CreateDataset(ds)
	require.NoError(t, err)
	assert.NotEmpty(t, ds.ID)

	got, ok := p.GetDataset(ds.ID)
	require.True(t, ok)
	assert.Equal(t, "qa-benchmark", got.Name)

	datasets := p.ListDatasets()
	assert.Len(t, datasets, 1)
}

func TestGetDataset_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, ok := p.GetDataset("nonexistent")
	assert.False(t, ok)
}

func TestRunCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	run := &EvalRun{DatasetID: "ds-1", Model: "gpt-4o"}
	err := p.CreateRun(run)
	require.NoError(t, err)
	assert.NotEmpty(t, run.ID)
	assert.Equal(t, "pending", run.Status)

	got, ok := p.GetRun(run.ID)
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", got.Model)
}

func TestCompleteRun(t *testing.T) {
	p := Init(nil, &mockLogger{})

	run := &EvalRun{DatasetID: "ds-1", Model: "gpt-4o"}
	p.CreateRun(run)

	metrics := map[string]float64{"accuracy": 0.95, "latency_ms": 120.5}
	err := p.CompleteRun(run.ID, metrics, 0.95)
	require.NoError(t, err)

	got, _ := p.GetRun(run.ID)
	assert.Equal(t, "completed", got.Status)
	assert.Equal(t, 0.95, got.Score)
	assert.NotNil(t, got.CompletedAt)
}

func TestCompleteRun_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.CompleteRun("nonexistent", nil, 0)
	assert.Error(t, err)
}

func TestListRuns_FilterByDataset(t *testing.T) {
	p := Init(nil, &mockLogger{})

	p.CreateRun(&EvalRun{DatasetID: "ds-1", Model: "gpt-4o"})
	p.CreateRun(&EvalRun{DatasetID: "ds-1", Model: "claude-3"})
	p.CreateRun(&EvalRun{DatasetID: "ds-2", Model: "gpt-4o"})

	ds1Runs := p.ListRuns("ds-1")
	assert.Len(t, ds1Runs, 2)

	allRuns := p.ListRuns("")
	assert.Len(t, allRuns, 3)
}

func TestCompareRuns(t *testing.T) {
	p := Init(nil, &mockLogger{})

	r1 := &EvalRun{DatasetID: "ds-1", Model: "gpt-4o", Metrics: map[string]float64{}}
	r2 := &EvalRun{DatasetID: "ds-1", Model: "claude-3", Metrics: map[string]float64{}}
	p.CreateRun(r1)
	p.CreateRun(r2)

	p.CompleteRun(r1.ID, map[string]float64{"accuracy": 0.92}, 0.92)
	p.CompleteRun(r2.ID, map[string]float64{"accuracy": 0.95}, 0.95)

	comp := p.CompareRuns([]string{r1.ID, r2.ID})
	require.NotNil(t, comp)
	assert.Equal(t, "claude-3", comp.BestModel)
	assert.Len(t, comp.Results, 2)
}

func TestCompareRuns_MissingRun(t *testing.T) {
	p := Init(nil, &mockLogger{})
	comp := p.CompareRuns([]string{"nonexistent"})
	assert.NotNil(t, comp)
	assert.Len(t, comp.Results, 0)
}
