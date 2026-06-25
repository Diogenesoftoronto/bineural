import { baseApi } from "@/lib/store/apis/baseApi";

export interface AuthTypeResponse {
	type: string;
	provider?: string;
}

export interface SCIMUser {
	id: string;
	external_id?: string;
	user_name: string;
	display_name?: string;
	email?: string;
	active: boolean;
	groups?: string[];
	created_at?: string;
	updated_at?: string;
}

export interface SCIMGroup {
	id: string;
	display_name: string;
	members?: string[];
}

export interface SCIMUsersResponse {
	users: SCIMUser[];
	total_count: number;
}

export interface SCIMGroupsResponse {
	groups: SCIMGroup[];
	total_count: number;
}

export type SCIMUserRequest = Omit<SCIMUser, "id" | "created_at" | "updated_at"> & { id?: string };
export type SCIMGroupRequest = Omit<SCIMGroup, "id"> & { id?: string };

export const scimApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		getAuthType: builder.query<AuthTypeResponse, void>({
			query: () => "/enterprise/sso/auth-type",
			providesTags: ["AuthType"],
		}),
		listSCIMUsers: builder.query<SCIMUsersResponse, void>({
			query: () => "/enterprise/scim/users",
			providesTags: ["SCIMUsers"],
		}),
		createSCIMUser: builder.mutation<{ message: string; user: SCIMUser }, SCIMUserRequest>({
			query: (body) => ({ url: "/enterprise/scim/users", method: "POST", body }),
			invalidatesTags: ["SCIMUsers", "SCIMGroups"],
		}),
		updateSCIMUser: builder.mutation<{ message: string; user: SCIMUser }, { id: string; body: Partial<SCIMUserRequest> }>({
			query: ({ id, body }) => ({ url: `/enterprise/scim/users/${id}`, method: "PUT", body }),
			invalidatesTags: ["SCIMUsers", "SCIMGroups"],
		}),
		deleteSCIMUser: builder.mutation<{ message: string }, string>({
			query: (id) => ({ url: `/enterprise/scim/users/${id}`, method: "DELETE" }),
			invalidatesTags: ["SCIMUsers", "SCIMGroups"],
		}),
		listSCIMGroups: builder.query<SCIMGroupsResponse, void>({
			query: () => "/enterprise/scim/groups",
			providesTags: ["SCIMGroups"],
		}),
		createSCIMGroup: builder.mutation<{ message: string; group: SCIMGroup }, SCIMGroupRequest>({
			query: (body) => ({ url: "/enterprise/scim/groups", method: "POST", body }),
			invalidatesTags: ["SCIMGroups"],
		}),
		updateSCIMGroup: builder.mutation<{ message: string; group: SCIMGroup }, { id: string; body: Partial<SCIMGroupRequest> }>({
			query: ({ id, body }) => ({ url: `/enterprise/scim/groups/${id}`, method: "PUT", body }),
			invalidatesTags: ["SCIMGroups", "SCIMUsers"],
		}),
		deleteSCIMGroup: builder.mutation<{ message: string }, string>({
			query: (id) => ({ url: `/enterprise/scim/groups/${id}`, method: "DELETE" }),
			invalidatesTags: ["SCIMGroups", "SCIMUsers"],
		}),
	}),
});

export const {
	useGetAuthTypeQuery,
	useListSCIMUsersQuery,
	useCreateSCIMUserMutation,
	useUpdateSCIMUserMutation,
	useDeleteSCIMUserMutation,
	useListSCIMGroupsQuery,
	useCreateSCIMGroupMutation,
	useUpdateSCIMGroupMutation,
	useDeleteSCIMGroupMutation,
} = scimApi;