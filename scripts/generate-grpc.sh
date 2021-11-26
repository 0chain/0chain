#!/usr/bin/env bash

protoc -I ./code/go/0chain.net/miner/minergrpc/proto \
   --go_out ./code/go/0chain.net/miner/minergrpc \
   --go_opt paths=source_relative \
   --go-grpc_out ./code/go/0chain.net/miner/minergrpc \
   --go-grpc_opt paths=source_relative \
   --grpc-gateway_out=allow_delete_body=true:. \
   --grpc-gateway_out ./code/go/0chain.net/miner/minergrpc \
   --grpc-gateway_opt paths=source_relative \
   --openapiv2_opt allow_delete_body=true \
   --openapiv2_out=./code/go/0chain.net/miner/openapi \
   ./code/go/0chain.net/miner/minergrpc/proto/*.proto