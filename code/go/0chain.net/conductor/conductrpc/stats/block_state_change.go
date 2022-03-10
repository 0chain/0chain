package stats

import (
	"encoding/json"
	"sync"
)

type (
	BlockStateChangeRequests struct {
		listMu sync.Mutex
		list   []*BlockStateChangeRequest
	}

	// BlockStateChangeRequest represents struct for collecting reports from the nodes
	// about sent block state change requests.
	BlockStateChangeRequest struct {
		NodeID string `json:"node_id"`
		Block  string `json:"block"`
	}
)

// NewBlockStateChangeRequests creates initialised BlockStateChangeRequests.
func NewBlockStateChangeRequests() *BlockStateChangeRequests {
	return &BlockStateChangeRequests{
		list: make([]*BlockStateChangeRequest, 0),
	}
}

// Add adds BlockStateChangeRequest to the list.
func (bi *BlockStateChangeRequests) Add(rep *BlockStateChangeRequest) {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	bi.list = append(bi.list, rep)
}

// CountWithHash counts number of stored BlockStateChangeRequest with BlockStateChangeRequest.Block
// equals to the provided hash.
func (bi *BlockStateChangeRequests) CountWithHash(hash string) int {
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

// Encode encodes BlockStateChangeRequest to the bytes.
func (req *BlockStateChangeRequest) Encode() ([]byte, error) {
	return json.Marshal(req)
}

// Decode decodes BlockStateChangeRequest from the bytes.
func (req *BlockStateChangeRequest) Decode(blob []byte) error {
	return json.Unmarshal(blob, req)
}
