import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { useListGuardrailRulesQuery, useCreateGuardrailRuleMutation, useDeleteGuardrailRuleMutation } from "@enterprise/lib/store/apis/guardrailsApi";

export default function guardrailsConfigurationView() {
	const [showCreate, setShowCreate] = useState(false);
	const { data: rules, isLoading } = useListGuardrailRulesQuery();
	const [createRule] = useCreateGuardrailRuleMutation();
	const [deleteRule] = useDeleteGuardrailRuleMutation();
	const [newRuleName, setNewRuleName] = useState("");
	const [newRuleType, setNewRuleType] = useState("regex");
	const [newRuleAction, setNewRuleAction] = useState("block");
	const [newRulePattern, setNewRulePattern] = useState("");

	const handleCreateRule = async () => {
		if (!newRuleName.trim() || !newRulePattern.trim()) return;
		await createRule({ name: newRuleName, type: newRuleType, pattern: newRulePattern, action: newRuleAction, enabled: true });
		setNewRuleName("");
		setNewRuleType("regex");
		setNewRuleAction("block");
		setNewRulePattern("");
		setShowCreate(false);
	};

	const handleDeleteRule = async (ruleId: string) => {
		await deleteRule(ruleId);
	};

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">Guardrails Configuration</h2>
					<p className="text-sm text-muted-foreground">Define input/output guardrail rules for content safety</p>
				</div>
				<Dialog open={showCreate} onOpenChange={setShowCreate}>
					<DialogTrigger asChild>
						<Button size="sm"><Plus className="mr-1 h-3 w-3" /> Add</Button>
					</DialogTrigger>
					<DialogContent>
						<DialogHeader><DialogTitle>Create Guardrails Rule</DialogTitle></DialogHeader>
						<div className="space-y-3 pt-2">
							<div><Label className="text-xs">Name</Label><Input className="h-8 text-xs mt-1" placeholder="Enter name" value={newRuleName} onChange={(e) => setNewRuleName(e.target.value)} /></div>
							<div><Label className="text-xs">Type</Label>
								<Select value={newRuleType} onValueChange={setNewRuleType}>
									<SelectTrigger className="h-8 text-xs mt-1"><SelectValue /></SelectTrigger>
									<SelectContent>
										<SelectItem value="regex">Regex</SelectItem>
										<SelectItem value="keyword">Keyword</SelectItem>
										<SelectItem value="ml">ML-based</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div><Label className="text-xs">Pattern</Label><Input className="h-8 text-xs mt-1" placeholder="Enter pattern or expression" value={newRulePattern} onChange={(e) => setNewRulePattern(e.target.value)} /></div>
							<div><Label className="text-xs">Action</Label>
								<Select value={newRuleAction} onValueChange={setNewRuleAction}>
									<SelectTrigger className="h-8 text-xs mt-1"><SelectValue /></SelectTrigger>
									<SelectContent>
										<SelectItem value="block">Block</SelectItem>
										<SelectItem value="flag">Flag</SelectItem>
										<SelectItem value="redact">Redact</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<Button size="sm" className="w-full" onClick={handleCreateRule}>Create</Button>
						</div>
					</DialogContent>
				</Dialog>
			</div>

			<Card>
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead className="text-xs">Name</TableHead>
							<TableHead className="text-xs">Type</TableHead>
							<TableHead className="text-xs">Action</TableHead>
							<TableHead className="text-xs">Status</TableHead>
							<TableHead className="text-xs">Last Updated</TableHead>
							<TableHead className="text-xs text-right">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading && <TableRow><TableCell colSpan={6} className="text-center text-xs text-muted-foreground py-8">Loading...</TableCell></TableRow>}
						{!isLoading && rules?.rules?.map((r) => (
							<TableRow key={String(r.id)}>
								<TableCell className="text-xs font-medium font-mono">{r.name}</TableCell>
								<TableCell className="text-xs"><Badge variant="outline" className="text-[10px]">{r.type}</Badge></TableCell>
								<TableCell className="text-xs"><Badge variant={r.action === "block" ? "destructive" : "default"} className="text-[10px]">{r.action}</Badge></TableCell>
								<TableCell className="text-xs"><Badge variant={r.enabled ? "default" : "secondary"} className="text-[10px]">{r.enabled ? "active" : "disabled"}</Badge></TableCell>
								<TableCell className="text-xs text-muted-foreground">{r.updated_at ? new Date(r.updated_at).toLocaleDateString() : "—"}</TableCell>
								<TableCell className="text-xs text-right">
									<Button variant="ghost" size="sm" className="h-6 text-xs text-destructive hover:text-destructive" onClick={() => handleDeleteRule(String(r.id))}>
										<Trash2 className="h-3 w-3" />
									</Button>
								</TableCell>
							</TableRow>
						))}
						{!isLoading && (!rules?.rules || rules.rules.length === 0) && <TableRow><TableCell colSpan={6} className="text-center text-xs text-muted-foreground py-8">No guardrail rules found</TableCell></TableRow>}
					</TableBody>
				</Table>
			</Card>
		</div>
	);
}
