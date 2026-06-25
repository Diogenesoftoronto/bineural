package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/transports/bifrost-http/enterprise/scim"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
	"github.com/valyala/fasthttp"
)

// SCIMHandler exposes SCIM user and group CRUD over HTTP.
type SCIMHandler struct {
	plugin *scim.SCIMPlugin
}

func NewSCIMHandler(plugin *scim.SCIMPlugin) *SCIMHandler {
	return &SCIMHandler{plugin: plugin}
}

func (h *SCIMHandler) RegisterRoutes(r *router.Router, middlewares ...schemas.BifrostHTTPMiddleware) {
	for _, prefix := range []string{"/api/enterprise/scim", "/api/scim/v2"} {
		r.GET(prefix+"/users", lib.ChainMiddlewares(h.listUsers, middlewares...))
		r.POST(prefix+"/users", lib.ChainMiddlewares(h.createUser, middlewares...))
		r.GET(prefix+"/users/{id}", lib.ChainMiddlewares(h.getUser, middlewares...))
		r.PUT(prefix+"/users/{id}", lib.ChainMiddlewares(h.updateUser, middlewares...))
		r.DELETE(prefix+"/users/{id}", lib.ChainMiddlewares(h.deleteUser, middlewares...))

		r.GET(prefix+"/Users", lib.ChainMiddlewares(h.listUsers, middlewares...))
		r.POST(prefix+"/Users", lib.ChainMiddlewares(h.createUser, middlewares...))
		r.GET(prefix+"/Users/{id}", lib.ChainMiddlewares(h.getUser, middlewares...))
		r.PUT(prefix+"/Users/{id}", lib.ChainMiddlewares(h.updateUser, middlewares...))
		r.DELETE(prefix+"/Users/{id}", lib.ChainMiddlewares(h.deleteUser, middlewares...))

		r.GET(prefix+"/groups", lib.ChainMiddlewares(h.listGroups, middlewares...))
		r.POST(prefix+"/groups", lib.ChainMiddlewares(h.createGroup, middlewares...))
		r.GET(prefix+"/groups/{id}", lib.ChainMiddlewares(h.getGroup, middlewares...))
		r.PUT(prefix+"/groups/{id}", lib.ChainMiddlewares(h.updateGroup, middlewares...))
		r.DELETE(prefix+"/groups/{id}", lib.ChainMiddlewares(h.deleteGroup, middlewares...))

		r.GET(prefix+"/Groups", lib.ChainMiddlewares(h.listGroups, middlewares...))
		r.POST(prefix+"/Groups", lib.ChainMiddlewares(h.createGroup, middlewares...))
		r.GET(prefix+"/Groups/{id}", lib.ChainMiddlewares(h.getGroup, middlewares...))
		r.PUT(prefix+"/Groups/{id}", lib.ChainMiddlewares(h.updateGroup, middlewares...))
		r.DELETE(prefix+"/Groups/{id}", lib.ChainMiddlewares(h.deleteGroup, middlewares...))
	}
}

func (h *SCIMHandler) ensurePlugin(ctx *fasthttp.RequestCtx) bool {
	if h == nil || h.plugin == nil {
		SendError(ctx, 503, "SCIM plugin not available")
		return false
	}
	return true
}

func scimPagination(ctx *fasthttp.RequestCtx) (int, int) {
	offset, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("offset")))
	limit, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))
	startIndex, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("startIndex")))
	count, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("count")))
	if startIndex > 0 {
		offset = startIndex - 1
	}
	if count > 0 {
		limit = count
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return offset, limit
}

func (h *SCIMHandler) listUsers(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	offset, limit := scimPagination(ctx)
	filter := string(ctx.QueryArgs().Peek("filter"))
	users := h.plugin.ListUsers(filter, offset, limit)
	SendJSON(ctx, map[string]any{"users": users, "Resources": users, "total_count": len(users), "totalResults": len(users), "startIndex": offset + 1, "itemsPerPage": limit})
}

func (h *SCIMHandler) createUser(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	var user scim.SCIMUser
	if err := json.Unmarshal(ctx.PostBody(), &user); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if err := h.plugin.CreateUser(&user); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create SCIM user: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]any{"message": "SCIM user created", "user": user}, 201)
}

func (h *SCIMHandler) getUser(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	user, ok := h.plugin.GetUser(ctx.UserValue("id").(string))
	if !ok {
		SendError(ctx, 404, "SCIM user not found")
		return
	}
	SendJSON(ctx, map[string]any{"user": user})
}

func (h *SCIMHandler) updateUser(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	var updates map[string]any
	if err := json.Unmarshal(ctx.PostBody(), &updates); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if rawGroups, ok := updates["groups"].([]any); ok {
		groups := make([]string, 0, len(rawGroups))
		for _, raw := range rawGroups {
			if group, ok := raw.(string); ok {
				groups = append(groups, group)
			}
		}
		updates["groups"] = groups
	}
	id := ctx.UserValue("id").(string)
	if err := h.plugin.UpdateUser(id, updates); err != nil {
		SendError(ctx, 404, err.Error())
		return
	}
	user, _ := h.plugin.GetUser(id)
	SendJSON(ctx, map[string]any{"message": "SCIM user updated", "user": user})
}

func (h *SCIMHandler) deleteUser(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	if err := h.plugin.DeleteUser(ctx.UserValue("id").(string)); err != nil {
		SendError(ctx, 404, err.Error())
		return
	}
	SendJSON(ctx, map[string]any{"message": "SCIM user deleted"})
}

func (h *SCIMHandler) listGroups(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	offset, limit := scimPagination(ctx)
	filter := string(ctx.QueryArgs().Peek("filter"))
	groups := h.plugin.ListGroups(filter, offset, limit)
	SendJSON(ctx, map[string]any{"groups": groups, "Resources": groups, "total_count": len(groups), "totalResults": len(groups), "startIndex": offset + 1, "itemsPerPage": limit})
}

func (h *SCIMHandler) createGroup(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	var group scim.SCIMGroup
	if err := json.Unmarshal(ctx.PostBody(), &group); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if err := h.plugin.CreateGroup(&group); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create SCIM group: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]any{"message": "SCIM group created", "group": group}, 201)
}

func (h *SCIMHandler) getGroup(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	group, ok := h.plugin.GetGroup(ctx.UserValue("id").(string))
	if !ok {
		SendError(ctx, 404, "SCIM group not found")
		return
	}
	SendJSON(ctx, map[string]any{"group": group})
}

func (h *SCIMHandler) updateGroup(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	var updates map[string]any
	if err := json.Unmarshal(ctx.PostBody(), &updates); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if rawMembers, ok := updates["members"].([]any); ok {
		members := make([]string, 0, len(rawMembers))
		for _, raw := range rawMembers {
			if member, ok := raw.(string); ok {
				members = append(members, member)
			}
		}
		updates["members"] = members
	}
	id := ctx.UserValue("id").(string)
	if err := h.plugin.UpdateGroup(id, updates); err != nil {
		SendError(ctx, 404, err.Error())
		return
	}
	group, _ := h.plugin.GetGroup(id)
	SendJSON(ctx, map[string]any{"message": "SCIM group updated", "group": group})
}

func (h *SCIMHandler) deleteGroup(ctx *fasthttp.RequestCtx) {
	if !h.ensurePlugin(ctx) {
		return
	}
	if err := h.plugin.DeleteGroup(ctx.UserValue("id").(string)); err != nil {
		SendError(ctx, 404, err.Error())
		return
	}
	SendJSON(ctx, map[string]any{"message": "SCIM group deleted"})
}
