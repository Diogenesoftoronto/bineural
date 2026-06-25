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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAppDispatch, useAppSelector } from "@/lib/store/hooks";
import {
	SCIMGroup,
	SCIMUser,
	useCreateSCIMGroupMutation,
	useCreateSCIMUserMutation,
	useDeleteSCIMGroupMutation,
	useDeleteSCIMUserMutation,
	useGetAuthTypeQuery,
	useListSCIMGroupsQuery,
	useListSCIMUsersQuery,
	useUpdateSCIMGroupMutation,
	useUpdateSCIMUserMutation,
} from "@enterprise/lib/store/apis/scimApi";
import { setScimProvider, setScimTab } from "@enterprise/lib/store/slices";
import { Pencil, Plus, RefreshCw, Settings, Trash2, UserCog, Users } from "lucide-react";
import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const splitList = (value: FormDataEntryValue | null): string[] =>
	String(value ?? "")
		.split(",")
		.map((item) => item.trim())
		.filter(Boolean);

export default function SCIMView() {
	const dispatch = useAppDispatch();
	const activeTab = useAppSelector((state) => state.scim.activeTab);
	const provider = useAppSelector((state) => state.scim.provider);
	const { data: authType, isLoading: authLoading } = useGetAuthTypeQuery();
	const { data: usersData, isLoading: usersLoading, refetch: refetchUsers } = useListSCIMUsersQuery();
	const { data: groupsData, isLoading: groupsLoading, refetch: refetchGroups } = useListSCIMGroupsQuery();
	const [createUser] = useCreateSCIMUserMutation();
	const [updateUser] = useUpdateSCIMUserMutation();
	const [deleteUser] = useDeleteSCIMUserMutation();
	const [createGroup] = useCreateSCIMGroupMutation();
	const [updateGroup] = useUpdateSCIMGroupMutation();
	const [deleteGroup] = useDeleteSCIMGroupMutation();

	const [showCreateUser, setShowCreateUser] = useState(false);
	const [showCreateGroup, setShowCreateGroup] = useState(false);
	const [editingUser, setEditingUser] = useState<SCIMUser | null>(null);
	const [editingGroup, setEditingGroup] = useState<SCIMGroup | null>(null);
	const [deletingUserId, setDeletingUserId] = useState<string | null>(null);
	const [deletingGroupId, setDeletingGroupId] = useState<string | null>(null);

	const users = usersData?.users ?? [];
	const groups = groupsData?.groups ?? [];

	const handleCreateUser = async (fd: FormData) => {
		await createUser({
			user_name: String(fd.get("user_name") ?? "").trim(),
			display_name: String(fd.get("display_name") ?? "").trim(),
			email: String(fd.get("email") ?? "").trim(),
			active: fd.get("active") === "on",
			groups: splitList(fd.get("groups")),
		});
		setShowCreateUser(false);
	};

	const handleUpdateUser = async (fd: FormData) => {
		if (!editingUser) return;
		await updateUser({
			id: editingUser.id,
			body: {
				user_name: String(fd.get("user_name") ?? editingUser.user_name).trim(),
				display_name: String(fd.get("display_name") ?? editingUser.display_name ?? "").trim(),
				email: String(fd.get("email") ?? editingUser.email ?? "").trim(),
				active: fd.get("active") === "on",
				groups: splitList(fd.get("groups")),
			},
		});
		setEditingUser(null);
	};

	const handleCreateGroup = async (fd: FormData) => {
		await createGroup({
			display_name: String(fd.get("display_name") ?? "").trim(),
			members: splitList(fd.get("members")),
		});
		setShowCreateGroup(false);
	};

	const handleUpdateGroup = async (fd: FormData) => {
		if (!editingGroup) return;
		await updateGroup({
			id: editingGroup.id,
			body: {
				display_name: String(fd.get("display_name") ?? editingGroup.display_name).trim(),
				members: splitList(fd.get("members")),
			},
		});
		setEditingGroup(null);
	};

	const handleDeleteUser = async () => {
		if (!deletingUserId) return;
		await deleteUser(deletingUserId);
		setDeletingUserId(null);
	};

	const handleDeleteGroup = async () => {
		if (!deletingGroupId) return;
		await deleteGroup(deletingGroupId);
		setDeletingGroupId(null);
	};

	const handleSync = () => {
		void refetchUsers();
		void refetchGroups();
	};

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">SCIM User Provisioning</h2>
					<p className="text-muted-foreground text-sm">
						Manage users and groups via SCIM protocol. {authType ? `Auth: ${authType.type}` : ""} {authLoading && "Loading..."}
					</p>
				</div>
				<div className="flex items-center gap-2">
					<Select value={provider ?? undefined} onValueChange={(v) => dispatch(setScimProvider(v))}>
						<SelectTrigger className="h-8 w-[160px] text-xs">
							<SelectValue placeholder="Provider" />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="okta">Okta</SelectItem>
							<SelectItem value="azure-ad">Azure AD</SelectItem>
							<SelectItem value="google">Google Workspace</SelectItem>
							<SelectItem value="onelogin">OneLogin</SelectItem>
						</SelectContent>
					</Select>
					<Button variant="outline" size="sm" onClick={handleSync}>
						<RefreshCw className="mr-1 h-3 w-3" />
						Sync
					</Button>
				</div>
			</div>

			<Tabs value={activeTab} onValueChange={(v) => dispatch(setScimTab(v as "users" | "groups" | "config"))}>
				<TabsList>
					<TabsTrigger value="users" className="gap-1.5">
						<Users className="h-3.5 w-3.5" />
						Users
						<Badge variant="secondary" className="ml-1 px-1 text-[10px]">
							{users.length}
						</Badge>
					</TabsTrigger>
					<TabsTrigger value="groups" className="gap-1.5">
						<UserCog className="h-3.5 w-3.5" />
						Groups
						<Badge variant="secondary" className="ml-1 px-1 text-[10px]">
							{groups.length}
						</Badge>
					</TabsTrigger>
					<TabsTrigger value="config" className="gap-1.5">
						<Settings className="h-3.5 w-3.5" />
						Configuration
					</TabsTrigger>
				</TabsList>

				<TabsContent value="users" className="mt-4 space-y-3">
					<div className="flex justify-end">
						<Button size="sm" onClick={() => setShowCreateUser(true)} data-testid="scim-user-add-btn">
							<Plus className="mr-1 h-3 w-3" /> Add user
						</Button>
					</div>
					<Card>
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead className="text-xs">Username</TableHead>
									<TableHead className="text-xs">Display Name</TableHead>
									<TableHead className="text-xs">Email</TableHead>
									<TableHead className="text-xs">Status</TableHead>
									<TableHead className="text-xs">Groups</TableHead>
									<TableHead className="text-right text-xs">Actions</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{usersLoading && (
									<TableRow>
										<TableCell colSpan={6} className="text-muted-foreground py-8 text-center text-xs">
											Loading...
										</TableCell>
									</TableRow>
								)}
								{!usersLoading &&
									users.map((u) => (
										<TableRow key={u.id}>
											<TableCell className="text-xs font-medium">{u.user_name}</TableCell>
											<TableCell className="text-muted-foreground text-xs">{u.display_name || "—"}</TableCell>
											<TableCell className="text-muted-foreground text-xs">{u.email || "—"}</TableCell>
											<TableCell className="text-xs">
												<Badge variant={u.active ? "default" : "secondary"} className="text-[10px]">
													{u.active ? "active" : "suspended"}
												</Badge>
											</TableCell>
											<TableCell className="text-muted-foreground text-xs">{u.groups?.join(", ") || "—"}</TableCell>
											<TableCell className="text-right text-xs">
												<div className="flex items-center justify-end gap-1">
													<Button
														variant="ghost"
														size="icon"
														className="h-6 w-6"
														onClick={() => setEditingUser(u)}
														data-testid="scim-user-edit-btn"
													>
														<Pencil className="h-3.5 w-3.5" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														className="text-muted-foreground hover:text-destructive h-6 w-6"
														onClick={() => setDeletingUserId(u.id)}
														data-testid="scim-user-delete-btn"
													>
														<Trash2 className="h-3.5 w-3.5" />
													</Button>
												</div>
											</TableCell>
										</TableRow>
									))}
								{!usersLoading && users.length === 0 && (
									<TableRow>
										<TableCell colSpan={6} className="text-muted-foreground py-8 text-center text-xs">
											No SCIM users found
										</TableCell>
									</TableRow>
								)}
							</TableBody>
						</Table>
					</Card>
				</TabsContent>

				<TabsContent value="groups" className="mt-4 space-y-3">
					<div className="flex justify-end">
						<Button size="sm" onClick={() => setShowCreateGroup(true)} data-testid="scim-group-add-btn">
							<Plus className="mr-1 h-3 w-3" /> Add group
						</Button>
					</div>
					<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
						{groupsLoading && (
							<Card>
								<CardContent className="text-muted-foreground py-8 text-center text-xs">Loading...</CardContent>
							</Card>
						)}
						{!groupsLoading &&
							groups.map((g) => (
								<Card key={g.id}>
									<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
										<CardTitle className="text-sm">{g.display_name}</CardTitle>
										<div className="flex items-center gap-1">
											<Button
												variant="ghost"
												size="icon"
												className="h-6 w-6"
												onClick={() => setEditingGroup(g)}
												data-testid="scim-group-edit-btn"
											>
												<Pencil className="h-3.5 w-3.5" />
											</Button>
											<Button
												variant="ghost"
												size="icon"
												className="text-muted-foreground hover:text-destructive h-6 w-6"
												onClick={() => setDeletingGroupId(g.id)}
												data-testid="scim-group-delete-btn"
											>
												<Trash2 className="h-3.5 w-3.5" />
											</Button>
										</div>
									</CardHeader>
									<CardContent>
										<p className="text-xs">
											<span className="font-medium">{g.members?.length ?? 0}</span> members
										</p>
										<p className="text-muted-foreground mt-2 truncate text-xs">{g.members?.join(", ") || "No members"}</p>
									</CardContent>
								</Card>
							))}
						{!groupsLoading && groups.length === 0 && (
							<Card>
								<CardContent className="text-muted-foreground py-8 text-center text-xs">No SCIM groups found</CardContent>
							</Card>
						)}
					</div>
				</TabsContent>

				<TabsContent value="config" className="mt-4">
					<Card>
						<CardHeader>
							<CardTitle className="text-sm">SCIM Endpoint Configuration</CardTitle>
						</CardHeader>
						<CardContent className="space-y-4">
							<div className="flex items-center justify-between">
								<div>
									<Label className="text-sm">Enable SCIM Provisioning</Label>
									<p className="text-muted-foreground text-xs">Allow external identity providers to sync users</p>
								</div>
								<Switch defaultChecked />
							</div>
							<div>
								<Label className="text-xs">SCIM Base URL</Label>
								<Input
									className="mt-1 h-8 text-xs"
									value={`${typeof window !== "undefined" ? window.location.origin : ""}/api/scim/v2`}
									readOnly
								/>
							</div>
							<div>
								<Label className="text-xs">Bearer Token</Label>
								<Input className="mt-1 h-8 text-xs" type="password" value="••••••••••••" readOnly />
							</div>
						</CardContent>
					</Card>
				</TabsContent>
			</Tabs>

			<UserDialog open={showCreateUser} title="Create User" onOpenChange={setShowCreateUser} onSubmit={handleCreateUser} />
			<UserDialog
				open={editingUser !== null}
				title="Edit User"
				user={editingUser ?? undefined}
				onOpenChange={(open) => !open && setEditingUser(null)}
				onSubmit={handleUpdateUser}
			/>
			<GroupDialog open={showCreateGroup} title="Create Group" onOpenChange={setShowCreateGroup} onSubmit={handleCreateGroup} />
			<GroupDialog
				open={editingGroup !== null}
				title="Edit Group"
				group={editingGroup ?? undefined}
				onOpenChange={(open) => !open && setEditingGroup(null)}
				onSubmit={handleUpdateGroup}
			/>

			<AlertDialog open={deletingUserId !== null} onOpenChange={(open) => !open && setDeletingUserId(null)}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete SCIM user?</AlertDialogTitle>
						<AlertDialogDescription>This removes the user from the SCIM directory.</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction onClick={handleDeleteUser} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
			<AlertDialog open={deletingGroupId !== null} onOpenChange={(open) => !open && setDeletingGroupId(null)}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete SCIM group?</AlertDialogTitle>
						<AlertDialogDescription>This removes the group and its membership list.</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction onClick={handleDeleteGroup} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}

function UserDialog({
	open,
	title,
	user,
	onOpenChange,
	onSubmit,
}: {
	open: boolean;
	title: string;
	user?: SCIMUser;
	onOpenChange: (open: boolean) => void;
	onSubmit: (fd: FormData) => void;
}) {
	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{title}</DialogTitle>
					<DialogDescription>Manage a SCIM-provisioned user.</DialogDescription>
				</DialogHeader>
				<form
					className="space-y-3 pt-2"
					onSubmit={(e) => {
						e.preventDefault();
						onSubmit(new FormData(e.currentTarget));
					}}
				>
					<div>
						<Label className="text-xs">Username</Label>
						<Input className="mt-1 h-8 text-xs" name="user_name" defaultValue={user?.user_name} required />
					</div>
					<div>
						<Label className="text-xs">Display Name</Label>
						<Input className="mt-1 h-8 text-xs" name="display_name" defaultValue={user?.display_name} />
					</div>
					<div>
						<Label className="text-xs">Email</Label>
						<Input className="mt-1 h-8 text-xs" name="email" type="email" defaultValue={user?.email} />
					</div>
					<div>
						<Label className="text-xs">Groups (comma-separated)</Label>
						<Input className="mt-1 h-8 text-xs" name="groups" defaultValue={user?.groups?.join(", ")} />
					</div>
					<div className="flex items-center justify-between rounded-md border border-zinc-200 px-3 py-2 dark:border-zinc-700">
						<div>
							<Label className="text-xs">Active</Label>
							<p className="text-muted-foreground text-[10px]">Inactive users are suspended.</p>
						</div>
						<Switch name="active" defaultChecked={user?.active ?? true} />
					</div>
					<DialogFooter>
						<Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
							Cancel
						</Button>
						<Button type="submit" size="sm" data-testid="scim-user-save-btn">
							Save
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}

function GroupDialog({
	open,
	title,
	group,
	onOpenChange,
	onSubmit,
}: {
	open: boolean;
	title: string;
	group?: SCIMGroup;
	onOpenChange: (open: boolean) => void;
	onSubmit: (fd: FormData) => void;
}) {
	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{title}</DialogTitle>
					<DialogDescription>Manage a SCIM group and members.</DialogDescription>
				</DialogHeader>
				<form
					className="space-y-3 pt-2"
					onSubmit={(e) => {
						e.preventDefault();
						onSubmit(new FormData(e.currentTarget));
					}}
				>
					<div>
						<Label className="text-xs">Name</Label>
						<Input className="mt-1 h-8 text-xs" name="display_name" defaultValue={group?.display_name} required />
					</div>
					<div>
						<Label className="text-xs">Members (comma-separated user IDs)</Label>
						<Input className="mt-1 h-8 text-xs" name="members" defaultValue={group?.members?.join(", ")} />
					</div>
					<DialogFooter>
						<Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
							Cancel
						</Button>
						<Button type="submit" size="sm" data-testid="scim-group-save-btn">
							Save
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}