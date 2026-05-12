package scopedkeys

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "scoped_api_keys"

type KeyScope struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"` // "read", "inference", "admin"
	Models   []string `json:"models,omitempty"`
}

type ScopedAPIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"`
	Prefix      string     `json:"prefix"`
	Scopes      []KeyScope `json:"scopes"`
	UserID      string     `json:"user_id,omitempty"`
	TeamID      string     `json:"team_id,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RateLimit   *int       `json:"rate_limit_rpm,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type Config struct {
	Enabled bool `json:"enabled"`
}

type ScopedKeysPlugin struct {
	mu      sync.RWMutex
	keys    map[string]*ScopedAPIKey
	enabled bool
	logger  schemas.Logger
}

func Init(config any, logger schemas.Logger) *ScopedKeysPlugin {
	p := &ScopedKeysPlugin{
		keys:   make(map[string]*ScopedAPIKey),
		logger: logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
		}
	}

	return p
}

func (p *ScopedKeysPlugin) GetName() string { return PluginName }
func (p *ScopedKeysPlugin) Cleanup() error  { return nil }

func (p *ScopedKeysPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func generateKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "bfk_" + hex.EncodeToString(b)
}

func (p *ScopedKeysPlugin) CreateKey(key *ScopedAPIKey) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if key.ID == "" {
		key.ID = fmt.Sprintf("key-%d", time.Now().UnixNano())
	}
	if key.Key == "" {
		key.Key = generateKey()
	}
	key.Prefix = key.Key[:8]
	key.Active = true
	key.CreatedAt = time.Now()
	if key.Scopes == nil {
		key.Scopes = []KeyScope{}
	}
	p.keys[key.ID] = key
	return nil
}

func (p *ScopedKeysPlugin) GetKey(id string) (*ScopedAPIKey, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	k, ok := p.keys[id]
	return k, ok
}

func (p *ScopedKeysPlugin) ListKeys(teamID string) []*ScopedAPIKey {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*ScopedAPIKey, 0)
	for _, k := range p.keys {
		if teamID != "" && k.TeamID != teamID {
			continue
		}
		result = append(result, k)
	}
	return result
}

func (p *ScopedKeysPlugin) ValidateKey(keyValue string) (*ScopedAPIKey, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, k := range p.keys {
		if k.Key == keyValue && k.Active {
			now := time.Now()
			if k.ExpiresAt != nil && now.After(*k.ExpiresAt) {
				return nil, false
			}
			k.LastUsedAt = &now
			return k, true
		}
	}
	return nil, false
}

func (p *ScopedKeysPlugin) CheckScope(keyValue, resource, action string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, k := range p.keys {
		if k.Key != keyValue || !k.Active {
			continue
		}
		for _, scope := range k.Scopes {
			if scope.Resource != resource && scope.Resource != "*" {
				continue
			}
			for _, a := range scope.Actions {
				if a == action || a == "*" {
					return true
				}
			}
		}
	}
	return false
}

func (p *ScopedKeysPlugin) RevokeKey(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	k, ok := p.keys[id]
	if !ok {
		return fmt.Errorf("key not found: %s", id)
	}
	k.Active = false
	return nil
}

var _ schemas.BasePlugin = (*ScopedKeysPlugin)(nil)
