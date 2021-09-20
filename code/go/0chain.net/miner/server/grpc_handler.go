package server

import (
	"context"

	"0chain.net/miner"
	"0chain.net/miner/convert"
	"0chain.net/miner/minergrpc"
	"github.com/0chain/errors"
)

type minerGRPCService struct {
	minergrpc.UnimplementedMinerServiceServer
}


func NewMinerGRPCService() *minerGRPCService {
	return &minerGRPCService{}
}

func (minerGRPCService) GetNotarizedBlock(ctx context.Context, req *minergrpc.GetNotarizedBlockRequest) (*minergrpc.GetNotarizedBlockResponse, error) {
	response, err := miner.GetNotarizedBlock(ctx, req.GetRound(), req.GetHash())
	if err != nil {
		return nil, errors.Wrap(err, "unable to get notarized block for request: " + req.String())
	}

	return convert.GetNotarizedBlockResponseCreator(response), nil
}