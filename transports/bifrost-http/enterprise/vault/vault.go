// Package vault provides an LLM plugin that resolves API keys from external
// secret managers (HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager,
// Azure Key Vault) instead of storing them in the Bifrost config file.
//
// When a provider key name follows the pattern "vault:path/to/secret#key_name",
// the plugin intercepts the request in PreLLMHook, resolves the secret from the
// configured vault provider, and injects the actual key into the request context.
package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "vault"

// VaultProvider is the interface for external secret backends.
type VaultProvider interface {
	// GetSecret retrieves a secret value at the given path and key.
	GetSecret(ctx context.Context, path string, key string) (string, error)
	// Name returns the provider identifier (e.g. "hashicorp", "aws").
	Name() string
}

// Config holds vault plugin configuration.
type Config struct {
	Enabled  bool             `json:"enabled"`
	Provider string           `json:"provider"` // "hashicorp", "aws", "gcp", "azure"
	Settings map[string]string `json:"settings,omitempty"`
}

// VaultPlugin resolves API keys from external secret managers.
type VaultPlugin struct {
	provider VaultProvider
	logger   schemas.Logger
	enabled  bool
}

// ---------------------------------------------------------------------------
// HashiCorp Vault provider
// ---------------------------------------------------------------------------

// HashiCorpVaultProvider reads secrets from HashiCorp Vault KV v2 engine.
type HashiCorpVaultProvider struct {
	address string
	token   string
	client  *http.Client
}

// NewHashiCorpVaultProvider creates a new HashiCorp Vault provider.
func NewHashiCorpVaultProvider(address, token string) *HashiCorpVaultProvider {
	return &HashiCorpVaultProvider{
		address: strings.TrimRight(address, "/"),
		token:   token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// hashicorpResponse models the relevant subset of the Vault KV v2 response.
type hashicorpResponse struct {
	Data struct {
		Data map[string]string `json:"data"`
	} `json:"data"`
}

func (p *HashiCorpVaultProvider) GetSecret(ctx context.Context, path string, key string) (string, error) {
	url := fmt.Sprintf("%s/v1/%s", p.address, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("vault: failed to create request: %w", err)
	}
	req.Header.Set("X-Vault-Token", p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vault: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result hashicorpResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("vault: failed to decode response: %w", err)
	}

	value, ok := result.Data.Data[key]
	if !ok {
		return "", fmt.Errorf("vault: key %q not found at path %q", key, path)
	}
	return value, nil
}

func (p *HashiCorpVaultProvider) Name() string { return "hashicorp" }

// ---------------------------------------------------------------------------
// AWS Secrets Manager provider (placeholder)
// ---------------------------------------------------------------------------

// AWSSecretsManagerProvider is a placeholder for AWS Secrets Manager integration.
type AWSSecretsManagerProvider struct{}

func (p *AWSSecretsManagerProvider) GetSecret(_ context.Context, _ string, _ string) (string, error) {
	return "", fmt.Errorf("AWS Secrets Manager integration requires AWS SDK — not yet available in open-source build")
}

func (p *AWSSecretsManagerProvider) Name() string { return "aws" }

// ---------------------------------------------------------------------------
// GCP Secret Manager provider (placeholder)
// ---------------------------------------------------------------------------

// GCPSecretManagerProvider is a placeholder for GCP Secret Manager integration.
type GCPSecretManagerProvider struct{}

func (p *GCPSecretManagerProvider) GetSecret(_ context.Context, _ string, _ string) (string, error) {
	return "", fmt.Errorf("GCP Secret Manager integration requires GCP SDK — not yet available in open-source build")
}

func (p *GCPSecretManagerProvider) Name() string { return "gcp" }

// ---------------------------------------------------------------------------
// Azure Key Vault provider (placeholder)
// ---------------------------------------------------------------------------

// AzureKeyVaultProvider is a placeholder for Azure Key Vault integration.
type AzureKeyVaultProvider struct{}

func (p *AzureKeyVaultProvider) GetSecret(_ context.Context, _ string, _ string) (string, error) {
	return "", fmt.Errorf("Azure Key Vault integration requires Azure SDK — not yet available in open-source build")
}

func (p *AzureKeyVaultProvider) Name() string { return "azure" }

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

// Init creates a new VaultPlugin from the given config and logger.
func Init(config any, logger schemas.Logger) *VaultPlugin {
	p := &VaultPlugin{
		logger:  logger,
		enabled: false,
	}

	if config == nil {
		return p
	}

	cfg, ok := config.(*Config)
	if !ok {
		// Try to unmarshal from map[string]any or similar
		logger.Warn("vault: unexpected config type %T, disabling plugin", config)
		return p
	}

	if !cfg.Enabled {
		return p
	}

	if cfg.Provider == "" {
		logger.Warn("vault: no provider specified, disabling plugin")
		return p
	}

	var provider VaultProvider
	switch cfg.Provider {
	case "hashicorp":
		address := cfg.Settings["address"]
		token := cfg.Settings["token"]
		if address == "" || token == "" {
			logger.Warn("vault: hashicorp provider requires 'address' and 'token' settings")
			return p
		}
		provider = NewHashiCorpVaultProvider(address, token)
	case "aws":
		provider = &AWSSecretsManagerProvider{}
	case "gcp":
		provider = &GCPSecretManagerProvider{}
	case "azure":
		provider = &AzureKeyVaultProvider{}
	default:
		logger.Warn("vault: unknown provider %q, disabling plugin", cfg.Provider)
		return p
	}

	p.provider = provider
	p.enabled = true
	logger.Info("vault plugin initialized with provider: %s", provider.Name())
	return p
}

// ---------------------------------------------------------------------------
// BasePlugin interface
// ---------------------------------------------------------------------------

func (p *VaultPlugin) GetName() string { return PluginName }
func (p *VaultPlugin) Cleanup() error   { return nil }

// ---------------------------------------------------------------------------
// LLMPlugin interface
// ---------------------------------------------------------------------------

// parseVaultRef parses a vault reference string of the form "vault:path/to/secret#key_name".
// Returns the path and key, or empty strings if the value is not a vault reference.
func parseVaultRef(value string) (path string, key string, ok bool) {
	if !strings.HasPrefix(value, "vault:") {
		return "", "", false
	}
	rest := value[len("vault:"):]
	idx := strings.LastIndex(rest, "#")
	if idx < 0 {
		return "", "", false
	}
	path = rest[:idx]
	key = rest[idx+1:]
	if path == "" || key == "" {
		return "", "", false
	}
	return path, key, true
}

// PreLLMHook checks if the API key name in the context is a vault reference
// and resolves it from the configured secret manager.
func (p *VaultPlugin) PreLLMHook(ctx *schemas.BifrostContext, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.LLMPluginShortCircuit, error) {
	if !p.enabled || p.provider == nil {
		return req, nil, nil
	}

	// Check the API key name in the context for a vault: reference
	keyNameRaw := ctx.Value(schemas.BifrostContextKeyAPIKeyName)
	keyName, ok := keyNameRaw.(string)
	if !ok || keyName == "" {
		return req, nil, nil
	}

	path, key, isRef := parseVaultRef(keyName)
	if !isRef {
		return req, nil, nil
	}

	secret, err := p.provider.GetSecret(ctx, path, key)
	if err != nil {
		p.logger.Warn("vault: failed to resolve secret at path=%s key=%s: %v", path, key, err)
		return req, nil, nil // Don't block the request; let key selection handle the missing key
	}

	// Inject the resolved secret as the direct key
	ctx.SetValue(schemas.BifrostContextKeyDirectKey, schemas.Key{
		Value: schemas.EnvVar{Val: secret},
	})
	// Clear the vault reference so it doesn't get used as a literal key name
	ctx.SetValue(schemas.BifrostContextKeyAPIKeyName, "")

	p.logger.Debug("vault: resolved secret for path=%s key=%s", path, key)
	return req, nil, nil
}

// PostLLMHook is a no-op for the vault plugin.
func (p *VaultPlugin) PostLLMHook(_ *schemas.BifrostContext, resp *schemas.BifrostResponse, bifrostErr *schemas.BifrostError) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
	return resp, bifrostErr, nil
}

// Compile-time interface checks.
var (
	_ schemas.BasePlugin = (*VaultPlugin)(nil)
	_ schemas.LLMPlugin  = (*VaultPlugin)(nil)
)
