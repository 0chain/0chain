package minerGRPC

import (
	"context"
	"time"

	"0chain.net/core/logging"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ratelimit "github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
)

const TIMEOUT_SECONDS = 10

func NewServerWithMiddlewares(limiter grpc_ratelimit.Limiter) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpc_zap.StreamServerInterceptor(logging.Logger),
			grpc_recovery.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			grpc_zap.UnaryServerInterceptor(logging.Logger),
			grpc_recovery.UnaryServerInterceptor(),
			//unaryDatabaseTransactionInjector(),
			grpc_ratelimit.UnaryServerInterceptor(limiter),
			unaryTimeoutInterceptor(), // should always be the lastest, to be "innermost"
		),
	)
}
func unaryTimeoutInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		deadline := time.Now().Add(time.Duration(TIMEOUT_SECONDS * time.Second))
		ctx, canceler := context.WithDeadline(ctx, deadline)
		defer canceler()

		return handler(ctx, req)
	}
}
