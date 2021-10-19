package handlers_test

import (
	"context"
	"log"
	"net"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"0chain.net/miner/server/grpc/handlers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// dialer
func dialer() func(context.Context, string) (net.Conn, error) {
	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()

	minerproto.RegisterMinerServiceServer(srv, handlers.NewMinerGRPCService())

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
}
