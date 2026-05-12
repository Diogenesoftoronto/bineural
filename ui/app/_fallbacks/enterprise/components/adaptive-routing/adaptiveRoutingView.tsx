import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { useCallback, useState } from "react";
import { Brain } from "lucide-react";

export default function AdaptiveRoutingView() {
	const [enabled, setEnabled] = useState(false);
	const [strategy, setStrategy] = useState("latency");

	const mockMetrics = [
		{ provider: "openai", latencyP50: 320, latencyP99: 1200, successRate: 99.8, activeKeys: 4, totalKeys: 5 },
		{ provider: "anthropic", latencyP50: 280, latencyP99: 900, successRate: 99.5, activeKeys: 3, totalKeys: 3 },
		{ provider: "google", latencyP50: 450, latencyP99: 2100, successRate: 98.2, activeKeys: 2, totalKeys: 3 },
	];

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold flex items-center gap-2"><Brain className="h-5 w-5" />Adaptive Load Balancing</h2>
					<p className="text-sm text-muted-foreground">Automatic, performance-based routing powered by real-time metrics</p>
				</div>
				<div className="flex items-center gap-3">
					<Label className="text-sm">Enabled</Label>
					<Switch checked={enabled} onCheckedChange={setEnabled} />
				</div>
			</div>

			{enabled && (
				<>
					<Card>
						<CardHeader><CardTitle className="text-sm">Routing Strategy</CardTitle></CardHeader>
						<CardContent className="space-y-4">
							<div className="grid grid-cols-2 gap-4">
								<div>
									<Label className="text-xs">Strategy</Label>
									<Select value={strategy} onValueChange={setStrategy}>
										<SelectTrigger className="h-8 text-xs mt-1"><SelectValue /></SelectTrigger>
										<SelectContent>
											<SelectItem value="latency">Lowest Latency</SelectItem>
											<SelectItem value="cost">Lowest Cost</SelectItem>
											<SelectItem value="availability">Highest Availability</SelectItem>
											<SelectItem value="balanced">Balanced</SelectItem>
										</SelectContent>
									</Select>
								</div>
								<div>
									<Label className="text-xs">Health Check Interval (ms)</Label>
									<Input className="h-8 text-xs mt-1" type="number" defaultValue="5000" />
								</div>
								<div>
									<Label className="text-xs">Cooldown Period (ms)</Label>
									<Input className="h-8 text-xs mt-1" type="number" defaultValue="30000" />
								</div>
								<div>
									<Label className="text-xs">Max Error Rate Before Failover (%)</Label>
									<Input className="h-8 text-xs mt-1" type="number" defaultValue="5" />
								</div>
							</div>
						</CardContent>
					</Card>

					<Card>
						<CardHeader><CardTitle className="text-sm">Provider Health Metrics</CardTitle></CardHeader>
						<CardContent>
							<div className="grid gap-3">
								{mockMetrics.map((m) => (
									<div key={m.provider} className="flex items-center justify-between rounded-lg border p-3">
										<div className="flex items-center gap-3">
											<Badge variant="outline" className="font-mono">{m.provider}</Badge>
											<span className="text-xs text-muted-foreground">Keys: {m.activeKeys}/{m.totalKeys}</span>
										</div>
										<div className="flex items-center gap-6 text-xs">
											<div><span className="text-muted-foreground">P50: </span><span className="font-medium">{m.latencyP50}ms</span></div>
											<div><span className="text-muted-foreground">P99: </span><span className="font-medium">{m.latencyP99}ms</span></div>
											<div><span className="text-muted-foreground">Success: </span><span className="font-medium">{m.successRate}%</span></div>
										</div>
									</div>
								))}
							</div>
						</CardContent>
					</Card>
				</>
			)}
		</div>
	);
}
