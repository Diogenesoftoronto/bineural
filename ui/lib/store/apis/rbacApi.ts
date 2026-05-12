import { baseApi } from "./baseApi";

export interface RBACRole {
	id: number;
	name: string;
	description: string;
	is_system: boolean;
	created_at: string;
	updated_at: string;
}

export interface RBACPermission {
	id: number;
	resource: string;
	action: string;
	description: string;
	created_at: string;
	updated_at: string;
}

export interface RBACRoleAssignment {
	id: number;
	user_id: string;
	role_id: number;
	team_id?: string;
	customer_id?: string;
	created_at: string;
	updated_at: string;
}

export interface CreateRoleRequest {
	name: string;
	description?: string;
	is_system?: boolean;
}

export interface CreatePermissionRequest {
	resource: string;
	action: string;
	description?: string;
}

export interface AssignRoleRequest {
	user_id: string;
	role_id: number;
	team_id?: string;
	customer_id?: string;
}

export const rbacApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		// Roles
		listRoles: builder.query<RBACRole[], void>({
			query: () => "/rbac/roles",
			providesTags: ["Roles"],
		}),
		createRole: builder.mutation<RBACRole, CreateRoleRequest>({
			query: (body) => ({ url: "/rbac/roles", method: "POST", body }),
			invalidatesTags: ["Roles"],
		}),
		getRole: builder.query<RBACRole, string>({
			query: (id) => `/rbac/roles/${id}`,
			providesTags: ["Roles"],
		}),
		updateRole: builder.mutation<RBACRole, { id: string; body: Partial<CreateRoleRequest> }>({
			query: ({ id, body }) => ({ url: `/rbac/roles/${id}`, method: "PUT", body }),
			invalidatesTags: ["Roles"],
		}),
		deleteRole: builder.mutation<void, string>({
			query: (id) => ({ url: `/rbac/roles/${id}`, method: "DELETE" }),
			invalidatesTags: ["Roles"],
		}),
		// Permissions
		listPermissions: builder.query<RBACPermission[], void>({
			query: () => "/rbac/permissions",
			providesTags: ["Permissions"],
		}),
		createPermission: builder.mutation<RBACPermission, CreatePermissionRequest>({
			query: (body) => ({ url: "/rbac/permissions", method: "POST", body }),
			invalidatesTags: ["Permissions"],
		}),
		deletePermission: builder.mutation<void, string>({
			query: (id) => ({ url: `/rbac/permissions/${id}`, method: "DELETE" }),
			invalidatesTags: ["Permissions"],
		}),
		// Role assignments
		assignRole: builder.mutation<RBACRoleAssignment, AssignRoleRequest>({
			query: (body) => ({ url: "/rbac/role-assignments", method: "POST", body }),
			invalidatesTags: ["Roles"],
		}),
		getUserRoleAssignments: builder.query<RBACRoleAssignment[], string>({
			query: (userId) => `/rbac/role-assignments?user_id=${userId}`,
			providesTags: ["Roles"],
		}),
		removeRoleAssignment: builder.mutation<void, string>({
			query: (id) => ({ url: `/rbac/role-assignments/${id}`, method: "DELETE" }),
			invalidatesTags: ["Roles"],
		}),
		// Role-permission mapping
		getRolePermissions: builder.query<RBACPermission[], string>({
			query: (roleId) => `/rbac/roles/${roleId}/permissions`,
			providesTags: ["Permissions"],
		}),
		addPermissionToRole: builder.mutation<void, { roleId: string; permissionId: number }>({
			query: ({ roleId, permissionId }) => ({
				url: `/rbac/roles/${roleId}/permissions`,
				method: "POST",
				body: { permission_id: permissionId },
			}),
			invalidatesTags: ["Roles", "Permissions"],
		}),
		removePermissionFromRole: builder.mutation<void, { roleId: string; permissionId: string }>({
			query: ({ roleId, permissionId }) => ({
				url: `/rbac/roles/${roleId}/permissions/${permissionId}`,
				method: "DELETE",
			}),
			invalidatesTags: ["Roles", "Permissions"],
		}),
	}),
});

export const {
	useListRolesQuery,
	useCreateRoleMutation,
	useGetRoleQuery,
	useUpdateRoleMutation,
	useDeleteRoleMutation,
	useListPermissionsQuery,
	useCreatePermissionMutation,
	useDeletePermissionMutation,
	useAssignRoleMutation,
	useGetUserRoleAssignmentsQuery,
	useRemoveRoleAssignmentMutation,
	useGetRolePermissionsQuery,
	useAddPermissionToRoleMutation,
	useRemovePermissionFromRoleMutation,
} = rbacApi;
