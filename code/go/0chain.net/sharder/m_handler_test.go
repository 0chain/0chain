package sharder

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
)

func init() {
	memoryStorage := memorystore.GetStorageProvider()
	block.SetupEntity(memoryStorage)
}

func TestLatestFinalizedBlockHandler(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")
	serverChain := chain.NewChainFromConfig()
	serverChain.LatestFinalizedBlock = b
	SetupSharderChain(serverChain)

	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "Test_LatestFinalizedBlockHandler_OK",
			want:    b,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := LatestFinalizedBlockHandler(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("LatestFinalizedBlockHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LatestFinalizedBlockHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_AcceptMessage(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

	sharderChain.Chain = chain.Provider().(*chain.Chain)
	sharderChain.AddBlock(b)

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   Stats
		BlockSyncStats *SyncStats
		TieringStats   *MinioStats
	}
	type args struct {
		entityName string
		entityID   string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test_Chain_AcceptMessage_Empty_Entity_TRUE",
			args: args{entityName: ""},
			want: true,
		},
		{
			name: "Test_Chain_AcceptMessage_Existing_Block_FALSE",
			fields: fields{
				Chain: sharderChain.Chain,
			},
			args: args{entityName: "block", entityID: b.Hash},
			want: false,
		},
		{
			name: "Test_Chain_AcceptMessage_Not_Existing_Block_TRUE",
			fields: fields{
				Chain: sharderChain.Chain,
			},
			args: args{entityName: "block"},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if got := sc.AcceptMessage(tt.args.entityName, tt.args.entityID); got != tt.want {
				t.Errorf("AcceptMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
