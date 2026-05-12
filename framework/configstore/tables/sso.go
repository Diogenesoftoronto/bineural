package tables

import "time"

// TableSSOProvider stores OIDC/SAML identity provider configurations.
type TableSSOProvider struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"type:varchar(255);not null;uniqueIndex:idx_sso_provider_name" json:"name"`
	Type         string `gorm:"type:varchar(20);not null" json:"type"` // oidc, saml
	Issuer       string `gorm:"type:varchar(512)" json:"issuer,omitempty"`
	ClientID     string `gorm:"type:varchar(512)" json:"client_id,omitempty"`
	ClientSecret string `gorm:"type:text" json:"client_secret,omitempty"` // encrypted at rest
	AuthURL      string `gorm:"type:varchar(1024)" json:"auth_url,omitempty"`
	TokenURL     string `gorm:"type:varchar(1024)" json:"token_url,omitempty"`
	UserInfoURL  string `gorm:"type:varchar(1024)" json:"user_info_url,omitempty"`
	RedirectURL  string `gorm:"type:varchar(1024)" json:"redirect_url,omitempty"`
	Scopes       string `gorm:"type:text" json:"scopes,omitempty"` // JSON-encoded []string
	Enabled      bool   `gorm:"default:true;not null" json:"enabled"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"index;not null" json:"updated_at"`
}

func (TableSSOProvider) TableName() string { return "governance_sso_providers" }

// TableSSOSession tracks active SSO authentication sessions.
type TableSSOSession struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	State       string `gorm:"type:varchar(255);not null;uniqueIndex:idx_sso_session_state" json:"state"`
	Provider    string `gorm:"type:varchar(255);not null;index:idx_sso_session_provider" json:"provider"`
	RedirectURL string `gorm:"type:varchar(1024)" json:"redirect_url,omitempty"`
	UserID      string `gorm:"type:varchar(255);index:idx_sso_session_user" json:"user_id,omitempty"`
	UserEmail   string `gorm:"type:varchar(255)" json:"user_email,omitempty"`
	IDToken     string `gorm:"type:text" json:"id_token,omitempty"`
	AccessToken string `gorm:"type:text" json:"access_token,omitempty"`
	ExpiresAt   time.Time `gorm:"index;not null" json:"expires_at"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

func (TableSSOSession) TableName() string { return "governance_sso_sessions" }
