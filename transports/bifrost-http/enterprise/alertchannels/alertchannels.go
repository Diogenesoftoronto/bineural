package alertchannels

import (
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "alert_channels"

type ChannelType string

const (
	ChannelTypeWebhook ChannelType = "webhook"
	ChannelTypeSlack   ChannelType = "slack"
	ChannelTypeEmail   ChannelType = "email"
	ChannelTypePagerDuty ChannelType = "pagerduty"
)

type AlertChannel struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Type      ChannelType `json:"type"`
	Config    map[string]string `json:"config,omitempty"`
	Enabled   bool        `json:"enabled"`
	CreatedAt time.Time   `json:"created_at"`
}

type AlertRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Condition   string   `json:"condition"`
	Threshold   float64  `json:"threshold,omitempty"`
	ChannelIDs  []string `json:"channel_ids"`
	Enabled     bool     `json:"enabled"`
}

type AlertEvent struct {
	RuleID    string    `json:"rule_id"`
	ChannelID string    `json:"channel_id"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}

type Config struct {
	Enabled  bool `json:"enabled"`
}

type AlertChannelsPlugin struct {
	mu       sync.RWMutex
	channels map[string]*AlertChannel
	rules    map[string]*AlertRule
	history  []AlertEvent
	enabled  bool
	logger   schemas.Logger
}

func Init(config any, logger schemas.Logger) *AlertChannelsPlugin {
	p := &AlertChannelsPlugin{
		channels: make(map[string]*AlertChannel),
		rules:    make(map[string]*AlertRule),
		history:  make([]AlertEvent, 0),
		logger:   logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
		}
	}

	return p
}

func (p *AlertChannelsPlugin) GetName() string { return PluginName }
func (p *AlertChannelsPlugin) Cleanup() error  { return nil }

func (p *AlertChannelsPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *AlertChannelsPlugin) CreateChannel(ch *AlertChannel) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ch.ID == "" {
		ch.ID = fmt.Sprintf("ch-%d", time.Now().UnixNano())
	}
	ch.CreatedAt = time.Now()
	p.channels[ch.ID] = ch
	return nil
}

func (p *AlertChannelsPlugin) GetChannel(id string) (*AlertChannel, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ch, ok := p.channels[id]
	return ch, ok
}

func (p *AlertChannelsPlugin) ListChannels() []*AlertChannel {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*AlertChannel, 0, len(p.channels))
	for _, ch := range p.channels {
		result = append(result, ch)
	}
	return result
}

func (p *AlertChannelsPlugin) DeleteChannel(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.channels[id]; !ok {
		return fmt.Errorf("channel not found: %s", id)
	}
	delete(p.channels, id)
	return nil
}

func (p *AlertChannelsPlugin) CreateRule(rule *AlertRule) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}
	p.rules[rule.ID] = rule
	return nil
}

func (p *AlertChannelsPlugin) GetRule(id string) (*AlertRule, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	r, ok := p.rules[id]
	return r, ok
}

func (p *AlertChannelsPlugin) ListRules() []*AlertRule {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*AlertRule, 0, len(p.rules))
	for _, r := range p.rules {
		result = append(result, r)
	}
	return result
}

func (p *AlertChannelsPlugin) FireAlert(ruleID, message, severity string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rule, ok := p.rules[ruleID]
	if !ok || !rule.Enabled {
		return
	}

	for _, chID := range rule.ChannelIDs {
		if ch, ok := p.channels[chID]; ok && ch.Enabled {
			p.history = append(p.history, AlertEvent{
				RuleID:    ruleID,
				ChannelID: chID,
				Message:   message,
				Severity:  severity,
				Timestamp: time.Now(),
			})
		}
	}
}

func (p *AlertChannelsPlugin) GetHistory(limit int) []AlertEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit > len(p.history) {
		limit = len(p.history)
	}
	result := make([]AlertEvent, limit)
	copy(result, p.history[len(p.history)-limit:])
	return result
}

var _ schemas.BasePlugin = (*AlertChannelsPlugin)(nil)
