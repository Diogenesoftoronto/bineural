package handlers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/configstore"
	"github.com/maximhq/bifrost/framework/configstore/tables"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
	"github.com/valyala/fasthttp"
)

// GuardrailsHandler manages HTTP requests for guardrail rule and profile operations.
type GuardrailsHandler struct {
	configStore configstore.ConfigStore
}

// NewGuardrailsHandler creates a new guardrails handler instance.
func NewGuardrailsHandler(configStore configstore.ConfigStore) *GuardrailsHandler {
	return &GuardrailsHandler{configStore: configStore}
}

// RegisterRoutes registers all guardrails-related routes.
func (h *GuardrailsHandler) RegisterRoutes(r *router.Router, middlewares ...schemas.BifrostHTTPMiddleware) {
	// Rules
	r.GET("/api/guardrails/rules", lib.ChainMiddlewares(h.listGuardrailRules, middlewares...))
	r.POST("/api/guardrails/rules", lib.ChainMiddlewares(h.createGuardrailRule, middlewares...))
	r.GET("/api/guardrails/rules/{id}", lib.ChainMiddlewares(h.getGuardrailRule, middlewares...))
	r.PUT("/api/guardrails/rules/{id}", lib.ChainMiddlewares(h.updateGuardrailRule, middlewares...))
	r.DELETE("/api/guardrails/rules/{id}", lib.ChainMiddlewares(h.deleteGuardrailRule, middlewares...))

	// Profiles
	r.GET("/api/guardrails/profiles", lib.ChainMiddlewares(h.listGuardrailProfiles, middlewares...))
	r.POST("/api/guardrails/profiles", lib.ChainMiddlewares(h.createGuardrailProfile, middlewares...))
	r.GET("/api/guardrails/profiles/{id}", lib.ChainMiddlewares(h.getGuardrailProfile, middlewares...))
	r.PUT("/api/guardrails/profiles/{id}", lib.ChainMiddlewares(h.updateGuardrailProfile, middlewares...))
	r.DELETE("/api/guardrails/profiles/{id}", lib.ChainMiddlewares(h.deleteGuardrailProfile, middlewares...))

	// Profile-Rule mapping
	r.POST("/api/guardrails/profiles/{id}/rules", lib.ChainMiddlewares(h.addRuleToProfile, middlewares...))
	r.GET("/api/guardrails/profiles/{id}/rules", lib.ChainMiddlewares(h.getProfileRules, middlewares...))
	r.DELETE("/api/guardrails/profiles/{id}/rules/{rule_id}", lib.ChainMiddlewares(h.removeRuleFromProfile, middlewares...))
}

// ─── Rules ────────────────────────────────────────────────────────────────────

func (h *GuardrailsHandler) listGuardrailRules(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	rules, err := h.configStore.ListGuardrailRules(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to list guardrail rules: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"rules": rules, "count": len(rules)})
}

func (h *GuardrailsHandler) createGuardrailRule(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var rule tables.TableGuardrailRule
	if err := json.Unmarshal(ctx.PostBody(), &rule); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if err := h.configStore.CreateGuardrailRule(ctx, &rule); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create guardrail rule: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "Guardrail rule created", "rule": rule}, 201)
}

func (h *GuardrailsHandler) getGuardrailRule(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	rule, err := h.configStore.GetGuardrailRule(ctx, id)
	if err != nil {
		if errors.Is(err, configstore.ErrNotFound) {
			SendError(ctx, 404, "Guardrail rule not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to get guardrail rule: %v", err))
		return
	}
	SendJSON(ctx, rule)
}

func (h *GuardrailsHandler) updateGuardrailRule(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	var rule tables.TableGuardrailRule
	if err := json.Unmarshal(ctx.PostBody(), &rule); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	rule.ID = mustParseUint(id)
	if err := h.configStore.UpdateGuardrailRule(ctx, &rule); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to update guardrail rule: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Guardrail rule updated"})
}

func (h *GuardrailsHandler) deleteGuardrailRule(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeleteGuardrailRule(ctx, id); err != nil {
		if errors.Is(err, configstore.ErrNotFound) {
			SendError(ctx, 404, "Guardrail rule not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to delete guardrail rule: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Guardrail rule deleted"})
}

// ─── Profiles ──────────────────────────────────────────────────────────────────

func (h *GuardrailsHandler) listGuardrailProfiles(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	profiles, err := h.configStore.ListGuardrailProfiles(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to list guardrail profiles: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"profiles": profiles, "count": len(profiles)})
}

func (h *GuardrailsHandler) createGuardrailProfile(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var profile tables.TableGuardrailProfile
	if err := json.Unmarshal(ctx.PostBody(), &profile); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if err := h.configStore.CreateGuardrailProfile(ctx, &profile); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create guardrail profile: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "Guardrail profile created", "profile": profile}, 201)
}

func (h *GuardrailsHandler) getGuardrailProfile(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	profile, err := h.configStore.GetGuardrailProfile(ctx, id)
	if err != nil {
		if errors.Is(err, configstore.ErrNotFound) {
			SendError(ctx, 404, "Guardrail profile not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to get guardrail profile: %v", err))
		return
	}
	SendJSON(ctx, profile)
}

func (h *GuardrailsHandler) updateGuardrailProfile(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	var profile tables.TableGuardrailProfile
	if err := json.Unmarshal(ctx.PostBody(), &profile); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	profile.ID = mustParseUint(id)
	if err := h.configStore.UpdateGuardrailProfile(ctx, &profile); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to update guardrail profile: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Guardrail profile updated"})
}

func (h *GuardrailsHandler) deleteGuardrailProfile(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeleteGuardrailProfile(ctx, id); err != nil {
		if errors.Is(err, configstore.ErrNotFound) {
			SendError(ctx, 404, "Guardrail profile not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to delete guardrail profile: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Guardrail profile deleted"})
}

// ─── Profile-Rule Mapping ──────────────────────────────────────────────────────

func (h *GuardrailsHandler) addRuleToProfile(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	profileID := ctx.UserValue("id").(string)
	var profileRule tables.TableGuardrailProfileRule
	if err := json.Unmarshal(ctx.PostBody(), &profileRule); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	profileRule.ProfileID = mustParseUint(profileID)
	if err := h.configStore.AddRuleToProfile(ctx, &profileRule); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to add rule to profile: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "Rule added to profile", "profile_rule": profileRule}, 201)
}

func (h *GuardrailsHandler) getProfileRules(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	profileID := ctx.UserValue("id").(string)
	rules, err := h.configStore.GetProfileRules(ctx, profileID)
	if err != nil {
		if errors.Is(err, configstore.ErrNotFound) {
			SendError(ctx, 404, "Guardrail profile not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to get profile rules: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"rules": rules, "count": len(rules)})
}

func (h *GuardrailsHandler) removeRuleFromProfile(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	profileID := ctx.UserValue("id").(string)
	ruleID := ctx.UserValue("rule_id").(string)
	if err := h.configStore.RemoveRuleFromProfile(ctx, profileID, ruleID); err != nil {
		if errors.Is(err, configstore.ErrNotFound) {
			SendError(ctx, 404, "Profile rule association not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to remove rule from profile: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Rule removed from profile"})
}

// mustParseUint parses a string as uint, returning 0 on failure.
func mustParseUint(s string) uint {
	var n uint
	fmt.Sscanf(s, "%d", &n)
	return n
}
