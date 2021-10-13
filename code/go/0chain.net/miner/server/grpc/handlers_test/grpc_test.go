package handlers_test

import (
	minerproto "0chain.net/miner/proto/api/src/proto"
	"0chain.net/miner/server/grpc/handlers"
	"google.golang.org/grpc"
)

// makeTestClient
func makeTestClient(conn *grpc.ClientConn) (minerproto.MinerServiceClient, error) {
	client := minerproto.NewMinerServiceClient(conn)
	return client, nil
}

// makeTestServer
func makeTestServer() (minerproto.MinerServiceServer, error) {
	server := handlers.NewMinerGRPCService()
	return server, nil
}
