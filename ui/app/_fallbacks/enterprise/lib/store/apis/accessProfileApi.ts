import { baseApi } from "@/lib/store/apis/baseApi";
import { GetUserAccessProfilesResponse } from "@enterprise/lib/types/accessProfile";

export const accessProfileApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		getUserAccessProfiles: builder.query<GetUserAccessProfilesResponse, string>({
			query: (userId) => `/enterprise/access-profiles?user_id=${userId}`,
			providesTags: ["AccessProfiles"],
		}),
	}),
});

export const { useGetUserAccessProfilesQuery } = accessProfileApi;
