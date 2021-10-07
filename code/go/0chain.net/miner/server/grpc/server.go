package server

import (
	"0chain.net/core/logging"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"0chain.net/miner/server/grpc/handlers"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
)

func NewGRPCServerWithMiddlewares() *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpc_zap.StreamServerInterceptor(logging.Logger),
			grpc_recovery.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			grpc_zap.UnaryServerInterceptor(logging.Logger),
			grpc_recovery.UnaryServerInterceptor(),
			unaryTimeoutInterceptor(), // should always be the lastest, to be "innermost"
		),
	)

	minerproto.RegisterMinerServiceServer(srv, handlers.NewMinerGRPCService())

	return srv
}
