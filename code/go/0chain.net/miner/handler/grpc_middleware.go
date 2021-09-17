package handler

import (
	"context"
	"net/http"
	"time"

	"0chain.net/core/logging"
	"github.com/gorilla/mux"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

const (
	TimeoutSeconds = 10 // to set deadline for requests
)

func NewGRPCServerWithMiddlewares(r *mux.Router) *grpc.Server {
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

	registerGRPCServices(r, srv)

	// adds grpc-web middleware
	wrappedServer := grpcweb.WrapServer(srv)
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if wrappedServer.IsGrpcWebRequest(r) {
				wrappedServer.ServeHTTP(w, r)
				return
			}
			h.ServeHTTP(w, r)
		})
	})

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
