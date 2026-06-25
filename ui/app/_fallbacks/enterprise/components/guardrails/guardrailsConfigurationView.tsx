import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import {
	GuardrailRule,
	useListGuardrailRulesQuery,
	useCreateGuardrailRuleMutation,
	useUpdateGuardrailRuleMutation,
	useDeleteGuardrailRuleMutation,
} from "@enterprise/lib/store/apis/guardrailsApi";

export default function guardrailsConfigurationView() {
	const [showCreate, setShowCreate] = useState(false);
	const [editingRule, setEditingRule] = useState<GuardrailRule | null>(null);
	const { data: rules, isLoading } = useListGuardrailRulesQuery();
	const [createRule] = useCreateGuardrailRuleMutation();
	const [updateRule] = useUpdateGuardrailRuleMutation();
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

	const handleUpdateRule = async () => {
		if (!editingRule || !editingRule.name.trim() || !editingRule.pattern.trim()) return;
		await updateRule({
			ruleId: String(editingRule.id),
			data: {
				name: editingRule.name,
				type: editingRule.type,
				pattern: editingRule.pattern,
				action: editingRule.action,
				enabled: editingRule.enabled,
			},
		});
		setEditingRule(null);
	};

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">Guardrails Configuration</h2>
					<p className="text-muted-foreground text-sm">Define input/output guardrail rules for content safety</p>
				</div>
				<Dialog open={showCreate} onOpenChange={setShowCreate}>
					<DialogTrigger asChild>
						<Button size="sm">
							<Plus className="mr-1 h-3 w-3" /> Add
						</Button>
					</DialogTrigger>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Create Guardrails Rule</DialogTitle>
						</DialogHeader>
						<div className="space-y-3 pt-2">
							<div>
								<Label className="text-xs">Name</Label>
								<Input
									className="mt-1 h-8 text-xs"
									placeholder="Enter name"
									value={newRuleName}
									onChange={(e) => setNewRuleName(e.target.value)}
								/>
							</div>
							<div>
								<Label className="text-xs">Type</Label>
								<Select value={newRuleType} onValueChange={setNewRuleType}>
									<SelectTrigger className="mt-1 h-8 text-xs">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="regex">Regex</SelectItem>
										<SelectItem value="keyword">Keyword</SelectItem>
										<SelectItem value="ml">ML-based</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div>
								<Label className="text-xs">Pattern</Label>
								<Input
									className="mt-1 h-8 text-xs"
									placeholder="Enter pattern or expression"
									value={newRulePattern}
									onChange={(e) => setNewRulePattern(e.target.value)}
								/>
							</div>
							<div>
								<Label className="text-xs">Action</Label>
								<Select value={newRuleAction} onValueChange={setNewRuleAction}>
									<SelectTrigger className="mt-1 h-8 text-xs">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="block">Block</SelectItem>
										<SelectItem value="flag">Flag</SelectItem>
										<SelectItem value="redact">Redact</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<Button size="sm" className="w-full" onClick={handleCreateRule}>
								Create
							</Button>
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
							<TableHead className="text-right text-xs">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading && (
							<TableRow>
								<TableCell colSpan={6} className="text-muted-foreground py-8 text-center text-xs">
									Loading...
								</TableCell>
							</TableRow>
						)}
						{!isLoading &&
							rules?.rules?.map((r) => (
								<TableRow key={String(r.id)}>
									<TableCell className="font-mono text-xs font-medium">{r.name}</TableCell>
									<TableCell className="text-xs">
										<Badge variant="outline" className="text-[10px]">
											{r.type}
										</Badge>
									</TableCell>
									<TableCell className="text-xs">
										<Badge variant={r.action === "block" ? "destructive" : "default"} className="text-[10px]">
											{r.action}
										</Badge>
									</TableCell>
									<TableCell className="text-xs">
										<Badge variant={r.enabled ? "default" : "secondary"} className="text-[10px]">
											{r.enabled ? "active" : "disabled"}
										</Badge>
									</TableCell>
									<TableCell className="text-muted-foreground text-xs">
										{r.updated_at ? new Date(r.updated_at).toLocaleDateString() : "—"}
									</TableCell>
									<TableCell className="text-right text-xs">
										<div className="flex items-center justify-end gap-1">
											<Button
												variant="ghost"
												size="icon"
												className="h-6 w-6"
												onClick={() => setEditingRule(r)}
												data-testid="guardrails-rule-edit-btn"
											>
												<Pencil className="h-3.5 w-3.5" />
											</Button>
											<Button
												variant="ghost"
												size="icon"
												className="text-destructive hover:text-destructive h-6 w-6"
												onClick={() => handleDeleteRule(String(r.id))}
											>
												<Trash2 className="h-3.5 w-3.5" />
											</Button>
										</div>
									</TableCell>
								</TableRow>
							))}
						{!isLoading && (!rules?.rules || rules.rules.length === 0) && (
							<TableRow>
								<TableCell colSpan={6} className="text-muted-foreground py-8 text-center text-xs">
									No guardrail rules found
								</TableCell>
							</TableRow>
						)}
					</TableBody>
				</Table>
			</Card>

			<Dialog open={editingRule !== null} onOpenChange={(open) => !open && setEditingRule(null)}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Edit Guardrails Rule</DialogTitle>
					</DialogHeader>
					{editingRule && (
						<div className="space-y-3 pt-2">
							<div>
								<Label className="text-xs">Name</Label>
								<Input
									className="mt-1 h-8 text-xs"
									value={editingRule.name}
									onChange={(e) => setEditingRule({ ...editingRule, name: e.target.value })}
								/>
							</div>
							<div>
								<Label className="text-xs">Type</Label>
								<Select value={editingRule.type} onValueChange={(type) => setEditingRule({ ...editingRule, type })}>
									<SelectTrigger className="mt-1 h-8 text-xs">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="regex">Regex</SelectItem>
										<SelectItem value="keyword">Keyword</SelectItem>
										<SelectItem value="ml">ML-based</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div>
								<Label className="text-xs">Pattern</Label>
								<Input
									className="mt-1 h-8 text-xs"
									value={editingRule.pattern}
									onChange={(e) => setEditingRule({ ...editingRule, pattern: e.target.value })}
								/>
							</div>
							<div>
								<Label className="text-xs">Action</Label>
								<Select value={editingRule.action} onValueChange={(action) => setEditingRule({ ...editingRule, action })}>
									<SelectTrigger className="mt-1 h-8 text-xs">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="block">Block</SelectItem>
										<SelectItem value="flag">Flag</SelectItem>
										<SelectItem value="redact">Redact</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div>
								<Label className="text-xs">Status</Label>
								<Select
									value={editingRule.enabled ? "active" : "disabled"}
									onValueChange={(status) => setEditingRule({ ...editingRule, enabled: status === "active" })}
								>
									<SelectTrigger className="mt-1 h-8 text-xs">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="active">Active</SelectItem>
										<SelectItem value="disabled">Disabled</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<DialogFooter>
								<Button type="button" variant="outline" size="sm" onClick={() => setEditingRule(null)}>
									Cancel
								</Button>
								<Button type="button" size="sm" onClick={handleUpdateRule} data-testid="guardrails-rule-save-btn">
									Save
								</Button>
							</DialogFooter>
						</div>
					)}
				</DialogContent>
			</Dialog>
		</div>
	);
}