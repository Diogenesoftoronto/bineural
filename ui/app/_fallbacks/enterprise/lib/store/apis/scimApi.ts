import { baseApi } from "@/lib/store/apis/baseApi";

export interface AuthTypeResponse {
	type: string;
	provider?: string;
}

export const scimApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		getAuthType: builder.query<AuthTypeResponse, void>({
			query: () => "/enterprise/sso/auth-type",
			providesTags: ["AuthType"],
		}),
	}),
});

export const { useGetAuthTypeQuery } = scimApi;
