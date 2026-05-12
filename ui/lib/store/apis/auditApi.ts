import { baseApi } from "./baseApi";

export interface AuditEntry {
	id: number;
	event_id: string;
	event_type: string;
	action: string;
	user_id: string;
	user_email: string;
	resource: string;
	resource_id: string;
	status_code: number;
	ip_address: string;
	user_agent: string;
	details: string;
	hmac_signature: string;
	timestamp: string;
}

export interface AuditListResponse {
	entries: AuditEntry[];
	total: number;
}

export interface AuditFilters {
	event_type?: string;
	user_id?: string;
	resource?: string;
	start_time?: string;
	end_time?: string;
	limit?: number;
	offset?: number;
}

export const auditApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		listAuditEntries: builder.query<AuditListResponse, AuditFilters | void>({
			query: (filters) => {
				const params: Record<string, string | number> = {};
				if (filters) {
					if (filters.event_type) params.event_type = filters.event_type;
					if (filters.user_id) params.user_id = filters.user_id;
					if (filters.resource) params.resource = filters.resource;
					if (filters.start_time) params.start_time = filters.start_time;
					if (filters.end_time) params.end_time = filters.end_time;
					if (filters.limit) params.limit = filters.limit;
					if (filters.offset) params.offset = filters.offset;
				}
				return { url: "/audit-logs", params };
			},
			providesTags: ["AuditLogs"],
		}),
		getAuditEntry: builder.query<AuditEntry, string>({
			query: (id) => `/audit-logs/${id}`,
			providesTags: ["AuditLogs"],
		}),
		getAuditEntriesByUser: builder.query<AuditEntry[], { userId: string; limit?: number; offset?: number }>({
			query: ({ userId, limit, offset }) => ({
				url: `/audit-logs/users/${userId}`,
				params: { limit: limit ?? 100, offset: offset ?? 0 },
			}),
			providesTags: ["AuditLogs"],
		}),
		getAuditEntriesByType: builder.query<AuditEntry[], { eventType: string; limit?: number; offset?: number }>({
			query: ({ eventType, limit, offset }) => ({
				url: `/audit-logs/types/${eventType}`,
				params: { limit: limit ?? 100, offset: offset ?? 0 },
			}),
			providesTags: ["AuditLogs"],
		}),
		deleteAuditEntriesBefore: builder.mutation<void, string>({
			query: (before) => ({ url: `/audit-logs/before?before=${encodeURIComponent(before)}`, method: "DELETE" }),
			invalidatesTags: ["AuditLogs"],
		}),
	}),
});

export const {
	useListAuditEntriesQuery,
	useGetAuditEntryQuery,
	useGetAuditEntriesByUserQuery,
	useGetAuditEntriesByTypeQuery,
	useDeleteAuditEntriesBeforeMutation,
} = auditApi;
