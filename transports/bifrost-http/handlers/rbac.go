package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/configstore"
	configstoreTables "github.com/maximhq/bifrost/framework/configstore/tables"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
	"github.com/valyala/fasthttp"
)

// RBACHandler manages HTTP requests for RBAC (Role-Based Access Control) operations.
type RBACHandler struct {
	configStore configstore.ConfigStore
}

// NewRBACHandler creates a new RBAC handler instance.
func NewRBACHandler(configStore configstore.ConfigStore) *RBACHandler {
	return &RBACHandler{configStore: configStore}
}

// RegisterRoutes registers all RBAC-related routes.
func (h *RBACHandler) RegisterRoutes(r *router.Router, middlewares ...schemas.BifrostHTTPMiddleware) {
	// Roles
	r.GET("/api/rbac/roles", lib.ChainMiddlewares(h.listRoles, middlewares...))
	r.POST("/api/rbac/roles", lib.ChainMiddlewares(h.createRole, middlewares...))
	r.GET("/api/rbac/roles/{id}", lib.ChainMiddlewares(h.getRole, middlewares...))
	r.PUT("/api/rbac/roles/{id}", lib.ChainMiddlewares(h.updateRole, middlewares...))
	r.DELETE("/api/rbac/roles/{id}", lib.ChainMiddlewares(h.deleteRole, middlewares...))

	// Permissions
	r.GET("/api/rbac/permissions", lib.ChainMiddlewares(h.listPermissions, middlewares...))
	r.POST("/api/rbac/permissions", lib.ChainMiddlewares(h.createPermission, middlewares...))
	r.DELETE("/api/rbac/permissions/{id}", lib.ChainMiddlewares(h.deletePermission, middlewares...))

	// Role Assignments
	r.POST("/api/rbac/role-assignments", lib.ChainMiddlewares(h.assignRoleToUser, middlewares...))
	r.GET("/api/rbac/role-assignments", lib.ChainMiddlewares(h.getUserRoleAssignments, middlewares...))
	r.DELETE("/api/rbac/role-assignments/{id}", lib.ChainMiddlewares(h.removeRoleAssignment, middlewares...))

	// Role-Permission mapping
	r.POST("/api/rbac/roles/{id}/permissions", lib.ChainMiddlewares(h.addPermissionToRole, middlewares...))
	r.GET("/api/rbac/roles/{id}/permissions", lib.ChainMiddlewares(h.getRolePermissions, middlewares...))
	r.DELETE("/api/rbac/roles/{id}/permissions/{permission_id}", lib.ChainMiddlewares(h.removePermissionFromRole, middlewares...))
}

// ─── Roles ─────────────────────────────────────────────────────────────────────

func (h *RBACHandler) listRoles(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	roles, err := h.configStore.ListRoles(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to list roles: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"roles": roles, "count": len(roles)})
}

func (h *RBACHandler) createRole(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var role configstoreTables.TableRBACRole
	if err := json.Unmarshal(ctx.PostBody(), &role); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if err := h.configStore.CreateRole(ctx, &role); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create role: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "Role created", "role": role}, 201)
}

func (h *RBACHandler) getRole(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	role, err := h.configStore.GetRole(ctx, id)
	if err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "Role not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to get role: %v", err))
		return
	}
	SendJSON(ctx, role)
}

func (h *RBACHandler) updateRole(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	var role configstoreTables.TableRBACRole
	if err := json.Unmarshal(ctx.PostBody(), &role); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	// Parse the path param ID as uint and set it on the role
	parsedID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid role ID: %v", err))
		return
	}
	role.ID = uint(parsedID)
	if err := h.configStore.UpdateRole(ctx, &role); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to update role: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Role updated"})
}

func (h *RBACHandler) deleteRole(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeleteRole(ctx, id); err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "Role not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to delete role: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Role deleted"})
}

// ─── Permissions ───────────────────────────────────────────────────────────────

func (h *RBACHandler) listPermissions(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	permissions, err := h.configStore.ListPermissions(ctx)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to list permissions: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"permissions": permissions, "count": len(permissions)})
}

func (h *RBACHandler) createPermission(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var permission configstoreTables.TableRBACPermission
	if err := json.Unmarshal(ctx.PostBody(), &permission); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if err := h.configStore.CreatePermission(ctx, &permission); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to create permission: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "Permission created", "permission": permission}, 201)
}

func (h *RBACHandler) deletePermission(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.DeletePermission(ctx, id); err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "Permission not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to delete permission: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Permission deleted"})
}

// ─── Role Assignments ─────────────────────────────────────────────────────────

func (h *RBACHandler) assignRoleToUser(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	var assignment configstoreTables.TableRBACRoleAssignment
	if err := json.Unmarshal(ctx.PostBody(), &assignment); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if err := h.configStore.AssignRoleToUser(ctx, &assignment); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to assign role: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "Role assigned", "assignment": assignment}, 201)
}

func (h *RBACHandler) getUserRoleAssignments(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	userID := string(ctx.QueryArgs().Peek("user_id"))
	if userID == "" {
		SendError(ctx, 400, "user_id query parameter is required")
		return
	}
	assignments, err := h.configStore.GetUserRoleAssignments(ctx, userID)
	if err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to get role assignments: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"assignments": assignments, "count": len(assignments)})
}

func (h *RBACHandler) removeRoleAssignment(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	id := ctx.UserValue("id").(string)
	if err := h.configStore.RemoveRoleAssignment(ctx, id); err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "Role assignment not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to remove role assignment: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Role assignment removed"})
}

// ─── Role-Permission Mapping ──────────────────────────────────────────────────

func (h *RBACHandler) addPermissionToRole(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	roleID := ctx.UserValue("id").(string)
	var rolePermission configstoreTables.TableRBACRolePermission
	if err := json.Unmarshal(ctx.PostBody(), &rolePermission); err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	// Set the role_id from the path parameter
	parsedRoleID, err := strconv.ParseUint(roleID, 10, 64)
	if err != nil {
		SendError(ctx, 400, fmt.Sprintf("Invalid role ID: %v", err))
		return
	}
	rolePermission.RoleID = uint(parsedRoleID)
	if err := h.configStore.AddPermissionToRole(ctx, &rolePermission); err != nil {
		SendError(ctx, 500, fmt.Sprintf("Failed to add permission to role: %v", err))
		return
	}
	SendJSONWithStatus(ctx, map[string]interface{}{"message": "Permission added to role"}, 201)
}

func (h *RBACHandler) getRolePermissions(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	roleID := ctx.UserValue("id").(string)
	permissions, err := h.configStore.GetRolePermissions(ctx, roleID)
	if err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "Role not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to get role permissions: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"permissions": permissions, "count": len(permissions)})
}

func (h *RBACHandler) removePermissionFromRole(ctx *fasthttp.RequestCtx) {
	if h.configStore == nil {
		SendError(ctx, 503, "Config store not available")
		return
	}
	roleID := ctx.UserValue("id").(string)
	permissionID := ctx.UserValue("permission_id").(string)
	if err := h.configStore.RemovePermissionFromRole(ctx, roleID, permissionID); err != nil {
		if err == configstore.ErrNotFound {
			SendError(ctx, 404, "Role or permission not found")
			return
		}
		SendError(ctx, 500, fmt.Sprintf("Failed to remove permission from role: %v", err))
		return
	}
	SendJSON(ctx, map[string]interface{}{"message": "Permission removed from role"})
}
