package convert

import (
	"0chain.net/chaincore/block"
	minerproto "0chain.net/miner/proto/api/src/proto"
)

func GetNotarizedBlockResponseCreator(resp interface{}) *minerproto.GetNotarizedBlockResponse {
	if resp == nil {
		return nil
	}

	b, _ := resp.(*block.Block)
	return &minerproto.GetNotarizedBlockResponse{Block: BlockToGRPCBlock(b)}
}
