package clustering

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

const PluginName = "clustering"

// Config for clustering plugin
type Config struct {
	Enabled           bool     `json:"enabled"`
	NodeID            string   `json:"node_id,omitempty"`
	Peers             []string `json:"peers,omitempty"`
	GossipIntervalSec int      `json:"gossip_interval_sec,omitempty"`
	SyncPort          int      `json:"sync_port,omitempty"`
}

// ClusterNode tracks known cluster members
type ClusterNode struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	LastSeen time.Time `json:"last_seen"`
	Healthy  bool      `json:"healthy"`
}

// ClusteringPlugin implements lightweight clustering primitives
type ClusteringPlugin struct {
	mu         sync.RWMutex
	nodeID     string
	peers      map[string]*ClusterNode
	enabled    bool
	logger     schemas.Logger
	stopChan   chan struct{}
}

// Init initializes the clustering plugin
func Init(config any, logger schemas.Logger) *ClusteringPlugin {
	p := &ClusteringPlugin{
		peers:    make(map[string]*ClusterNode),
		logger:   logger,
		stopChan: make(chan struct{}),
	}

	if config != nil {
		if cfg, ok := config.(*Config); ok {
			p.enabled = cfg.Enabled
			p.nodeID = cfg.NodeID
			for _, peer := range cfg.Peers {
				p.peers[peer] = &ClusterNode{
					ID:       hashPeer(peer),
					Address:  peer,
					LastSeen: time.Now(),
					Healthy:  true,
				}
			}
		}
	}

	if p.nodeID == "" {
		p.nodeID = fmt.Sprintf("node-%d", time.Now().Unix())
	}

	return p
}

func (p *ClusteringPlugin) GetName() string { return PluginName }
func (p *ClusteringPlugin) Cleanup() error {
	close(p.stopChan)
	return nil
}

// IsEnabled returns whether clustering is active
func (p *ClusteringPlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// GetNodeID returns this node's ID
func (p *ClusteringPlugin) GetNodeID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.nodeID
}

// GetPeers returns known cluster peers
func (p *ClusteringPlugin) GetPeers() []ClusterNode {
	p.mu.RLock()
	defer p.mu.RUnlock()

	peers := make([]ClusterNode, 0, len(p.peers))
	for _, peer := range p.peers {
		peers = append(peers, *peer)
	}
	return peers
}

// MarkPeerHealthy/ Unhealthy
func (p *ClusteringPlugin) MarkPeerHealthy(peerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer, ok := p.peers[peerID]; ok {
		peer.Healthy = true
		peer.LastSeen = time.Now()
	}
}

// SelectNodeForKey returns which cluster node should handle a given key (consistent hashing)
func (p *ClusteringPlugin) SelectNodeForKey(key string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled || len(p.peers) == 0 {
		return p.nodeID
	}

	// Include self in ring
	allNodes := []string{p.nodeID}
	for id := range p.peers {
		allNodes = append(allNodes, id)
	}

	// Simple consistent hash: select node with highest hash value after key hash
	targetHash := hashString(key)
	bestNode := p.nodeID
	bestDiff := uint64(0)

	for _, nodeID := range allNodes {
		nodeHash := hashString(nodeID)
		var diff uint64
		if nodeHash >= targetHash {
			diff = nodeHash - targetHash
		} else {
			diff = ^uint64(0) - targetHash + nodeHash
		}
		if bestDiff == 0 || diff < bestDiff {
			bestDiff = diff
			bestNode = nodeID
		}
	}

	return bestNode
}

func hashPeer(peer string) string {
	h := fnv.New64a()
	h.Write([]byte(peer))
	return fmt.Sprintf("%x", h.Sum64())[:12]
}

func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

var _ schemas.BasePlugin = (*ClusteringPlugin)(nil)
