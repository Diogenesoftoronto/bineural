package scim

import (
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "scim"

type Config struct {
	Enabled       bool   `json:"enabled"`
	Endpoint      string `json:"endpoint,omitempty"`
	BearerToken   string `json:"bearer_token,omitempty"`
	DefaultRoleID string `json:"default_role_id,omitempty"`
}

type SCIMUser struct {
	ID          string    `json:"id"`
	ExternalID  string    `json:"external_id,omitempty"`
	UserName    string    `json:"user_name"`
	DisplayName string    `json:"display_name,omitempty"`
	Email       string    `json:"email,omitempty"`
	Active      bool      `json:"active"`
	Groups      []string  `json:"groups,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SCIMGroup struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Members     []string `json:"members,omitempty"`
}

type SCIMPlugin struct {
	mu      sync.RWMutex
	users   map[string]*SCIMUser
	groups  map[string]*SCIMGroup
	enabled bool
	logger  schemas.Logger
}

func Init(config any, logger schemas.Logger) *SCIMPlugin {
	p := &SCIMPlugin{
		users:  make(map[string]*SCIMUser),
		groups: make(map[string]*SCIMGroup),
		logger: logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
		}
	}

	return p
}

func (p *SCIMPlugin) GetName() string { return PluginName }
func (p *SCIMPlugin) Cleanup() error  { return nil }

func (p *SCIMPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *SCIMPlugin) CreateUser(user *SCIMUser) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if user.ID == "" {
		user.ID = fmt.Sprintf("scim-%d", time.Now().UnixNano())
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	p.users[user.ID] = user
	return nil
}

func (p *SCIMPlugin) GetUser(id string) (*SCIMUser, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	u, ok := p.users[id]
	if !ok {
		return nil, false
	}
	return u, true
}

func (p *SCIMPlugin) ListUsers(filter string, offset, limit int) []*SCIMUser {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*SCIMUser, 0)
	skipped := 0
	for _, u := range p.users {
		if filter != "" && u.UserName != filter && u.Email != filter {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if len(result) >= limit {
			break
		}
		result = append(result, u)
	}
	return result
}

func (p *SCIMPlugin) UpdateUser(id string, updates map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	u, ok := p.users[id]
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}

	if name, ok := updates["display_name"].(string); ok {
		u.DisplayName = name
	}
	if userName, ok := updates["user_name"].(string); ok {
		u.UserName = userName
	}
	if email, ok := updates["email"].(string); ok {
		u.Email = email
	}
	if active, ok := updates["active"].(bool); ok {
		u.Active = active
	}
	if groups, ok := updates["groups"].([]string); ok {
		u.Groups = groups
	}
	u.UpdatedAt = time.Now()
	return nil
}

func (p *SCIMPlugin) DeleteUser(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.users[id]; !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	delete(p.users, id)
	return nil
}

func (p *SCIMPlugin) CreateGroup(group *SCIMGroup) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if group.ID == "" {
		group.ID = fmt.Sprintf("grp-%d", time.Now().UnixNano())
	}
	if group.Members == nil {
		group.Members = []string{}
	}
	p.groups[group.ID] = group
	return nil
}

func (p *SCIMPlugin) GetGroup(id string) (*SCIMGroup, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	g, ok := p.groups[id]
	return g, ok
}

func (p *SCIMPlugin) ListGroups(filter string, offset, limit int) []*SCIMGroup {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*SCIMGroup, 0)
	skipped := 0
	for _, g := range p.groups {
		if filter != "" && g.DisplayName != filter {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if limit > 0 && len(result) >= limit {
			break
		}
		result = append(result, g)
	}
	return result
}

func (p *SCIMPlugin) UpdateGroup(id string, updates map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	g, ok := p.groups[id]
	if !ok {
		return fmt.Errorf("group not found: %s", id)
	}

	if displayName, ok := updates["display_name"].(string); ok {
		g.DisplayName = displayName
	}
	if members, ok := updates["members"].([]string); ok {
		g.Members = members
	}
	return nil
}

func (p *SCIMPlugin) DeleteGroup(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.groups[id]; !ok {
		return fmt.Errorf("group not found: %s", id)
	}
	delete(p.groups, id)
	return nil
}

func (p *SCIMPlugin) AddGroupMember(groupID, userID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	g, ok := p.groups[groupID]
	if !ok {
		return fmt.Errorf("group not found: %s", groupID)
	}
	g.Members = append(g.Members, userID)
	return nil
}

func (p *SCIMPlugin) RemoveGroupMember(groupID, userID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	g, ok := p.groups[groupID]
	if !ok {
		return fmt.Errorf("group not found: %s", groupID)
	}
	for i, m := range g.Members {
		if m == userID {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			return nil
		}
	}
	return nil
}

var _ schemas.BasePlugin = (*SCIMPlugin)(nil)
