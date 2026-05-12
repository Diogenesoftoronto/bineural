import { cn } from "@/lib/utils";

interface Props {
	className?: string;
	icon: React.ReactNode;
	title: string;
	description: string;
	featurePath?: string;
	align?: "middle" | "top";
	testIdPrefix?: string;
}

export default function ContactUsView({ icon, title, description, className, featurePath, align = "middle", testIdPrefix }: Props) {
	return (
		<div className={cn("flex flex-col items-center gap-4 text-center", align === "middle" ? "justify-center" : "justify-start", className)}>
			<div className="text-muted-foreground">{icon}</div>
			<div className="flex flex-col gap-1">
				<h1 className="text-xl font-medium">{title}</h1>
				<div className="text-muted-foreground mt-2 max-w-[600px] text-sm font-normal">{description}</div>
				{featurePath && (
					<p className="text-muted-foreground mt-2 text-xs">
						Navigate to <span className="font-mono">{featurePath}</span> to configure this feature.
					</p>
				)}
			</div>
		</div>
	);
}
