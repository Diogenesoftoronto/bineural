import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alertDialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
	useCreateVirtualKeyMutation,
	useDeleteVirtualKeyMutation,
	useGetVirtualKeysQuery,
	useUpdateVirtualKeyMutation,
} from "@/lib/store/apis/governanceApi";
import { VirtualKey } from "@/lib/types/governance";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

const rpmFromKey = (key: VirtualKey): number => key.rate_limit?.request_max_limit ?? 0;
const scopesFromKey = (key: VirtualKey): string => key.description ?? "";
const dateOnly = (value?: string): string => (value ? new Date(value).toISOString().slice(0, 10) : "—");

const rpmRateLimit = (rpm: number) =>
	rpm > 0
		? {
				request_max_limit: rpm,
				request_reset_duration: "1m",
			}
		: undefined;

export default function APIKeysView() {
	const { data, isLoading } = useGetVirtualKeysQuery({ sort_by: "created_at", order: "desc" });
	const [createVirtualKey] = useCreateVirtualKeyMutation();
	const [updateVirtualKey] = useUpdateVirtualKeyMutation();
	const [deleteVirtualKey] = useDeleteVirtualKeyMutation();
	const [showCreate, setShowCreate] = useState(false);
	const [editing, setEditing] = useState<VirtualKey | null>(null);
	const [deletingId, setDeletingId] = useState<string | null>(null);

	const items = data?.virtual_keys ?? [];
	const formSwitchStatus = (fd: FormData): boolean => fd.get("status") === "on";

	const handleCreate = async (fd: FormData) => {
		const name = String(fd.get("name") ?? "").trim();
		if (!name) return;
		const rpm = Number(fd.get("rateLimitRpm") ?? 0);
		await createVirtualKey({
			name,
			description: String(fd.get("scopes") ?? "").trim(),
			is_active: true,
			rate_limit: rpmRateLimit(rpm),
		});
		setShowCreate(false);
	};

	const handleUpdate = async (fd: FormData) => {
		if (!editing) return;
		const rpm = Number(fd.get("rateLimitRpm") ?? 0);
		await updateVirtualKey({
			vkId: editing.id,
			data: {
				name: String(fd.get("name") ?? editing.name),
				description: String(fd.get("scopes") ?? scopesFromKey(editing)),
				is_active: formSwitchStatus(fd),
				rate_limit: rpmRateLimit(rpm),
				team_id: editing.team_id,
				customer_id: editing.customer_id,
			},
		});
		setEditing(null);
	};

	const handleDelete = async () => {
		if (!deletingId) return;
		await deleteVirtualKey(deletingId);
		setDeletingId(null);
	};

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">Scoped API Keys</h2>
					<p className="text-muted-foreground text-sm">Create and manage API keys with granular scope-based access control</p>
				</div>
				<Dialog open={showCreate} onOpenChange={setShowCreate}>
					<DialogTrigger asChild>
						<Button size="sm" data-testid="api-keys-add-btn">
							<Plus className="mr-1 h-3 w-3" /> Add
						</Button>
					</DialogTrigger>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Create Scoped API Key</DialogTitle>
							<DialogDescription>Provision a new virtual key with explicit scopes.</DialogDescription>
						</DialogHeader>
						<form
							className="space-y-3 pt-2"
							onSubmit={(e) => {
								e.preventDefault();
								void handleCreate(new FormData(e.currentTarget));
							}}
						>
							<div>
								<Label className="text-xs">Name</Label>
								<Input className="mt-1 h-8 text-xs" name="name" placeholder="key:env-purpose" required />
							</div>
							<div>
								<Label className="text-xs">Scopes (comma-separated)</Label>
								<Input className="mt-1 h-8 text-xs" name="scopes" placeholder="models:read, chat:invoke" />
							</div>
							<div>
								<Label className="text-xs">Rate limit (RPM)</Label>
								<Input className="mt-1 h-8 text-xs" name="rateLimitRpm" type="number" min={0} defaultValue={1000} />
							</div>
							<DialogFooter>
								<Button type="button" variant="outline" size="sm" onClick={() => setShowCreate(false)}>
									Cancel
								</Button>
								<Button type="submit" size="sm" data-testid="api-keys-create-btn">
									Create
								</Button>
							</DialogFooter>
						</form>
					</DialogContent>
				</Dialog>
			</div>

			<Card>
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead className="text-xs">Name</TableHead>
							<TableHead className="text-xs">Status</TableHead>
							<TableHead className="text-xs">Scopes</TableHead>
							<TableHead className="text-xs">Rate (RPM)</TableHead>
							<TableHead className="text-xs">Created</TableHead>
							<TableHead className="text-xs">Last Updated</TableHead>
							<TableHead className="text-right text-xs">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading && (
							<TableRow>
								<TableCell colSpan={7} className="text-muted-foreground py-8 text-center text-xs">
									Loading...
								</TableCell>
							</TableRow>
						)}
						{!isLoading &&
							items.map((item) => (
								<TableRow key={item.id}>
									<TableCell className="font-mono text-xs font-medium">{item.name}</TableCell>
									<TableCell className="text-xs">
										<Badge variant={item.is_active ? "default" : "secondary"} className="text-[10px]">
											{item.is_active ? "active" : "disabled"}
										</Badge>
									</TableCell>
									<TableCell className="text-muted-foreground max-w-[240px] truncate text-xs">{scopesFromKey(item) || "—"}</TableCell>
									<TableCell className="text-xs tabular-nums">{rpmFromKey(item).toLocaleString()}</TableCell>
									<TableCell className="text-muted-foreground text-xs">{dateOnly(item.created_at)}</TableCell>
									<TableCell className="text-muted-foreground text-xs">{dateOnly(item.updated_at)}</TableCell>
									<TableCell className="text-right text-xs">
										<div className="flex items-center justify-end gap-1">
											<Button
												variant="ghost"
												size="icon"
												className="h-6 w-6"
												onClick={() => setEditing(item)}
												data-testid="api-keys-edit-btn"
											>
												<Pencil className="h-3.5 w-3.5" />
											</Button>
											<Button
												variant="ghost"
												size="icon"
												className="text-muted-foreground hover:text-destructive h-6 w-6"
												onClick={() => setDeletingId(item.id)}
												data-testid="api-keys-delete-btn"
											>
												<Trash2 className="h-3.5 w-3.5" />
											</Button>
										</div>
									</TableCell>
								</TableRow>
							))}
						{!isLoading && items.length === 0 && (
							<TableRow>
								<TableCell colSpan={7} className="text-muted-foreground py-8 text-center text-xs">
									No scoped API keys found
								</TableCell>
							</TableRow>
						)}
					</TableBody>
				</Table>
			</Card>

			<Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Edit Scoped API Key</DialogTitle>
						<DialogDescription>Update the key's name, scopes, rate limit, or status.</DialogDescription>
					</DialogHeader>
					{editing && (
						<form
							className="space-y-3 pt-2"
							onSubmit={(e) => {
								e.preventDefault();
								void handleUpdate(new FormData(e.currentTarget));
							}}
						>
							<div>
								<Label className="text-xs">Name</Label>
								<Input className="mt-1 h-8 text-xs" name="name" defaultValue={editing.name} required />
							</div>
							<div>
								<Label className="text-xs">Scopes (comma-separated)</Label>
								<Input className="mt-1 h-8 text-xs" name="scopes" defaultValue={scopesFromKey(editing)} />
							</div>
							<div>
								<Label className="text-xs">Rate limit (RPM)</Label>
								<Input className="mt-1 h-8 text-xs" name="rateLimitRpm" type="number" min={0} defaultValue={rpmFromKey(editing)} />
							</div>
							<div className="flex items-center justify-between rounded-md border border-zinc-200 px-3 py-2 dark:border-zinc-700">
								<div>
									<Label className="text-xs">Active</Label>
									<p className="text-muted-foreground text-[10px]">Disabled keys reject all requests.</p>
								</div>
								<Switch name="status" defaultChecked={editing.is_active} />
							</div>
							<DialogFooter>
								<Button type="button" variant="outline" size="sm" onClick={() => setEditing(null)}>
									Cancel
								</Button>
								<Button type="submit" size="sm" data-testid="api-keys-save-btn">
									Save
								</Button>
							</DialogFooter>
						</form>
					)}
				</DialogContent>
			</Dialog>

			<AlertDialog open={deletingId !== null} onOpenChange={(open) => !open && setDeletingId(null)}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete scoped API key?</AlertDialogTitle>
						<AlertDialogDescription>
							This will revoke the key immediately. Any clients using it will start receiving 401 errors.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={handleDelete}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
							data-testid="api-keys-confirm-delete-btn"
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}