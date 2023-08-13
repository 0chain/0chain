package stats

import (
	"encoding/json"
	"sync"
)

type (
	BlockRequests struct {
		listMu sync.Mutex
		list   []*BlockRequest
	}

	// BlockRequest represents struct for collecting reports from the nodes
	// about handled block's requests.
	BlockRequest struct {
		NodeID string `json:"miner_id"`
		Hash   string `json:"hash"`
		Round  int    `json:"round"`

		// optional field
		SenderID string `json:"sender_id,omitempty"`
	}

	BlockFromSharder struct {
		Round int64 `json:"round"`
		Hash string `json:"hash"`
		GeneratorId string `json:"miner_id"` 
	}
)

// NewBlockRequests creates initialised BlockRequests.
func NewBlockRequests() *BlockRequests {
	return &BlockRequests{
		list: make([]*BlockRequest, 0),
	}
}

// Add adds BlockRequest to the list.
func (bi *BlockRequests) Add(rep *BlockRequest) {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	bi.list = append(bi.list, rep)
}

// GetByHashOrRound looks for BlockRequest with provided hash or round or hash and round both.
// Returns nil if BlockRequest was not found.
func (bi *BlockRequests) GetByHashOrRound(hash string, round int) *BlockRequest {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	for _, stats := range bi.list {
		onlyHash := stats.Hash == hash && stats.Round == 0
		onlyRound := stats.Round == round && stats.Hash == ""
		hashAndRound := stats.Hash == hash && stats.Round == round
		if onlyHash || onlyRound || hashAndRound {
			return stats
		}
	}
	return nil
}

// GetByHash looks for BlockRequest with provided hash.
// Returns nil if BlockRequest was not found.
func (bi *BlockRequests) GetByHash(hash string) *BlockRequest {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	for _, stats := range bi.list {
		if stats.Hash == hash {
			return stats
		}
	}
	return nil
}

// GetBySenderIDAndHash looks for BlockRequest with provided senderID and hash.
// Returns nil if BlockRequest was not found.
func (bi *BlockRequests) GetBySenderIDAndHash(senderID, hash string) *BlockRequest {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	for _, stats := range bi.list {
		if stats.SenderID == senderID && stats.Hash == hash {
			return stats
		}
	}
	return nil
}

// CountWithHash counts number of stored BlockRequest with BlockRequest.Hash
// equals to the provided hash.
func (bi *BlockRequests) CountWithHash(hash string) int {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	numReq := 0
	for _, req := range bi.list {
		if req.Hash == hash {
			numReq++
		}
	}
	return numReq
}

// Encode encodes BlockRequest to the bytes.
func (br *BlockRequest) Encode() ([]byte, error) {
	return json.Marshal(br)
}

// Decode decodes BlockRequest from the bytes.
func (br *BlockRequest) Decode(blob []byte) error {
	return json.Unmarshal(blob, br)
}

// Encode encodes BlockRequest to the bytes.
func (br *BlockFromSharder) Encode() ([]byte, error) {
	return json.Marshal(br)
}

// Decode decodes BlockRequest from the bytes.
func (br *BlockFromSharder) Decode(blob []byte) error {
	return json.Unmarshal(blob, br)
}
