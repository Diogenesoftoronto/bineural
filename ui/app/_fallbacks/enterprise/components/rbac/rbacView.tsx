import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";
import { Plus, Shield, Trash2 } from "lucide-react";
import { useListRolesQuery, useCreateRoleMutation, useDeleteRoleMutation } from "@enterprise/lib/store/apis/rbacApi";

export default function RBACView() {
	const [showCreate, setShowCreate] = useState(false);
	const { data: roles, isLoading } = useListRolesQuery();
	const [createRole] = useCreateRoleMutation();
	const [deleteRole] = useDeleteRoleMutation();
	const [newRoleName, setNewRoleName] = useState("");
	const [newRoleDesc, setNewRoleDesc] = useState("");

	const handleCreateRole = async () => {
		if (!newRoleName.trim()) return;
		await createRole({ name: newRoleName, description: newRoleDesc });
		setNewRoleName("");
		setNewRoleDesc("");
		setShowCreate(false);
	};

	const handleDeleteRole = async (roleId: string) => {
		await deleteRole(roleId);
	};

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">Roles & Permissions</h2>
					<p className="text-sm text-muted-foreground">Configure RBAC roles and permission policies</p>
				</div>
				<Dialog open={showCreate} onOpenChange={setShowCreate}>
					<DialogTrigger asChild>
						<Button size="sm"><Plus className="mr-1 h-3 w-3" /> Create Role</Button>
					</DialogTrigger>
					<DialogContent>
						<DialogHeader><DialogTitle>Create Custom Role</DialogTitle></DialogHeader>
						<div className="space-y-3 pt-2">
							<div><Label className="text-xs">Role Name</Label><Input className="h-8 text-xs mt-1" placeholder="e.g. ML Engineer" value={newRoleName} onChange={(e) => setNewRoleName(e.target.value)} /></div>
							<div><Label className="text-xs">Description</Label><Input className="h-8 text-xs mt-1" placeholder="Describe this role's access level" value={newRoleDesc} onChange={(e) => setNewRoleDesc(e.target.value)} /></div>
							<div><Label className="text-xs">Base Permission Set</Label>
								<Select><SelectTrigger className="h-8 text-xs mt-1"><SelectValue placeholder="Select base" /></SelectTrigger>
									<SelectContent><SelectItem value="viewer">Viewer</SelectItem><SelectItem value="editor">Editor</SelectItem><SelectItem value="admin">Admin</SelectItem></SelectContent>
								</Select>
							</div>
							<Button size="sm" className="w-full" onClick={handleCreateRole}>Create Role</Button>
						</div>
					</DialogContent>
				</Dialog>
			</div>

			<Card>
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead className="text-xs">Role</TableHead>
							<TableHead className="text-xs">Description</TableHead>
							<TableHead className="text-xs">Type</TableHead>
							<TableHead className="text-xs">Users</TableHead>
							<TableHead className="text-xs">Permissions</TableHead>
							<TableHead className="text-xs text-right">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading && <TableRow><TableCell colSpan={6} className="text-center text-xs text-muted-foreground py-8">Loading...</TableCell></TableRow>}
						{!isLoading && roles?.roles?.map((r) => (
							<TableRow key={String(r.id)}>
								<TableCell className="text-xs font-medium">
									<div className="flex items-center gap-1.5"><Shield className="h-3 w-3" />{r.name}</div>
								</TableCell>
								<TableCell className="text-xs text-muted-foreground">{r.description}</TableCell>
								<TableCell className="text-xs"><Badge variant={r.is_system ? "secondary" : "outline"} className="text-[10px]">{r.is_system ? "System" : "Custom"}</Badge></TableCell>
								<TableCell className="text-xs">{r.user_count}</TableCell>
								<TableCell className="text-xs">{r.permissions_count}</TableCell>
								<TableCell className="text-xs text-right">
									{!r.is_system && (
										<Button variant="ghost" size="sm" className="h-6 text-xs text-destructive hover:text-destructive" onClick={() => handleDeleteRole(String(r.id))}>
											<Trash2 className="h-3 w-3" />
										</Button>
									)}
								</TableCell>
							</TableRow>
						))}
						{!isLoading && (!roles?.roles || roles.roles.length === 0) && <TableRow><TableCell colSpan={6} className="text-center text-xs text-muted-foreground py-8">No roles found</TableCell></TableRow>}
					</TableBody>
				</Table>
			</Card>
		</div>
	);
}
