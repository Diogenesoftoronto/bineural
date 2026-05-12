import type { BaseQueryFn, FetchArgs, FetchBaseQueryError } from "@reduxjs/toolkit/query/react";
import { getAccessToken, isTokenExpired, clearOAuthStorage } from "./tokenManager";

export function createBaseQueryWithRefresh(baseQuery: BaseQueryFn): BaseQueryFn {
	return async (args: FetchArgs | string, api: any, extraOptions: any) => {
		let adjustedArgs = args;

		if (typeof args !== "string" && args.headers) {
			const token = await getAccessToken();
			if (token) {
				args.headers = {
					...args.headers,
					Authorization: `Bearer ${token}`,
				};
			}
		} else if (typeof args === "string") {
			const token = await getAccessToken();
			if (token) {
				adjustedArgs = {
					url: args,
					headers: { Authorization: `Bearer ${token}` },
				};
			}
		}

		let result = await baseQuery(adjustedArgs, api, extraOptions);

		if (result.error && (result.error as FetchBaseQueryError).status === 401) {
			if (isTokenExpired()) {
				clearOAuthStorage();
				if (typeof window !== "undefined" && !window.location.pathname.includes("/login")) {
					window.location.href = "/login";
				}
			}
		}

		return result;
	};
}
