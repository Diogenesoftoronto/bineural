package tables

import "time"

// TableProviderSubscription tracks external provider subscription plans and spend.
// Links a provider+key to a subscription plan with a monthly spend limit.
type TableProviderSubscription struct {
	ID           string  `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Provider     string  `gorm:"type:varchar(255);index;not null" json:"provider"`
	KeyID        *string `gorm:"type:varchar(255);index" json:"key_id,omitempty"`
	PlanName     string  `gorm:"type:varchar(255);not null" json:"plan_name"`
	MonthlyLimit float64 `gorm:"not null" json:"monthly_limit"`              // Monthly spend limit in USD
	CurrentSpend float64 `gorm:"default:0" json:"current_spend"`             // Current month spend in USD
	ResetDate    string  `gorm:"type:varchar(50);not null" json:"reset_date"` // Day of month for spend reset (e.g., "1", "15")
	LastResetAt  time.Time `gorm:"index" json:"last_reset_at"`                // When spend was last reset
	IsActive     bool    `gorm:"default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableProviderSubscription) TableName() string { return "governance_provider_subscriptions" }

// TableSaaSBillingTier defines internal SaaS billing tiers for Bifrost users.
// Each tier has a monthly price, included token allocation, and overage pricing.
type TableSaaSBillingTier struct {
	ID             string  `gorm:"primaryKey;type:varchar(255)" json:"id"`
	TierName       string  `gorm:"type:varchar(255);uniqueIndex;not null" json:"tier_name"`
	MonthlyPrice   float64 `gorm:"not null" json:"monthly_price"`                 // Price in USD
	IncludedTokens int64   `gorm:"not null" json:"included_tokens"`                // Token allocation included
	OverageRate    float64 `gorm:"default:0" json:"overage_rate"`                // Per-million-token overage rate in USD
	MaxRequests    *int64  `json:"max_requests,omitempty"`                        // Optional request cap per month
	Features       string  `gorm:"type:text" json:"-"`                            // JSON serialized map[string]bool
	FeaturesParsed map[string]bool `gorm:"-" json:"features,omitempty"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableSaaSBillingTier) TableName() string { return "governance_saas_billing_tiers" }

// TableUserSubscription links a virtual key or user to a SaaS billing tier.
type TableUserSubscription struct {
	ID           string  `gorm:"primaryKey;type:varchar(255)" json:"id"`
	VirtualKeyID *string `gorm:"type:varchar(255);index" json:"virtual_key_id,omitempty"`
	UserID       *string `gorm:"type:varchar(255);index" json:"user_id,omitempty"`
	TierID       string  `gorm:"type:varchar(255);index;not null" json:"tier_id"`
	CurrentSpend float64 `gorm:"default:0" json:"current_spend"`            // Current month spend in USD
	TokensUsed   int64   `gorm:"default:0" json:"tokens_used"`               // Current month tokens consumed
	RequestsUsed int64   `gorm:"default:0" json:"requests_used"`            // Current month requests consumed
	IsActive     bool    `gorm:"default:true" json:"is_active"`
	StartedAt    time.Time `gorm:"index;not null" json:"started_at"`

	// Relationships
	Tier *TableSaaSBillingTier `gorm:"foreignKey:TierID" json:"tier,omitempty"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableUserSubscription) TableName() string { return "governance_user_subscriptions" }
