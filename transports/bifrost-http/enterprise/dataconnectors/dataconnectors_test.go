package dataconnectors

import (
	"encoding/json"
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
	cfg := &Config{
		Enabled: true,
		Connectors: []DataConnector{
			{ID: "conn-1", Name: "my-datadog", Type: ConnectorDatadog, Enabled: true},
		},
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
}

func TestConnectorCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	conn := &DataConnector{Name: "test-connector", Type: ConnectorDatadog, Config: ConnectorConfig{}, Enabled: true}
	err := p.CreateConnector(conn)
	require.NoError(t, err)
	assert.NotEmpty(t, conn.ID)

	got, ok := p.GetConnector(conn.ID)
	require.True(t, ok)
	assert.Equal(t, "prod-datadog", got.Name)
	assert.Equal(t, ConnectorDatadog, got.Type)

	conns := p.ListConnectors("")
	assert.Len(t, conns, 1)
}

func TestListConnectors_FilterByType(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateConnector(&DataConnector{Name: "dd", Type: ConnectorDatadog, Enabled: true})
	p.CreateConnector(&DataConnector{Name: "bq", Type: ConnectorBigQuery, Enabled: true})

	dd := p.ListConnectors(ConnectorDatadog)
	assert.Len(t, dd, 1)
	assert.Equal(t, "dd", dd[0].Name)

	all := p.ListConnectors("")
	assert.Len(t, all, 2)
}

func TestUpdateConnector(t *testing.T) {
	p := Init(nil, &mockLogger{})
	conn := &DataConnector{Name: "dd", Type: ConnectorDatadog, Enabled: true}
	p.CreateConnector(conn)

	err := p.UpdateConnector(conn.ID, map[string]any{"name": "dd-updated", "enabled": false})
	require.NoError(t, err)

	got, _ := p.GetConnector(conn.ID)
	assert.Equal(t, "dd-updated", got.Name)
	assert.False(t, got.Enabled)
}

func TestUpdateConnector_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.UpdateConnector("nonexistent", map[string]any{"name": "x"})
	assert.Error(t, err)
}

func TestDeleteConnector(t *testing.T) {
	p := Init(nil, &mockLogger{})
	conn := &DataConnector{Name: "dd", Type: ConnectorDatadog, Enabled: true}
	p.CreateConnector(conn)

	err := p.DeleteConnector(conn.ID)
	require.NoError(t, err)
	_, ok := p.GetConnector(conn.ID)
	assert.False(t, ok)
}

func TestDeleteConnector_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.DeleteConnector("nonexistent")
	assert.Error(t, err)
}

func TestIngest(t *testing.T) {
	p := Init(nil, &mockLogger{})
	conn := &DataConnector{Name: "dd", Type: ConnectorDatadog, Config: ConnectorConfig{}, Enabled: true}
	p.CreateConnector(conn)

	payload := json.RawMessage(`{"metric": "cpu", "value": 85.5}`)
	err := p.Ingest(conn.ID, "metric", payload)
	require.NoError(t, err)

	events := p.GetEvents(conn.ID, 10)
	assert.Len(t, events, 1)
	assert.Equal(t, "metric", events[0].EventType)
}

func TestIngest_DisabledConnector(t *testing.T) {
	p := Init(nil, &mockLogger{})
	conn := &DataConnector{Name: "dd", Type: ConnectorDatadog, Enabled: false}
	p.CreateConnector(conn)

	err := p.Ingest(conn.ID, "metric", nil)
	assert.Error(t, err)
}

func TestIngest_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.Ingest("nonexistent", "metric", nil)
	assert.Error(t, err)
}

func TestGetEvents_MultipleConnectors(t *testing.T) {
	p := Init(nil, &mockLogger{})
	c1 := &DataConnector{Name: "dd", Type: ConnectorDatadog, Enabled: true}
	c2 := &DataConnector{Name: "bq", Type: ConnectorBigQuery, Enabled: true}
	p.CreateConnector(c1)
	p.CreateConnector(c2)

	p.Ingest(c1.ID, "metric", json.RawMessage(`{}`))
	p.Ingest(c2.ID, "query", json.RawMessage(`{}`))
	p.Ingest(c1.ID, "alert", json.RawMessage(`{}`))

	c1Events := p.GetEvents(c1.ID, 10)
	assert.Len(t, c1Events, 2)

	allEvents := p.GetEvents("", 10)
	assert.Len(t, allEvents, 3)
}

func TestGetEvents_Limit(t *testing.T) {
	p := Init(nil, &mockLogger{})
	conn := &DataConnector{Name: "dd", Type: ConnectorDatadog, Enabled: true}
	p.CreateConnector(conn)

	for i := 0; i < 5; i++ {
		p.Ingest(conn.ID, "event", json.RawMessage(`{}`))
	}

	events := p.GetEvents(conn.ID, 3)
	assert.Len(t, events, 3)
}

func TestConnectorTypes(t *testing.T) {
	assert.Equal(t, ConnectorType("datadog"), ConnectorDatadog)
	assert.Equal(t, ConnectorType("bigquery"), ConnectorBigQuery)
}
