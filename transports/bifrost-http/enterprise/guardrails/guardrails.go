package guardrails

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "guardrails"

// RuleType represents the type of guardrail rule
type RuleType string

const (
	RuleTypeBlocklist   RuleType = "blocklist"
	RuleTypeAllowlist   RuleType = "allowlist"
	RuleTypeRegex       RuleType = "regex"
	RuleTypePII         RuleType = "pii"
	RuleTypeContent     RuleType = "content"
	RuleTypeRateLimit   RuleType = "rate_limit"
)

// RuleAction represents what to do when a rule matches
type RuleAction string

const (
	ActionBlock   RuleAction = "block"
	ActionLog     RuleAction = "log"
	ActionMask    RuleAction = "mask"
	ActionFlag    RuleAction = "flag"
)

// Rule defines a guardrail rule
type Rule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        RuleType    `json:"type"`
	Pattern     string      `json:"pattern,omitempty"`      // For regex/blocklist
	Patterns    []string    `json:"patterns,omitempty"`     // For blocklist multiple
	ContentType string      `json:"content_type,omitempty"` // "input", "output", "both"
	Action      RuleAction  `json:"action"`
	Message     string      `json:"message,omitempty"`
	Enabled     bool        `json:"enabled"`
	compiled    *regexp.Regexp
}

// Profile groups rules together
type Profile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	RuleIDs     []string `json:"rule_ids"`
	Enabled     bool     `json:"enabled"`
}

// Config for guardrails plugin
type Config struct {
	Rules    []Rule    `json:"rules,omitempty"`
	Profiles []Profile `json:"profiles,omitempty"`
	Enabled  bool      `json:"enabled"`
}

// Violation represents a triggered guardrail
type Violation struct {
	RuleID  string     `json:"rule_id"`
	RuleName string    `json:"rule_name"`
	Type    RuleType   `json:"type"`
	Action  RuleAction `json:"action"`
	Message string     `json:"message,omitempty"`
	Matched string     `json:"matched,omitempty"`
}

// GuardrailResult is the outcome of evaluation
type GuardrailResult struct {
	Allowed    bool        `json:"allowed"`
	Violations []Violation `json:"violations,omitempty"`
	MaskedText string      `json:"masked_text,omitempty"`
}

// GuardrailsPlugin implements content safety guardrails
type GuardrailsPlugin struct {
	mu        sync.RWMutex
	rules     map[string]*Rule
	profiles  map[string]*Profile
	enabled   bool
	logger    schemas.Logger
}

// Init initializes the guardrails plugin
func Init(config any, logger schemas.Logger) *GuardrailsPlugin {
	p := &GuardrailsPlugin{
		rules:    make(map[string]*Rule),
		profiles: make(map[string]*Profile),
		logger:   logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			for i := range cfg.Rules {
				rule := &cfg.Rules[i]
				p.AddRule(rule)
			}
			for i := range cfg.Profiles {
				prof := &cfg.Profiles[i]
				p.profiles[prof.ID] = prof
			}
		}
	}

	return p
}

func (p *GuardrailsPlugin) GetName() string { return PluginName }
func (p *GuardrailsPlugin) Cleanup() error  { return nil }

// AddRule adds a compiled rule
func (p *GuardrailsPlugin) AddRule(rule *Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	if rule.Type == RuleTypeRegex && rule.Pattern != "" {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		rule.compiled = re
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.rules[rule.ID] = rule
	return nil
}

// Evaluate checks content against all enabled rules
func (p *GuardrailsPlugin) Evaluate(content string, contentType string) GuardrailResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := GuardrailResult{Allowed: true}
	if !p.enabled {
		return result
	}

	for _, rule := range p.rules {
		if !rule.Enabled {
			continue
		}
		if rule.ContentType != "" && rule.ContentType != "both" && rule.ContentType != contentType {
			continue
		}

		var matched string
		var hit bool

		switch rule.Type {
		case RuleTypeBlocklist:
			for _, pattern := range rule.Patterns {
				if strings.Contains(strings.ToLower(content), strings.ToLower(pattern)) {
					hit = true
					matched = pattern
					break
				}
			}
		case RuleTypeRegex:
			if rule.compiled != nil && rule.compiled.MatchString(content) {
				hit = true
				matched = rule.Pattern
			}
		case RuleTypePII:
			// Basic PII patterns (emails, SSNs, phone numbers)
			if matchedPII := detectPII(content); matchedPII != "" {
				hit = true
				matched = matchedPII
			}
		case RuleTypeContent:
			if rule.Pattern != "" && strings.Contains(content, rule.Pattern) {
				hit = true
				matched = rule.Pattern
			}
		}

		if hit {
			v := Violation{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Type:     rule.Type,
				Action:   rule.Action,
				Message:  rule.Message,
				Matched:  matched,
			}
			result.Violations = append(result.Violations, v)

			if rule.Action == ActionBlock {
				result.Allowed = false
			}
			if rule.Action == ActionMask {
				result.MaskedText = maskContent(content, matched)
			}
		}
	}

	return result
}

// PreLLMHook evaluates guardrails on request before LLM call
func (p *GuardrailsPlugin) PreLLMHook(ctx *schemas.BifrostContext, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.LLMPluginShortCircuit, error) {
	if !p.enabled {
		return req, nil, nil
	}

	var content string
	if req.ChatRequest != nil && len(req.ChatRequest.Input) > 0 {
		for _, msg := range req.ChatRequest.Input {
			if msg.Content != nil && msg.Content.ContentStr != nil {
				content += *msg.Content.ContentStr + " "
			}
		}
	} else if req.TextCompletionRequest != nil && req.TextCompletionRequest.Input != nil && req.TextCompletionRequest.Input.PromptStr != nil {
		content = *req.TextCompletionRequest.Input.PromptStr
	}

	if content == "" {
		return req, nil, nil
	}

	result := p.Evaluate(content, "input")
	if !result.Allowed {
		msg := "Content blocked by guardrails"
		if len(result.Violations) > 0 && result.Violations[0].Message != "" {
			msg = result.Violations[0].Message
		}
		return nil, &schemas.LLMPluginShortCircuit{
			Error: &schemas.BifrostError{
				IsBifrostError: true,
				StatusCode:     &[]int{400}[0],
				Error: &schemas.ErrorField{
					Message: msg,
					Type:    &[]string{"guardrails_violation"}[0],
				},
			},
		}, nil
	}

	return req, nil, nil
}

// PostLLMHook evaluates guardrails on response after LLM call
func (p *GuardrailsPlugin) PostLLMHook(ctx *schemas.BifrostContext, resp *schemas.BifrostResponse, bifrostErr *schemas.BifrostError) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
	if !p.enabled || bifrostErr != nil {
		return resp, bifrostErr, nil
	}

	var content string
	if resp != nil && resp.ChatResponse != nil && len(resp.ChatResponse.Choices) > 0 && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice != nil && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message != nil && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message.Content != nil && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message.Content.ContentStr != nil {
		content = *resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message.Content.ContentStr
	} else if resp != nil && resp.TextCompletionResponse != nil && len(resp.TextCompletionResponse.Choices) > 0 && resp.TextCompletionResponse.Choices[0].TextCompletionResponseChoice != nil && resp.TextCompletionResponse.Choices[0].TextCompletionResponseChoice.Text != nil {
		content = *resp.TextCompletionResponse.Choices[0].TextCompletionResponseChoice.Text
	}

	if content == "" {
		return resp, bifrostErr, nil
	}

	result := p.Evaluate(content, "output")
	if !result.Allowed {
		msg := "Content blocked by guardrails"
		if len(result.Violations) > 0 && result.Violations[0].Message != "" {
			msg = result.Violations[0].Message
		}
		return nil, &schemas.BifrostError{
			IsBifrostError: true,
			StatusCode:     &[]int{400}[0],
			Error: &schemas.ErrorField{
				Message: msg,
				Type:    &[]string{"guardrails_violation"}[0],
			},
		}, nil
	}

	return resp, bifrostErr, nil
}

func detectPII(text string) string {
	// Simple PII patterns
	patterns := map[string]*regexp.Regexp{
		"email":    regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		"ssn":      regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		"phone":    regexp.MustCompile(`\b\d{3}-\d{3}-\d{4}\b`),
		"credit":   regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
	}

	for name, re := range patterns {
		if re.MatchString(text) {
			return name
		}
	}
	return ""
}

func maskContent(content, pattern string) string {
	return strings.ReplaceAll(content, pattern, "[REDACTED]")
}

var _ schemas.LLMPlugin = (*GuardrailsPlugin)(nil)
var _ schemas.BasePlugin = (*GuardrailsPlugin)(nil)
