#!/usr/bin/env bash

protoc -I ./code/go/0chain.net/miner/minerGRPC/proto --go-grpc_out=. --go_out=. --grpc-gateway_out=. --openapiv2_out=./code/go/0chain.net/miner/openapi ./code/go/0chain.net/miner/minerGRPC/proto/miner.proto