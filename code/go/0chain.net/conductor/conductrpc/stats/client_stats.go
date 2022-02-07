package stats

import (
	"sync"
)

type (
	// NodesClientStats represents struct with maps containing
	// needed nodes client stats.
	NodesClientStats struct {
		blockStateChangeMu sync.Mutex

		// BlockStateChange represents map which stores block state change requests stats.
		// minerID -> BlockStateChangeRequests
		BlockStateChange map[string]*BlockStateChangeRequests
	}
)

// NewNodesClientStats creates initialised NodesClientStats.
func NewNodesClientStats() *NodesClientStats {
	return &NodesClientStats{
		BlockStateChange: make(map[string]*BlockStateChangeRequests),
	}
}

// AddBlockStateChangeStats takes needed info from the BlockStateChangeRequest
// and inserts it to the NodesClientStats.BlockStateChange map.
func (nss *NodesClientStats) AddBlockStateChangeStats(rep *BlockStateChangeRequest) {
	nss.blockStateChangeMu.Lock()
	defer nss.blockStateChangeMu.Unlock()

	_, ok := nss.BlockStateChange[rep.NodeID]
	if !ok {
		nss.BlockStateChange[rep.NodeID] = NewBlockStateChangeRequests()
	}
	nss.BlockStateChange[rep.NodeID].Add(rep)
}
