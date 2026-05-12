package piiredactor

import (
	"regexp"
	"sync"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "pii_redactor"

type PIICategory string

const (
	PIIEmail      PIICategory = "email"
	PIISSN        PIICategory = "ssn"
	PIIPhone      PIICategory = "phone"
	PIICreditCard PIICategory = "credit_card"
	PIIIPAddress  PIICategory = "ip_address"
	PIIDate        PIICategory = "date"
	PIICustom      PIICategory = "custom"
)

type RedactionRule struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Category PIICategory `json:"category"`
	Pattern  string      `json:"pattern,omitempty"`
	Action   string      `json:"action"` // "mask", "replace", "remove"
	Replace  string      `json:"replace,omitempty"`
	Enabled  bool        `json:"enabled"`
	compiled *regexp.Regexp
}

type RedactionProfile struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	RuleIDs []string `json:"rule_ids"`
	Enabled bool     `json:"enabled"`
}

type Config struct {
	Rules    []RedactionRule    `json:"rules,omitempty"`
	Profiles []RedactionProfile `json:"profiles,omitempty"`
	Enabled  bool               `json:"enabled"`
}

type RedactionResult struct {
	Original   string             `json:"original"`
	Redacted    string             `json:"redacted"`
	Findings    []RedactionFinding `json:"findings,omitempty"`
	HasRedactions bool            `json:"has_redactions"`
}

type RedactionFinding struct {
	RuleID   string      `json:"rule_id"`
	Category PIICategory `json:"category"`
	Match    string      `json:"match"`
	Position [2]int      `json:"position"`
}

type PIIRedactorPlugin struct {
	mu       sync.RWMutex
	rules    map[string]*RedactionRule
	profiles map[string]*RedactionProfile
	enabled  bool
	logger   schemas.Logger
}

var builtinPatterns = map[PIICategory]*regexp.Regexp{
	PIIEmail:      regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
	PIISSN:        regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	PIIPhone:      regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
	PIICreditCard: regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
	PIIIPAddress:  regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
	PIIDate:       regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),
}

func Init(config any, logger schemas.Logger) *PIIRedactorPlugin {
	p := &PIIRedactorPlugin{
		rules:    make(map[string]*RedactionRule),
		profiles: make(map[string]*RedactionProfile),
		logger:   logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			for i := range cfg.Rules {
				p.addRule(&cfg.Rules[i])
			}
			for i := range cfg.Profiles {
				p.profiles[cfg.Profiles[i].ID] = &cfg.Profiles[i]
			}
		}
	}

	return p
}

func (p *PIIRedactorPlugin) GetName() string { return PluginName }
func (p *PIIRedactorPlugin) Cleanup() error  { return nil }

func (p *PIIRedactorPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *PIIRedactorPlugin) addRule(rule *RedactionRule) error {
	if rule.ID == "" {
		return nil
	}

	if rule.Pattern != "" {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return err
		}
		rule.compiled = re
	} else if builtin, ok := builtinPatterns[rule.Category]; ok {
		rule.compiled = builtin
	}

	p.rules[rule.ID] = rule
	return nil
}

func (p *PIIRedactorPlugin) AddRule(rule *RedactionRule) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.addRule(rule)
}

func (p *PIIRedactorPlugin) Redact(content string, profileID string) RedactionResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := RedactionResult{
		Original: content,
		Redacted: content,
	}

	if !p.enabled {
		return result
	}

	activeRules := make(map[string]*RedactionRule)
	if profileID != "" {
		if prof, ok := p.profiles[profileID]; ok && prof.Enabled {
			for _, rID := range prof.RuleIDs {
				if r, ok := p.rules[rID]; ok && r.Enabled {
					activeRules[rID] = r
				}
			}
		}
	} else {
		for id, r := range p.rules {
			if r.Enabled {
				activeRules[id] = r
			}
		}
	}

	redacted := content
	for _, rule := range activeRules {
		re := rule.compiled
		if re == nil {
			continue
		}

		matches := re.FindAllStringIndex(redacted, -1)
		for i := len(matches) - 1; i >= 0; i-- {
			loc := matches[i]
			matched := redacted[loc[0]:loc[1]]

			result.Findings = append(result.Findings, RedactionFinding{
				RuleID:   rule.ID,
				Category: rule.Category,
				Match:    matched,
				Position: [2]int{loc[0], loc[1]},
			})

			replacement := "[REDACTED]"
			switch rule.Action {
			case "replace":
				if rule.Replace != "" {
					replacement = rule.Replace
				}
			case "remove":
				replacement = ""
			}

			redacted = redacted[:loc[0]] + replacement + redacted[loc[1]:]
		}
	}

	result.Redacted = redacted
	result.HasRedactions = len(result.Findings) > 0
	return result
}

func (p *PIIRedactorPlugin) PreLLMHook(ctx *schemas.BifrostContext, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.LLMPluginShortCircuit, error) {
	if !p.enabled {
		return req, nil, nil
	}

	var content string
	var contentPtr **string

	if req.ChatRequest != nil && len(req.ChatRequest.Input) > 0 {
		for i := range req.ChatRequest.Input {
			msg := &req.ChatRequest.Input[i]
			if msg.Content != nil && msg.Content.ContentStr != nil {
				result := p.Redact(*msg.Content.ContentStr, "")
				if result.HasRedactions {
					*msg.Content.ContentStr = result.Redacted
				}
				content = *msg.Content.ContentStr
				contentPtr = &msg.Content.ContentStr
			}
		}
	} else if req.TextCompletionRequest != nil && req.TextCompletionRequest.Input != nil && req.TextCompletionRequest.Input.PromptStr != nil {
		result := p.Redact(*req.TextCompletionRequest.Input.PromptStr, "")
		if result.HasRedactions {
			*req.TextCompletionRequest.Input.PromptStr = result.Redacted
		}
	}

	_ = content
	_ = contentPtr
	return req, nil, nil
}

func (p *PIIRedactorPlugin) PostLLMHook(ctx *schemas.BifrostContext, resp *schemas.BifrostResponse, bifrostErr *schemas.BifrostError) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
	if !p.enabled || bifrostErr != nil || resp == nil {
		return resp, bifrostErr, nil
	}

	if resp.ChatResponse != nil && len(resp.ChatResponse.Choices) > 0 && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice != nil && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message != nil && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message.Content != nil && resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message.Content.ContentStr != nil {
		result := p.Redact(*resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message.Content.ContentStr, "")
		if result.HasRedactions {
			*resp.ChatResponse.Choices[0].ChatNonStreamResponseChoice.Message.Content.ContentStr = result.Redacted
		}
	} else if resp.TextCompletionResponse != nil && len(resp.TextCompletionResponse.Choices) > 0 && resp.TextCompletionResponse.Choices[0].TextCompletionResponseChoice != nil && resp.TextCompletionResponse.Choices[0].TextCompletionResponseChoice.Text != nil {
		result := p.Redact(*resp.TextCompletionResponse.Choices[0].TextCompletionResponseChoice.Text, "")
		if result.HasRedactions {
			*resp.TextCompletionResponse.Choices[0].TextCompletionResponseChoice.Text = result.Redacted
		}
	}

	return resp, bifrostErr, nil
}

var _ schemas.LLMPlugin = (*PIIRedactorPlugin)(nil)
var _ schemas.BasePlugin = (*PIIRedactorPlugin)(nil)
