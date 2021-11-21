package handlers

import (
	minerproto "0chain.net/miner/proto/api/src/proto"
)

// minerGRPCService
type minerGRPCService struct {
	*minerproto.UnimplementedMinerServiceServer
}

// NewMinerGRPCService
func NewMinerGRPCService() *minerGRPCService {
	return &minerGRPCService{
		UnimplementedMinerServiceServer: &minerproto.UnimplementedMinerServiceServer{},
	}
}
