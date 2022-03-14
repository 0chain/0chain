package conductrpc

import (
	"encoding/json"

	"0chain.net/conductor/conductrpc/stats"
)

type (
	BlockRequest struct {
		Req     *stats.BlockRequest  `json:"req"`
		ReqType stats.BlockRequestor `json:"req_type"`
	}
)

func newBlockRequest(req *stats.BlockRequest, reqType stats.BlockRequestor) *BlockRequest {
	return &BlockRequest{
		Req:     req,
		ReqType: reqType,
	}
}

func (br *BlockRequest) Encode() ([]byte, error) {
	return json.Marshal(br)
}

func (br *BlockRequest) Decode(blob []byte) error {
	return json.Unmarshal(blob, br)
}
