package chain

import (
	"context"
	"net/http"

	"0chain.net/chaincore/block"

	"0chain.net/miner/minerGRPC"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func RegisterGRPCMinerChainServer(server *grpc.Server) {
	minerChainService := NewGRPCMinerChainService(GetServerChain())
	grpcGatewayHandler := runtime.NewServeMux()

	minerGRPC.RegisterChainServer(server, minerChainService)
	_ = minerGRPC.RegisterChainHandlerServer(context.Background(), grpcGatewayHandler, minerChainService)

	http.Handle("/", grpcGatewayHandler)
}

func NewGRPCMinerChainService(chain IChain) *minerChainGRPCService {
	return &minerChainGRPCService{
		ServerChain: chain,
	}
}

type IChain interface {
	GetLatestFinalizedBlockSummary() *block.BlockSummary
}

type minerChainGRPCService struct {
	ServerChain IChain
	minerGRPC.UnimplementedChainServer
}

func (m *minerChainGRPCService) GetLatestFinalizedBlockSummary(ctx context.Context, req *minerGRPC.GetLatestFinalizedBlockSummaryRequest) (*minerGRPC.GetLatestFinalizedBlockSummaryResponse, error) {

	//summary := m.ServerChain.GetLatestFinalizedBlockSummary()

	//return BlockSummaryToGRPCBlockSummary(summary), nil
	return nil, nil
}
