package server

import (
	"context"
	"fmt"
	"slices"

	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/plugins/compat"
	"github.com/maximhq/bifrost/plugins/governance"
	"github.com/maximhq/bifrost/plugins/logging"
	"github.com/maximhq/bifrost/plugins/maxim"
	"github.com/maximhq/bifrost/plugins/otel"
	"github.com/maximhq/bifrost/plugins/prompts"
	"github.com/maximhq/bifrost/plugins/semanticcache"
	"github.com/maximhq/bifrost/plugins/telemetry"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/accessprofiles"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/alertchannels"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/audit"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/clustering"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/dataconnectors"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/evals"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/guardrails"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/largepayload"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/loadbalancer"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/piiredactor"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/promptdeploy"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/quotatracker"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/rbac"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/scim"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/scopedkeys"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/sso"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/usergovernance"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/vault"
	"github.com/maximhq/bifrost/transports/bifrost-http/handlers"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
)

// InferPluginTypes determines which interface types a plugin implements
func InferPluginTypes(plugin schemas.BasePlugin) []schemas.PluginType {
	var types []schemas.PluginType
	if _, ok := plugin.(schemas.LLMPlugin); ok {
		types = append(types, schemas.PluginTypeLLM)
	}
	if _, ok := plugin.(schemas.MCPPlugin); ok {
		types = append(types, schemas.PluginTypeMCP)
	}
	if _, ok := plugin.(schemas.HTTPTransportPlugin); ok {
		types = append(types, schemas.PluginTypeHTTP)
	}
	return types
}

// Single-plugin methods used plugin create/update

// InstantiatePlugin creates a plugin instance but does NOT register it
// Registration is done separately via Config.RegisterPlugin()
func InstantiatePlugin(ctx context.Context, name string, path *string, pluginConfig any, bifrostConfig *lib.Config) (schemas.BasePlugin, error) {
	// Custom plugin (has path)
	if path != nil {
		return loadCustomPlugin(ctx, path, pluginConfig, bifrostConfig)
	}

	// Built-in plugin (by name)
	return loadBuiltinPlugin(ctx, name, pluginConfig, bifrostConfig)
}

// loadBuiltinPlugin instantiates a built-in plugin by name
func loadBuiltinPlugin(ctx context.Context, name string, pluginConfig any, bifrostConfig *lib.Config) (schemas.BasePlugin, error) {
	switch name {
	case telemetry.PluginName:
		telConfig := &telemetry.Config{
			CustomLabels: bifrostConfig.ClientConfig.PrometheusLabels,
		}
		// Merge push gateway config if provided (e.g., from config file or UI update)
		if pluginConfig != nil {
			extraConfig, err := MarshalPluginConfig[telemetry.Config](pluginConfig)
			if err == nil && extraConfig != nil && extraConfig.PushGateway != nil {
				telConfig.PushGateway = extraConfig.PushGateway
			}
		}
		return telemetry.Init(telConfig, bifrostConfig.ModelCatalog, logger)

	case prompts.PluginName:
		return prompts.Init(ctx, bifrostConfig.ConfigStore, logger)

	case logging.PluginName:
		loggingConfig, err := MarshalPluginConfig[logging.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal logging plugin config: %w", err)
		}
		return logging.Init(ctx, loggingConfig, logger, bifrostConfig.LogsStore,
			bifrostConfig.ModelCatalog, bifrostConfig.MCPCatalog)

	case governance.PluginName:
		governanceConfig, err := MarshalPluginConfig[governance.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal governance plugin config: %w", err)
		}
		inMemoryStore := &GovernanceInMemoryStore{Config: bifrostConfig}
		return governance.Init(ctx, governanceConfig, logger, bifrostConfig.ConfigStore,
			bifrostConfig.GovernanceConfig, bifrostConfig.ModelCatalog,
			bifrostConfig.MCPCatalog, inMemoryStore)

	case maxim.PluginName:
		maximConfig, err := MarshalPluginConfig[maxim.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal maxim plugin config: %w", err)
		}
		return maxim.Init(maximConfig, logger)

	case semanticcache.PluginName:
		semanticConfig, err := MarshalPluginConfig[semanticcache.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal semantic cache plugin config: %w", err)
		}
		return semanticcache.Init(ctx, semanticConfig, logger, bifrostConfig.VectorStore)

	case otel.PluginName:
		otelConfig, err := MarshalPluginConfig[otel.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal otel plugin config: %w", err)
		}
		return otel.Init(ctx, otelConfig, logger, bifrostConfig.ModelCatalog, handlers.GetVersion())

	case compat.PluginName:
		compatConfig, err := MarshalPluginConfig[compat.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal compat plugin config: %w", err)
		}
		return compat.Init(*compatConfig, logger, bifrostConfig.ModelCatalog)

	case rbac.PluginName:
		rbacConfig, err := MarshalPluginConfig[rbac.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal rbac plugin config: %w", err)
		}
		return rbac.Init(rbacConfig, logger), nil

	case audit.PluginName:
		auditConfig, err := MarshalPluginConfig[audit.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal audit plugin config: %w", err)
		}
		return audit.Init(auditConfig, logger), nil

	case guardrails.PluginName:
		guardConfig, err := MarshalPluginConfig[guardrails.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal guardrails plugin config: %w", err)
		}
		return guardrails.Init(guardConfig, logger), nil

	case sso.PluginName:
		ssoConfig, err := MarshalPluginConfig[sso.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal sso plugin config: %w", err)
		}
		return sso.Init(ssoConfig, logger), nil

	case clustering.PluginName:
		clusterConfig, err := MarshalPluginConfig[clustering.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal clustering plugin config: %w", err)
		}
		return clustering.Init(clusterConfig, logger), nil

	case loadbalancer.PluginName:
		lbConfig, err := MarshalPluginConfig[loadbalancer.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal adaptive_loadbalancer plugin config: %w", err)
		}
		return loadbalancer.Init(lbConfig, logger), nil

	case scim.PluginName:
		scimConfig, err := MarshalPluginConfig[scim.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal scim plugin config: %w", err)
		}
		return scim.Init(scimConfig, logger), nil

	case alertchannels.PluginName:
		acConfig, err := MarshalPluginConfig[alertchannels.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal alert_channels plugin config: %w", err)
		}
		return alertchannels.Init(acConfig, logger), nil

	case evals.PluginName:
		evalConfig, err := MarshalPluginConfig[evals.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal evals plugin config: %w", err)
		}
		return evals.Init(evalConfig, logger), nil

	case piiredactor.PluginName:
		piiConfig, err := MarshalPluginConfig[piiredactor.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal pii_redactor plugin config: %w", err)
		}
		return piiredactor.Init(piiConfig, logger), nil

	case usergovernance.PluginName:
		ugConfig, err := MarshalPluginConfig[usergovernance.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user_governance plugin config: %w", err)
		}
		return usergovernance.Init(ugConfig, logger), nil

	case accessprofiles.PluginName:
		apConfig, err := MarshalPluginConfig[accessprofiles.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal access_profiles plugin config: %w", err)
		}
		return accessprofiles.Init(apConfig, logger), nil

	case scopedkeys.PluginName:
		skConfig, err := MarshalPluginConfig[scopedkeys.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal scoped_api_keys plugin config: %w", err)
		}
		return scopedkeys.Init(skConfig, logger), nil

	case promptdeploy.PluginName:
		pdConfig, err := MarshalPluginConfig[promptdeploy.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal prompt_deployments plugin config: %w", err)
		}
		return promptdeploy.Init(pdConfig, logger), nil

	case dataconnectors.PluginName:
		dcConfig, err := MarshalPluginConfig[dataconnectors.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data_connectors plugin config: %w", err)
		}
		return dataconnectors.Init(dcConfig, logger), nil

	case largepayload.PluginName:
		lpConfig, err := MarshalPluginConfig[largepayload.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal large_payload plugin config: %w", err)
		}
		return largepayload.Init(lpConfig, logger), nil

	case quotatracker.PluginName:
		qtConfig, err := MarshalPluginConfig[quotatracker.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal quota_tracker plugin config: %w", err)
		}
		return quotatracker.Init(qtConfig, logger), nil

	case vault.PluginName:
		vaultConfig, err := MarshalPluginConfig[vault.Config](pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal vault plugin config: %w", err)
		}
		return vault.Init(vaultConfig, logger), nil

	default:
		return nil, fmt.Errorf("unknown built-in plugin: %s", name)
	}
}

// loadCustomPlugin loads a plugin from a shared object file
func loadCustomPlugin(ctx context.Context, path *string, pluginConfig any, bifrostConfig *lib.Config) (schemas.BasePlugin, error) {
	logger.Info("loading custom plugin from path %s", *path)

	plugin, err := bifrostConfig.PluginLoader.LoadPlugin(*path, pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom plugin: %w", err)
	}
	return plugin, nil
}

// LoadPlugins loads the plugins for the server.
func (s *BifrostHTTPServer) LoadPlugins(ctx context.Context) error {
	// Load built-in plugins first (order matters)
	if err := s.loadBuiltinPlugins(ctx); err != nil {
		return err
	}
	// Load custom plugins from config
	if err := s.loadCustomPlugins(ctx); err != nil {
		return err
	}
	// Sort all plugins by placement group and order
	s.Config.SortAndRebuildPlugins()
	return nil
}

// getPluginConfig retrieves a plugin's config from PluginConfigs by name
func (s *BifrostHTTPServer) getPluginConfig(name string) *schemas.PluginConfig {
	for _, cfg := range s.Config.PluginConfigs {
		if cfg.Name == name {
			return cfg
		}
	}
	return nil
}

// loadBuiltinPlugins loads required built-in plugins in specific order
func (s *BifrostHTTPServer) loadBuiltinPlugins(ctx context.Context) error {
	builtinPlacement := schemas.Ptr(schemas.PluginPlacementBuiltin)

	// 1. Telemetry (always first - tracks everything)
	if err := s.registerPluginWithStatus(ctx, telemetry.PluginName, nil, nil, true); err != nil {
		return err
	}
	s.Config.SetPluginOrderInfo(telemetry.PluginName, builtinPlacement, schemas.Ptr(1))

	// 2. Prompts (requires config store for prompt repository; disabled in enterprise)
	if s.Config.ConfigStore != nil && ctx.Value(schemas.BifrostContextKeyIsEnterprise) == nil {
		s.registerPluginWithStatus(ctx, prompts.PluginName, nil, nil, false)
	} else {
		s.markPluginDisabled(prompts.PluginName)
	}
	s.Config.SetPluginOrderInfo(prompts.PluginName, builtinPlacement, schemas.Ptr(2))

	// 3. Logging (if enabled)
	if (s.Config.ClientConfig.EnableLogging == nil || *s.Config.ClientConfig.EnableLogging) && s.Config.LogsStore != nil {
		config := &logging.Config{
			DisableContentLogging: &s.Config.ClientConfig.DisableContentLogging,
			LoggingHeaders:        &s.Config.ClientConfig.LoggingHeaders,
		}
		s.registerPluginWithStatus(ctx, logging.PluginName, nil, config, false)
	} else {
		s.markPluginDisabled(logging.PluginName)
	}
	s.Config.SetPluginOrderInfo(logging.PluginName, builtinPlacement, schemas.Ptr(3))

	// 4. Governance (if enabled and not enterprise)
	if ctx.Value(schemas.BifrostContextKeyIsEnterprise) == nil {
		config := &governance.Config{
			IsVkMandatory:         &s.Config.ClientConfig.EnforceAuthOnInference,
			RequiredHeaders:       &s.Config.ClientConfig.RequiredHeaders,
			DisableAutoToolInject: &s.Config.ClientConfig.MCPDisableAutoToolInject,
			RoutingChainMaxDepth:  &s.Config.ClientConfig.RoutingChainMaxDepth,
		}
		s.registerPluginWithStatus(ctx, governance.PluginName, nil, config, false)
	} else {
		s.markPluginDisabled(governance.PluginName)
	}
	s.Config.SetPluginOrderInfo(governance.PluginName, builtinPlacement, schemas.Ptr(4))

	// 5. OTEL (if configured in PluginConfigs)
	otelConfig := s.getPluginConfig(otel.PluginName)
	if otelConfig != nil && otelConfig.Enabled {
		s.registerPluginWithStatus(ctx, otel.PluginName, nil, otelConfig.Config, false)
	} else {
		s.markPluginDisabled(otel.PluginName)
	}
	s.Config.SetPluginOrderInfo(otel.PluginName, builtinPlacement, schemas.Ptr(5))

	// 6. Semantic Cache (if configured in PluginConfigs)
	semanticCacheConfig := s.getPluginConfig(semanticcache.PluginName)
	if semanticCacheConfig != nil && semanticCacheConfig.Enabled {
		s.registerPluginWithStatus(ctx, semanticcache.PluginName, nil, semanticCacheConfig.Config, false)
	} else {
		s.markPluginDisabled(semanticcache.PluginName)
	}
	s.Config.SetPluginOrderInfo(semanticcache.PluginName, builtinPlacement, schemas.Ptr(6))

	// 7. Compat (if any compat feature is enabled in ClientConfig)
	cc := s.Config.ClientConfig.Compat
	compatCfg := &compat.Config{
		ConvertTextToChat:      cc.ConvertTextToChat,
		ConvertChatToResponses: cc.ConvertChatToResponses,
		ShouldDropParams:       cc.ShouldDropParams,
		ShouldConvertParams:    cc.ShouldConvertParams,
	}
	s.registerPluginWithStatus(ctx, compat.PluginName, nil, compatCfg, false)
	s.Config.SetPluginOrderInfo(compat.PluginName, builtinPlacement, schemas.Ptr(7))

	// 8. Maxim (if configured in PluginConfigs)
	maximConfig := s.getPluginConfig(maxim.PluginName)
	if maximConfig != nil && maximConfig.Enabled {
		s.registerPluginWithStatus(ctx, maxim.PluginName, nil, maximConfig.Config, false)
	} else {
		s.markPluginDisabled(maxim.PluginName)
	}
	s.Config.SetPluginOrderInfo(maxim.PluginName, builtinPlacement, schemas.Ptr(8))

	// 9-14. Enterprise plugins (when enterprise mode is enabled)
	if ctx.Value(schemas.BifrostContextKeyIsEnterprise) != nil {
		// RBAC
		rbacConfig := s.getPluginConfig(rbac.PluginName)
		if rbacConfig != nil && rbacConfig.Enabled {
			s.registerPluginWithStatus(ctx, rbac.PluginName, nil, rbacConfig.Config, false)
		} else {
			s.markPluginDisabled(rbac.PluginName)
		}
		s.Config.SetPluginOrderInfo(rbac.PluginName, builtinPlacement, schemas.Ptr(9))

		// Audit Logs
		auditConfig := s.getPluginConfig(audit.PluginName)
		if auditConfig != nil && auditConfig.Enabled {
			s.registerPluginWithStatus(ctx, audit.PluginName, nil, auditConfig.Config, false)
		} else {
			s.markPluginDisabled(audit.PluginName)
		}
		s.Config.SetPluginOrderInfo(audit.PluginName, builtinPlacement, schemas.Ptr(10))

		// Guardrails
		guardConfig := s.getPluginConfig(guardrails.PluginName)
		if guardConfig != nil && guardConfig.Enabled {
			s.registerPluginWithStatus(ctx, guardrails.PluginName, nil, guardConfig.Config, false)
		} else {
			s.markPluginDisabled(guardrails.PluginName)
		}
		s.Config.SetPluginOrderInfo(guardrails.PluginName, builtinPlacement, schemas.Ptr(11))

		// SSO
		ssoConfig := s.getPluginConfig(sso.PluginName)
		if ssoConfig != nil && ssoConfig.Enabled {
			s.registerPluginWithStatus(ctx, sso.PluginName, nil, ssoConfig.Config, false)
		} else {
			s.markPluginDisabled(sso.PluginName)
		}
		s.Config.SetPluginOrderInfo(sso.PluginName, builtinPlacement, schemas.Ptr(12))

		// Clustering
		clusterConfig := s.getPluginConfig(clustering.PluginName)
		if clusterConfig != nil && clusterConfig.Enabled {
			s.registerPluginWithStatus(ctx, clustering.PluginName, nil, clusterConfig.Config, false)
		} else {
			s.markPluginDisabled(clustering.PluginName)
		}
		s.Config.SetPluginOrderInfo(clustering.PluginName, builtinPlacement, schemas.Ptr(13))

		// Adaptive Load Balancer
		lbConfig := s.getPluginConfig(loadbalancer.PluginName)
		if lbConfig != nil && lbConfig.Enabled {
			s.registerPluginWithStatus(ctx, loadbalancer.PluginName, nil, lbConfig.Config, false)
		} else {
			s.markPluginDisabled(loadbalancer.PluginName)
		}
		s.Config.SetPluginOrderInfo(loadbalancer.PluginName, builtinPlacement, schemas.Ptr(14))

		// SCIM
		scimCfg := s.getPluginConfig(scim.PluginName)
		if scimCfg != nil && scimCfg.Enabled {
			s.registerPluginWithStatus(ctx, scim.PluginName, nil, scimCfg.Config, false)
		} else {
			s.markPluginDisabled(scim.PluginName)
		}
		s.Config.SetPluginOrderInfo(scim.PluginName, builtinPlacement, schemas.Ptr(15))

		// Alert Channels
		acCfg := s.getPluginConfig(alertchannels.PluginName)
		if acCfg != nil && acCfg.Enabled {
			s.registerPluginWithStatus(ctx, alertchannels.PluginName, nil, acCfg.Config, false)
		} else {
			s.markPluginDisabled(alertchannels.PluginName)
		}
		s.Config.SetPluginOrderInfo(alertchannels.PluginName, builtinPlacement, schemas.Ptr(16))

		// Evals
		evalCfg := s.getPluginConfig(evals.PluginName)
		if evalCfg != nil && evalCfg.Enabled {
			s.registerPluginWithStatus(ctx, evals.PluginName, nil, evalCfg.Config, false)
		} else {
			s.markPluginDisabled(evals.PluginName)
		}
		s.Config.SetPluginOrderInfo(evals.PluginName, builtinPlacement, schemas.Ptr(17))

		// PII Redactor
		piiCfg := s.getPluginConfig(piiredactor.PluginName)
		if piiCfg != nil && piiCfg.Enabled {
			s.registerPluginWithStatus(ctx, piiredactor.PluginName, nil, piiCfg.Config, false)
		} else {
			s.markPluginDisabled(piiredactor.PluginName)
		}
		s.Config.SetPluginOrderInfo(piiredactor.PluginName, builtinPlacement, schemas.Ptr(18))

		// User Governance
		ugCfg := s.getPluginConfig(usergovernance.PluginName)
		if ugCfg != nil && ugCfg.Enabled {
			s.registerPluginWithStatus(ctx, usergovernance.PluginName, nil, ugCfg.Config, false)
		} else {
			s.markPluginDisabled(usergovernance.PluginName)
		}
		s.Config.SetPluginOrderInfo(usergovernance.PluginName, builtinPlacement, schemas.Ptr(19))

		// Access Profiles
		apCfg := s.getPluginConfig(accessprofiles.PluginName)
		if apCfg != nil && apCfg.Enabled {
			s.registerPluginWithStatus(ctx, accessprofiles.PluginName, nil, apCfg.Config, false)
		} else {
			s.markPluginDisabled(accessprofiles.PluginName)
		}
		s.Config.SetPluginOrderInfo(accessprofiles.PluginName, builtinPlacement, schemas.Ptr(20))

		// Scoped API Keys
		skCfg := s.getPluginConfig(scopedkeys.PluginName)
		if skCfg != nil && skCfg.Enabled {
			s.registerPluginWithStatus(ctx, scopedkeys.PluginName, nil, skCfg.Config, false)
		} else {
			s.markPluginDisabled(scopedkeys.PluginName)
		}
		s.Config.SetPluginOrderInfo(scopedkeys.PluginName, builtinPlacement, schemas.Ptr(21))

		// Prompt Deployments
		pdCfg := s.getPluginConfig(promptdeploy.PluginName)
		if pdCfg != nil && pdCfg.Enabled {
			s.registerPluginWithStatus(ctx, promptdeploy.PluginName, nil, pdCfg.Config, false)
		} else {
			s.markPluginDisabled(promptdeploy.PluginName)
		}
		s.Config.SetPluginOrderInfo(promptdeploy.PluginName, builtinPlacement, schemas.Ptr(22))

		// Data Connectors
		dcCfg := s.getPluginConfig(dataconnectors.PluginName)
		if dcCfg != nil && dcCfg.Enabled {
			s.registerPluginWithStatus(ctx, dataconnectors.PluginName, nil, dcCfg.Config, false)
		} else {
			s.markPluginDisabled(dataconnectors.PluginName)
		}
		s.Config.SetPluginOrderInfo(dataconnectors.PluginName, builtinPlacement, schemas.Ptr(23))

		// Large Payload
		lpCfg := s.getPluginConfig(largepayload.PluginName)
		if lpCfg != nil && lpCfg.Enabled {
			s.registerPluginWithStatus(ctx, largepayload.PluginName, nil, lpCfg.Config, false)
		} else {
			s.markPluginDisabled(largepayload.PluginName)
		}
		s.Config.SetPluginOrderInfo(largepayload.PluginName, builtinPlacement, schemas.Ptr(24))

		// Quota Tracker
		qtCfg := s.getPluginConfig(quotatracker.PluginName)
		if qtCfg != nil && qtCfg.Enabled {
			s.registerPluginWithStatus(ctx, quotatracker.PluginName, nil, qtCfg.Config, false)
		} else {
			s.markPluginDisabled(quotatracker.PluginName)
		}
		s.Config.SetPluginOrderInfo(quotatracker.PluginName, builtinPlacement, schemas.Ptr(25))

		// Vault
		vaultCfg := s.getPluginConfig(vault.PluginName)
		if vaultCfg != nil && vaultCfg.Enabled {
			s.registerPluginWithStatus(ctx, vault.PluginName, nil, vaultCfg.Config, false)
		} else {
			s.markPluginDisabled(vault.PluginName)
		}
		s.Config.SetPluginOrderInfo(vault.PluginName, builtinPlacement, schemas.Ptr(26))

		// Wire cross-plugin integrations
		s.wireEnterpriseIntegrations()
	}

	return nil
}

// wireEnterpriseIntegrations connects enterprise plugins to each other for cross-cutting behavior.
func (s *BifrostHTTPServer) wireEnterpriseIntegrations() {
	// Guardrails → Audit: log guardrail violations as audit entries
	guardrailsPlugin, _ := lib.FindPluginAs[*guardrails.GuardrailsPlugin](s.Config, guardrails.PluginName)
	auditPlugin, _ := lib.FindPluginAs[*audit.AuditPlugin](s.Config, audit.PluginName)
	if guardrailsPlugin != nil && auditPlugin != nil {
		guardrailsPlugin.OnViolation(func(v *guardrails.Violation, contentType string) {
			auditPlugin.Log(audit.AuditLogEntry{
				EventType: audit.EventSecurity,
				Action:    "guardrail_violation",
				Resource:  "guardrail:" + string(v.Type),
				Status:    string(v.Action),
				Details: map[string]any{
					"rule_id":      v.RuleID,
					"rule_name":    v.RuleName,
					"content_type": contentType,
					"message":      v.Message,
					"matched":      v.Matched,
				},
			})
		})
	}

	// QuotaTracker → AlertChannels: fire alert channel notifications on quota threshold crossings
	qtPlugin, _ := lib.FindPluginAs[*quotatracker.QuotaTrackerPlugin](s.Config, quotatracker.PluginName)
	acPlugin, _ := lib.FindPluginAs[*alertchannels.AlertChannelsPlugin](s.Config, alertchannels.PluginName)
	if qtPlugin != nil && acPlugin != nil {
		qtPlugin.RegisterAlertCallback(func(alert *quotatracker.QuotaAlert) {
			severity := "warning"
			if alert.Meter != nil && alert.Meter.UtilizationPercent >= 95 {
				severity = "critical"
			}
			meterKey := "unknown"
			if alert.Meter != nil {
				meterKey = alert.Meter.Key
			}
			acPlugin.FireAlert("quota_alert", fmt.Sprintf("Quota alert: %s at %.1f%% utilization", meterKey, alert.Meter.UtilizationPercent), severity)
		})
	}
}

// loadCustomPlugins loads plugins from PluginConfigs
func (s *BifrostHTTPServer) loadCustomPlugins(ctx context.Context) error {
	for _, cfg := range s.Config.PluginConfigs {
		// Skip built-ins (already loaded)
		if lib.IsBuiltinPlugin(cfg.Name) {
			continue
		}
		// Handle disabled plugins
		if !cfg.Enabled {
			// For custom plugins with a path, verify to get the real plugin name
			if cfg.Path != nil {
				pluginName, err := s.Config.PluginLoader.VerifyBasePlugin(*cfg.Path)
				if err != nil {
					logger.Error("failed to verify disabled plugin %s: %v", cfg.Name, err)
					continue
				}
				// Store plugin status without instantiating (no Init() call, no resource usage)
				// Note: We can't determine types without instantiating, so pass empty slice
				s.Config.UpdatePluginOverallStatus(pluginName, cfg.Name, schemas.PluginStatusDisabled,
					[]string{fmt.Sprintf("plugin %s is disabled", cfg.Name)}, []schemas.PluginType{})
			} else {
				// Built-in plugin - use cfg.Name directly
				s.Config.UpdatePluginOverallStatus(cfg.Name, cfg.Name, schemas.PluginStatusDisabled,
					[]string{fmt.Sprintf("plugin %s is disabled", cfg.Name)}, []schemas.PluginType{})
			}
			continue
		}

		// Plugin is enabled - instantiate it
		plugin, err := InstantiatePlugin(ctx, cfg.Name, cfg.Path, cfg.Config, s.Config)
		if err != nil {
			// Skip enterprise plugins silently
			if slices.Contains(enterprisePlugins, cfg.Name) {
				continue
			}
			logger.Error("failed to load plugin %s: %v", cfg.Name, err)
			// Use cfg.Name since plugin may be nil when InstantiatePlugin returns an error
			s.Config.UpdatePluginOverallStatus(cfg.Name, cfg.Name, schemas.PluginStatusError,
				[]string{fmt.Sprintf("error loading plugin %s: %v", cfg.Name, err)}, []schemas.PluginType{})
			continue
		}

		// Ensure plugin is not nil before using it (defensive check)
		if plugin == nil {
			logger.Error("plugin %s instantiated but returned nil", cfg.Name)
			s.Config.UpdatePluginOverallStatus(cfg.Name, cfg.Name, schemas.PluginStatusError,
				[]string{fmt.Sprintf("plugin %s instantiated but returned nil", cfg.Name)}, []schemas.PluginType{})
			continue
		}

		// Register enabled plugin and mark as active
		s.Config.ReloadPlugin(plugin)
		s.Config.SetPluginOrderInfo(plugin.GetName(), cfg.Placement, cfg.Order)
		s.Config.UpdatePluginOverallStatus(plugin.GetName(), cfg.Name, schemas.PluginStatusActive,
			[]string{fmt.Sprintf("plugin %s initialized successfully", cfg.Name)}, InferPluginTypes(plugin))
	}
	return nil
}
