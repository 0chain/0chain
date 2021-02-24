package sharder_test

import (
	"0chain.net/core/common"
	"context"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/sharder"
)

func TestChain_GetRoundChannel(t *testing.T) {
	t.Parallel()

	sc := sharder.GetSharderChain()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	tests := []struct {
		name   string
		fields fields
		want   chan *round.Round
	}{
		{
			name: "Test_Chain_GetRoundChannel_OK",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			want: sc.RoundChannel,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if got := sc.GetRoundChannel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRoundChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockChannel(t *testing.T) {
	t.Parallel()

	sc := sharder.GetSharderChain()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	tests := []struct {
		name   string
		fields fields
		want   chan *block.Block
	}{
		{
			name: "Test_Chain_GetBlockChannel_OK",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			want: sc.BlockChannel,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if got := sc.GetBlockChannel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockHash(t *testing.T) {
	sc := sharder.GetSharderChain()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	type args struct {
		ctx         context.Context
		roundNumber int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test_Chain_GetBlockHash_ERR",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			args: args{
				ctx:         common.GetRootContext(),
				roundNumber: -1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			got, err := sc.GetBlockHash(tt.args.ctx, tt.args.roundNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBlockHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}
