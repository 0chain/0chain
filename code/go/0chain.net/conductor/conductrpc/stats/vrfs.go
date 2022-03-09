package stats

import (
	"encoding/json"
	"sync"
)

type (
	VRFSRequests struct {
		listMu sync.Mutex
		list   []*VRFSRequest
	}

	// VRFSRequest represents struct for collecting reports from the nodes
	// about handled vrf share requests.
	VRFSRequest struct {
		NodeID       string `json:"node_id"`
		SenderID     string `json:"sender_id"`
		Round        int64  `json:"round"`
		TimeoutCount int    `json:"timeout_count"`
	}
)

// NewVRFSRequests creates initialised VRFSRequests.
func NewVRFSRequests() *VRFSRequests {
	return &VRFSRequests{
		list: make([]*VRFSRequest, 0),
	}
}

// Add adds VRFSRequests to the list.
func (bi *VRFSRequests) Add(rep *VRFSRequest) {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	bi.list = append(bi.list, rep)
}

func (bi *VRFSRequests) GetByRound(round int64) []*VRFSRequest {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	res := make([]*VRFSRequest, 0)
	for _, req := range bi.list {
		if req.Round == round {
			res = append(res, req)
		}
	}
	return res
}

// Encode encodes VRFSRequest to the bytes.
func (br *VRFSRequest) Encode() ([]byte, error) {
	return json.Marshal(br)
}

// Decode decodes VRFSRequest from the bytes.
func (br *VRFSRequest) Decode(blob []byte) error {
	return json.Unmarshal(blob, br)
}
