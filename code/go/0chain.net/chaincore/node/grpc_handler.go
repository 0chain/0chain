package node

import (
	"context"
	"net/http"
	"strings"

	"github.com/0chain/0chain/code/go/0chain.net/core/encryption"

	"github.com/0chain/0chain/code/go/0chain.net/miner/minerGRPC"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func RegisterGRPCMinerNodeService(server *grpc.Server) {
	minerNodeService := NewGRPCMinerNodeService(Self)
	grpcGatewayHandler := runtime.NewServeMux()

	minerGRPC.RegisterMinerNodeServer(server, minerNodeService)
	_ = minerGRPC.RegisterMinerNodeHandlerServer(context.Background(), grpcGatewayHandler, minerNodeService)

	// TODO i dont think this works, all requests will come to grpc gateway - check blobber
	http.Handle("/", grpcGatewayHandler)
}

type ISelfNode interface {
	Underlying() *Node
	GetSignatureScheme() encryption.SignatureScheme
	SetSignatureScheme(signatureScheme encryption.SignatureScheme)
	Sign(hash string) (string, error)
	TimeStampSignature() (string, string, string, error)
	IsEqual(node *Node) bool
	SetNodeIfPublicKeyIsEqual(node *Node)
}

func NewGRPCMinerNodeService(self ISelfNode) *minerNodeGRPCService {
	return &minerNodeGRPCService{
		self: self,
	}
}

type minerNodeGRPCService struct {
	self ISelfNode
	minerGRPC.UnimplementedMinerNodeServer
}

func (m *minerNodeGRPCService) WhoAmI(ctx context.Context, req *minerGRPC.WhoAmIRequest) (*minerGRPC.WhoAmIResponse, error) {

	var resp = &minerGRPC.WhoAmIResponse{}

	if m.self != nil {
		var data = &strings.Builder{}
		m.self.Underlying().Print(data)
		resp.Data = data.String()
	}

	return resp, nil
}
