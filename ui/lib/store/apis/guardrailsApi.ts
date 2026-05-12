import { baseApi } from "./baseApi";

export interface GuardrailRule {
	id: number;
	name: string;
	type: string;
	pattern?: string;
	patterns?: string;
	content_type?: string;
	action: string;
	message?: string;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface GuardrailProfile {
	id: number;
	name: string;
	description?: string;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateRuleRequest {
	name: string;
	type: string;
	pattern?: string;
	patterns?: string;
	content_type?: string;
	action: string;
	message?: string;
	enabled?: boolean;
}

export interface CreateProfileRequest {
	name: string;
	description?: string;
	enabled?: boolean;
}

export const guardrailsApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		// Rules
		listGuardrailRules: builder.query<GuardrailRule[], void>({
			query: () => "/guardrails/rules",
			providesTags: ["GuardrailRules"],
		}),
		createGuardrailRule: builder.mutation<GuardrailRule, CreateRuleRequest>({
			query: (body) => ({ url: "/guardrails/rules", method: "POST", body }),
			invalidatesTags: ["GuardrailRules"],
		}),
		getGuardrailRule: builder.query<GuardrailRule, string>({
			query: (id) => `/guardrails/rules/${id}`,
			providesTags: ["GuardrailRules"],
		}),
		updateGuardrailRule: builder.mutation<GuardrailRule, { id: string; body: Partial<CreateRuleRequest> }>({
			query: ({ id, body }) => ({ url: `/guardrails/rules/${id}`, method: "PUT", body }),
			invalidatesTags: ["GuardrailRules"],
		}),
		deleteGuardrailRule: builder.mutation<void, string>({
			query: (id) => ({ url: `/guardrails/rules/${id}`, method: "DELETE" }),
			invalidatesTags: ["GuardrailRules"],
		}),
		// Profiles
		listGuardrailProfiles: builder.query<GuardrailProfile[], void>({
			query: () => "/guardrails/profiles",
			providesTags: ["Guardrails"],
		}),
		createGuardrailProfile: builder.mutation<GuardrailProfile, CreateProfileRequest>({
			query: (body) => ({ url: "/guardrails/profiles", method: "POST", body }),
			invalidatesTags: ["Guardrails"],
		}),
		getGuardrailProfile: builder.query<GuardrailProfile, string>({
			query: (id) => `/guardrails/profiles/${id}`,
			providesTags: ["Guardrails"],
		}),
		updateGuardrailProfile: builder.mutation<GuardrailProfile, { id: string; body: Partial<CreateProfileRequest> }>({
			query: ({ id, body }) => ({ url: `/guardrails/profiles/${id}`, method: "PUT", body }),
			invalidatesTags: ["Guardrails"],
		}),
		deleteGuardrailProfile: builder.mutation<void, string>({
			query: (id) => ({ url: `/guardrails/profiles/${id}`, method: "DELETE" }),
			invalidatesTags: ["Guardrails"],
		}),
		// Profile-Rule mapping
		getProfileRules: builder.query<GuardrailRule[], string>({
			query: (profileId) => `/guardrails/profiles/${profileId}/rules`,
			providesTags: ["GuardrailRules"],
		}),
		addRuleToProfile: builder.mutation<void, { profileId: string; ruleId: number }>({
			query: ({ profileId, ruleId }) => ({
				url: `/guardrails/profiles/${profileId}/rules`,
				method: "POST",
				body: { rule_id: ruleId },
			}),
			invalidatesTags: ["Guardrails", "GuardrailRules"],
		}),
		removeRuleFromProfile: builder.mutation<void, { profileId: string; ruleId: string }>({
			query: ({ profileId, ruleId }) => ({
				url: `/guardrails/profiles/${profileId}/rules/${ruleId}`,
				method: "DELETE",
			}),
			invalidatesTags: ["Guardrails", "GuardrailRules"],
		}),
	}),
});

export const {
	useListGuardrailRulesQuery,
	useCreateGuardrailRuleMutation,
	useGetGuardrailRuleQuery,
	useUpdateGuardrailRuleMutation,
	useDeleteGuardrailRuleMutation,
	useListGuardrailProfilesQuery,
	useCreateGuardrailProfileMutation,
	useGetGuardrailProfileQuery,
	useUpdateGuardrailProfileMutation,
	useDeleteGuardrailProfileMutation,
	useGetProfileRulesQuery,
	useAddRuleToProfileMutation,
	useRemoveRuleFromProfileMutation,
} = guardrailsApi;
