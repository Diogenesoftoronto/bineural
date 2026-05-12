import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";
import { Download } from "lucide-react";
import { useListAuditEntriesQuery, useDeleteAuditEntriesBeforeMutation } from "@enterprise/lib/store/apis/auditApi";

export default function AuditLogsView() {
	const [eventType, setEventType] = useState<string>("all");
	const [offset, setOffset] = useState(0);
	const { data, isLoading } = useListAuditEntriesQuery({
		event_type: eventType !== "all" ? eventType : undefined,
		limit: 50,
		offset,
	});
	const [deleteBefore] = useDeleteAuditEntriesBeforeMutation();

	return (
		<div className="flex h-full flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h2 className="text-lg font-semibold">Audit Logs</h2>
					<p className="text-sm text-muted-foreground">Track all system events for compliance and security review</p>
				</div>
				<div className="flex items-center gap-2">
					<Select value={eventType} onValueChange={(v) => { setEventType(v); setOffset(0); }}>
						<SelectTrigger className="w-[140px] h-8 text-xs"><SelectValue /></SelectTrigger>
						<SelectContent>
							<SelectItem value="all">All Events</SelectItem>
							<SelectItem value="auth">Authentication</SelectItem>
							<SelectItem value="config">Configuration</SelectItem>
							<SelectItem value="guardrail">Guardrails</SelectItem>
							<SelectItem value="key">Virtual Keys</SelectItem>
						</SelectContent>
					</Select>
					<Button variant="outline" size="sm"><Download className="mr-1 h-3 w-3" /> Export</Button>
				</div>
			</div>
			<Card>
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead className="text-xs">Timestamp</TableHead>
							<TableHead className="text-xs">User</TableHead>
							<TableHead className="text-xs">Action</TableHead>
							<TableHead className="text-xs">Resource</TableHead>
							<TableHead className="text-xs">Status</TableHead>
							<TableHead className="text-xs">IP</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading && <TableRow><TableCell colSpan={6} className="text-center text-xs text-muted-foreground py-8">Loading...</TableCell></TableRow>}
						{!isLoading && data?.entries?.map((l) => (
							<TableRow key={l.event_id}>
								<TableCell className="text-xs text-muted-foreground whitespace-nowrap">{new Date(l.timestamp).toLocaleString()}</TableCell>
								<TableCell className="text-xs font-medium">{l.user_email || l.user_id}</TableCell>
								<TableCell className="text-xs"><Badge variant="outline" className="text-[10px] font-mono">{l.action}</Badge></TableCell>
								<TableCell className="text-xs text-muted-foreground">{l.resource}</TableCell>
								<TableCell className="text-xs"><Badge variant={l.status_code >= 400 ? "destructive" : "default"} className="text-[10px]">{l.status_code >= 400 ? "error" : "success"}</Badge></TableCell>
								<TableCell className="text-xs text-muted-foreground font-mono">{l.ip_address}</TableCell>
							</TableRow>
						))}
						{!isLoading && (!data?.entries || data.entries.length === 0) && <TableRow><TableCell colSpan={6} className="text-center text-xs text-muted-foreground py-8">No audit logs found</TableCell></TableRow>}
					</TableBody>
				</Table>
			</Card>
			{!isLoading && data?.entries && data.entries.length > 0 && (
				<div className="flex justify-center">
					<Button variant="outline" size="sm" onClick={() => setOffset((prev) => prev + 50)}>
						Load More
					</Button>
				</div>
			)}
		</div>
	);
}
