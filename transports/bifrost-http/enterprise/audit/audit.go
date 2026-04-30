package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "audit"

// EventType represents the type of audit event
type EventType string

const (
	EventAuth          EventType = "auth"
	EventAuthz         EventType = "authz"
	EventConfigChange  EventType = "config_change"
	EventDataAccess    EventType = "data_access"
	EventSecurity      EventType = "security"
	EventInference     EventType = "inference"
	EventAdmin         EventType = "admin"
)

// AuditLogEntry represents a single audit log entry
type AuditLogEntry struct {
	ID        string            `json:"id"`
	Timestamp int64             `json:"timestamp"`
	EventType EventType         `json:"event_type"`
	Action    string            `json:"action"`
	UserID    string            `json:"user_id,omitempty"`
	UserName  string            `json:"user_name,omitempty"`
	Resource  string            `json:"resource,omitempty"`
	ResourceID string           `json:"resource_id,omitempty"`
	Status    string            `json:"status"`
	Details   map[string]any    `json:"details,omitempty"`
	ClientIP  string            `json:"client_ip,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	HMAC      string            `json:"hmac"`
}

// Config for audit plugin
type Config struct {
	SecretKey       string   `json:"secret_key,omitempty"`       // HMAC signing key
	BufferSize      int      `json:"buffer_size,omitempty"`      // Internal buffer size
	ExportWebhook   string   `json:"export_webhook,omitempty"`   // Optional webhook for export
	RetentionDays   int      `json:"retention_days,omitempty"`   // Days to retain logs
}

// AuditPlugin implements immutable audit logging with HMAC signatures
type AuditPlugin struct {
	mu         sync.RWMutex
	entries    []AuditLogEntry
	secretKey  []byte
	bufferSize int
	hook       string
	logger     schemas.Logger
}

// Init initializes the audit plugin
func Init(config any, logger schemas.Logger) *AuditPlugin {
	p := &AuditPlugin{
		entries:    make([]AuditLogEntry, 0, 1000),
		bufferSize: 10000,
		secretKey:  []byte("bifrost-default-audit-key"),
		logger:     logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			if cfg.SecretKey != "" {
				p.secretKey = []byte(cfg.SecretKey)
			}
			if cfg.BufferSize > 0 {
				p.bufferSize = cfg.BufferSize
			}
			p.hook = cfg.ExportWebhook
		}
	}

	return p
}

func (p *AuditPlugin) GetName() string { return PluginName }
func (p *AuditPlugin) Cleanup() error  { return nil }

// Log records an audit event
func (p *AuditPlugin) Log(entry AuditLogEntry) {
	entry.Timestamp = time.Now().UnixMilli()
	entry.ID = fmt.Sprintf("%d-%s", entry.Timestamp, hashUserID(entry.UserID))
	entry.HMAC = p.sign(entry)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.entries = append(p.entries, entry)
	if len(p.entries) > p.bufferSize {
		p.entries = p.entries[len(p.entries)-p.bufferSize:]
	}
}

// sign generates an HMAC signature for the audit entry
func (p *AuditPlugin) sign(entry AuditLogEntry) string {
	data, _ := json.Marshal(map[string]any{
		"id":         entry.ID,
		"timestamp":  entry.Timestamp,
		"event_type": entry.EventType,
		"action":     entry.Action,
		"user_id":    entry.UserID,
		"resource":   entry.Resource,
		"resource_id": entry.ResourceID,
		"status":     entry.Status,
	})
	mac := hmac.New(sha256.New, p.secretKey)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks if the HMAC signature is valid for an entry
func (p *AuditPlugin) Verify(entry AuditLogEntry) bool {
	expected := p.sign(entry)
	return hmac.Equal([]byte(entry.HMAC), []byte(expected))
}

// Query retrieves audit logs with filters
func (p *AuditPlugin) Query(eventType *EventType, userID *string, resource *string, limit int) []AuditLogEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit <= 0 || limit > len(p.entries) {
		limit = len(p.entries)
	}

	results := make([]AuditLogEntry, 0, limit)
	for i := len(p.entries) - 1; i >= 0 && len(results) < limit; i-- {
		e := p.entries[i]
		if eventType != nil && e.EventType != *eventType {
			continue
		}
		if userID != nil && e.UserID != *userID {
			continue
		}
		if resource != nil && e.Resource != *resource {
			continue
		}
		results = append(results, e)
	}
	return results
}

// HTTPTransportPostHook intercepts responses and logs access events
func (p *AuditPlugin) HTTPTransportPostHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest, resp *schemas.HTTPResponse) error {
	userID, _ := ctx.Value(schemas.BifrostContextKeyUserID).(string)
	userName, _ := ctx.Value(schemas.BifrostContextKeyUserName).(string)

	var eventType EventType
	var action string

	if userID != "" {
		eventType = EventAuthz
		action = req.Method + " " + req.Path
	} else {
		eventType = EventDataAccess
		action = req.Method + " " + req.Path
	}

	status := "success"
	if resp != nil && resp.StatusCode >= 400 {
		status = "failure"
	}

	p.Log(AuditLogEntry{
		EventType: eventType,
		Action:    action,
		UserID:    userID,
		UserName:  userName,
		Resource:  req.Path,
		Status:    status,
		Details: map[string]any{
			"method":      req.Method,
			"status_code": resp.StatusCode,
			"path":        req.Path,
		},
	})

	return nil
}

// HTTPTransportPreHook is a no-op for audit
func (p *AuditPlugin) HTTPTransportPreHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest) (*schemas.HTTPResponse, error) {
	return nil, nil
}

func hashUserID(userID string) string {
	h := sha256.New()
	h.Write([]byte(userID))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (p *AuditPlugin) HTTPTransportStreamChunkHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest, chunk *schemas.BifrostStreamChunk) (*schemas.BifrostStreamChunk, error) {
	return chunk, nil
}

var _ schemas.HTTPTransportPlugin = (*AuditPlugin)(nil)
var _ schemas.BasePlugin = (*AuditPlugin)(nil)
