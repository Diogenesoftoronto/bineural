import { baseApi } from "@/lib/store/apis/baseApi";
import { LargePayloadConfig } from "@enterprise/lib/types/largePayload";

export const largePayloadApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		getLargePayloadConfig: builder.query<LargePayloadConfig, void>({
			query: () => "/enterprise/large-payload/config",
			providesTags: ["LargePayloadConfig"],
		}),
		updateLargePayloadConfig: builder.mutation<void, LargePayloadConfig>({
			query: (config) => ({
				url: "/enterprise/large-payload/config",
				method: "PUT",
				body: config,
			}),
			invalidatesTags: ["LargePayloadConfig"],
		}),
	}),
});

export const { useGetLargePayloadConfigQuery, useUpdateLargePayloadConfigMutation } = largePayloadApi;
