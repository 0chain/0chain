package handlers

import (
	"context"

	"0chain.net/miner"
	"0chain.net/miner/convert"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/0chain/errors"
)

type minerGRPCService struct {
	*minerproto.UnimplementedMinerServiceServer
}

func NewMinerGRPCService() *minerGRPCService {
	return &minerGRPCService{
		UnimplementedMinerServiceServer: &minerproto.UnimplementedMinerServiceServer{},
	}
}

func (m *minerGRPCService) GetNotarizedBlock(ctx context.Context, req *minerproto.GetNotarizedBlockRequest) (*minerproto.GetNotarizedBlockResponse, error) {
	response, err := miner.GetNotarizedBlock(ctx, req.GetRound(), req.GetHash())
	if err != nil {
		return nil, errors.Wrap(err, "unable to get notarized block for request: "+req.String())
	}

	return convert.GetNotarizedBlockResponseCreator(response), nil
}
