package tables

import "time"

// TableGuardrailRule represents a guardrail rule definition.
type TableGuardrailRule struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(255);not null;uniqueIndex:idx_guardrail_rule_name" json:"name"`
	Type        string `gorm:"type:varchar(50);not null" json:"type"` // blocklist, allowlist, regex, pii, content, rate_limit
	Pattern     string `gorm:"type:text" json:"pattern,omitempty"`
	Patterns    string `gorm:"type:text" json:"patterns,omitempty"`    // JSON-encoded []string
	ContentType string `gorm:"type:varchar(20)" json:"content_type,omitempty"` // input, output, both
	Action      string `gorm:"type:varchar(20);not null" json:"action"` // block, log, mask, flag
	Message     string `gorm:"type:text" json:"message,omitempty"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableGuardrailRule) TableName() string { return "governance_guardrail_rules" }

// TableGuardrailProfile groups guardrail rules together.
type TableGuardrailProfile struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(255);not null;uniqueIndex:idx_guardrail_profile_name" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableGuardrailProfile) TableName() string { return "governance_guardrail_profiles" }

// TableGuardrailProfileRule is the join table linking profiles to rules.
type TableGuardrailProfileRule struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	ProfileID uint `gorm:"uniqueIndex:idx_profile_rule;not null;index:idx_guardrail_profile_id" json:"profile_id"`
	RuleID    uint `gorm:"uniqueIndex:idx_profile_rule;not null;index:idx_guardrail_rule_id" json:"rule_id"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

func (TableGuardrailProfileRule) TableName() string { return "governance_guardrail_profile_rules" }
