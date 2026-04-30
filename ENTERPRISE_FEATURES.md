# Bifrost Enterprise — Unlocked Features

This fork implements the premium enterprise features that Maxim AI gates behind custom pricing tiers. All features are now available in the open-source build.

## How to Enable Enterprise Mode

Add to your `config.json`:

```json
{
  "enable_enterprise": true,
  "plugins": {
    "rbac": { "enabled": true, "config": {} },
    "audit": { "enabled": true, "config": {} },
    "guardrails": { "enabled": true, "config": {} },
    "sso": { "enabled": true, "config": {} },
    "clustering": { "enabled": true, "config": {} },
    "adaptive_loadbalancer": { "enabled": true, "config": {} }
  }
}
```

## Implemented Enterprise Features

### 1. RBAC (Role-Based Access Control)
**File**: `transports/bifrost-http/enterprise/rbac/`

- System roles: Admin, Developer, Viewer
- Custom role creation with per-resource CRUD permissions
- 14 protected resources (virtual keys, teams, budgets, guardrails, etc.)
- HTTP middleware intercepts all API calls and enforces permissions

### 2. Audit Logs
**File**: `transports/bifrost-http/enterprise/audit/`

- Immutable HMAC-signed audit log entries
- Event types: auth, authz, config_change, data_access, security, inference, admin
- Queryable by event type, user, resource
- Configurable buffer size and retention

### 3. Guardrails / Content Safety
**File**: `transports/bifrost-http/enterprise/guardrails/`

- Rule types: blocklist, allowlist, regex, PII detection, content filtering
- Actions: block, log, mask, flag
- Input/output evaluation via PreLLMHook and PostLLMHook
- Built-in PII detection (emails, SSNs, phone numbers, credit cards)
- Profile-based rule grouping

### 4. SSO / OIDC
**File**: `transports/bifrost-http/enterprise/sso/`

- Multi-provider OIDC support (Okta, Entra, Keycloak, Google Workspace)
- OAuth 2.0 authorization code flow with state parameter validation
- User claim extraction (ID, email, name, groups)
- Configurable scopes and redirect URLs

### 5. Clustering
**File**: `transports/bifrost-http/enterprise/clustering/`

- Node ID assignment with peer discovery
- Consistent hashing for key-to-node routing
- Peer health tracking
- Configurable gossip interval and sync port

### 6. Adaptive Load Balancing
**File**: `transports/bifrost-http/enterprise/loadbalancer/`

- Real-time per-key performance metrics
- 5-second weight recalculation based on:
  - Error rate (50% weight)
  - Latency (20% weight)
  - Utilization (5% weight)
- Weighted random key selection with configurable exploration (25%)
- Automatic recovery for degraded keys

## Architecture

All enterprise features are implemented as:
- **Built-in plugins** in `transports/bifrost-http/enterprise/`
- Registered in plugin loading system (`server/plugins.go`)
- Conditionally loaded when `enable_enterprise: true`
- Context flag `BifrostContextKeyIsEnterprise` gates enterprise-only behavior
- No external dependencies beyond existing Bifrost infrastructure

## License

Same as upstream: Apache 2.0
