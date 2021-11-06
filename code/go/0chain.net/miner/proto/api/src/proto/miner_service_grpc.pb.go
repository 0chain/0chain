// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// MinerServiceClient is the client API for MinerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MinerServiceClient interface {
	GetNotarizedBlock(ctx context.Context, in *GetNotarizedBlockRequest, opts ...grpc.CallOption) (*GetNotarizedBlockResponse, error)
	//
	GetWalletStats(ctx context.Context, in *GetWalletStatsRequest, opts ...grpc.CallOption) (*GetWalletStatsResponse, error)
}

type minerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMinerServiceClient(cc grpc.ClientConnInterface) MinerServiceClient {
	return &minerServiceClient{cc}
}

func (c *minerServiceClient) GetNotarizedBlock(ctx context.Context, in *GetNotarizedBlockRequest, opts ...grpc.CallOption) (*GetNotarizedBlockResponse, error) {
	out := new(GetNotarizedBlockResponse)
	err := c.cc.Invoke(ctx, "/miner.MinerService/GetNotarizedBlock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *minerServiceClient) GetWalletStats(ctx context.Context, in *GetWalletStatsRequest, opts ...grpc.CallOption) (*GetWalletStatsResponse, error) {
	out := new(GetWalletStatsResponse)
	err := c.cc.Invoke(ctx, "/miner.MinerService/GetWalletStats", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MinerServiceServer is the server API for MinerService service.
// All implementations should embed UnimplementedMinerServiceServer
// for forward compatibility
type MinerServiceServer interface {
	GetNotarizedBlock(context.Context, *GetNotarizedBlockRequest) (*GetNotarizedBlockResponse, error)
	//
	GetWalletStats(context.Context, *GetWalletStatsRequest) (*GetWalletStatsResponse, error)
}

// UnimplementedMinerServiceServer should be embedded to have forward compatible implementations.
type UnimplementedMinerServiceServer struct {
}

func (UnimplementedMinerServiceServer) GetNotarizedBlock(context.Context, *GetNotarizedBlockRequest) (*GetNotarizedBlockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetNotarizedBlock not implemented")
}
func (UnimplementedMinerServiceServer) GetWalletStats(context.Context, *GetWalletStatsRequest) (*GetWalletStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetWalletStats not implemented")
}

// UnsafeMinerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MinerServiceServer will
// result in compilation errors.
type UnsafeMinerServiceServer interface {
	mustEmbedUnimplementedMinerServiceServer()
}

func RegisterMinerServiceServer(s grpc.ServiceRegistrar, srv MinerServiceServer) {
	s.RegisterService(&MinerService_ServiceDesc, srv)
}

func _MinerService_GetNotarizedBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetNotarizedBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MinerServiceServer).GetNotarizedBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/miner.MinerService/GetNotarizedBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MinerServiceServer).GetNotarizedBlock(ctx, req.(*GetNotarizedBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MinerService_GetWalletStats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetWalletStatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MinerServiceServer).GetWalletStats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/miner.MinerService/GetWalletStats",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MinerServiceServer).GetWalletStats(ctx, req.(*GetWalletStatsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// MinerService_ServiceDesc is the grpc.ServiceDesc for MinerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MinerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "miner.MinerService",
	HandlerType: (*MinerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetNotarizedBlock",
			Handler:    _MinerService_GetNotarizedBlock_Handler,
		},
		{
			MethodName: "GetWalletStats",
			Handler:    _MinerService_GetWalletStats_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "miner_service.proto",
}
