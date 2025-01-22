package node

import "sync"

// VCAddNodesList - a cache for vc_add register node list ids
// which are node ids that are going to be added to new magic block
type VCAddNodesList struct {
	sync.RWMutex
	nodes []string
}

// NewVCAddNodesList - create a new register nodes cache
func NewVCAddNodesList() *VCAddNodesList {
	return &VCAddNodesList{
		nodes: make([]string, 0),
	}
}

// ReplaceAll - replace all registered node IDs in the cache
func (rnc *VCAddNodesList) ReplaceAll(nodes []string) {
	rnc.Lock()
	defer rnc.Unlock()
	rnc.nodes = make([]string, len(nodes))
	copy(rnc.nodes, nodes)
}

// Contains - check if a node ID exists in the cache
func (rnc *VCAddNodesList) Contains(id string) bool {
	rnc.RLock()
	defer rnc.RUnlock()
	for _, nodeID := range rnc.nodes {
		if nodeID == id {
			return true
		}
	}
	return false
}
