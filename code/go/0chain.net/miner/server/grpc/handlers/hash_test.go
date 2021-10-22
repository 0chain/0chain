package handlers

import (
	"context"
	"reflect"
	"testing"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

func newGRPCMinerService() *minerGRPCService {
	return &minerGRPCService{}
}

func Test_minerGRPCService_Hash(t *testing.T) {
	type fields struct {
		UnimplementedMinerServiceServer minerproto.UnimplementedMinerServiceServer
	}
	type args struct {
		ctx context.Context
		req *minerproto.HashRequest
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *httpbody.HttpBody
		wantErr bool
	}{
		//TOOD: Add test cases.
		{
			name: "Test_minerGRPCService_Hash",
			fields: fields{
				UnimplementedMinerServiceServer: minerproto.UnimplementedMinerServiceServer{},
			},
			args: args{
				ctx: context.Background(),
				req: &minerproto.HashRequest{
					Text: "test",
				},
			},
			want: &httpbody.HttpBody{
				Data: []byte("test"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &minerGRPCService{
				UnimplementedMinerServiceServer: tt.fields.UnimplementedMinerServiceServer,
			}
			got, err := m.Hash(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("minerGRPCService.Hash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("minerGRPCService.Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}
