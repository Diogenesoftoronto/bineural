package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/quotatracker"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
	"github.com/valyala/fasthttp"
)

// QuotaHandler manages HTTP requests for quota tracking operations
type QuotaHandler struct {
	tracker *quotatracker.QuotaTrackerPlugin
}

// NewQuotaHandler creates a new quota handler instance
func NewQuotaHandler(tracker *quotatracker.QuotaTrackerPlugin) *QuotaHandler {
	return &QuotaHandler{tracker: tracker}
}

// RegisterRoutes registers all quota-related routes
func (h *QuotaHandler) RegisterRoutes(r *router.Router, middlewares ...schemas.BifrostHTTPMiddleware) {
	r.GET("/api/quotas", lib.ChainMiddlewares(h.getQuotas, middlewares...))
	r.GET("/api/quotas/{scope}/{scope_id}", lib.ChainMiddlewares(h.getQuotasByScope, middlewares...))
	r.POST("/api/quotas/meters", lib.ChainMiddlewares(h.registerMeter, middlewares...))
	r.POST("/api/quotas/alerts", lib.ChainMiddlewares(h.registerAlertConfig, middlewares...))
}

// getQuotas handles GET /api/quotas — list all quota meters with utilization
func (h *QuotaHandler) getQuotas(ctx *fasthttp.RequestCtx) {
	if h.tracker == nil {
		SendError(ctx, 503, "Quota tracker not available")
		return
	}
	scope := string(ctx.QueryArgs().Peek("scope"))
	meters := h.tracker.GetMeters(scope)
	SendJSON(ctx, map[string]interface{}{
		"meters": meters,
		"count":  len(meters),
	})
}

// getQuotasByScope handles GET /api/quotas/{scope}/{scope_id} — per-scope detail
func (h *QuotaHandler) getQuotasByScope(ctx *fasthttp.RequestCtx) {
	if h.tracker == nil {
		SendError(ctx, 503, "Quota tracker not available")
		return
	}
	scope := ctx.UserValue("scope").(string)
	scopeID := ctx.UserValue("scope_id").(string)
	meters := h.tracker.GetMeters(scope)
	// Filter to match scope_id
	filtered := make([]*quotatracker.QuotaMeter, 0)
	for _, m := range meters {
		if m.ScopeID == scopeID {
			filtered = append(filtered, m)
		}
	}
	SendJSON(ctx, map[string]interface{}{
		"scope":     scope,
		"scope_id":  scopeID,
		"meters":    filtered,
		"count":     len(filtered),
	})
}

// registerMeter handles POST /api/quotas/meters — register/update a quota meter
func (h *QuotaHandler) registerMeter(ctx *fasthttp.RequestCtx) {
	if h.tracker == nil {
		SendError(ctx, 503, "Quota tracker not available")
		return
	}
	var meter quotatracker.QuotaMeter
	if err := json.Unmarshal(ctx.PostBody(), &meter); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	h.tracker.RegisterMeter(&meter)
	SendJSON(ctx, map[string]interface{}{"message": "Meter registered", "meter": meter})
}

// registerAlertConfig handles POST /api/quotas/alerts — configure alert thresholds
func (h *QuotaHandler) registerAlertConfig(ctx *fasthttp.RequestCtx) {
	if h.tracker == nil {
		SendError(ctx, 503, "Quota tracker not available")
		return
	}
	var config quotatracker.QuotaAlertConfig
	if err := json.Unmarshal(ctx.PostBody(), &config); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	h.tracker.RegisterAlertConfig(&config)
	SendJSON(ctx, map[string]interface{}{"message": "Alert config registered"})
}
