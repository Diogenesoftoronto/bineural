package dataconnectors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "data_connectors"

// ── Connector types (existing) ──────────────────────────────────────────────

type ConnectorType string

const (
	ConnectorDatadog  ConnectorType = "datadog"
	ConnectorBigQuery ConnectorType = "bigquery"
)

type ConnectorConfig struct {
	APIKey    string            `json:"api_key,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	DatasetID string            `json:"dataset_id,omitempty"`
	TableID   string            `json:"table_id,omitempty"`
	Endpoints map[string]string `json:"endpoints,omitempty"`
	Custom    map[string]string `json:"custom,omitempty"`
}

type DataConnector struct {
	ID         string          `json:"id"`
	Name       string         `json:"name"`
	Type       ConnectorType   `json:"type"`
	Config     ConnectorConfig `json:"config"`
	Enabled    bool            `json:"enabled"`
	LastSyncAt *time.Time      `json:"last_sync_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ── Destination types (new) ──────────────────────────────────────────────────

// DestinationType enumerates the external systems to which logs can be dispatched.
type DestinationType string

const (
	DestTypeWebhook  DestinationType = "webhook"
	DestTypeDatadog  DestinationType = "datadog"
	DestTypeBigQuery DestinationType = "bigquery"
)

// Destination describes a single external dispatch target.
type Destination struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Type    DestinationType   `json:"type"`
	Config  map[string]string `json:"config,omitempty"` // type-specific config
	Enabled bool              `json:"enabled"`
}

// ── Shared event type ────────────────────────────────────────────────────────

type IngestEvent struct {
	ConnectorID string          `json:"connector_id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	Timestamp   time.Time       `json:"timestamp"`
}

// ── Plugin config ────────────────────────────────────────────────────────────

type Config struct {
	Enabled      bool          `json:"enabled"`
	Connectors   []DataConnector `json:"connectors,omitempty"`
	Destinations []Destination   `json:"destinations,omitempty"`
}

// ── Plugin struct ────────────────────────────────────────────────────────────

type DataConnectorsPlugin struct {
	mu          sync.RWMutex
	connectors  map[string]*DataConnector
	destinations map[string]*Destination
	events      []IngestEvent
	enabled     bool
	logger      schemas.Logger
	client      *http.Client
	stopCh      chan struct{}
}

// ── Initialisation ───────────────────────────────────────────────────────────

func Init(config any, logger schemas.Logger) *DataConnectorsPlugin {
	p := &DataConnectorsPlugin{
		connectors:   make(map[string]*DataConnector),
		destinations:  make(map[string]*Destination),
		events:       make([]IngestEvent, 0),
		logger:       logger,
		client:       &http.Client{Timeout: 30 * time.Second},
		stopCh:       make(chan struct{}),
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			for i := range cfg.Connectors {
				c := &cfg.Connectors[i]
				if c.ID == "" {
					c.ID = fmt.Sprintf("conn-%d", time.Now().UnixNano())
				}
				c.CreatedAt = time.Now()
				p.connectors[c.ID] = c
			}
			for i := range cfg.Destinations {
				d := &cfg.Destinations[i]
				if d.ID == "" {
					d.ID = fmt.Sprintf("dest-%d", time.Now().UnixNano())
				}
				p.destinations[d.ID] = d
			}
		}
	}

	// Start periodic flush goroutine.
	go p.periodicFlush()

	return p
}

// periodicFlush drains the buffer every 5 seconds or when stopCh is closed.
func (p *DataConnectorsPlugin) periodicFlush() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.flush()
		case <-p.stopCh:
			p.flush() // final drain on shutdown
			return
		}
	}
}

// ── BasePlugin interface ─────────────────────────────────────────────────────

func (p *DataConnectorsPlugin) GetName() string { return PluginName }

func (p *DataConnectorsPlugin) Cleanup() error {
	close(p.stopCh)
	return nil
}

func (p *DataConnectorsPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// ── Connector CRUD (unchanged logic, kept as-is) ─────────────────────────────

func (p *DataConnectorsPlugin) CreateConnector(conn *DataConnector) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn.ID == "" {
		conn.ID = fmt.Sprintf("conn-%d", time.Now().UnixNano())
	}
	conn.CreatedAt = time.Now()
	p.connectors[conn.ID] = conn
	return nil
}

func (p *DataConnectorsPlugin) GetConnector(id string) (*DataConnector, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	c, ok := p.connectors[id]
	return c, ok
}

func (p *DataConnectorsPlugin) ListConnectors(connType ConnectorType) []*DataConnector {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*DataConnector, 0)
	for _, c := range p.connectors {
		if connType == "" || c.Type == connType {
			result = append(result, c)
		}
	}
	return result
}

func (p *DataConnectorsPlugin) UpdateConnector(id string, updates map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	c, ok := p.connectors[id]
	if !ok {
		return fmt.Errorf("connector not found: %s", id)
	}

	if name, ok := updates["name"].(string); ok {
		c.Name = name
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		c.Enabled = enabled
	}
	return nil
}

func (p *DataConnectorsPlugin) DeleteConnector(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.connectors[id]; !ok {
		return fmt.Errorf("connector not found: %s", id)
	}
	delete(p.connectors, id)
	return nil
}

// ── Destination CRUD ─────────────────────────────────────────────────────────

// ListDestinations returns all destinations, optionally filtered by type.
func (p *DataConnectorsPlugin) ListDestinations(destType DestinationType) []*Destination {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*Destination, 0)
	for _, d := range p.destinations {
		if destType == "" || d.Type == destType {
			result = append(result, d)
		}
	}
	return result
}

// AddDestination registers a new dispatch destination.
func (p *DataConnectorsPlugin) AddDestination(dest *Destination) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if dest.ID == "" {
		dest.ID = fmt.Sprintf("dest-%d", time.Now().UnixNano())
	}
	if _, exists := p.destinations[dest.ID]; exists {
		return fmt.Errorf("destination already exists: %s", dest.ID)
	}
	p.destinations[dest.ID] = dest
	return nil
}

// RemoveDestination removes a dispatch destination by ID.
func (p *DataConnectorsPlugin) RemoveDestination(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.destinations[id]; !ok {
		return fmt.Errorf("destination not found: %s", id)
	}
	delete(p.destinations, id)
	return nil
}

// ── Ingest & dispatch ────────────────────────────────────────────────────────

// Ingest buffers the event and dispatches it to all enabled destinations.
func (p *DataConnectorsPlugin) Ingest(connID string, eventType string, payload json.RawMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn, ok := p.connectors[connID]
	if !ok {
		return fmt.Errorf("connector not found: %s", connID)
	}
	if !conn.Enabled {
		return fmt.Errorf("connector is disabled: %s", connID)
	}

	event := IngestEvent{
		ConnectorID: connID,
		EventType:   eventType,
		Payload:     payload,
		Timestamp:   time.Now(),
	}

	p.events = append(p.events, event)

	now := time.Now()
	conn.LastSyncAt = &now

	// Snapshot enabled destinations for non-blocking dispatch.
	dests := make([]*Destination, 0)
	for _, d := range p.destinations {
		if d.Enabled {
			dests = append(dests, d)
		}
	}

	// Non-blocking dispatch to each destination.
	for _, dest := range dests {
		go func(d *Destination, ev IngestEvent) {
			defer func() {
				if r := recover(); r != nil {
					p.logger.Warn("data_connectors: dispatch panic recovered", "destination", d.ID, "error", r)
				}
			}()
			if err := p.dispatch(d, ev); err != nil {
				p.logger.Warn("data_connectors: dispatch failed", "destination", d.ID, "error", err)
			}
		}(dest, event)
	}

	return nil
}

// dispatch sends the event to a single destination.
func (p *DataConnectorsPlugin) dispatch(dest *Destination, event IngestEvent) error {
	switch dest.Type {
	case DestTypeWebhook:
		return p.dispatchWebhook(dest, event)
	case DestTypeDatadog:
		return p.dispatchDatadog(dest, event)
	case DestTypeBigQuery:
		return fmt.Errorf("BigQuery integration requires Google Cloud SDK — not yet available in open-source build")
	default:
		return fmt.Errorf("unknown destination type: %s", dest.Type)
	}
}

// dispatchWebhook POSTs the event as JSON to the configured URL.
func (p *DataConnectorsPlugin) dispatchWebhook(dest *Destination, event IngestEvent) error {
	url, ok := dest.Config["url"]
	if !ok || strings.TrimSpace(url) == "" {
		return fmt.Errorf("webhook destination %s: missing or empty 'url' config", dest.ID)
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("webhook destination %s: marshal event: %w", dest.ID, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook destination %s: create request: %w", dest.ID, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook destination %s: send request: %w", dest.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook destination %s: unexpected status %d", dest.ID, resp.StatusCode)
	}
	return nil
}

// dispatchDatadog POSTs the event payload to the Datadog Logs Intake API.
func (p *DataConnectorsPlugin) dispatchDatadog(dest *Destination, event IngestEvent) error {
	apiKey, ok := dest.Config["api_key"]
	if !ok || strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("datadog destination %s: missing 'api_key' config", dest.ID)
	}

	site := dest.Config["site"]
	if strings.TrimSpace(site) == "" {
		site = "datadoghq.com"
	}

	endpoint := fmt.Sprintf("https://http-intake.logs.%s/v1/input", site)

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("datadog destination %s: marshal event: %w", dest.ID, err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("datadog destination %s: create request: %w", dest.ID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("datadog destination %s: send request: %w", dest.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("datadog destination %s: unexpected status %d", dest.ID, resp.StatusCode)
	}
	return nil
}

// ── Flush ─────────────────────────────────────────────────────────────────────

// flush drains all buffered events and dispatches each to every enabled destination.
func (p *DataConnectorsPlugin) flush() {
	p.mu.Lock()
	events := p.events
	p.events = make([]IngestEvent, 0)

	dests := make([]*Destination, 0)
	for _, d := range p.destinations {
		if d.Enabled {
			dests = append(dests, d)
		}
	}
	p.mu.Unlock()

	if len(events) == 0 || len(dests) == 0 {
		return
	}

	for _, event := range events {
		for _, dest := range dests {
			go func(d *Destination, ev IngestEvent) {
				defer func() {
					if r := recover(); r != nil {
						p.logger.Warn("data_connectors: flush dispatch panic recovered", "destination", d.ID, "error", r)
					}
				}()
				if err := p.dispatch(d, ev); err != nil {
					p.logger.Warn("data_connectors: flush dispatch failed", "destination", d.ID, "error", err)
				}
			}(dest, event)
		}
	}
}

// ── Event retrieval (unchanged) ──────────────────────────────────────────────

func (p *DataConnectorsPlugin) GetEvents(connID string, limit int) []IngestEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]IngestEvent, 0)
	for i := len(p.events) - 1; i >= 0 && len(result) < limit; i-- {
		if connID == "" || p.events[i].ConnectorID == connID {
			result = append(result, p.events[i])
		}
	}
	return result
}

var _ schemas.BasePlugin = (*DataConnectorsPlugin)(nil)
