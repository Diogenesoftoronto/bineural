package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/configstore"
	"github.com/maximhq/bifrost/framework/configstore/tables"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
	"github.com/valyala/fasthttp"
)

// SubscriptionHandler manages HTTP requests for subscription operations
type SubscriptionHandler struct {
	configStore configstore.ConfigStore
}

// NewSubscriptionHandler creates a new subscription handler instance
func NewSubscriptionHandler(configStore configstore.ConfigStore) *SubscriptionHandler {
	return &SubscriptionHandler{configStore: configStore}
}

// RegisterRoutes registers all subscription-related routes
func (h *SubscriptionHandler) RegisterRoutes(r *router.Router, middlewares ...schemas.BifrostHTTPMiddleware) {
	r.GET("/api/subscriptions", lib.ChainMiddlewares(h.getProviderSubscriptions, middlewares...))
	r.POST("/api/subscriptions", lib.ChainMiddlewares(h.createProviderSubscription, middlewares...))
	r.GET("/api/subscriptions/{id}", lib.ChainMiddlewares(h.getProviderSubscription, middlewares...))
	r.PUT("/api/subscriptions/{id}", lib.ChainMiddlewares(h.updateProviderSubscription, middlewares...))
	r.DELETE("/api/subscriptions/{id}", lib.ChainMiddlewares(h.deleteProviderSubscription, middlewares...))

	r.GET("/api/billing/tiers", lib.ChainMiddlewares(h.getSaaSBillingTiers, middlewares...))
	r.POST("/api/billing/tiers", lib.ChainMiddlewares(h.createSaaSBillingTier, middlewares...))
	r.PUT("/api/billing/tiers/{id}", lib.ChainMiddlewares(h.updateSaaSBillingTier, middlewares...))
	r.DELETE("/api/billing/tiers/{id}", lib.ChainMiddlewares(h.deleteSaaSBillingTier, middlewares...))

	r.GET("/api/billing/user-subscriptions", lib.ChainMiddlewares(h.getUserSubscriptions, middlewares...))
	r.POST("/api/billing/user-subscriptions", lib.ChainMiddlewares(h.createUserSubscription, middlewares...))
	r.PUT("/api/billing/user-subscriptions/{id}", lib.ChainMiddlewares(h.updateUserSubscription, middlewares...))
	r.DELETE("/api/billing/user-subscriptions/{id}", lib.ChainMiddlewares(h.deleteUserSubscription, middlewares...))
}

// ─── Provider Subscriptions ────────────────────────────────────────────────────

func (h *SubscriptionHandler) getProviderSubscriptions(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	subs, err := h.configStore.GetProviderSubscriptions(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get subscriptions: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"subscriptions": subs, "count": len(subs)})
}

func (h *SubscriptionHandler) createProviderSubscription(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var sub tables.TableProviderSubscription
	if err := json.Unmarshal(ctx.PostBody(), &sub); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if sub.ID == "" {
		sub.ID = fmt.Sprintf("sub-%d", time.Now().UnixNano())
	}
	sub.LastResetAt = time.Now()
	if err := h.configStore.CreateProviderSubscription(ctx, &sub); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create subscription: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Subscription created", "subscription": sub})
}

func (h *SubscriptionHandler) getProviderSubscription(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	sub, err := h.configStore.GetProviderSubscription(ctx, id)
	if err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "Subscription not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to get subscription: %v", err))
		return
	}
	SendJSON(ctx, sub)
}

func (h *SubscriptionHandler) updateProviderSubscription(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	var sub tables.TableProviderSubscription
	if err := json.Unmarshal(ctx.PostBody(), &sub); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	sub.ID = id
	if err := h.configStore.UpdateProviderSubscription(ctx, &sub); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to update subscription: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Subscription updated"})
}

func (h *SubscriptionHandler) deleteProviderSubscription(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeleteProviderSubscription(ctx, id); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to delete subscription: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Subscription deleted"})
}

// ─── SaaS Billing Tiers ───────────────────────────────────────────────────────

func (h *SubscriptionHandler) getSaaSBillingTiers(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	tiers, err := h.configStore.GetSaaSBillingTiers(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get billing tiers: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"tiers": tiers, "count": len(tiers)})
}

func (h *SubscriptionHandler) createSaaSBillingTier(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var tier tables.TableSaaSBillingTier
	if err := json.Unmarshal(ctx.PostBody(), &tier); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if tier.ID == "" {
		tier.ID = fmt.Sprintf("tier-%d", time.Now().UnixNano())
	}
	if err := h.configStore.CreateSaaSBillingTier(ctx, &tier); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create billing tier: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Billing tier created", "tier": tier})
}

func (h *SubscriptionHandler) updateSaaSBillingTier(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	var tier tables.TableSaaSBillingTier
	if err := json.Unmarshal(ctx.PostBody(), &tier); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	tier.ID = id
	if err := h.configStore.UpdateSaaSBillingTier(ctx, &tier); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to update billing tier: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Billing tier updated"})
}

func (h *SubscriptionHandler) deleteSaaSBillingTier(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeleteSaaSBillingTier(ctx, id); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to delete billing tier: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Billing tier deleted"})
}

// ─── User Subscriptions ───────────────────────────────────────────────────────

func (h *SubscriptionHandler) getUserSubscriptions(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	if vkID := string(ctx.QueryArgs().Peek("virtual_key_id")); vkID != "" {
		subs, err := h.configStore.GetUserSubscriptionsByVirtualKeyID(ctx, vkID)
		if err != nil {
			SendError(ctx, 500, fmt.Sprintf("Failed to get user subscriptions: %v", err))
			return
		}
		SendJSON(ctx, map[string]interface{}{"subscriptions": subs, "count": len(subs)})
		return
	}
	if userID := string(ctx.QueryArgs().Peek("user_id")); userID != "" {
		subs, err := h.configStore.GetUserSubscriptionsByUserID(ctx, userID)
		if err != nil {
			SendError(ctx, 500, fmt.Sprintf("Failed to get user subscriptions: %v", err))
			return
		}
		SendJSON(ctx, map[string]interface{}{"subscriptions": subs, "count": len(subs)})
		return
	}
	subs, err := h.configStore.GetUserSubscriptions(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get user subscriptions: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"subscriptions": subs, "count": len(subs)})
}

func (h *SubscriptionHandler) createUserSubscription(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var sub tables.TableUserSubscription
	if err := json.Unmarshal(ctx.PostBody(), &sub); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if sub.ID == "" {
		sub.ID = fmt.Sprintf("usub-%d", time.Now().UnixNano())
	}
	if sub.StartedAt.IsZero() {
		sub.StartedAt = time.Now()
	}
	if err := h.configStore.CreateUserSubscription(ctx, &sub); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create user subscription: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "User subscription created", "subscription": sub})
}

func (h *SubscriptionHandler) updateUserSubscription(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	var sub tables.TableUserSubscription
	if err := json.Unmarshal(ctx.PostBody(), &sub); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	sub.ID = id
	if err := h.configStore.UpdateUserSubscription(ctx, &sub); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to update user subscription: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "User subscription updated"})
}

func (h *SubscriptionHandler) deleteUserSubscription(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeleteUserSubscription(ctx, id); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to delete user subscription: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "User subscription deleted"})
}
