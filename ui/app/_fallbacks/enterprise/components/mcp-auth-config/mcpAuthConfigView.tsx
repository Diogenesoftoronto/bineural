import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useState } from "react";
import { Plus, Settings } from "lucide-react";

export default function mcpAuthConfigView() {
	const [showCreate, setShowCreate] = useState(false);

	const mockItems = [
		{ id: "1", name: "config:oauth-github", status: "active", updated: "2026-04-30" },
		{ id: "2", name: "config:api-key-openai", status: "active", updated: "2026-04-29" },
	];

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">MCP Auth Configuration</h2>
					<p className="text-sm text-muted-foreground">Configure authentication for MCP server connections</p>
				</div>
				<Dialog open={showCreate} onOpenChange={setShowCreate}>
					<DialogTrigger asChild>
						<Button size="sm"><Plus className="mr-1 h-3 w-3" /> Add</Button>
					</DialogTrigger>
					<DialogContent>
						<DialogHeader><DialogTitle>Create MCP Auth Configuratio</DialogTitle></DialogHeader>
						<div className="space-y-3 pt-2">
							<div><Label className="text-xs">Name</Label><Input className="h-8 text-xs mt-1" placeholder="Enter name" /></div>
							<Button size="sm" className="w-full" onClick={() => setShowCreate(false)}>Create</Button>
						</div>
					</DialogContent>
				</Dialog>
			</div>

			<Card>
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead className="text-xs">Name</TableHead>
							<TableHead className="text-xs">Status</TableHead>
							<TableHead className="text-xs">Last Updated</TableHead>
							<TableHead className="text-xs text-right">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{mockItems.map((item) => (
							<TableRow key={item.id}>
								<TableCell className="text-xs font-medium font-mono">{item.name}</TableCell>
								<TableCell className="text-xs"><Badge variant="default" className="text-[10px]">{item.status}</Badge></TableCell>
								<TableCell className="text-xs text-muted-foreground">{item.updated}</TableCell>
								<TableCell className="text-xs text-right"><Button variant="ghost" size="sm" className="h-6 text-xs">Edit</Button></TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
			</Card>
		</div>
	);
}
