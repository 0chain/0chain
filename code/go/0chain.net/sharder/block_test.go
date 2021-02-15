package sharder

import (
	"context"
	"os"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
)

func init() {
	SetupBlockSummaries()

	common.SetupRootContext(node.GetNodeContext())

	if err := os.MkdirAll("data/rocksdb/state", 0700); err != nil {
		panic(err)
	}

	memoryStorage := memorystore.GetStorageProvider()
	block.SetupBlockSummaryEntity(memoryStorage)
	block.SetupBlockSummaryDB()
	block.SetupBlockSummaryEntity(ememorystore.GetStorageProvider())

	logging.InitLogging("testing")
}

func TestNewBlockSummaries(t *testing.T) {
	tests := []struct {
		name string
		want *BlockSummaries
	}{
		{
			name: "Test_NewBlockSummaries_OK",
			want: datastore.GetEntityMetadata("block_summaries").Instance().(*BlockSummaries),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBlockSummaries(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBlockSummaries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockSummariesProvider(t *testing.T) {
	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "Test_BlockSummariesProvider_OK",
			want: &BlockSummaries{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BlockSummariesProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BlockSummariesProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockBySummary(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.HashBlock()

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
		ctx context.Context
		bs  *block.BlockSummary
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.Block
		wantErr bool
	}{
		{
			name:    "Test_Chain_GetBlockBySummary_OK",
			fields:  fields{Chain: sharderChain.Chain},
			args:    args{bs: &block.BlockSummary{Hash: b.Hash}},
			want:    b,
			wantErr: false,
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
			got, err := sc.GetBlockBySummary(tt.args.ctx, tt.args.bs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockBySummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockBySummary() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockFromHash(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.HashBlock()

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
		ctx      context.Context
		hash     string
		roundNum int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.Block
		wantErr bool
	}{
		{
			name:    "Test_Chain_GetBlockFromHash_OK",
			fields:  fields{Chain: sharderChain.Chain},
			args:    args{hash: b.Hash},
			want:    b,
			wantErr: false,
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
			got, err := sc.GetBlockFromHash(tt.args.ctx, tt.args.hash, tt.args.roundNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockFromHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockFromHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_StoreBlockSummaryFromBlock(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.HashBlock()

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
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_Chain_StoreBlockSummaryFromBlock_OK",
			args:    args{ctx: common.GetRootContext(), b: b},
			wantErr: false,
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
			if err := sc.StoreBlockSummaryFromBlock(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("StoreBlockSummaryFromBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_StoreBlockSummary(t *testing.T) {
	t.Parallel()

	bs := datastore.GetEntityMetadata("block_summary").Instance().(*block.BlockSummary)

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
		bs  *block.BlockSummary
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "TestChain_StoreBlockSummary_OK",
			args:    args{ctx: common.GetRootContext(), bs: bs},
			wantErr: false,
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
			if err := sc.StoreBlockSummary(tt.args.ctx, tt.args.bs); (err != nil) != tt.wantErr {
				t.Errorf("StoreBlockSummary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
