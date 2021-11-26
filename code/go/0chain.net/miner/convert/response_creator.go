package convert

import (
	"0chain.net/chaincore/block"
	"0chain.net/miner/minergrpc"
)

func GetNotarizedBlockResponseCreator(resp interface{}) *minergrpc.GetNotarizedBlockResponse {
	if resp == nil {
		return nil
	}

	b, _ := resp.(*block.Block)
	return &minergrpc.GetNotarizedBlockResponse{Block: BlockToGRPCBlock(b)}
}