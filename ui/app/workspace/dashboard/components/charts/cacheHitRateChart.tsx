import type { TokenHistogramResponse } from "@/lib/types/logs";
import { useMemo } from "react";
import { Area, AreaChart, Bar, BarChart, CartesianGrid, Line, ComposedChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { CHART_COLORS, formatFullTimestamp, formatTimestamp } from "../../utils/chartUtils";
import { ChartErrorBoundary } from "./chartErrorBoundary";
import type { ChartType } from "./chartTypeToggle";

interface CacheHitRateChartProps {
	data: TokenHistogramResponse | null;
	chartType: ChartType;
	startTime: number;
	endTime: number;
}

function CustomTooltip({ active, payload }: any) {
	if (!active || !payload || !payload.length) return null;
	const data = payload[0]?.payload;
	if (!data) return null;
	return (
		<div className="rounded-sm border border-zinc-200 bg-white px-3 py-2 shadow-lg dark:border-zinc-700 dark:bg-zinc-900">
			<div className="mb-1 text-xs text-zinc-500">{formatFullTimestamp(data.timestamp)}</div>
			<div className="space-y-1 text-sm">
				<div className="flex items-center justify-between gap-4">
					<span className="text-zinc-600 dark:text-zinc-400">Hit rate</span>
					<span className="font-medium">{data.hit_rate.toFixed(1)}%</span>
				</div>
				<div className="flex items-center justify-between gap-4">
					<span className="flex items-center gap-1.5">
						<span className="h-2 w-2 rounded-full" style={{ backgroundColor: CHART_COLORS.cachedReadTokens }} />
						<span className="text-zinc-600 dark:text-zinc-400">Cached</span>
					</span>
					<span className="font-medium">{data.cached_read_tokens.toLocaleString()}</span>
				</div>
				<div className="flex items-center justify-between gap-4">
					<span className="flex items-center gap-1.5">
						<span className="h-2 w-2 rounded-full" style={{ backgroundColor: CHART_COLORS.promptTokens }} />
						<span className="text-zinc-600 dark:text-zinc-400">Total prompt</span>
					</span>
					<span className="font-medium">{data.prompt_tokens.toLocaleString()}</span>
				</div>
			</div>
		</div>
	);
}

export function CacheHitRateChart({ data, chartType, startTime, endTime }: CacheHitRateChartProps) {
	const chartData = useMemo(() => {
		if (!data?.buckets || !data.bucket_size_seconds) return [];
		return data.buckets.map((bucket, index) => {
			const hitRate = bucket.prompt_tokens > 0 ? (bucket.cached_read_tokens / bucket.prompt_tokens) * 100 : 0;
			return {
				timestamp: bucket.timestamp,
				prompt_tokens: bucket.prompt_tokens,
				cached_read_tokens: bucket.cached_read_tokens,
				hit_rate: hitRate,
				index,
				formattedTime: formatTimestamp(bucket.timestamp, data.bucket_size_seconds),
			};
		});
	}, [data]);

	if (!data?.buckets || chartData.length === 0) {
		return <div className="text-muted-foreground flex h-full items-center justify-center text-sm">No data available</div>;
	}

	const commonProps = {
		data: chartData,
		margin: { top: 6, right: 4, left: 4, bottom: 0 },
	};

	return (
		<ChartErrorBoundary resetKey={`${startTime}-${endTime}-${chartData.length}`}>
			<ResponsiveContainer width="100%" height="100%">
				{chartType === "bar" ? (
					<BarChart {...commonProps} barCategoryGap={1}>
						<CartesianGrid strokeDasharray="3 3" vertical={false} className="stroke-zinc-200 dark:stroke-zinc-700" />
						<XAxis
							dataKey="index"
							type="number"
							domain={[-0.5, chartData.length - 0.5]}
							tick={{ fontSize: 11, className: "fill-zinc-500", dy: 5 }}
							tickLine={false}
							axisLine={false}
							tickFormatter={(idx) => chartData[Math.round(idx)]?.formattedTime || ""}
							interval="preserveStartEnd"
						/>
						<YAxis
							yAxisId="left"
							tick={{ fontSize: 11, className: "fill-zinc-500" }}
							tickLine={false}
							axisLine={false}
							width={50}
							domain={[0, (dataMax: number) => Math.max(dataMax, 1)]}
							allowDataOverflow={false}
						/>
						<YAxis
							yAxisId="right"
							orientation="right"
							tick={{ fontSize: 11, className: "fill-zinc-500" }}
							tickLine={false}
							axisLine={false}
							width={50}
							domain={[0, 100]}
							tickFormatter={(v) => `${v}%`}
						/>
						<Tooltip content={<CustomTooltip />} cursor={{ fill: "#8c8c8f", fillOpacity: 0.15 }} />
						<Bar
							yAxisId="left"
							isAnimationActive={false}
							dataKey="cached_read_tokens"
							fill={CHART_COLORS.cachedReadTokens}
							fillOpacity={0.9}
							radius={[2, 2, 0, 0]}
							barSize={30}
						/>
						<Bar
							yAxisId="left"
							isAnimationActive={false}
							dataKey="prompt_tokens"
							fill={CHART_COLORS.promptTokens}
							fillOpacity={0.5}
							radius={[2, 2, 0, 0]}
							barSize={30}
						/>
						<Line
							yAxisId="right"
							type="monotone"
							dataKey="hit_rate"
							stroke="#f59e0b"
							strokeWidth={2}
							dot={false}
							isAnimationActive={false}
						/>
					</BarChart>
				) : (
					<ComposedChart {...commonProps}>
						<CartesianGrid strokeDasharray="3 3" vertical={false} className="stroke-zinc-200 dark:stroke-zinc-700" />
						<XAxis
							dataKey="index"
							type="number"
							domain={[-0.5, chartData.length - 0.5]}
							tick={{ fontSize: 11, className: "fill-zinc-500" }}
							tickLine={false}
							axisLine={false}
							tickFormatter={(idx) => chartData[Math.round(idx)]?.formattedTime || ""}
							interval="preserveStartEnd"
						/>
						<YAxis
							yAxisId="left"
							tick={{ fontSize: 11, className: "fill-zinc-500" }}
							tickLine={false}
							axisLine={false}
							width={50}
							domain={[0, (dataMax: number) => Math.max(dataMax, 1)]}
							allowDataOverflow={false}
						/>
						<YAxis
							yAxisId="right"
							orientation="right"
							tick={{ fontSize: 11, className: "fill-zinc-500" }}
							tickLine={false}
							axisLine={false}
							width={50}
							domain={[0, 100]}
							tickFormatter={(v) => `${v}%`}
						/>
						<Tooltip content={<CustomTooltip />} />
						<Area
							yAxisId="left"
							isAnimationActive={false}
							type="monotone"
							dataKey="cached_read_tokens"
							stroke={CHART_COLORS.cachedReadTokens}
							fill={CHART_COLORS.cachedReadTokens}
							fillOpacity={0.7}
						/>
						<Line
							yAxisId="right"
							type="monotone"
							dataKey="hit_rate"
							stroke="#f59e0b"
							strokeWidth={2}
							dot={false}
							isAnimationActive={false}
						/>
					</ComposedChart>
				)}
			</ResponsiveContainer>
		</ChartErrorBoundary>
	);
}
