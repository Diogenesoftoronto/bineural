import { LargePayloadConfig, DefaultLargePayloadConfig } from "@enterprise/lib/types/largePayload";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { useCallback } from "react";

export interface LargePayloadSettingsFragmentProps {
	config: LargePayloadConfig;
	onConfigChange: (config: LargePayloadConfig) => void;
	controlsDisabled: boolean;
}

function formatBytes(bytes: number): string {
	if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
	if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(0)} MB`;
	if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`;
	return `${bytes} B`;
}

function parseByteInput(value: string): number {
	const match = value.match(/^(\d+(?:\.\d+)?)\s*(gb|mb|kb|b)?$/i);
	if (!match) return 0;
	const num = parseFloat(match[1]);
	const unit = (match[2] || "b").toLowerCase();
	switch (unit) {
		case "gb": return Math.round(num * 1024 * 1024 * 1024);
		case "mb": return Math.round(num * 1024 * 1024);
		case "kb": return Math.round(num * 1024);
		default: return Math.round(num);
	}
}

export default function LargePayloadSettingsFragment({ config, onConfigChange, controlsDisabled }: LargePayloadSettingsFragmentProps) {
	const effectiveConfig = { ...DefaultLargePayloadConfig, ...config };

	const handleToggle = useCallback((checked: boolean) => {
		onConfigChange({ ...effectiveConfig, enabled: checked });
	}, [effectiveConfig, onConfigChange]);

	const handleFieldChange = useCallback((field: keyof LargePayloadConfig, value: string) => {
		const bytes = parseByteInput(value);
		if (bytes > 0) {
			onConfigChange({ ...effectiveConfig, [field]: bytes });
		}
	}, [effectiveConfig, onConfigChange]);

	const fields: { key: keyof LargePayloadConfig; label: string; description: string }[] = [
		{ key: "request_threshold_bytes", label: "Request Threshold", description: "Payloads above this size use streaming" },
		{ key: "response_threshold_bytes", label: "Response Threshold", description: "Responses above this size use streaming" },
		{ key: "prefetch_size_bytes", label: "Prefetch Size", description: "Initial bytes to buffer before streaming" },
		{ key: "max_payload_bytes", label: "Max Payload", description: "Maximum allowed payload size" },
		{ key: "truncated_log_bytes", label: "Truncated Log Size", description: "Maximum bytes logged for large payloads" },
	];

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<div className="space-y-0.5">
					<Label className="text-sm font-medium">Large Payload Handling</Label>
					<p className="text-xs text-muted-foreground">
						Configure streaming thresholds and size limits for large request/response payloads
					</p>
				</div>
				<Switch
					checked={effectiveConfig.enabled}
					onCheckedChange={handleToggle}
					disabled={controlsDisabled}
				/>
			</div>

			{effectiveConfig.enabled && (
				<div className="grid grid-cols-2 gap-4 pt-2">
					{fields.map(({ key, label, description }) => (
						<div key={key} className="space-y-1.5">
							<Label className="text-xs font-medium">{label}</Label>
							<Input
								type="text"
								value={formatBytes(effectiveConfig[key] as number)}
								onChange={(e) => handleFieldChange(key, e.target.value)}
								disabled={controlsDisabled}
								className="h-8 text-xs"
							/>
							<p className="text-[10px] text-muted-foreground">{description}</p>
						</div>
					))}
				</div>
			)}
		</div>
	);
}
