package gateway

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

// Run
func Run(dialAddr string) error {
	log := grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
	grpclog.SetLoggerV2(log)

	conn, err := grpc.DialContext(
		context.Background(),
		dialAddr,
		//grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(cert, "")),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	gwmux := runtime.NewServeMux()
	if err := minerproto.RegisterMinerServiceHandler(context.Background(), gwmux, conn); err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "11000"
	}

	gatewayAddr := net.JoinHostPort("0.0.0.0", port)

	gwServer := &http.Server{
		Addr: gatewayAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/") {
				gwmux.ServeHTTP(w, r)
				return
			}
		}),
	}
	// Empty parameters mean use the TLS Config specified with the server.
	if strings.ToLower(os.Getenv("SERVE_HTTP")) == "true" {
		log.Info("Serving gRPC-Gateway and OpenAPI Documentation on http://", gatewayAddr)
		return fmt.Errorf("serving gRPC-Gateway server: %w", gwServer.ListenAndServe())
	}

	log.Info("Serving gRPC-Gateway and OpenAPI Documentation on https://", gatewayAddr)
	return fmt.Errorf("serving gRPC-Gateway server: %w", gwServer.ListenAndServeTLS("", ""))
}
