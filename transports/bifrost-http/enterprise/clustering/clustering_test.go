package clustering

import (
	"fmt"
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
	assert.NotEmpty(t, p.GetNodeID())
}

func TestInit_WithConfig(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		NodeID:  "node-1",
		Peers:   []string{"node-2:8080", "node-3:8080"},
	}
	p := Init(cfg, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
	assert.Equal(t, "node-1", p.GetNodeID())

	peers := p.GetPeers()
	assert.Len(t, peers, 2)
}

func TestInit_DefaultNodeID(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	nodeID := p.GetNodeID()
	assert.Contains(t, nodeID, "node-")
}

func TestGetPeers(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		NodeID:  "node-1",
		Peers:   []string{"node-2:8080", "node-3:8080"},
	}
	p := Init(cfg, &mockLogger{})

	peers := p.GetPeers()
	assert.Len(t, peers, 2)

	peerIDs := map[string]bool{}
	for _, p := range peers {
		peerIDs[p.ID] = true
		assert.True(t, p.Healthy)
	}
	assert.Len(t, peerIDs, 2)
}

func TestMarkPeerHealthy(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		NodeID:  "node-1",
		Peers:   []string{"node-2:8080"},
	}
	p := Init(cfg, &mockLogger{})

	peers := p.GetPeers()
	require.Len(t, peers, 1)
	peerID := peers[0].ID

	p.MarkPeerHealthy(peerID)

	peers = p.GetPeers()
	assert.True(t, peers[0].Healthy)
}

func TestSelectNodeForKey_SingleNode(t *testing.T) {
	p := Init(&Config{Enabled: true, NodeID: "node-1"}, &mockLogger{})

	node := p.SelectNodeForKey("some-key")
	assert.Equal(t, "node-1", node)
}

func TestSelectNodeForKey_MultipleNodes(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		NodeID:  "node-1",
		Peers:   []string{"node-2:8080", "node-3:8080"},
	}
	p := Init(cfg, &mockLogger{})

	node1 := p.SelectNodeForKey("test-key")
	node2 := p.SelectNodeForKey("test-key")
	assert.Equal(t, node1, node2) // Consistent hashing should return same node for same key
}

func TestSelectNodeForKey_DifferentKeys(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		NodeID:  "node-1",
		Peers:   []string{"node-2:8080", "node-3:8080", "node-4:8080"},
	}
	p := Init(cfg, &mockLogger{})

	nodes := map[string]bool{}
	for i := 0; i < 100; i++ {
		node := p.SelectNodeForKey(fmt.Sprintf("key-%d", i))
		nodes[node] = true
	}
	assert.Greater(t, len(nodes), 1, "consistent hashing should distribute across nodes")
}

func TestSelectNodeForKey_Disabled(t *testing.T) {
	p := Init(&Config{Enabled: false, NodeID: "node-1"}, &mockLogger{})
	node := p.SelectNodeForKey("any-key")
	assert.Equal(t, "node-1", node)
}

func TestCleanup(t *testing.T) {
	p := Init(&Config{Enabled: true, NodeID: "node-1"}, &mockLogger{})
	err := p.Cleanup()
	assert.Nil(t, err)
}
