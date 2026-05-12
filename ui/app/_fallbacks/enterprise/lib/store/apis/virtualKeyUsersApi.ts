import { baseApi } from "@/lib/store/apis/baseApi";
import { User } from "@enterprise/lib/types/user";

export interface GetVirtualKeyUsersResponse {
	users: User[];
}

export const virtualKeyUsersApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		getVirtualKeyUsers: builder.query<GetVirtualKeyUsersResponse, string>({
			query: (vkId) => `/enterprise/virtual-keys/${vkId}/users`,
			providesTags: (_result, _error, vkId) => [{ type: "VirtualKeyUsers", id: vkId }],
		}),
	}),
});

export const { useGetVirtualKeyUsersQuery } = virtualKeyUsersApi;
