package promptdeploy

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

func TestVersionCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	v := &PromptVersion{PromptID: "prompt-1", Version: 1, Content: "You are a helpful assistant.", Variables: []string{"name"}}
	err := p.CreateVersion(v)
	require.NoError(t, err)
	assert.NotEmpty(t, v.ID)

	got, ok := p.GetVersion(v.ID)
	require.True(t, ok)
	assert.Equal(t, "You are a helpful assistant.", got.Content)
	assert.Equal(t, 1, got.Version)
}

func TestListVersions_FilterByPrompt(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateVersion(&PromptVersion{PromptID: "p-1", Version: 1, Content: "v1"})
	p.CreateVersion(&PromptVersion{PromptID: "p-1", Version: 2, Content: "v2"})
	p.CreateVersion(&PromptVersion{PromptID: "p-2", Version: 1, Content: "v1"})

	p1Versions := p.ListVersions("p-1")
	assert.Len(t, p1Versions, 2)

	all := p.ListVersions("")
	assert.Len(t, all, 3)
}

func TestDeploymentCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	deploy := &PromptDeployment{
		PromptID:      "prompt-1",
		Name:          "production",
		Strategy:      StrategyRolling,
		ActiveVersion: "pv-1",
	}
	err := p.CreateDeployment(deploy)
	require.NoError(t, err)
	assert.NotEmpty(t, deploy.ID)
	assert.Equal(t, "active", deploy.Status)

	got, ok := p.GetDeployment(deploy.ID)
	require.True(t, ok)
	assert.Equal(t, "production", got.Name)
	assert.Equal(t, StrategyRolling, got.Strategy)

	deploys := p.ListDeployments("prompt-1")
	assert.Len(t, deploys, 1)
}

func TestDeploymentStrategies(t *testing.T) {
	assert.Equal(t, DeploymentStrategy("rolling"), StrategyRolling)
	assert.Equal(t, DeploymentStrategy("canary"), StrategyCanary)
	assert.Equal(t, DeploymentStrategy("blue_green"), StrategyBlueGreen)
	assert.Equal(t, DeploymentStrategy("shadow"), StrategyShadow)
}

func TestPromoteCanary(t *testing.T) {
	p := Init(nil, &mockLogger{})

	deploy := &PromptDeployment{
		PromptID:      "p-1",
		Name:          "canary-test",
		Strategy:      StrategyCanary,
		ActiveVersion: "pv-1",
		CanaryVersion: "pv-2",
		CanaryPercent: 10,
	}
	p.CreateDeployment(deploy)

	err := p.PromoteCanary(deploy.ID)
	require.NoError(t, err)

	got, _ := p.GetDeployment(deploy.ID)
	assert.Equal(t, "pv-2", got.ActiveVersion)
	assert.Empty(t, got.CanaryVersion)
	assert.Equal(t, 0, got.CanaryPercent)
}

func TestPromoteCanary_NoCanary(t *testing.T) {
	p := Init(nil, &mockLogger{})
	deploy := &PromptDeployment{PromptID: "p-1", Name: "no-canary", Strategy: StrategyRolling, ActiveVersion: "pv-1"}
	p.CreateDeployment(deploy)

	err := p.PromoteCanary(deploy.ID)
	assert.Error(t, err)
}

func TestPromoteCanary_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.PromoteCanary("nonexistent")
	assert.Error(t, err)
}

func TestRollback(t *testing.T) {
	p := Init(nil, &mockLogger{})

	deploy := &PromptDeployment{
		PromptID:      "p-1",
		Name:          "rollback-test",
		Strategy:      StrategyCanary,
		ActiveVersion: "pv-1",
		CanaryVersion: "pv-2",
		CanaryPercent: 20,
	}
	p.CreateDeployment(deploy)

	err := p.Rollback(deploy.ID)
	require.NoError(t, err)

	got, _ := p.GetDeployment(deploy.ID)
	assert.Empty(t, got.CanaryVersion)
	assert.Equal(t, 0, got.CanaryPercent)
	assert.Equal(t, "rolled_back", got.Status)
}

func TestRollback_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.Rollback("nonexistent")
	assert.Error(t, err)
}

func TestResolvePrompt(t *testing.T) {
	p := Init(nil, &mockLogger{})

	p.CreateVersion(&PromptVersion{ID: "pv-1", PromptID: "p-1", Version: 1, Content: "active content"})
	p.CreateVersion(&PromptVersion{ID: "pv-2", PromptID: "p-1", Version: 2, Content: "canary content"})

	p.CreateDeployment(&PromptDeployment{
		PromptID:      "p-1",
		Name:          "prod",
		Strategy:      StrategyCanary,
		ActiveVersion: "pv-1",
		CanaryVersion: "pv-2",
		Status:        "active",
	})

	activeV := p.ResolvePrompt("p-1", false)
	require.NotNil(t, activeV)
	assert.Equal(t, "active content", activeV.Content)

	canaryV := p.ResolvePrompt("p-1", true)
	require.NotNil(t, canaryV)
	assert.Equal(t, "canary content", canaryV.Content)
}

func TestResolvePrompt_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	result := p.ResolvePrompt("nonexistent", false)
	assert.Nil(t, result)
}
