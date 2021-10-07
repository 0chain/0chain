package server

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

const (
	TimeoutSeconds = 10 // to set deadline for requests
)

func unaryTimeoutInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		deadline := time.Now().Add(TimeoutSeconds * time.Second)
		ctx, canceler := context.WithDeadline(ctx, deadline)
		defer canceler()

		return handler(ctx, req)
	}
}
