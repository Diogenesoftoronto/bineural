package sso

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "sso"

// ProviderConfig for OIDC/SAML providers
type ProviderConfig struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // "oidc" or "saml"
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"user_info_url"`
	RedirectURL  string `json:"redirect_url"`
	Scopes       []string `json:"scopes,omitempty"`
}

// Config for SSO plugin
type Config struct {
	Providers []ProviderConfig `json:"providers,omitempty"`
	Enabled   bool             `json:"enabled"`
}

// UserClaim represents a decoded user from SSO
type UserClaim struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Groups    []string `json:"groups,omitempty"`
	Provider  string   `json:"provider"`
}

// PendingAuth tracks in-flight auth flows
type PendingAuth struct {
	State       string
	Provider    string
	RedirectURL string
	ExpiresAt   time.Time
}

// SSOPlugin implements OIDC authentication
type SSOPlugin struct {
	mu          sync.RWMutex
	providers   map[string]*ProviderConfig
	sessions    map[string]*PendingAuth
	logger      schemas.Logger
	enabled     bool
}

// Init initializes the SSO plugin
func Init(config any, logger schemas.Logger) *SSOPlugin {
	p := &SSOPlugin{
		providers: make(map[string]*ProviderConfig),
		sessions:  make(map[string]*PendingAuth),
		logger:    logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			for i := range cfg.Providers {
				prov := &cfg.Providers[i]
				p.providers[prov.Name] = prov
			}
		}
	}

	return p
}

func (p *SSOPlugin) GetName() string { return PluginName }
func (p *SSOPlugin) Cleanup() error  { return nil }

// IsEnabled returns whether SSO is configured
func (p *SSOPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled && len(p.providers) > 0
}

// GetProviders returns available SSO providers
func (p *SSOPlugin) GetProviders() []ProviderConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	providers := make([]ProviderConfig, 0, len(p.providers))
	for _, prov := range p.providers {
		providers = append(providers, *prov)
	}
	return providers
}

// GenerateState creates a random state parameter for OAuth flow
func (p *SSOPlugin) GenerateState(provider string) string {
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessions[state] = &PendingAuth{
		State:     state,
		Provider:  provider,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	return state
}

// GetAuthURL returns the authorization URL for an SSO provider
func (p *SSOPlugin) GetAuthURL(providerName, redirectURI string) (string, error) {
	p.mu.RLock()
	prov, ok := p.providers[providerName]
	p.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("provider not found: %s", providerName)
	}

	state := p.GenerateState(providerName)

	u, err := url.Parse(prov.AuthURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("client_id", prov.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	q.Set("scope", strings.Join(append([]string{"openid", "profile", "email"}, prov.Scopes...), " "))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// VerifyState checks and consumes a state parameter
func (p *SSOPlugin) VerifyState(state string) (*PendingAuth, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	auth, ok := p.sessions[state]
	if !ok || time.Now().After(auth.ExpiresAt) {
		return nil, false
	}

	delete(p.sessions, state)
	return auth, true
}

// BuildUserFromToken creates user claims from token data
func (p *SSOPlugin) BuildUserFromToken(providerName string, tokenData map[string]any) (*UserClaim, error) {
	p.mu.RLock()
	prov, ok := p.providers[providerName]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	claim := &UserClaim{
		Provider: providerName,
	}

	if sub, ok := tokenData["sub"].(string); ok {
		claim.ID = sub
	}
	if email, ok := tokenData["email"].(string); ok {
		claim.Email = email
	}
	if name, ok := tokenData["name"].(string); ok {
		claim.Name = name
	}
	if groups, ok := tokenData["groups"].([]any); ok {
		for _, g := range groups {
			if gs, ok := g.(string); ok {
				claim.Groups = append(claim.Groups, gs)
			}
		}
	}

	if claim.ID == "" && claim.Email != "" {
		claim.ID = fmt.Sprintf("%s:%s", prov.Name, claim.Email)
	}

	return claim, nil
}

// HTTPTransportPreHook intercepts requests for SSO session validation
func (p *SSOPlugin) HTTPTransportPreHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest) (*schemas.HTTPResponse, error) {
	return nil, nil
}

// HTTPTransportPostHook is a no-op for SSO
func (p *SSOPlugin) HTTPTransportPostHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest, resp *schemas.HTTPResponse) error {
	return nil
}

func (p *SSOPlugin) HTTPTransportStreamChunkHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest, chunk *schemas.BifrostStreamChunk) (*schemas.BifrostStreamChunk, error) {
	return chunk, nil
}

var _ schemas.HTTPTransportPlugin = (*SSOPlugin)(nil)
var _ schemas.BasePlugin = (*SSOPlugin)(nil)
