package largepayload

import (
	"sync"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "large_payload"

type LargePayloadConfig struct {
	Enabled            bool `json:"enabled"`
	MaxNormalBytes     int  `json:"max_normal_bytes,omitempty"`
	MaxLargeBytes      int  `json:"max_large_bytes,omitempty"`
	ChunkSize          int  `json:"chunk_size,omitempty"`
	StreamThreshold    int  `json:"stream_threshold_bytes,omitempty"`
}

type Config struct {
	Enabled          bool `json:"enabled"`
	MaxNormalBytes   int  `json:"max_normal_bytes,omitempty"`
	MaxLargeBytes    int  `json:"max_large_bytes,omitempty"`
	ChunkSize        int  `json:"chunk_size,omitempty"`
	StreamThreshold  int  `json:"stream_threshold_bytes,omitempty"`
}

type LargePayloadPlugin struct {
	mu              sync.RWMutex
	maxNormalBytes  int
	maxLargeBytes   int
	chunkSize       int
	streamThreshold int
	enabled         bool
	logger          schemas.Logger
}

func Init(config any, logger schemas.Logger) *LargePayloadPlugin {
	p := &LargePayloadPlugin{
		maxNormalBytes:  10 * 1024 * 1024, // 10MB
		maxLargeBytes:   100 * 1024 * 1024, // 100MB
		chunkSize:       1024 * 1024,       // 1MB
		streamThreshold: 5 * 1024 * 1024,   // 5MB
		logger:          logger,
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			if cfg.MaxNormalBytes > 0 {
				p.maxNormalBytes = cfg.MaxNormalBytes
			}
			if cfg.MaxLargeBytes > 0 {
				p.maxLargeBytes = cfg.MaxLargeBytes
			}
			if cfg.ChunkSize > 0 {
				p.chunkSize = cfg.ChunkSize
			}
			if cfg.StreamThreshold > 0 {
				p.streamThreshold = cfg.StreamThreshold
			}
		}
	}

	return p
}

func (p *LargePayloadPlugin) GetName() string { return PluginName }
func (p *LargePayloadPlugin) Cleanup() error  { return nil }

func (p *LargePayloadPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *LargePayloadPlugin) ShouldUseLargeMode(payloadSize int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return false
	}
	return payloadSize > p.streamThreshold
}

func (p *LargePayloadPlugin) IsWithinLimit(payloadSize int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return payloadSize <= p.maxNormalBytes
	}
	return payloadSize <= p.maxLargeBytes
}

func (p *LargePayloadPlugin) GetMaxNormalBytes() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.maxNormalBytes
}

func (p *LargePayloadPlugin) GetMaxLargeBytes() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.maxLargeBytes
}

func (p *LargePayloadPlugin) GetChunkSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.chunkSize
}

func (p *LargePayloadPlugin) GetStreamThreshold() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.streamThreshold
}

func (p *LargePayloadPlugin) SetConfig(maxNormal, maxLarge, chunkSize, threshold int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if maxNormal > 0 {
		p.maxNormalBytes = maxNormal
	}
	if maxLarge > 0 {
		p.maxLargeBytes = maxLarge
	}
	if chunkSize > 0 {
		p.chunkSize = chunkSize
	}
	if threshold > 0 {
		p.streamThreshold = threshold
	}
}

var _ schemas.BasePlugin = (*LargePayloadPlugin)(nil)
