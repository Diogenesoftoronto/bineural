import { baseApi } from "@/lib/store/apis/baseApi";

export interface GuardrailRule {
	id: string | number;
	name: string;
	type: string;
	pattern: string;
	action: string;
	enabled: boolean;
	updated_at?: string;
}

export interface ListGuardrailRulesResponse {
	rules: GuardrailRule[];
	total_count: number;
}

export interface CreateGuardrailRuleRequest {
	name: string;
	type: string;
	pattern: string;
	action: string;
	enabled: boolean;
}

export interface GuardrailProfile {
	id: string | number;
	name: string;
	rules: string[];
	updated_at?: string;
}

export interface ListGuardrailProfilesResponse {
	profiles: GuardrailProfile[];
	total_count: number;
}

export interface CreateGuardrailProfileRequest {
	name: string;
	rules: string[];
}

export const guardrailsApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		listGuardrailRules: builder.query<ListGuardrailRulesResponse, void>({
			query: () => "/enterprise/guardrails/rules",
			providesTags: ["GuardrailRules"],
		}),
		createGuardrailRule: builder.mutation<{ message: string; rule: GuardrailRule }, CreateGuardrailRuleRequest>({
			query: (data) => ({
				url: "/enterprise/guardrails/rules",
				method: "POST",
				body: data,
			}),
			invalidatesTags: ["GuardrailRules"],
		}),
		deleteGuardrailRule: builder.mutation<{ message: string }, string>({
			query: (ruleId) => ({
				url: `/enterprise/guardrails/rules/${ruleId}`,
				method: "DELETE",
			}),
			invalidatesTags: ["GuardrailRules"],
		}),
		listGuardrailProfiles: builder.query<ListGuardrailProfilesResponse, void>({
			query: () => "/enterprise/guardrails/profiles",
			providesTags: ["Guardrails"],
		}),
		createGuardrailProfile: builder.mutation<{ message: string; profile: GuardrailProfile }, CreateGuardrailProfileRequest>({
			query: (data) => ({
				url: "/enterprise/guardrails/profiles",
				method: "POST",
				body: data,
			}),
			invalidatesTags: ["Guardrails"],
		}),
		deleteGuardrailProfile: builder.mutation<{ message: string }, string>({
			query: (profileId) => ({
				url: `/enterprise/guardrails/profiles/${profileId}`,
				method: "DELETE",
			}),
			invalidatesTags: ["Guardrails"],
		}),
	}),
});

export const {
	useListGuardrailRulesQuery,
	useCreateGuardrailRuleMutation,
	useDeleteGuardrailRuleMutation,
	useListGuardrailProfilesQuery,
	useCreateGuardrailProfileMutation,
	useDeleteGuardrailProfileMutation,
} = guardrailsApi;
