import { useGetAuthTypeQuery } from "@enterprise/lib/store/apis/scimApi";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useState } from "react";
import { useAppDispatch, useAppSelector } from "@/lib/store/hooks";
import { setScimTab, setScimProvider } from "@enterprise/lib/store/slices";
import { Plus, RefreshCw, Users, UserCog, Settings } from "lucide-react";

export default function SCIMView() {
	const dispatch = useAppDispatch();
	const activeTab = useAppSelector((state) => state.scim.activeTab);
	const provider = useAppSelector((state) => state.scim.provider);
	const { data: authType, isLoading } = useGetAuthTypeQuery();
	const [showAddUser, setShowAddUser] = useState(false);

	const mockUsers = [
		{ id: "usr_1", name: "Alice Johnson", email: "alice@example.com", status: "active", groups: ["Engineering", "Admin"] },
		{ id: "usr_2", name: "Bob Smith", email: "bob@example.com", status: "active", groups: ["Engineering"] },
		{ id: "usr_3", name: "Carol White", email: "carol@example.com", status: "suspended", groups: ["Sales"] },
	];

	const mockGroups = [
		{ id: "grp_1", name: "Engineering", members: 12, description: "Engineering team" },
		{ id: "grp_2", name: "Admin", members: 3, description: "System administrators" },
		{ id: "grp_3", name: "Sales", members: 8, description: "Sales team" },
	];

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">SCIM User Provisioning</h2>
					<p className="text-sm text-muted-foreground">
						Manage users and groups via SCIM protocol. {authType ? `Auth: ${authType}` : ""} {isLoading && "Loading..."}
					</p>
				</div>
				<div className="flex items-center gap-2">
					<Select value={provider ?? undefined} onValueChange={(v) => dispatch(setScimProvider(v))}>
						<SelectTrigger className="w-[160px] h-8 text-xs">
							<SelectValue placeholder="Provider" />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="okta">Okta</SelectItem>
							<SelectItem value="azure-ad">Azure AD</SelectItem>
							<SelectItem value="google">Google Workspace</SelectItem>
							<SelectItem value="onelogin">OneLogin</SelectItem>
						</SelectContent>
					</Select>
					<Button variant="outline" size="sm">
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
						<Badge variant="secondary" className="ml-1 text-[10px] px-1">{mockUsers.length}</Badge>
					</TabsTrigger>
					<TabsTrigger value="groups" className="gap-1.5">
						<UserCog className="h-3.5 w-3.5" />
						Groups
						<Badge variant="secondary" className="ml-1 text-[10px] px-1">{mockGroups.length}</Badge>
					</TabsTrigger>
					<TabsTrigger value="config" className="gap-1.5">
						<Settings className="h-3.5 w-3.5" />
						Configuration
					</TabsTrigger>
				</TabsList>

				<TabsContent value="users" className="mt-4">
					<div className="flex items-center justify-between mb-4">
						<Input placeholder="Search users..." className="max-w-[300px] h-8 text-xs" />
						<Dialog open={showAddUser} onOpenChange={setShowAddUser}>
							<DialogTrigger asChild>
								<Button size="sm"><Plus className="mr-1 h-3 w-3" /> Add User</Button>
							</DialogTrigger>
							<DialogContent>
								<DialogHeader><DialogTitle>Provision New User</DialogTitle></DialogHeader>
								<div className="space-y-3 pt-2">
									<div><Label className="text-xs">Full Name</Label><Input className="h-8 text-xs mt-1" placeholder="John Doe" /></div>
									<div><Label className="text-xs">Email</Label><Input className="h-8 text-xs mt-1" placeholder="john@example.com" /></div>
									<div><Label className="text-xs">Groups</Label><Input className="h-8 text-xs mt-1" placeholder="Engineering, Admin" /></div>
									<Button size="sm" className="w-full" onClick={() => setShowAddUser(false)}>Provision User</Button>
								</div>
							</DialogContent>
						</Dialog>
					</div>
					<Card>
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead className="text-xs">Name</TableHead>
									<TableHead className="text-xs">Email</TableHead>
									<TableHead className="text-xs">Status</TableHead>
									<TableHead className="text-xs">Groups</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{mockUsers.map((u) => (
									<TableRow key={u.id}>
										<TableCell className="text-xs font-medium">{u.name}</TableCell>
										<TableCell className="text-xs text-muted-foreground">{u.email}</TableCell>
										<TableCell className="text-xs">
											<Badge variant={u.status === "active" ? "default" : "secondary"} className="text-[10px]">{u.status}</Badge>
										</TableCell>
										<TableCell className="text-xs text-muted-foreground">{u.groups.join(", ")}</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					</Card>
				</TabsContent>

				<TabsContent value="groups" className="mt-4">
					<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
						{mockGroups.map((g) => (
							<Card key={g.id}>
								<CardHeader className="pb-2">
									<CardTitle className="text-sm">{g.name}</CardTitle>
								</CardHeader>
								<CardContent>
									<p className="text-xs text-muted-foreground">{g.description}</p>
									<p className="text-xs mt-2"><span className="font-medium">{g.members}</span> members</p>
								</CardContent>
							</Card>
						))}
					</div>
				</TabsContent>

				<TabsContent value="config" className="mt-4">
					<Card>
						<CardHeader><CardTitle className="text-sm">SCIM Endpoint Configuration</CardTitle></CardHeader>
						<CardContent className="space-y-4">
							<div className="flex items-center justify-between">
								<div><Label className="text-sm">Enable SCIM Provisioning</Label><p className="text-xs text-muted-foreground">Allow external identity providers to sync users</p></div>
								<Switch defaultChecked />
							</div>
							<div>
								<Label className="text-xs">SCIM Base URL</Label>
								<Input className="h-8 text-xs mt-1" value={`${window.location.origin}/scim/v2`} readOnly />
							</div>
							<div>
								<Label className="text-xs">Bearer Token</Label>
								<Input className="h-8 text-xs mt-1" type="password" value="••••••••••••" readOnly />
							</div>
							<div className="flex items-center justify-between">
								<div><Label className="text-sm">Auto-Provision on First Login</Label><p className="text-xs text-muted-foreground">Create local user on first SSO sign-in</p></div>
								<Switch defaultChecked />
							</div>
						</CardContent>
					</Card>
				</TabsContent>
			</Tabs>
		</div>
	);
}
