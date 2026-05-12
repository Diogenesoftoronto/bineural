package evals

import (
	"fmt"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "evals"

type EvalMetricType string

const (
	MetricTypeAccuracy   EvalMetricType = "accuracy"
	MetricTypeLatency    EvalMetricType = "latency"
	MetricTypeCost       EvalMetricType = "cost"
	MetricTypeQuality    EvalMetricType = "quality"
	MetricTypeSafety     EvalMetricType = "safety"
	MetricTypeCustom     EvalMetricType = "custom"
)

type EvalDataset struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	NumEntries  int       `json:"num_entries"`
	CreatedAt   time.Time `json:"created_at"`
}

type EvalRun struct {
	ID           string         `json:"id"`
	DatasetID    string         `json:"dataset_id"`
	Model        string         `json:"model"`
	Metrics      map[string]float64 `json:"metrics"`
	Score        float64        `json:"score"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}

type EvalComparison struct {
	RunIDs    []string           `json:"run_ids"`
	Results   map[string]EvalRun `json:"results"`
	BestModel string             `json:"best_model"`
}

type Config struct {
	Enabled bool `json:"enabled"`
}

type EvalsPlugin struct {
	mu       sync.RWMutex
	datasets map[string]*EvalDataset
	runs     map[string]*EvalRun
	enabled  bool
	logger   schemas.Logger
}

func Init(config any, logger schemas.Logger) *EvalsPlugin {
	p := &EvalsPlugin{
		datasets: make(map[string]*EvalDataset),
		runs:     make(map[string]*EvalRun),
		logger:   logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
		}
	}

	return p
}

func (p *EvalsPlugin) GetName() string { return PluginName }
func (p *EvalsPlugin) Cleanup() error  { return nil }

func (p *EvalsPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *EvalsPlugin) CreateDataset(ds *EvalDataset) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ds.ID == "" {
		ds.ID = fmt.Sprintf("ds-%d", time.Now().UnixNano())
	}
	ds.CreatedAt = time.Now()
	p.datasets[ds.ID] = ds
	return nil
}

func (p *EvalsPlugin) GetDataset(id string) (*EvalDataset, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ds, ok := p.datasets[id]
	return ds, ok
}

func (p *EvalsPlugin) ListDatasets() []*EvalDataset {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*EvalDataset, 0, len(p.datasets))
	for _, ds := range p.datasets {
		result = append(result, ds)
	}
	return result
}

func (p *EvalsPlugin) CreateRun(run *EvalRun) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if run.ID == "" {
		run.ID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	if run.Status == "" {
		run.Status = "pending"
	}
	if run.Metrics == nil {
		run.Metrics = make(map[string]float64)
	}
	run.CreatedAt = time.Now()
	p.runs[run.ID] = run
	return nil
}

func (p *EvalsPlugin) GetRun(id string) (*EvalRun, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	r, ok := p.runs[id]
	return r, ok
}

func (p *EvalsPlugin) CompleteRun(id string, metrics map[string]float64, score float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	run, ok := p.runs[id]
	if !ok {
		return fmt.Errorf("run not found: %s", id)
	}

	now := time.Now()
	run.Metrics = metrics
	run.Score = score
	run.Status = "completed"
	run.CompletedAt = &now
	return nil
}

func (p *EvalsPlugin) ListRuns(datasetID string) []*EvalRun {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*EvalRun, 0)
	for _, r := range p.runs {
		if datasetID == "" || r.DatasetID == datasetID {
			result = append(result, r)
		}
	}
	return result
}

func (p *EvalsPlugin) CompareRuns(runIDs []string) *EvalComparison {
	p.mu.RLock()
	defer p.mu.RUnlock()

	comp := &EvalComparison{
		RunIDs:  runIDs,
		Results: make(map[string]EvalRun),
	}

	bestScore := -1.0
	for _, id := range runIDs {
		if r, ok := p.runs[id]; ok {
			comp.Results[id] = *r
			if r.Score > bestScore {
				bestScore = r.Score
				comp.BestModel = r.Model
			}
		}
	}

	return comp
}

var _ schemas.BasePlugin = (*EvalsPlugin)(nil)
