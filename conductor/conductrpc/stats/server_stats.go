package stats

import (
	"sync"
)

type (
	// NodesServerStats represents struct with maps containing
	// needed nodes server stats.
	NodesServerStats struct {
		blockMu sync.Mutex

		// Block represents map which stores fetching block stats.
		// minerID -> BlockRequests
		Block map[string]*BlockRequests

		vrfsMu sync.Mutex

		// VRFS represents map which stores vrfs requests stats.
		// minerID -> VRFSRequests
		VRFS map[string]*VRFSRequests
	}
)

// NewNodesServerStats creates initialised NodesServerStats.
func NewNodesServerStats() *NodesServerStats {
	return &NodesServerStats{
		Block: make(map[string]*BlockRequests),
		VRFS:  make(map[string]*VRFSRequests),
	}
}

// AddBlockStats takes needed info from the BlockRequest and inserts it to the NodesServerStats.Block map.
func (nss *NodesServerStats) AddBlockStats(rep *BlockRequest) {
	nss.blockMu.Lock()
	defer nss.blockMu.Unlock()

	_, ok := nss.Block[rep.NodeID]
	if !ok {
		nss.Block[rep.NodeID] = NewBlockRequests()
	}
	nss.Block[rep.NodeID].Add(rep)
}

// AddVRFSStats takes needed info from the VRFSRequest and inserts it to the NodesServerStats.VRFS map.
func (nss *NodesServerStats) AddVRFSStats(rep *VRFSRequest) {
	nss.vrfsMu.Lock()
	defer nss.vrfsMu.Unlock()

	_, ok := nss.VRFS[rep.NodeID]
	if !ok {
		nss.VRFS[rep.NodeID] = NewVRFSRequests()
	}
	nss.VRFS[rep.NodeID].Add(rep)
}
