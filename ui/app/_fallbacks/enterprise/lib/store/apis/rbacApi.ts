import { baseApi } from "@/lib/store/apis/baseApi";

export interface Role {
	id: string | number;
	name: string;
	description: string;
	is_system: boolean;
	user_count: number;
	permissions_count: number;
}

export interface ListRolesResponse {
	roles: Role[];
	total_count: number;
}

export interface CreateRoleRequest {
	name: string;
	description: string;
}

export interface Permission {
	id: string;
	name: string;
	resource: string;
	operation: string;
}

export interface ListPermissionsResponse {
	permissions: Permission[];
	total_count: number;
}

export interface GetRolePermissionsResponse {
	permissions: Permission[];
}

export const rbacApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		listRoles: builder.query<ListRolesResponse, void>({
			query: () => "/enterprise/rbac/roles",
			providesTags: ["Roles"],
		}),
		createRole: builder.mutation<{ message: string; role: Role }, CreateRoleRequest>({
			query: (data) => ({
				url: "/enterprise/rbac/roles",
				method: "POST",
				body: data,
			}),
			invalidatesTags: ["Roles"],
		}),
		deleteRole: builder.mutation<{ message: string }, string>({
			query: (roleId) => ({
				url: `/enterprise/rbac/roles/${roleId}`,
				method: "DELETE",
			}),
			invalidatesTags: ["Roles"],
		}),
		listPermissions: builder.query<ListPermissionsResponse, void>({
			query: () => "/enterprise/rbac/permissions",
			providesTags: ["Permissions"],
		}),
		getRolePermissions: builder.query<GetRolePermissionsResponse, string>({
			query: (roleId) => `/enterprise/rbac/roles/${roleId}/permissions`,
			providesTags: (result, error, roleId) => [{ type: "Permissions", id: roleId }],
		}),
	}),
});

export const { useListRolesQuery, useCreateRoleMutation, useDeleteRoleMutation, useListPermissionsQuery, useGetRolePermissionsQuery } = rbacApi;
