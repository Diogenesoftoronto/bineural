import { createContext, useContext, useState, useCallback } from "react";

export enum RbacResource {
	GuardrailsConfig = "GuardrailsConfig",
	GuardrailsProviders = "GuardrailsProviders",
	GuardrailRules = "GuardrailRules",
	UserProvisioning = "UserProvisioning",
	Cluster = "Cluster",
	Settings = "Settings",
	Users = "Users",
	Logs = "Logs",
	Observability = "Observability",
	VirtualKeys = "VirtualKeys",
	ModelProvider = "ModelProvider",
	Plugins = "Plugins",
	MCPGateway = "MCPGateway",
	AdaptiveRouter = "AdaptiveRouter",
	AuditLogs = "AuditLogs",
	Customers = "Customers",
	Teams = "Teams",
	RBAC = "RBAC",
	Governance = "Governance",
	RoutingRules = "RoutingRules",
	PIIRedactor = "PIIRedactor",
	PromptRepository = "PromptRepository",
	PromptDeploymentStrategy = "PromptDeploymentStrategy",
	AccessProfiles = "AccessProfiles",
}

export enum RbacOperation {
	Read = "Read",
	View = "View",
	Create = "Create",
	Update = "Update",
	Delete = "Delete",
	Download = "Download",
}

interface RbacContextType {
	isAllowed: (resource: RbacResource, operation: RbacOperation) => boolean;
	permissions: Record<string, Record<string, boolean>>;
	isLoading: boolean;
	refetch: () => void;
}

const RbacContext = createContext<RbacContextType | null>(null);

function loadPermissions(): Record<string, Record<string, boolean>> {
	try {
		const stored = localStorage.getItem("bifrost_rbac_permissions");
		if (stored) return JSON.parse(stored);
	} catch {}
	return {};
}

export function RbacProvider({ children }: { children: React.ReactNode }) {
	const [permissions, setPermissions] = useState<Record<string, Record<string, boolean>>>(loadPermissions);

	const refetch = useCallback(() => {
		setPermissions(loadPermissions());
	}, []);

	const isAllowed = useCallback(
		(resource: RbacResource, operation: RbacOperation): boolean => {
			if (permissions[resource]?.[operation]) return true;
			if (permissions["*"]?.[operation]) return true;
			if (permissions[resource]?.["*"]) return true;
			return Object.keys(permissions).length === 0;
		},
		[permissions],
	);

	return (
		<RbacContext.Provider value={{ isAllowed, permissions, isLoading: false, refetch }}>
			{children}
		</RbacContext.Provider>
	);
}

export function useRbac(resource: RbacResource, operation: RbacOperation): boolean {
	const context = useContext(RbacContext);
	if (!context) return true;
	return context.isAllowed(resource, operation);
}

export function useRbacContext() {
	const context = useContext(RbacContext);
	if (!context) return { isAllowed: () => true, permissions: {}, isLoading: false, refetch: () => {} };
	return context;
}
