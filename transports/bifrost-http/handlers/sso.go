package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/configstore"
	"github.com/maximhq/bifrost/framework/configstore/tables"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
	"github.com/valyala/fasthttp"
)

// SSOHandler manages HTTP requests for SSO (Single Sign-On) operations.
type SSOHandler struct {
	configStore configstore.ConfigStore
}

// NewSSOHandler creates a new SSO handler instance.
func NewSSOHandler(configStore configstore.ConfigStore) *SSOHandler {
	return &SSOHandler{configStore: configStore}
}

// RegisterRoutes registers all SSO-related routes.
func (h *SSOHandler) RegisterRoutes(r *router.Router, middlewares ...schemas.BifrostHTTPMiddleware) {
	// Provider management (authenticated)
	r.GET("/api/sso/providers", lib.ChainMiddlewares(h.listSSOProviders, middlewares...))
	r.POST("/api/sso/providers", lib.ChainMiddlewares(h.createSSOProvider, middlewares...))
	r.GET("/api/sso/providers/{id}", lib.ChainMiddlewares(h.getSSOProvider, middlewares...))
	r.PUT("/api/sso/providers/{id}", lib.ChainMiddlewares(h.updateSSOProvider, middlewares...))
	r.DELETE("/api/sso/providers/{id}", lib.ChainMiddlewares(h.deleteSSOProvider, middlewares...))

	// Authentication flow (public, no auth middleware)
	r.GET("/api/sso/auth/{provider_name}", h.initiateSSOAuth)
	r.GET("/api/sso/callback", h.handleSSOCallback)
}

// ─── Provider Management ──────────────────────────────────────────────────────

func (h *SSOHandler) listSSOProviders(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	providers, err := h.configStore.ListSSOProviders(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to list SSO providers: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"providers": providers, "count": len(providers)})
}

func (h *SSOHandler) createSSOProvider(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var provider tables.TableSSOProvider
	if err := json.Unmarshal(ctx.PostBody(), &provider); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if provider.Name == "" {
		SendError(ctx, 400, "Provider name is required")
		return
	}
	if err := h.configStore.CreateSSOProvider(ctx, &provider); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create SSO provider: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "SSO provider created", "provider": provider}, 201)
}

func (h *SSOHandler) getSSOProvider(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	provider, err := h.configStore.GetSSOProvider(ctx, id)
	if err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "SSO provider not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to get SSO provider: %v", err))
		return
	}
	SendJSON(ctx, provider)
}

func (h *SSOHandler) updateSSOProvider(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	var provider tables.TableSSOProvider
	if err := json.Unmarshal(ctx.PostBody(), &provider); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	provider.ID = parseUint(id)
	if err := h.configStore.UpdateSSOProvider(ctx, &provider); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to update SSO provider: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "SSO provider updated"})
}

func (h *SSOHandler) deleteSSOProvider(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeleteSSOProvider(ctx, id); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to delete SSO provider: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "SSO provider deleted"})
}

// ─── Authentication Flow ──────────────────────────────────────────────────────

func (h *SSOHandler) initiateSSOAuth(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	providerName := ctx.UserValue("provider_name").(string)
	provider, err := h.configStore.GetSSOProviderByName(ctx, providerName)
	if err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, fmt.Sprintf("SSO provider not found: %s", providerName))
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to look up SSO provider: %v", err))
		return
	}
	if !provider.Enabled {
		SendError(ctx, 400, fmt.Sprintf("SSO provider %s is disabled", providerName))
		return
	}

	// Generate state (nonce) for CSRF protection
	state, err := generateState()
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to generate state: %v", err))
		return
	}

	// Determine the redirect_uri: use the provider's configured redirect URL,
	// or fall back to constructing one from the incoming request.
	redirectURI := provider.RedirectURL
	if redirectURI == "" {
		scheme := "http"
		if string(ctx.URI().Scheme()) == "https" {
			scheme = "https"
		}
		host := string(ctx.Host())
		redirectURI = fmt.Sprintf("%s://%s/api/sso/callback", scheme, host)
	}

	// Create an SSO session
	session := &tables.TableSSOSession{
		State:       state,
		Provider:    providerName,
		RedirectURL: redirectURI,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	if err := h.configStore.CreateSSOSession(ctx, session); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create SSO session: %v", err))
		return
	}

	// Build the authorization redirect URL
	authURL, err := buildAuthRedirect(provider.AuthURL, provider.ClientID, redirectURI, state, provider.Scopes)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to build auth URL: %v", err))
		return
	}

	ctx.Redirect(authURL, 302)
}

func (h *SSOHandler) handleSSOCallback(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}

	state := string(ctx.QueryArgs().Peek("state"))
	code := string(ctx.QueryArgs().Peek("code"))
	if state == "" || code == "" {
		SendError(ctx, 400, "Missing state or code parameter")
		return
	}

	// Look up the session by state
	session, err := h.configStore.GetSSOSessionByState(ctx, state)
	if err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 400, "Invalid or expired SSO session")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to look up SSO session: %v", err))
		return
	}

	// Look up the provider
	provider, err := h.configStore.GetSSOProviderByName(ctx, session.Provider)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to look up SSO provider: %v", err))
		return
	}

	// Exchange the authorization code for tokens
	tokenResponse, err := exchangeCodeForToken(provider.TokenURL, code, session.RedirectURL, provider.ClientID, provider.ClientSecret)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to exchange code for token: %v", err))
		return
	}

	// Fetch user info from the provider
	userInfo, err := fetchUserInfo(provider.UserInfoURL, tokenResponse.AccessToken)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to fetch user info: %v", err))
		return
	}

	// Extract user identity claims
	userID, _ := userInfo["sub"].(string)
	userEmail, _ := userInfo["email"].(string)

	// Update session with user identity and tokens:
	// delete the old session and recreate with updated fields since
	// ConfigStore does not expose an UpdateSSOSession method.
	session.UserID = userID
	session.UserEmail = userEmail
	session.IDToken = tokenResponse.IDToken
	session.AccessToken = tokenResponse.AccessToken

	sessionIDStr := fmt.Sprintf("%d", session.ID)
	if err := h.configStore.DeleteSSOSession(ctx, sessionIDStr); err != nil {
		logger.Warn(fmt.Sprintf("Failed to delete old SSO session for update: %v", err))
	}
	session.ID = 0 // let DB auto-assign a new ID
	if err := h.configStore.CreateSSOSession(ctx, session); err != nil {
		logger.Warn(fmt.Sprintf("Failed to persist updated SSO session: %v", err))
	}

	// Return user claims as JSON
	SendJSON(ctx, map[string]interface{}{
		"user_id":    userID,
		"email":      userEmail,
		"provider":   session.Provider,
		"id_token":   tokenResponse.IDToken,
		"expires_at": session.ExpiresAt,
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// generateState creates a cryptographically random state string for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildAuthRedirect constructs the full authorization URL with query parameters.
func buildAuthRedirect(authURL, clientID, redirectURI, state, scopesJSON string) (string, error) {
	u, err := url.Parse(authURL)
	if err != nil {
		return "", fmt.Errorf("invalid auth URL: %w", err)
	}

	var scopes []string
	if scopesJSON != "" {
		if err := json.Unmarshal([]byte(scopesJSON), &scopes); err != nil {
			// Fall back to treating the raw string as a single scope
			scopes = strings.Split(scopesJSON, ",")
		}
	}
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("response_type", "code")
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// tokenResponse represents the response from an OIDC token endpoint.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// exchangeCodeForToken exchanges an authorization code for tokens at the provider's token endpoint.
func exchangeCodeForToken(tokenURL, code, redirectURI, clientID, clientSecret string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// fetchUserInfo retrieves user information from the provider's userinfo endpoint.
func fetchUserInfo(userInfoURL, accessToken string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info response: %w", err)
	}

	return userInfo, nil
}

// parseUint parses a string as uint for table ID fields.
func parseUint(s string) uint {
	var n uint
	fmt.Sscanf(s, "%d", &n)
	return n
}
