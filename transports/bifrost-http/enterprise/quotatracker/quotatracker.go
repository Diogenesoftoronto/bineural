// Package quotatracker provides a multi-level quota tracking system for Bifrost.
// It tracks quotas at provider, model, virtual-key, team, and user levels,
// emitting utilization status and enforcing limits via PreLLMHook.
// Pattern inspired by Plexus's meter-based quota checker.
package quotatracker

import (
	"fmt"
	"sync"
	"time"

	bifrost "github.com/maximhq/bifrost/core"
	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "quota_tracker"

// MeterKind classifies the type of quota meter.
type MeterKind string

const (
	MeterKindBalance   MeterKind = "balance"   // Spend-based (e.g., monthly budget remaining)
	MeterKindAllowance MeterKind = "allowance" // Rate-limit window (e.g., daily/weekly/rolling token cap)
)

// MeterStatus represents the health of a quota meter.
type MeterStatus string

const (
	MeterStatusOk        MeterStatus = "ok"
	MeterStatusWarning   MeterStatus = "warning"   // >80% utilized
	MeterStatusCritical  MeterStatus = "critical"  // >95% utilized
	MeterStatusExhausted MeterStatus = "exhausted" // 100% utilized
)

// QuotaMeter represents a single tracked quota dimension.
// Modeled after Plexus's Meter type with balance and allowance kinds.
type QuotaMeter struct {
	Key               string     `json:"key"`
	Label             string     `json:"label"`
	Kind              MeterKind  `json:"kind"`
	Unit              string     `json:"unit"` // "tokens", "requests", "usd"
	Limit             float64   `json:"limit,omitempty"`
	Used              float64   `json:"used,omitempty"`
	Remaining         float64   `json:"remaining,omitempty"`
	UtilizationPercent float64   `json:"utilization_percent"`
	Status            MeterStatus `json:"status"`
	// Allowance-specific fields
	PeriodValue  *int    `json:"period_value,omitempty"`  // e.g., 1, 7, 30
	PeriodUnit   *string `json:"period_unit,omitempty"`   // "day", "hour", "minute"
	PeriodCycle  *string `json:"period_cycle,omitempty"`  // "rolling", "daily", "weekly", "monthly"
	ResetsAt     *string `json:"resets_at,omitempty"`
	// Scope fields
	Scope     string `json:"scope"`      // "provider", "model", "virtual_key", "team", "user"
	ScopeID   string `json:"scope_id"`   // ID of the scoped entity
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
}

// QuotaAlertConfig defines alert thresholds for a quota scope.
type QuotaAlertConfig struct {
	Scope          string  `json:"scope"`
	ScopeID        string  `json:"scope_id"`
	WarningThreshold  float64 `json:"warning_threshold"`  // Default: 0.8 (80%)
	CriticalThreshold float64 `json:"critical_threshold"` // Default: 0.95 (95%)
	WebhookURL     *string `json:"webhook_url,omitempty"`
}

// QuotaAlert represents a triggered quota alert.
type QuotaAlert struct {
	Meter      *QuotaMeter     `json:"meter"`
	Threshold  float64         `json:"threshold"`
	TriggeredAt time.Time      `json:"triggered_at"`
	PreviousStatus MeterStatus `json:"previous_status"`
}

// AlertCallback is called when a quota meter crosses a threshold.
type AlertCallback func(alert *QuotaAlert)

// Config holds the quota tracker configuration.
type Config struct {
	Enabled           bool              `json:"enabled"`
	WarningThreshold  float64           `json:"warning_threshold,omitempty"`  // Default: 0.8
	CriticalThreshold float64           `json:"critical_threshold,omitempty"` // Default: 0.95
	AlertCallbacks    []AlertCallback   `json:"-"`
}

// QuotaTrackerPlugin implements an LLMPlugin that tracks quotas across multiple scopes.
type QuotaTrackerPlugin struct {
	mu       sync.RWMutex
	meters   map[string]*QuotaMeter   // key: "scope:scope_id:meter_key"
	alerts   map[string]*QuotaAlertConfig
	logger   schemas.Logger
	enabled  bool
	warningThreshold  float64
	criticalThreshold float64
	alertCallbacks []AlertCallback
}

// Init creates a new QuotaTrackerPlugin.
func Init(config any, logger schemas.Logger) *QuotaTrackerPlugin {
	p := &QuotaTrackerPlugin{
		meters:           make(map[string]*QuotaMeter),
		alerts:           make(map[string]*QuotaAlertConfig),
		logger:           logger,
		warningThreshold: 0.8,
		criticalThreshold: 0.95,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			if cfg.WarningThreshold > 0 {
				p.warningThreshold = cfg.WarningThreshold
			}
			if cfg.CriticalThreshold > 0 {
				p.criticalThreshold = cfg.CriticalThreshold
			}
			p.alertCallbacks = cfg.AlertCallbacks
		}
	}

	return p
}

func (p *QuotaTrackerPlugin) GetName() string { return PluginName }
func (p *QuotaTrackerPlugin) Cleanup() error  { return nil }

func (p *QuotaTrackerPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// buildMeterKey generates a unique key for a meter in the map.
func buildMeterKey(scope, scopeID, mKey string) string {
	return fmt.Sprintf("%s:%s:%s", scope, scopeID, mKey)
}

// RegisterMeter adds or updates a quota meter.
func (p *QuotaTrackerPlugin) RegisterMeter(meter *QuotaMeter) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := buildMeterKey(meter.Scope, meter.ScopeID, meter.Key)
	p.meters[key] = meter
}

// RecordUsage increments a meter's used value and updates its status.
func (p *QuotaTrackerPlugin) RecordUsage(scope, scopeID, mKey string, amount float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := buildMeterKey(scope, scopeID, mKey)
	meter, ok := p.meters[key]
	if !ok {
		return
	}

	previousStatus := meter.Status
	meter.Used += amount
	if meter.Limit > 0 {
		meter.Remaining = meter.Limit - meter.Used
		if meter.Remaining < 0 {
			meter.Remaining = 0
		}
		meter.UtilizationPercent = (meter.Used / meter.Limit) * 100
		meter.Status = computeStatus(meter.UtilizationPercent, p.warningThreshold*100, p.criticalThreshold*100)
	}

	// Fire alerts if status changed
	if meter.Status != previousStatus {
		p.fireAlert(&QuotaAlert{
			Meter:         meter,
			Threshold:     getThresholdForStatus(meter.Status, p.warningThreshold, p.criticalThreshold),
			TriggeredAt:   time.Now(),
			PreviousStatus: previousStatus,
		})
	}
}

// CheckQuota checks if a request is allowed under the current quota.
// Returns true if allowed, false if quota is exhausted.
func (p *QuotaTrackerPlugin) CheckQuota(scope, scopeID, mKey string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	key := buildMeterKey(scope, scopeID, mKey)
	meter, ok := p.meters[key]
	if !ok {
		return true // No meter registered = no limit
	}
	return meter.Status != MeterStatusExhausted
}

// GetMeters returns all registered meters, optionally filtered by scope.
func (p *QuotaTrackerPlugin) GetMeters(scope string) []*QuotaMeter {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*QuotaMeter, 0, len(p.meters))
	for _, meter := range p.meters {
		if scope == "" || meter.Scope == scope {
			result = append(result, meter)
		}
	}
	return result
}

// GetMeter returns a specific meter by scope, scope ID, and meter key.
func (p *QuotaTrackerPlugin) GetMeter(scope, scopeID, mKey string) *QuotaMeter {
	p.mu.RLock()
	defer p.mu.RUnlock()

	key := buildMeterKey(scope, scopeID, mKey)
	return p.meters[key]
}

// RegisterAlertConfig registers alert configuration for a quota scope.
func (p *QuotaTrackerPlugin) RegisterAlertConfig(config *QuotaAlertConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := buildMeterKey(config.Scope, config.ScopeID, "alert")
	p.alerts[key] = config
}

// RegisterAlertCallback adds a callback that fires when a quota threshold is crossed.
func (p *QuotaTrackerPlugin) RegisterAlertCallback(cb AlertCallback) {
	p.alertCallbacks = append(p.alertCallbacks, cb)
}

func (p *QuotaTrackerPlugin) fireAlert(alert *QuotaAlert) {
	// Call registered callbacks (non-blocking)
	for _, cb := range p.alertCallbacks {
		go func(callback AlertCallback) {
			defer func() {
				if r := recover(); r != nil {
					p.logger.Warn("quota alert callback panicked: %v", r)
				}
			}()
			callback(alert)
		}(cb)
	}
}

// PreLLMHook checks quotas before an LLM request. Short-circuits if exhausted.
func (p *QuotaTrackerPlugin) PreLLMHook(ctx *schemas.BifrostContext, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.LLMPluginShortCircuit, error) {
	if !p.enabled {
		return req, nil, nil
	}

	// Resolve provider and model from the request
	var provider, model string
	switch {
	case req.ChatRequest != nil:
		provider = string(req.ChatRequest.Provider)
		model = req.ChatRequest.Model
	case req.TextCompletionRequest != nil:
		provider = string(req.TextCompletionRequest.Provider)
		model = req.TextCompletionRequest.Model
	case req.ResponsesRequest != nil:
		provider = string(req.ResponsesRequest.Provider)
		model = req.ResponsesRequest.Model
	case req.EmbeddingRequest != nil:
		provider = string(req.EmbeddingRequest.Provider)
		model = req.EmbeddingRequest.Model
	}

	// Check provider-level quota
	if !p.CheckQuota("provider", provider, "balance") {
		return req, &schemas.LLMPluginShortCircuit{
			Error: &schemas.BifrostError{
				StatusCode:    func() *int { v := 429; return &v }(),
				IsBifrostError: true,
				Error: &schemas.ErrorField{
					Message: fmt.Sprintf("Provider %s spend quota exhausted", provider),
				},
				ExtraFields: schemas.BifrostErrorExtraFields{
					Provider:    schemas.ModelProvider(provider),
					RequestType: req.RequestType,
				},
			},
		}, nil
	}

	// Check model-level token allowance
	if !p.CheckQuota("model", provider+"/"+model, "allowance") {
		return req, &schemas.LLMPluginShortCircuit{
			Error: &schemas.BifrostError{
				StatusCode:    func() *int { v := 429; return &v }(),
				IsBifrostError: true,
				Error: &schemas.ErrorField{
					Message: fmt.Sprintf("Model %s/%s token allowance exhausted", provider, model),
				},
				ExtraFields: schemas.BifrostErrorExtraFields{
					Provider:    schemas.ModelProvider(provider),
					RequestType: req.RequestType,
				},
			},
		}, nil
	}

	return req, nil, nil
}

// PostLLMHook records usage against all relevant quota meters after a request.
func (p *QuotaTrackerPlugin) PostLLMHook(ctx *schemas.BifrostContext, result *schemas.BifrostResponse, bifrostErr *schemas.BifrostError) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
	if !p.enabled {
		return result, bifrostErr, nil
	}

	// Only track successful requests for quota purposes
	if bifrostErr != nil || result == nil {
		return result, bifrostErr, nil
	}

	var inputTokens, outputTokens int
	var cost float64

	switch {
	case result.ChatResponse != nil && result.ChatResponse.Usage != nil:
		inputTokens = result.ChatResponse.Usage.PromptTokens
		outputTokens = result.ChatResponse.Usage.CompletionTokens
		if result.ChatResponse.Usage.Cost != nil {
			cost = result.ChatResponse.Usage.Cost.TotalCost
		}
	case result.TextCompletionResponse != nil && result.TextCompletionResponse.Usage != nil:
		inputTokens = result.TextCompletionResponse.Usage.PromptTokens
		outputTokens = result.TextCompletionResponse.Usage.CompletionTokens
	case result.ResponsesResponse != nil && result.ResponsesResponse.Usage != nil:
		inputTokens = result.ResponsesResponse.Usage.InputTokens
		outputTokens = result.ResponsesResponse.Usage.OutputTokens
	case result.ResponsesStreamResponse != nil && result.ResponsesStreamResponse.Response != nil && result.ResponsesStreamResponse.Response.Usage != nil:
		inputTokens = result.ResponsesStreamResponse.Response.Usage.InputTokens
		outputTokens = result.ResponsesStreamResponse.Response.Usage.OutputTokens
	}

	totalTokens := float64(inputTokens + outputTokens)
	_, provider, _, _ := bifrost.GetResponseFields(result, bifrostErr)
	model := ""
	if result != nil {
		switch {
		case result.ChatResponse != nil:
			model = result.ChatResponse.Model
		case result.TextCompletionResponse != nil:
			model = result.TextCompletionResponse.Model
		case result.ResponsesResponse != nil:
			model = result.ResponsesResponse.Model
		case result.ResponsesStreamResponse != nil:
			model = result.ResponsesStreamResponse.Response.Model
		}
	}

	// Record usage at each scope level
	p.RecordUsage("provider", string(provider), "balance", cost)
	p.RecordUsage("provider", string(provider), "allowance", totalTokens)
	p.RecordUsage("model", string(provider)+"/"+model, "allowance", totalTokens)

	// Virtual key level (if VK context available)
	if vkID := bifrost.GetStringFromContext(ctx, schemas.BifrostContextKeyGovernanceVirtualKeyID); vkID != "" {
		p.RecordUsage("virtual_key", vkID, "balance", cost)
		p.RecordUsage("virtual_key", vkID, "allowance", totalTokens)
	}

	// Team level
	if teamID := bifrost.GetStringFromContext(ctx, schemas.BifrostContextKeyGovernanceTeamID); teamID != "" {
		p.RecordUsage("team", teamID, "balance", cost)
	}

	// User level (enterprise)
	if userID := bifrost.GetStringFromContext(ctx, schemas.BifrostContextKeyGovernanceUserID); userID != "" {
		p.RecordUsage("user", userID, "balance", cost)
		p.RecordUsage("user", userID, "allowance", totalTokens)
	}

	return result, bifrostErr, nil
}

// computeStatus determines meter status from utilization percentage.
func computeStatus(utilization, warningPct, criticalPct float64) MeterStatus {
	if utilization >= criticalPct {
		return MeterStatusCritical
	}
	if utilization >= warningPct {
		return MeterStatusWarning
	}
	return MeterStatusOk
}

// getThresholdForStatus returns the threshold value that triggered a status change.
func getThresholdForStatus(status MeterStatus, warning, critical float64) float64 {
	switch status {
	case MeterStatusCritical:
		return critical
	case MeterStatusWarning:
		return warning
	default:
		return 0
	}
}

var _ schemas.BasePlugin = (*QuotaTrackerPlugin)(nil)
