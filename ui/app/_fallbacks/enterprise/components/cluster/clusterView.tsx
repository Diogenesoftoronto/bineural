import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { useState } from "react";
import { Network, RefreshCw } from "lucide-react";

export default function ClusterView() {
	const mockNodes = [
		{ id: "node-1", address: "10.0.1.10:8080", status: "healthy", role: "leader", uptime: "72h 15m", connections: 342, cpu: 45, memory: 62 },
		{ id: "node-2", address: "10.0.1.11:8080", status: "healthy", role: "follower", uptime: "72h 14m", connections: 289, cpu: 38, memory: 55 },
		{ id: "node-3", address: "10.0.1.12:8080", status: "healthy", role: "follower", uptime: "48h 30m", connections: 198, cpu: 22, memory: 41 },
	];

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">Cluster Status</h2>
					<p className="text-sm text-muted-foreground">P2P clustering with gossip-based state synchronization</p>
				</div>
				<Button variant="outline" size="sm"><RefreshCw className="mr-1 h-3 w-3" /> Refresh</Button>
			</div>

			<div className="grid gap-4 md:grid-cols-3">
				<Card><CardContent className="pt-4"><div className="text-xs text-muted-foreground">Active Nodes</div><div className="text-2xl font-bold">{mockNodes.length}</div></CardContent></Card>
				<Card><CardContent className="pt-4"><div className="text-xs text-muted-foreground">Cluster Health</div><div className="text-2xl font-bold text-green-600">Healthy</div></CardContent></Card>
				<Card><CardContent className="pt-4"><div className="text-xs text-muted-foreground">Total Connections</div><div className="text-2xl font-bold">{mockNodes.reduce((s, n) => s + n.connections, 0)}</div></CardContent></Card>
			</div>

			<div className="grid gap-4 md:grid-cols-3">
				{mockNodes.map((n) => (
					<Card key={n.id}>
						<CardHeader className="pb-2">
							<CardTitle className="text-sm flex items-center justify-between">
								<span className="flex items-center gap-1.5"><Network className="h-3.5 w-3.5" />{n.id}</span>
								<Badge variant={n.status === "healthy" ? "default" : "destructive"} className="text-[10px]">{n.status}</Badge>
							</CardTitle>
						</CardHeader>
						<CardContent className="space-y-3">
							<div className="text-xs text-muted-foreground">{n.address}</div>
							<div className="flex items-center justify-between text-xs"><span>Role</span><Badge variant="outline" className="text-[10px]">{n.role}</Badge></div>
							<div className="flex items-center justify-between text-xs"><span>Uptime</span><span>{n.uptime}</span></div>
							<div className="space-y-1">
								<div className="flex items-center justify-between text-xs"><span>CPU</span><span>{n.cpu}%</span></div>
								<Progress value={n.cpu} className="h-1.5" />
							</div>
							<div className="space-y-1">
								<div className="flex items-center justify-between text-xs"><span>Memory</span><span>{n.memory}%</span></div>
								<Progress value={n.memory} className="h-1.5" />
							</div>
							<div className="flex items-center justify-between text-xs"><span>Connections</span><span>{n.connections}</span></div>
						</CardContent>
					</Card>
				))}
			</div>
		</div>
	);
}
