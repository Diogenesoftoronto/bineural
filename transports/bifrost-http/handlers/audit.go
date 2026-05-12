// Package handlers provides HTTP request handlers for the Bifrost HTTP transport.
// This file contains audit log handlers for querying and managing audit entries.
package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/logstore"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
	"github.com/valyala/fasthttp"
)

// AuditHandler manages HTTP requests for audit log operations.
type AuditHandler struct {
	logStore logstore.LogStore
}

// NewAuditHandler creates a new audit handler instance.
func NewAuditHandler(logStore logstore.LogStore) *AuditHandler {
	return &AuditHandler{logStore: logStore}
}

// RegisterRoutes registers all audit-related routes.
func (h *AuditHandler) RegisterRoutes(r *router.Router, middlewares ...schemas.BifrostHTTPMiddleware) {
	r.GET("/api/audit-logs", lib.ChainMiddlewares(h.getAuditEntries, middlewares...))
	r.GET("/api/audit-logs/{id}", lib.ChainMiddlewares(h.getAuditEntryByID, middlewares...))
	r.GET("/api/audit-logs/users/{user_id}", lib.ChainMiddlewares(h.getAuditEntriesByUser, middlewares...))
	r.GET("/api/audit-logs/types/{event_type}", lib.ChainMiddlewares(h.getAuditEntriesByType, middlewares...))
	r.DELETE("/api/audit-logs/before", lib.ChainMiddlewares(h.deleteAuditEntriesBefore, middlewares...))
}

// getAuditEntries handles GET /api/audit-logs — List audit entries with filtering.
func (h *AuditHandler) getAuditEntries(ctx *fasthttp.RequestCtx) {
	if h.logStore == nil {
		SendError(ctx, 503, "Log store not available")
		return
	}

	filter := logstore.AuditFilter{
		EventType: string(ctx.QueryArgs().Peek("event_type")),
		UserID:    string(ctx.QueryArgs().Peek("user_id")),
		Resource:  string(ctx.QueryArgs().Peek("resource")),
	}

	if startTimeStr := string(ctx.QueryArgs().Peek("start_time")); startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			SendError(ctx, 400, fmt.Sprintf("Invalid start_time: %v", err))
			return
		}
		filter.StartTime = &t
	}

	if endTimeStr := string(ctx.QueryArgs().Peek("end_time")); endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			SendError(ctx, 400, fmt.Sprintf("Invalid end_time format: %v", err))
			return
		}
		filter.EndTime = &t
	}

	limit := 100
	if limitStr := string(ctx.QueryArgs().Peek("limit")); limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v < 0 {
			SendError(ctx, 400, "Invalid limit")
			return
		}
		limit = v
	}

	offset := 0
	if offsetStr := string(ctx.QueryArgs().Peek("offset")); offsetStr != "" {
		v, err := strconv.Atoi(offsetStr)
		if err != nil || v < 0 {
			SendError(ctx, 400, "Invalid offset")
			return
		}
		offset = v
	}

	filter.Limit = limit
	filter.Offset = offset

	entries, total, err := h.logStore.GetAuditEntries(ctx, filter)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get audit entries: %v", err))
		return
	}

	SendJSON(ctx, map[string]interface{}{
		"entries": entries,
		"total":   total,
	})
}

// getAuditEntryByID handles GET /api/audit-logs/{id} — Get audit entry by ID.
func (h *AuditHandler) getAuditEntryByID(ctx *fasthttp.RequestCtx) {
	if h.logStore == nil {
		SendError(ctx, 503, "Log store not available")
		return
	}

	id, ok := ctx.UserValue("id").(string)
	if !ok || id == "" {
		SendError(ctx, 400, "id is required")
		return
	}

	entry, err := h.logStore.GetAuditEntryByID(ctx, id)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get audit entry: %v", err))
		return
	}
	if entry == nil {
		SendError(ctx, 404, "Audit entry not found")
		return
	}

	SendJSON(ctx, entry)
}

// getAuditEntriesByUser handles GET /api/audit-logs/users/{user_id} — Get audit entries for a user.
func (h *AuditHandler) getAuditEntriesByUser(ctx *fasthttp.RequestCtx) {
	if h.logStore == nil {
		SendError(ctx, 503, "Log store not available")
		return
	}

	userID, ok := ctx.UserValue("user_id").(string)
	if !ok || userID == "" {
		SendError(ctx, 400, "user_id is required")
		return
	}

	limit := 100
	if limitStr := string(ctx.QueryArgs().Peek("limit")); limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v < 0 {
			SendError(ctx, 400, "Invalid limit")
			return
		}
		limit = v
	}

	offset := 0
	if offsetStr := string(ctx.QueryArgs().Peek("offset")); offsetStr != "" {
		v, err := strconv.Atoi(offsetStr)
		if err != nil || v < 0 {
			SendError(ctx, 400, "Invalid offset")
			return
		}
		offset = v
	}

	entries, err := h.logStore.GetAuditEntriesByUser(ctx, userID, limit, offset)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get audit entries by user: %v", err))
		return
	}

	SendJSON(ctx, map[string]interface{}{
		"entries": entries,
	})
}

// getAuditEntriesByType handles GET /api/audit-logs/types/{event_type} — Get audit entries by type.
func (h *AuditHandler) getAuditEntriesByType(ctx *fasthttp.RequestCtx) {
	if h.logStore == nil {
		SendError(ctx, 503, "Log store not available")
		return
	}

	eventType, ok := ctx.UserValue("event_type").(string)
	if !ok || eventType == "" {
		SendError(ctx, 400, "event_type is required")
		return
	}

	limit := 100
	if limitStr := string(ctx.QueryArgs().Peek("limit")); limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v < 0 {
			SendError(ctx, 400, "Invalid limit")
			return
		}
		limit = v
	}

	offset := 0
	if offsetStr := string(ctx.QueryArgs().Peek("offset")); offsetStr != "" {
		v, err := strconv.Atoi(offsetStr)
		if err != nil || v < 0 {
			SendError(ctx, 400, "Invalid offset")
			return
		}
		offset = v
	}

	entries, err := h.logStore.GetAuditEntriesByType(ctx, eventType, limit, offset)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get audit entries by type: %v", err))
		return
	}

	SendJSON(ctx, map[string]interface{}{
		"entries": entries,
	})
}

// deleteAuditEntriesBefore handles DELETE /api/audit-logs/before — Delete entries before a timestamp.
func (h *AuditHandler) deleteAuditEntriesBefore(ctx *fasthttp.RequestCtx) {
	if h.logStore == nil {
		SendError(ctx, 503, "Log store not available")
		return
	}

	beforeStr := string(ctx.QueryArgs().Peek("before"))
	if beforeStr == "" {
		SendError(ctx, 400, "before query parameter is required")
		return
	}

	before, err := time.Parse(time.RFC3339, beforeStr)
	if err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid before timestamp: %v", err))
		return
	}

	if err := h.logStore.DeleteAuditEntriesBefore(ctx, before); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to delete audit entries: %v", err))
		return
	}

	SendJSON(ctx, map[string]interface{}{
		"message": "Audit entries deleted",
	})
}
