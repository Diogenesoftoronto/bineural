// Chart utility functions for the dashboard

// Format timestamp based on bucket size
export function formatTimestamp(timestamp: string, bucketSizeSeconds: number): string {
	const date = new Date(timestamp);

	if (bucketSizeSeconds >= 86400) {
		// Daily buckets: "Jan 20"
		return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
	} else if (bucketSizeSeconds >= 3600) {
		// Hourly buckets: "10:00"
		return date.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit", hour12: false });
	} else {
		// Sub-hourly: "10:15"
		return date.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit", hour12: false });
	}
}

// Format full timestamp for tooltip
export function formatFullTimestamp(timestamp: string): string {
	const date = new Date(timestamp);
	return date.toLocaleString("en-US", {
		month: "short",
		day: "numeric",
		hour: "2-digit",
		minute: "2-digit",
		hour12: false,
	});
}

// Format cost values
export function formatCost(cost: number): string {
	if (cost < 0.01) {
		return `$${cost.toFixed(4)}`;
	}
	return `$${cost.toFixed(2)}`;
}

// Format token values
export function formatTokens(tokens: number): string {
	if (tokens >= 1000000) {
		return `${(tokens / 1000000).toFixed(1)}M`;
	}
	if (tokens >= 1000) {
		return `${(tokens / 1000).toFixed(1)}K`;
	}
	return tokens.toLocaleString();
}

// Color palette for models
export const MODEL_COLORS = [
	"#10b981", // emerald-500
	"#3b82f6", // blue-500
	"#f59e0b", // amber-500
	"#ef4444", // red-500
	"#8b5cf6", // violet-500
	"#ec4899", // pink-500
	"#06b6d4", // cyan-500
	"#84cc16", // lime-500
];

// Get color for a model by index
export function getModelColor(index: number): string {
	return MODEL_COLORS[index % MODEL_COLORS.length];
}

// Format latency values
export function formatLatency(ms: number): string {
	if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`;
	return `${ms.toFixed(0)}ms`;
}

// Latency chart color palette
export const LATENCY_COLORS = {
	avg: "#06b6d4", // cyan-500
	p90: "#3b82f6", // blue-500
	p95: "#f59e0b", // amber-500
	p99: "#ef4444", // red-500
};

// Shared CSS class constants for chart card headers
export const CHART_HEADER_ACTIONS_CLASS = "flex min-w-0 w-full flex-col-reverse gap-2";
export const CHART_HEADER_LEGEND_CLASS = "flex min-h-5 min-w-0 flex-wrap items-center gap-2 pl-2 text-xs";
export const CHART_HEADER_CONTROLS_CLASS = "flex items-center justify-end gap-2";

// Chart colors
export const CHART_COLORS = {
	success: "#10b981", // emerald-500
	error: "#ef4444", // red-500
	promptTokens: "#3b82f6", // blue-500
	completionTokens: "#10b981", // emerald-500
	totalTokens: "#8b5cf6", // violet-500
	cost: "#f59e0b", // amber-500
	cachedReadTokens: "#06b6d4", // cyan-500
	energyJoules: "#eab308", // yellow-500
	billedCostUSD: "#f97316", // orange-500
	avgPowerWatts: "#84cc16", // lime-500
	tokensPerSec: "#ec4899", // pink-500
};

const ENERGY_UNITS: Array<[number, string]> = [
	[1, "J"],
	[1_000, "kJ"],
	[1_000_000, "MJ"],
];

// Format energy in joules as a compact human-readable string.
export function formatEnergy(joules: number): string {
	const abs = Math.abs(joules);
	if (abs === 0) return "0 J";
	for (let i = ENERGY_UNITS.length - 1; i >= 0; i--) {
		const [factor, unit] = ENERGY_UNITS[i];
		if (abs >= factor) {
			const v = joules / factor;
			const fixed = v >= 100 ? 0 : v >= 10 ? 1 : 2;
			return `${v.toFixed(fixed)} ${unit}`;
		}
	}
	return `${joules.toFixed(2)} J`;
}

// Format an instantaneous power value (W).
export function formatWatts(watts: number): string {
	const abs = Math.abs(watts);
	if (abs >= 1_000_000) return `${(watts / 1_000_000).toFixed(2)} MW`;
	if (abs >= 1_000) return `${(watts / 1_000).toFixed(1)} kW`;
	if (abs >= 1) return `${watts.toFixed(1)} W`;
	if (abs >= 0.001) return `${(watts * 1_000).toFixed(0)} mW`;
	return `${watts.toFixed(3)} W`;
}

// Format tokens-per-second.
export function formatTokensPerSec(tps: number): string {
	if (tps >= 1000) return `${(tps / 1000).toFixed(1)}k tok/s`;
	if (tps >= 1) return `${tps.toFixed(1)} tok/s`;
	return `${tps.toFixed(2)} tok/s`;
}