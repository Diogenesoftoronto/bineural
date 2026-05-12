import { baseApi } from "@/lib/store/apis/baseApi";

export interface AuditEntry {
	event_id: string;
	timestamp: string;
	user_id: string;
	user_email: string;
	action: string;
	resource: string;
	status_code: number;
	ip_address: string;
}

export interface ListAuditEntriesResponse {
	entries: AuditEntry[];
	total_count: number;
}

export interface ListAuditEntriesParams {
	event_type?: string;
	limit?: number;
	offset?: number;
}

export const auditApi = baseApi.injectEndpoints({
	endpoints: (builder) => ({
		listAuditEntries: builder.query<ListAuditEntriesResponse, ListAuditEntriesParams | void>({
			query: (params) => ({
				url: "/enterprise/audit",
				params: {
					...(params?.event_type && { event_type: params.event_type }),
					...(params?.limit && { limit: params.limit }),
					...(params?.offset !== undefined && { offset: params.offset }),
				},
			}),
			providesTags: ["AuditLogs"],
		}),
		getAuditEntriesByType: builder.query<ListAuditEntriesResponse, { event_type: string; limit?: number; offset?: number }>({
			query: (params) => ({
				url: "/enterprise/audit",
				params: {
					event_type: params.event_type,
					...(params.limit && { limit: params.limit }),
					...(params.offset !== undefined && { offset: params.offset }),
				},
			}),
			providesTags: ["AuditLogs"],
		}),
		deleteAuditEntriesBefore: builder.mutation<{ message: string }, { before: string }>({
			query: (data) => ({
				url: "/enterprise/audit",
				method: "DELETE",
				body: data,
			}),
			invalidatesTags: ["AuditLogs"],
		}),
	}),
});

export const { useListAuditEntriesQuery, useGetAuditEntriesByTypeQuery, useDeleteAuditEntriesBeforeMutation } = auditApi;
