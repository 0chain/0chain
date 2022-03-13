package stats

import (
	"encoding/json"
	"sync"
)

type (
	MinerNotarisedBlockRequests struct {
		listMu sync.Mutex
		list   []*MinerNotarisedBlockRequest
	}

	// MinerNotarisedBlockRequest represents struct for collecting reports from the nodes
	// about sent notarised block requests.
	MinerNotarisedBlockRequest struct {
		NodeID string `json:"node_id"`
		Round  int    `json:"round"`
		Block  string `json:"block"`
	}
)

// NewMinerNotarisedBlockRequests creates initialised MinerNotarisedBlockRequests.
func NewMinerNotarisedBlockRequests() *MinerNotarisedBlockRequests {
	return &MinerNotarisedBlockRequests{
		list: make([]*MinerNotarisedBlockRequest, 0),
	}
}

// Add adds MinerNotarisedBlockRequest to the list.
func (bi *MinerNotarisedBlockRequests) Add(rep *MinerNotarisedBlockRequest) {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	bi.list = append(bi.list, rep)
}

// CountWithHash counts number of stored MinerNotarisedBlockRequest with MinerNotarisedBlockRequest.Block
// equals to the provided hash.
func (bi *MinerNotarisedBlockRequests) CountWithHash(hash string) int {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	numReq := 0
	for _, req := range bi.list {
		if req.Block == hash {
			numReq++
		}
	}
	return numReq
}

// Encode encodes MinerNotarisedBlockRequest to the bytes.
func (req *MinerNotarisedBlockRequest) Encode() ([]byte, error) {
	return json.Marshal(req)
}

// Decode decodes MinerNotarisedBlockRequest from the bytes.
func (req *MinerNotarisedBlockRequest) Decode(blob []byte) error {
	return json.Unmarshal(blob, req)
}
