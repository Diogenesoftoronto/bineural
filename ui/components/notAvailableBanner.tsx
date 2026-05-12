import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Database } from "lucide-react";

const NotAvailableBanner = () => {
	return (
		<div className="h-base flex items-center justify-center p-4">
			<div className="w-full max-w-md">
				<Alert className="border-destructive/50 text-destructive/50 dark:text-destructive/70 dark:border-destructive/70 [&>svg]:text-destructive dark:bg-card bg-red-50">
					<AlertTitle className="flex items-center gap-2">
						<Database className="dark:text-destructive/70 text-destructive/50 h-4 w-4" />
						Config store setup is missing.
					</AlertTitle>
					<AlertDescription className="mt-2 space-y-2 text-xs">
						<div>The UI requires a database connection to store configuration data, but no database is currently configured.</div>
						<div className="text-muted-foreground">
							To enable the UI, please add the database settings to your config.json. Set{" "}
							<code className="rounded bg-muted px-1 py-0.5 text-xs font-mono">config_store</code>{" "}
							with your database connection string under the{" "}
							<code className="rounded bg-muted px-1 py-0.5 text-xs font-mono">framework</code> section.
						</div>
					</AlertDescription>
				</Alert>
			</div>
		</div>
	);
};

export default NotAvailableBanner;