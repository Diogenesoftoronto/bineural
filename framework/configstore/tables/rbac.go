package tables

import "time"

// TableRBACRole represents a role definition in the RBAC system.
type TableRBACRole struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	IsSystem    bool   `gorm:"default:false;not null" json:"is_system"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableRBACRole) TableName() string { return "governance_rbac_roles" }

// TableRBACPermission represents a permission (resource+action pair) in the RBAC system.
type TableRBACPermission struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Resource    string `gorm:"type:varchar(255);not null;uniqueIndex:idx_resource_action" json:"resource"`
	Action      string `gorm:"type:varchar(255);not null;uniqueIndex:idx_resource_action" json:"action"`
	Description string `gorm:"type:text" json:"description"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableRBACPermission) TableName() string { return "governance_rbac_permissions" }

// TableRBACRolePermission is the join table linking roles to permissions.
type TableRBACRolePermission struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	RoleID       uint `gorm:"uniqueIndex:idx_role_permission;not null" json:"role_id"`
	PermissionID uint `gorm:"uniqueIndex:idx_role_permission;not null" json:"permission_id"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableRBACRolePermission) TableName() string { return "governance_rbac_role_permissions" }

// TableRBACRoleAssignment maps a user to a role, optionally scoped to a team or customer.
type TableRBACRoleAssignment struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     string `gorm:"type:varchar(255);not null;index:idx_user_role" json:"user_id"`
	RoleID     uint   `gorm:"not null;index:idx_user_role" json:"role_id"`
	TeamID     string `gorm:"type:varchar(255);index" json:"team_id,omitempty"`
	CustomerID string `gorm:"type:varchar(255);index" json:"customer_id,omitempty"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableRBACRoleAssignment) TableName() string { return "governance_rbac_role_assignments" }
