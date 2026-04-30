package rbac

import (
	"fmt"
	"strings"
	"sync"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "rbac"

// Permission represents a CRUD operation
type Permission string

const (
	PermissionCreate Permission = "create"
	PermissionRead   Permission = "read"
	PermissionUpdate Permission = "update"
	PermissionDelete Permission = "delete"
)

// Resource represents a Bifrost entity that can be protected
type Resource string

const (
	ResourceVirtualKeys   Resource = "virtual_keys"
	ResourceTeams         Resource = "teams"
	ResourceCustomers     Resource = "customers"
	ResourceBudgets       Resource = "budgets"
	ResourceRateLimits    Resource = "rate_limits"
	ResourceProviders     Resource = "providers"
	ResourceKeys          Resource = "keys"
	ResourceRoutingRules  Resource = "routing_rules"
	ResourceModelConfigs  Resource = "model_configs"
	ResourcePrompts       Resource = "prompts"
	ResourceMCPClients    Resource = "mcp_clients"
	ResourceUsers         Resource = "users"
	ResourceAuditLogs     Resource = "audit_logs"
	ResourceGuardrails    Resource = "guardrails"
	ResourceSystemConfig  Resource = "system_config"
)

// Role represents a role definition
type Role struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Permissions map[Resource][]Permission `json:"permissions"`
	IsSystem    bool                  `json:"is_system"`
}

// UserRoleAssignment maps a user to roles
type UserRoleAssignment struct {
	UserID    string   `json:"user_id"`
	RoleIDs   []string `json:"role_ids"`
	TeamID    *string  `json:"team_id,omitempty"`
	CustomerID *string `json:"customer_id,omitempty"`
}

// System roles
var (
	SystemRoleAdmin = Role{
		ID:       "system:admin",
		Name:     "Admin",
		IsSystem: true,
		Permissions: map[Resource][]Permission{
			ResourceVirtualKeys:  {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceTeams:        {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceCustomers:    {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceBudgets:      {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceRateLimits:   {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceProviders:    {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceKeys:         {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceRoutingRules: {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceModelConfigs: {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourcePrompts:      {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceMCPClients:   {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceUsers:        {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceAuditLogs:    {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceGuardrails:   {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceSystemConfig: {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
		},
	}

	SystemRoleDeveloper = Role{
		ID:       "system:developer",
		Name:     "Developer",
		IsSystem: true,
		Permissions: map[Resource][]Permission{
			ResourceVirtualKeys:  {PermissionRead},
			ResourceTeams:        {PermissionRead},
			ResourceCustomers:    {PermissionRead},
			ResourceBudgets:      {PermissionRead},
			ResourceRateLimits:   {PermissionRead},
			ResourceProviders:    {PermissionRead},
			ResourceKeys:         {PermissionRead},
			ResourceRoutingRules: {PermissionRead},
			ResourceModelConfigs: {PermissionRead},
			ResourcePrompts:      {PermissionCreate, PermissionRead, PermissionUpdate, PermissionDelete},
			ResourceMCPClients:   {PermissionRead},
			ResourceAuditLogs:    {PermissionRead},
			ResourceGuardrails:   {PermissionRead},
		},
	}

	SystemRoleViewer = Role{
		ID:       "system:viewer",
		Name:     "Viewer",
		IsSystem: true,
		Permissions: map[Resource][]Permission{
			ResourceVirtualKeys:  {PermissionRead},
			ResourceTeams:        {PermissionRead},
			ResourceCustomers:    {PermissionRead},
			ResourceBudgets:      {PermissionRead},
			ResourceRateLimits:   {PermissionRead},
			ResourceProviders:    {PermissionRead},
			ResourceKeys:         {PermissionRead},
			ResourceRoutingRules: {PermissionRead},
			ResourceModelConfigs: {PermissionRead},
			ResourcePrompts:      {PermissionRead},
			ResourceMCPClients:   {PermissionRead},
			ResourceAuditLogs:    {PermissionRead},
			ResourceGuardrails:   {PermissionRead},
		},
	}
)

// RBACPlugin implements role-based access control
type RBACPlugin struct {
	mu          sync.RWMutex
	roles       map[string]*Role
	assignments map[string]*UserRoleAssignment
	logger      schemas.Logger
}

// Config for RBAC plugin
type Config struct {
	Roles       []Role                 `json:"roles,omitempty"`
	Assignments []UserRoleAssignment  `json:"assignments,omitempty"`
}

// Init initializes the RBAC plugin
func Init(config any, logger schemas.Logger) *RBACPlugin {
	p := &RBACPlugin{
		roles:       make(map[string]*Role),
		assignments: make(map[string]*UserRoleAssignment),
		logger:      logger,
	}

	// Register system roles
	p.roles[SystemRoleAdmin.ID] = &SystemRoleAdmin
	p.roles[SystemRoleDeveloper.ID] = &SystemRoleDeveloper
	p.roles[SystemRoleViewer.ID] = &SystemRoleViewer

	// Load custom config
	if config != nil {
		if cfg, ok := config.(*Config); ok {
			for i := range cfg.Roles {
				role := &cfg.Roles[i]
				if !role.IsSystem {
					p.roles[role.ID] = role
				}
			}
			for i := range cfg.Assignments {
				asst := &cfg.Assignments[i]
				p.assignments[asst.UserID] = asst
			}
		}
	}

	return p
}

func (p *RBACPlugin) GetName() string { return PluginName }

func (p *RBACPlugin) Cleanup() error { return nil }

// Can checks if a user has a specific permission on a resource
func (p *RBACPlugin) Can(userID string, resource Resource, perm Permission) bool {
	if userID == "" {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	assignment, ok := p.assignments[userID]
	if !ok {
		return false
	}

	for _, roleID := range assignment.RoleIDs {
		role, ok := p.roles[roleID]
		if !ok {
			continue
		}
		perms, ok := role.Permissions[resource]
		if !ok {
			continue
		}
		for _, p := range perms {
			if p == perm {
				return true
			}
		}
	}
	return false
}

// IsAdmin checks if user has admin role
func (p *RBACPlugin) IsAdmin(userID string) bool {
	return p.Can(userID, ResourceSystemConfig, PermissionDelete)
}

// GetRoles returns all roles
func (p *RBACPlugin) GetRoles() []*Role {
	p.mu.RLock()
	defer p.mu.RUnlock()

	roles := make([]*Role, 0, len(p.roles))
	for _, r := range p.roles {
		roles = append(roles, r)
	}
	return roles
}

// AddRole adds a custom role
func (p *RBACPlugin) AddRole(role *Role) error {
	if role.ID == "" {
		return fmt.Errorf("role ID is required")
	}
	if strings.HasPrefix(role.ID, "system:") {
		return fmt.Errorf("cannot modify system roles")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.roles[role.ID] = role
	return nil
}

// AssignRoles assigns roles to a user
func (p *RBACPlugin) AssignRoles(userID string, roleIDs []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.assignments[userID] = &UserRoleAssignment{
		UserID:  userID,
		RoleIDs: roleIDs,
	}
}

// HTTPTransportPreHook intercepts requests and checks permissions
func (p *RBACPlugin) HTTPTransportPreHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest) (*schemas.HTTPResponse, error) {
	userID, _ := ctx.Value(schemas.BifrostContextKeyUserID).(string)
	if userID == "" {
		return nil, nil // No user, skip RBAC (let auth middleware handle it)
	}

	resource, action := p.pathToResource(req.Path, req.Method)
	if resource == "" {
		return nil, nil
	}

	if !p.Can(userID, resource, action) {
		return &schemas.HTTPResponse{
			StatusCode: 403,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"error":"forbidden","message":"access denied"}`),
		}, nil
	}

	return nil, nil
}

// pathToResource maps API paths to RBAC resources and HTTP methods to permissions
func (p *RBACPlugin) pathToResource(path string, method string) (Resource, Permission) {
	var action Permission
	switch method {
	case "GET":
		action = PermissionRead
	case "POST":
		action = PermissionCreate
	case "PUT", "PATCH":
		action = PermissionUpdate
	case "DELETE":
		action = PermissionDelete
	default:
		return "", ""
	}

	path = strings.ToLower(path)
	switch {
	case strings.Contains(path, "/virtual-key"):
		return ResourceVirtualKeys, action
	case strings.Contains(path, "/team"):
		return ResourceTeams, action
	case strings.Contains(path, "/customer"):
		return ResourceCustomers, action
	case strings.Contains(path, "/budget"):
		return ResourceBudgets, action
	case strings.Contains(path, "/rate-limit"):
		return ResourceRateLimits, action
	case strings.Contains(path, "/provider"):
		return ResourceProviders, action
	case strings.Contains(path, "/key"):
		return ResourceKeys, action
	case strings.Contains(path, "/routing-rule"):
		return ResourceRoutingRules, action
	case strings.Contains(path, "/model-config"):
		return ResourceModelConfigs, action
	case strings.Contains(path, "/prompt"):
		return ResourcePrompts, action
	case strings.Contains(path, "/mcp"):
		return ResourceMCPClients, action
	case strings.Contains(path, "/user"):
		return ResourceUsers, action
	case strings.Contains(path, "/audit-log"):
		return ResourceAuditLogs, action
	case strings.Contains(path, "/guardrail"):
		return ResourceGuardrails, action
	case strings.Contains(path, "/config"):
		return ResourceSystemConfig, action
	default:
		return "", ""
	}
}

func (p *RBACPlugin) HTTPTransportPostHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest, resp *schemas.HTTPResponse) error {
	return nil
}

func (p *RBACPlugin) HTTPTransportStreamChunkHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest, chunk *schemas.BifrostStreamChunk) (*schemas.BifrostStreamChunk, error) {
	return chunk, nil
}

// Ensure RBACPlugin implements required interfaces
var _ schemas.HTTPTransportPlugin = (*RBACPlugin)(nil)
var _ schemas.BasePlugin = (*RBACPlugin)(nil)
