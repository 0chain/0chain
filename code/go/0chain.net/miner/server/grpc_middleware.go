package server

import (
	"context"
	"time"

	"0chain.net/core/logging"
	"0chain.net/miner/minergrpc"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
)

const (
	TimeoutSeconds = 10 // to set deadline for requests
)

func NewGRPCServerWithMiddlewares() *grpc.Server {
	srv := grpc.NewServer(
		//grpc.Creds(credentials.NewServerTLSFromCert(cert)),
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

	minergrpc.RegisterMinerServiceServer(srv, NewMinerGRPCService())

	return srv
}

func unaryTimeoutInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		deadline := time.Now().Add(TimeoutSeconds * time.Second)
		ctx, canceler := context.WithDeadline(ctx, deadline)
		defer canceler()

		return handler(ctx, req)
	}
}
