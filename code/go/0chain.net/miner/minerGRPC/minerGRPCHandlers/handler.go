package minerGRPCHandlers

import (
	"context"
	"net/http"

	"0chain.net/miner/minerGRPC"

	"0chain.net/chaincore/chain"

	"0chain.net/chaincore/node"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func RegisterGRPCMinerService(server *grpc.Server) {
	minerService := NewGRPCMinerService()
	grpcGatewayHandler := runtime.NewServeMux()

	minerGRPC.RegisterMinerServer(server, minerService)
	_ = minerGRPC.RegisterMinerHandlerServer(context.Background(), grpcGatewayHandler, minerService)

	// TODO i dont think this works, all requests will come to grpc gateway - check blobber
	http.Handle("/", grpcGatewayHandler)
}

type nodeService interface {
	WhoAmI(ctx context.Context, req *minerGRPC.WhoAmIRequest) (*minerGRPC.WhoAmIResponse, error)
}

type chainService interface {
	GetLatestFinalizedBlockSummary(ctx context.Context, req *minerGRPC.GetLatestFinalizedBlockSummaryRequest) (*minerGRPC.GetLatestFinalizedBlockSummaryResponse, error)
}

type minerGRPCService struct {
	chain chainService
	node  nodeService
	minerGRPC.UnimplementedMinerServer
}

func (m *minerGRPCService) WhoAmI(ctx context.Context, req *minerGRPC.WhoAmIRequest) (*minerGRPC.WhoAmIResponse, error) {
	return m.node.WhoAmI(ctx, req)
}

func (m *minerGRPCService) GetLatestFinalizedBlockSummary(ctx context.Context, req *minerGRPC.GetLatestFinalizedBlockSummaryRequest) (*minerGRPC.GetLatestFinalizedBlockSummaryResponse, error) {
	return m.chain.GetLatestFinalizedBlockSummary(ctx, req)
}

func NewGRPCMinerService() *minerGRPCService {
	return &minerGRPCService{
		chain: chain.NewGRPCMinerChainService(chain.GetServerChain()),
		node:  node.NewGRPCMinerNodeService(node.Self),
	}
}
