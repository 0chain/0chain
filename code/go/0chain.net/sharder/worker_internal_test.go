package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"context"
	"testing"
)

func TestChain_hasBlockTransactions(t *testing.T) {
	// case 1
	b := block.NewBlock("", 1)
	b.Txns = append(b.Txns, &transaction.Transaction{})

	// case 2
	b2 := block.NewBlock("", 1)
	txn := &transaction.Transaction{}
	txn.Hash = encryption.Hash("data")
	b2.Txns = append(b2.Txns, txn)

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
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TestChain_hasBlockTransactions_FALSE",
			args: args{
				ctx: common.GetRootContext(),
				b:   b,
			},
			want: false,
		},
		{
			name: "TestChain_hasBlockTransactions_TRUE",
			args: args{
				ctx: common.GetRootContext(),
				b:   b2,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if got := sc.hasBlockTransactions(tt.args.ctx, tt.args.b); got != tt.want {
				t.Errorf("hasBlockTransactions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_getPruneCountRoundStorage(t *testing.T) {
	sc := GetSharderChain()

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
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "TestChain_getPruneCountRoundStorage_OK",
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			f := sc.getPruneCountRoundStorage()
			if got := f(sc.MagicBlockStorage); got != chain.DefaultCountPruneRoundStorage {
				t.Errorf("getPruneCountRoundStorage() = %v, want %v", got, chain.DefaultCountPruneRoundStorage)
			}
			if got := f(round.NewRoundStartingStorage()); got != chain.DefaultCountPruneRoundStorage {
				t.Errorf("getPruneCountRoundStorage() = %v, want %v", got, chain.DefaultCountPruneRoundStorage)
			}
		})
	}
}
