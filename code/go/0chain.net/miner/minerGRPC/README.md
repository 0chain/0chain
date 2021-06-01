# GRPC Endpoints

## Development

Modify the '.proto' file in `code/go/0chain.net/miner/minerGRPC/proto/blobber.proto` and run 
`scripts/generate-grpc.sh` to add new api's.

## Installation

Install the [protoc](https://grpc.io/docs/protoc-installation/) command line interface.

```
go install \
github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
google.golang.org/protobuf/cmd/protoc-gen-go \
google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

Run this command to install all the GRPC related binaries required to generate GRPC related files using `protoc` CLI.

Now you can run the script in `scripts/generate-grpc.sh`.

## Plugins

* [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) 
plugin is being used to expose a REST api for grpc incompatible clients.

## Testing

The current grpc implementation supports server reflection in development environment.
You can interact with the api using https://github.com/gusaul/grpcox.

Make sure the server is running on `--deployment_mode 0` to use server reflection.

You can use https://github.com/vektra/mockery to generate mocks for tests.

## Documentation

Basic documentation can be found here - https://grpc.io/docs/languages/go/basics/.

Advanced documentation can be found here - https://github.com/grpc/grpc-go/tree/master/Documentation.


