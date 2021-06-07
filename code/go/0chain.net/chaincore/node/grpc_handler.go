package node

import (
	"context"
	"strings"

	"0chain.net/core/encryption"

	"0chain.net/miner/minerGRPC"
)

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
