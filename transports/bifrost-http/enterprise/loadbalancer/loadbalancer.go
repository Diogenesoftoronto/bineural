package loadbalancer

import (
	"math"
	"math/rand/v2"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "adaptive_loadbalancer"

// Config for adaptive load balancing
type Config struct {
	Enabled              bool    `json:"enabled"`
	WeightUpdateInterval int     `json:"weight_update_interval_sec,omitempty"` // default 5s
	ErrorWeight          float64 `json:"error_weight,omitempty"`               // default 0.5
	LatencyWeight        float64 `json:"latency_weight,omitempty"`             // default 0.2
	UtilWeight           float64 `json:"util_weight,omitempty"`                // default 0.05
	ExplorationPercent   float64 `json:"exploration_percent,omitempty"`        // default 0.25
}

// KeyMetrics tracks per-key performance
type KeyMetrics struct {
	KeyID         string
	TotalRequests int64
	ErrorCount    int64
	TotalLatency  int64 // ms
	LastUpdated   time.Time
	CurrentWeight float64
}

// AdaptiveLoadBalancer implements intelligent request routing
type AdaptiveLoadBalancer struct {
	mu              sync.RWMutex
	metrics         map[string]*KeyMetrics
	enabled         bool
	updateInterval  time.Duration
	errorWeight     float64
	latencyWeight   float64
	utilWeight      float64
	exploration     float64
	logger          schemas.Logger
	stopChan        chan struct{}
}

// Init initializes the adaptive load balancer
func Init(config any, logger schemas.Logger) *AdaptiveLoadBalancer {
	p := &AdaptiveLoadBalancer{
		metrics:        make(map[string]*KeyMetrics),
		updateInterval: 5 * time.Second,
		errorWeight:    0.5,
		latencyWeight:  0.2,
		utilWeight:     0.05,
		exploration:    0.25,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			if cfg.WeightUpdateInterval > 0 {
				p.updateInterval = time.Duration(cfg.WeightUpdateInterval) * time.Second
			}
			if cfg.ErrorWeight > 0 {
				p.errorWeight = cfg.ErrorWeight
			}
			if cfg.LatencyWeight > 0 {
				p.latencyWeight = cfg.LatencyWeight
			}
			if cfg.UtilWeight > 0 {
				p.utilWeight = cfg.UtilWeight
			}
			if cfg.ExplorationPercent > 0 {
				p.exploration = cfg.ExplorationPercent
			}
		}
	}

	if p.enabled {
		go p.updateWeightsLoop()
	}

	return p
}

func (p *AdaptiveLoadBalancer) GetName() string { return PluginName }

func (p *AdaptiveLoadBalancer) Cleanup() error {
	close(p.stopChan)
	return nil
}

// IsEnabled returns whether adaptive LB is active
func (p *AdaptiveLoadBalancer) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// RecordResult records the outcome of a request
func (p *AdaptiveLoadBalancer) RecordResult(keyID string, latencyMs int64, isError bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	m, ok := p.metrics[keyID]
	if !ok {
		m = &KeyMetrics{KeyID: keyID, CurrentWeight: 1.0}
		p.metrics[keyID] = m
	}

	m.TotalRequests++
	m.TotalLatency += latencyMs
	if isError {
		m.ErrorCount++
	}
	m.LastUpdated = time.Now()
}

// SelectKey chooses a key based on adaptive weights with exploration
func (p *AdaptiveLoadBalancer) SelectKey(keyIDs []string) string {
	if len(keyIDs) == 0 {
		return ""
	}
	if len(keyIDs) == 1 {
		return keyIDs[0]
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return keyIDs[rand.IntN(len(keyIDs))]
	}

	// Exploration: pick random key some percentage of the time
	if rand.Float64() < p.exploration {
		return keyIDs[rand.IntN(len(keyIDs))]
	}

	// Exploitation: weighted random based on current weights
	type weighted struct {
		id     string
		weight float64
	}
	weights := make([]weighted, 0, len(keyIDs))
	var totalWeight float64

	for _, id := range keyIDs {
		m, ok := p.metrics[id]
		w := 1.0
		if ok && m.CurrentWeight > 0 {
			w = m.CurrentWeight
		}
		weights = append(weights, weighted{id: id, weight: w})
		totalWeight += w
	}

	if totalWeight <= 0 {
		return keyIDs[rand.IntN(len(keyIDs))]
	}

	// Weighted random selection
	r := rand.Float64() * totalWeight
	for _, w := range weights {
		r -= w.weight
		if r <= 0 {
			return w.id
		}
	}

	return keyIDs[len(keyIDs)-1]
}

// updateWeightsLoop periodically recalculates key weights
func (p *AdaptiveLoadBalancer) updateWeightsLoop() {
	ticker := time.NewTicker(p.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.recalculateWeights()
		case <-p.stopChan:
			return
		}
	}
}

func (p *AdaptiveLoadBalancer) recalculateWeights() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, m := range p.metrics {
		if m.TotalRequests == 0 {
			m.CurrentWeight = 1.0
			continue
		}

		errorRate := float64(m.ErrorCount) / float64(m.TotalRequests)
		avgLatency := float64(m.TotalLatency) / float64(m.TotalRequests)

		// Normalize latency (clamp to 0-1 range assuming max 30s)
		normLatency := math.Min(avgLatency/30000.0, 1.0)

		// Utilization approximation (how heavily used)
		util := math.Min(float64(m.TotalRequests)/1000.0, 1.0)

		// Weight = product of health factors
		health := (1.0 - errorRate*p.errorWeight) *
			(1.0 - normLatency*p.latencyWeight) *
			(1.0 - util*p.utilWeight)

		// Ensure minimum weight for recovery
		m.CurrentWeight = math.Max(health, 0.1)
	}
}

// PreLLMHook intercepts to track key selections
func (p *AdaptiveLoadBalancer) PreLLMHook(ctx *schemas.BifrostContext, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.LLMPluginShortCircuit, error) {
	return req, nil, nil
}

// PostLLMHook records latency/errors for adaptive routing
func (p *AdaptiveLoadBalancer) PostLLMHook(ctx *schemas.BifrostContext, resp *schemas.BifrostResponse, bifrostErr *schemas.BifrostError) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
	if !p.enabled {
		return resp, bifrostErr, nil
	}

	keyID := ""
	if v := ctx.Value(schemas.BifrostContextKeySelectedKeyID); v != nil {
		keyID = v.(string)
	}
	if keyID == "" {
		return resp, bifrostErr, nil
	}

	isError := bifrostErr != nil
	var latency int64
	if resp != nil {
		latency = resp.GetExtraFields().Latency
	}

	p.RecordResult(keyID, latency, isError)
	return resp, bifrostErr, nil
}

var _ schemas.LLMPlugin = (*AdaptiveLoadBalancer)(nil)
var _ schemas.BasePlugin = (*AdaptiveLoadBalancer)(nil)
