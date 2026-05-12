package usergovernance

import (
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "user_governance"

type User struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Email         string  `json:"email,omitempty"`
	RoleID        string  `json:"role_id,omitempty"`
	BudgetLimit   *float64 `json:"budget_limit,omitempty"`
	RateLimitRPM  *int    `json:"rate_limit_rpm,omitempty"`
	TeamID        string  `json:"team_id,omitempty"`
	BusinessUnitID string `json:"business_unit_id,omitempty"`
	Active        bool    `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
}

type Team struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	LeaderID    string   `json:"leader_id,omitempty"`
	MemberIDs   []string `json:"member_ids,omitempty"`
	BudgetLimit *float64 `json:"budget_limit,omitempty"`
}

type BusinessUnit struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	TeamIDs     []string `json:"team_ids,omitempty"`
	BudgetLimit *float64 `json:"budget_limit,omitempty"`
}

type UserRanking struct {
	UserID    string  `json:"user_id"`
	UserName  string  `json:"user_name"`
	TotalReqs int64   `json:"total_requests"`
	TotalCost float64 `json:"total_cost"`
	Rank      int     `json:"rank"`
}

type Config struct {
	Enabled bool `json:"enabled"`
}

type UserGovernancePlugin struct {
	mu            sync.RWMutex
	users         map[string]*User
	teams         map[string]*Team
	businessUnits map[string]*BusinessUnit
	usage         map[string]*usageEntry
	enabled       bool
	logger        schemas.Logger
}

type usageEntry struct {
	totalRequests int64
	totalCost     float64
}

func Init(config any, logger schemas.Logger) *UserGovernancePlugin {
	p := &UserGovernancePlugin{
		users:         make(map[string]*User),
		teams:         make(map[string]*Team),
		businessUnits: make(map[string]*BusinessUnit),
		usage:         make(map[string]*usageEntry),
		logger:        logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
		}
	}

	return p
}

func (p *UserGovernancePlugin) GetName() string { return PluginName }
func (p *UserGovernancePlugin) Cleanup() error  { return nil }

func (p *UserGovernancePlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *UserGovernancePlugin) CreateUser(user *User) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if user.ID == "" {
		user.ID = fmt.Sprintf("usr-%d", time.Now().UnixNano())
	}
	user.CreatedAt = time.Now()
	if user.Active == false && user.ID != "" {
		user.Active = true
	}
	p.users[user.ID] = user
	p.usage[user.ID] = &usageEntry{}
	return nil
}

func (p *UserGovernancePlugin) GetUser(id string) (*User, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	u, ok := p.users[id]
	return u, ok
}

func (p *UserGovernancePlugin) ListUsers(teamID string, offset, limit int) []*User {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*User, 0)
	skipped := 0
	for _, u := range p.users {
		if teamID != "" && u.TeamID != teamID {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if limit > 0 && len(result) >= limit {
			break
		}
		result = append(result, u)
	}
	return result
}

func (p *UserGovernancePlugin) UpdateUser(id string, updates map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	u, ok := p.users[id]
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}

	if name, ok := updates["name"].(string); ok {
		u.Name = name
	}
	if email, ok := updates["email"].(string); ok {
		u.Email = email
	}
	if roleID, ok := updates["role_id"].(string); ok {
		u.RoleID = roleID
	}
	if teamID, ok := updates["team_id"].(string); ok {
		u.TeamID = teamID
	}
	if buID, ok := updates["business_unit_id"].(string); ok {
		u.BusinessUnitID = buID
	}
	if budget, ok := updates["budget_limit"].(float64); ok {
		u.BudgetLimit = &budget
	}
	if rpm, ok := updates["rate_limit_rpm"].(int); ok {
		u.RateLimitRPM = &rpm
	}
	if active, ok := updates["active"].(bool); ok {
		u.Active = active
	}

	return nil
}

func (p *UserGovernancePlugin) DeleteUser(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.users[id]; !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	delete(p.users, id)
	delete(p.usage, id)
	return nil
}

func (p *UserGovernancePlugin) CreateTeam(team *Team) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if team.ID == "" {
		team.ID = fmt.Sprintf("team-%d", time.Now().UnixNano())
	}
	if team.MemberIDs == nil {
		team.MemberIDs = []string{}
	}
	p.teams[team.ID] = team
	return nil
}

func (p *UserGovernancePlugin) GetTeam(id string) (*Team, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	t, ok := p.teams[id]
	return t, ok
}

func (p *UserGovernancePlugin) ListTeams() []*Team {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*Team, 0, len(p.teams))
	for _, t := range p.teams {
		result = append(result, t)
	}
	return result
}

func (p *UserGovernancePlugin) CreateBusinessUnit(bu *BusinessUnit) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bu.ID == "" {
		bu.ID = fmt.Sprintf("bu-%d", time.Now().UnixNano())
	}
	if bu.TeamIDs == nil {
		bu.TeamIDs = []string{}
	}
	p.businessUnits[bu.ID] = bu
	return nil
}

func (p *UserGovernancePlugin) GetBusinessUnit(id string) (*BusinessUnit, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	bu, ok := p.businessUnits[id]
	return bu, ok
}

func (p *UserGovernancePlugin) ListBusinessUnits() []*BusinessUnit {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*BusinessUnit, 0, len(p.businessUnits))
	for _, bu := range p.businessUnits {
		result = append(result, bu)
	}
	return result
}

func (p *UserGovernancePlugin) RecordUsage(userID string, cost float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.usage[userID]
	if !ok {
		e = &usageEntry{}
		p.usage[userID] = e
	}
	e.totalRequests++
	e.totalCost += cost
}

func (p *UserGovernancePlugin) CheckBudget(userID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	u, ok := p.users[userID]
	if !ok || u.BudgetLimit == nil {
		return true
	}

	e, ok := p.usage[userID]
	if !ok {
		return true
	}

	return e.totalCost < *u.BudgetLimit
}

func (p *UserGovernancePlugin) CheckRateLimit(userID string, currentRPM int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	u, ok := p.users[userID]
	if !ok || u.RateLimitRPM == nil {
		return true
	}

	return currentRPM <= *u.RateLimitRPM
}

func (p *UserGovernancePlugin) GetUserRankings(limit int) []UserRanking {
	p.mu.RLock()
	defer p.mu.RUnlock()

	rankings := make([]UserRanking, 0)
	for id, e := range p.usage {
		u, ok := p.users[id]
		name := id
		if ok {
			name = u.Name
		}
		rankings = append(rankings, UserRanking{
			UserID:    id,
			UserName:  name,
			TotalReqs: e.totalRequests,
			TotalCost: e.totalCost,
		})
	}

	for i := 0; i < len(rankings); i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[j].TotalCost > rankings[i].TotalCost {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	for i := range rankings {
		rankings[i].Rank = i + 1
	}

	if limit > 0 && len(rankings) > limit {
		rankings = rankings[:limit]
	}
	return rankings
}

var _ schemas.BasePlugin = (*UserGovernancePlugin)(nil)
