package accessprofiles

import (
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "access_profiles"

type PermissionSet struct {
	Resource  string   `json:"resource"`
	Actions   []string `json:"actions"` // "read", "create", "update", "delete"
}

type AccessProfile struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Permissions []PermissionSet `json:"permissions"`
	UserIDs     []string       `json:"user_ids,omitempty"`
	TeamIDs     []string       `json:"team_ids,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type Config struct {
	Enabled  bool            `json:"enabled"`
	Profiles []AccessProfile `json:"profiles,omitempty"`
}

type AccessProfilesPlugin struct {
	mu       sync.RWMutex
	profiles map[string]*AccessProfile
	enabled  bool
	logger   schemas.Logger
}

func Init(config any, logger schemas.Logger) *AccessProfilesPlugin {
	p := &AccessProfilesPlugin{
		profiles: make(map[string]*AccessProfile),
		logger:   logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			for i := range cfg.Profiles {
				prof := &cfg.Profiles[i]
				if prof.ID == "" {
					prof.ID = fmt.Sprintf("ap-%d", time.Now().UnixNano())
				}
				prof.CreatedAt = time.Now()
				prof.UpdatedAt = time.Now()
				p.profiles[prof.ID] = prof
			}
		}
	}

	return p
}

func (p *AccessProfilesPlugin) GetName() string { return PluginName }
func (p *AccessProfilesPlugin) Cleanup() error  { return nil }

func (p *AccessProfilesPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *AccessProfilesPlugin) CreateProfile(profile *AccessProfile) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if profile.ID == "" {
		profile.ID = fmt.Sprintf("ap-%d", time.Now().UnixNano())
	}
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()
	if profile.UserIDs == nil {
		profile.UserIDs = []string{}
	}
	if profile.TeamIDs == nil {
		profile.TeamIDs = []string{}
	}
	p.profiles[profile.ID] = profile
	return nil
}

func (p *AccessProfilesPlugin) GetProfile(id string) (*AccessProfile, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	prof, ok := p.profiles[id]
	return prof, ok
}

func (p *AccessProfilesPlugin) ListProfiles() []*AccessProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*AccessProfile, 0, len(p.profiles))
	for _, prof := range p.profiles {
		result = append(result, prof)
	}
	return result
}

func (p *AccessProfilesPlugin) UpdateProfile(id string, updates map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	prof, ok := p.profiles[id]
	if !ok {
		return fmt.Errorf("profile not found: %s", id)
	}

	if name, ok := updates["name"].(string); ok {
		prof.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		prof.Description = desc
	}
	prof.UpdatedAt = time.Now()
	return nil
}

func (p *AccessProfilesPlugin) DeleteProfile(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.profiles[id]; !ok {
		return fmt.Errorf("profile not found: %s", id)
	}
	delete(p.profiles, id)
	return nil
}

func (p *AccessProfilesPlugin) CheckAccess(userID, resource, action string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, prof := range p.profiles {
		hasUser := false
		for _, uid := range prof.UserIDs {
			if uid == userID {
				hasUser = true
				break
			}
		}
		if !hasUser {
			continue
		}

		for _, perm := range prof.Permissions {
			if perm.Resource != resource {
				continue
			}
			for _, a := range perm.Actions {
				if a == action || a == "*" {
					return true
				}
			}
		}
	}
	return false
}

var _ schemas.BasePlugin = (*AccessProfilesPlugin)(nil)
