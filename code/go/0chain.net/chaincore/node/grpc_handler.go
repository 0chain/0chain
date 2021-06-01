package node

import (
	"context"
	"net/http"
	"strings"

	"0chain.net/miner/minerGRPC"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func RegisterGRPCMinerNodeService(server *grpc.Server) {
	minerNodeService := newGRPCMinerNodeService()
	grpcGatewayHandler := runtime.NewServeMux()

	minerGRPC.RegisterMinerServer(server, minerNodeService)
	_ = minerGRPC.RegisterMinerHandlerServer(context.Background(), grpcGatewayHandler, minerNodeService)

	// TODO i dont think this works, all requests will come to grpc gateway - check blobber
	http.Handle("/", grpcGatewayHandler)
}

type minerNodeGRPCService struct {
	minerGRPC.UnimplementedMinerServer
}

func (m *minerNodeGRPCService) WhoAmI(ctx context.Context, req *minerGRPC.WhoAmIRequest) (*minerGRPC.WhoAmIResponse, error) {

	var resp = &minerGRPC.WhoAmIResponse{}

	if Self != nil {
		var data = &strings.Builder{}
		Self.Underlying().Print(data)
		resp.Data = data.String()
	}

	return resp, nil
}

func newGRPCMinerNodeService() *minerNodeGRPCService {
	return &minerNodeGRPCService{}
}
