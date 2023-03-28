package stats

import (
	"sync"
)

type (
	// NodesClientStats represents struct with maps containing
	// needed nodes client stats.
	NodesClientStats struct {
		// BlockStateChange represents map which stores block state change requests stats.
		// minerID -> BlockRequests
		BlockStateChange map[string]*BlockRequests

		minerNotarisedBlockMu sync.Mutex

		// MinerNotarisedBlock represents map which stores miner notarised block requests stats.
		// minerID -> BlockRequests
		MinerNotarisedBlock map[string]*BlockRequests

		fbMu sync.Mutex

		// FB represents map which stores miner notarised block requests stats.
		// minerID -> BlockRequests
		FB map[string]*BlockRequests
	}
)

// NewNodesClientStats creates initialised NodesClientStats.
func NewNodesClientStats() *NodesClientStats {
	return &NodesClientStats{
		BlockStateChange:    make(map[string]*BlockRequests),
		MinerNotarisedBlock: make(map[string]*BlockRequests),
		FB:                  make(map[string]*BlockRequests),
	}
}

func (ncs *NodesClientStats) AddBlockStats(request *BlockRequest, requestorType BlockRequestor) {
	switch requestorType {
	case BRBlockStateChange:
		ncs.addBlockStateChangeStats(request)

	case BRMinerNotarisedBlock:
		ncs.addMinerNotarisedBlockStats(request)

	case BRFB:
		ncs.addFBStats(request)
	}
}

// addBlockStateChangeStats takes needed info from the BlockStateChangeRequest
// and inserts it to the NodesClientStats.BlockStateChange map.
func (ncs *NodesClientStats) addBlockStateChangeStats(rep *BlockRequest) {
	ncs.minerNotarisedBlockMu.Lock()
	defer ncs.minerNotarisedBlockMu.Unlock()

	_, ok := ncs.BlockStateChange[rep.NodeID]
	if !ok {
		ncs.BlockStateChange[rep.NodeID] = NewBlockRequests()
	}
	ncs.BlockStateChange[rep.NodeID].Add(rep)
}

// addMinerNotarisedBlockStats takes needed info from the MinerNotarisedBlockRequest
// and inserts it to the NodesClientStats.MinerNotarisedBlock map.
func (ncs *NodesClientStats) addMinerNotarisedBlockStats(rep *BlockRequest) {
	ncs.fbMu.Lock()
	defer ncs.fbMu.Unlock()

	_, ok := ncs.MinerNotarisedBlock[rep.NodeID]
	if !ok {
		ncs.MinerNotarisedBlock[rep.NodeID] = NewBlockRequests()
	}
	ncs.MinerNotarisedBlock[rep.NodeID].Add(rep)
}

// AddFBStats takes needed info from the MinerNotarisedBlockRequest
// and inserts it to the NodesClientStats.FB map.
func (ncs *NodesClientStats) addFBStats(rep *BlockRequest) {
	ncs.fbMu.Lock()
	defer ncs.fbMu.Unlock()

	_, ok := ncs.FB[rep.NodeID]
	if !ok {
		ncs.FB[rep.NodeID] = NewBlockRequests()
	}
	ncs.FB[rep.NodeID].Add(rep)
}
