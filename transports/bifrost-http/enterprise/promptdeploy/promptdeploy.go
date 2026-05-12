package promptdeploy

import (
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "prompt_deployments"

type DeploymentStrategy string

const (
	StrategyRolling   DeploymentStrategy = "rolling"
	StrategyCanary     DeploymentStrategy = "canary"
	StrategyBlueGreen  DeploymentStrategy = "blue_green"
	StrategyShadow     DeploymentStrategy = "shadow"
)

type PromptVersion struct {
	ID        string    `json:"id"`
	PromptID  string   `json:"prompt_id"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	Variables []string  `json:"variables,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PromptDeployment struct {
	ID           string             `json:"id"`
	PromptID     string             `json:"prompt_id"`
	Name         string             `json:"name"`
	Strategy     DeploymentStrategy `json:"strategy"`
	ActiveVersion  string           `json:"active_version"`
	CanaryVersion  string           `json:"canary_version,omitempty"`
	CanaryPercent  int              `json:"canary_percent,omitempty"`
	Status       string             `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type Config struct {
	Enabled bool `json:"enabled"`
}

type PromptDeployPlugin struct {
	mu          sync.RWMutex
	versions    map[string]*PromptVersion
	deployments map[string]*PromptDeployment
	enabled     bool
	logger      schemas.Logger
}

func Init(config any, logger schemas.Logger) *PromptDeployPlugin {
	p := &PromptDeployPlugin{
		versions:    make(map[string]*PromptVersion),
		deployments: make(map[string]*PromptDeployment),
		logger:      logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
		}
	}

	return p
}

func (p *PromptDeployPlugin) GetName() string { return PluginName }
func (p *PromptDeployPlugin) Cleanup() error  { return nil }

func (p *PromptDeployPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *PromptDeployPlugin) CreateVersion(version *PromptVersion) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if version.ID == "" {
		version.ID = fmt.Sprintf("pv-%d", time.Now().UnixNano())
	}
	version.CreatedAt = time.Now()
	p.versions[version.ID] = version
	return nil
}

func (p *PromptDeployPlugin) GetVersion(id string) (*PromptVersion, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.versions[id]
	return v, ok
}

func (p *PromptDeployPlugin) ListVersions(promptID string) []*PromptVersion {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*PromptVersion, 0)
	for _, v := range p.versions {
		if promptID == "" || v.PromptID == promptID {
			result = append(result, v)
		}
	}
	return result
}

func (p *PromptDeployPlugin) CreateDeployment(deploy *PromptDeployment) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if deploy.ID == "" {
		deploy.ID = fmt.Sprintf("deploy-%d", time.Now().UnixNano())
	}
	if deploy.Status == "" {
		deploy.Status = "active"
	}
	deploy.CreatedAt = time.Now()
	deploy.UpdatedAt = time.Now()
	p.deployments[deploy.ID] = deploy
	return nil
}

func (p *PromptDeployPlugin) GetDeployment(id string) (*PromptDeployment, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d, ok := p.deployments[id]
	return d, ok
}

func (p *PromptDeployPlugin) ListDeployments(promptID string) []*PromptDeployment {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*PromptDeployment, 0)
	for _, d := range p.deployments {
		if promptID == "" || d.PromptID == promptID {
			result = append(result, d)
		}
	}
	return result
}

func (p *PromptDeployPlugin) PromoteCanary(deploymentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	d, ok := p.deployments[deploymentID]
	if !ok {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}
	if d.CanaryVersion == "" {
		return fmt.Errorf("no canary version to promote")
	}

	d.ActiveVersion = d.CanaryVersion
	d.CanaryVersion = ""
	d.CanaryPercent = 0
	d.UpdatedAt = time.Now()
	return nil
}

func (p *PromptDeployPlugin) Rollback(deploymentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	d, ok := p.deployments[deploymentID]
	if !ok {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	d.CanaryVersion = ""
	d.CanaryPercent = 0
	d.Status = "rolled_back"
	d.UpdatedAt = time.Now()
	return nil
}

func (p *PromptDeployPlugin) ResolvePrompt(promptID string, isCanary bool) *PromptVersion {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, d := range p.deployments {
		if d.PromptID != promptID || d.Status != "active" {
			continue
		}

		versionID := d.ActiveVersion
		if isCanary && d.CanaryVersion != "" {
			versionID = d.CanaryVersion
		}

		if v, ok := p.versions[versionID]; ok {
			return v
		}
	}
	return nil
}

var _ schemas.BasePlugin = (*PromptDeployPlugin)(nil)
