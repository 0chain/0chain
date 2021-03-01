package sharder_test

import (
	"context"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/sharder"
)

func TestNewBlockSummaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *sharder.BlockSummaries
	}{
		{
			name: "Test_NewBlockSummaries_OK",
			want: datastore.GetEntityMetadata("block_summaries").Instance().(*sharder.BlockSummaries),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sharder.NewBlockSummaries(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBlockSummaries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockSummariesProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "Test_BlockSummariesProvider_OK",
			want: &sharder.BlockSummaries{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sharder.BlockSummariesProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BlockSummariesProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockBySummary(t *testing.T) {
	sc := sharder.GetSharderChain()

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")
	sc.AddBlock(b)

	cacheB := block.NewBlock("", 1)
	cacheB.Hash = encryption.Hash("another data")
	if err := sc.BlockTxnCache.Add(cacheB.Hash, cacheB); err != nil {
		t.Fatal(err)
	}

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
			name: "Test_Chain_GetBlockBySummary_OK",
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
			args:    args{bs: &block.BlockSummary{Hash: b.Hash}},
			want:    b,
			wantErr: false,
		},
		{
			name: "Test_Chain_Unknown_Block_ERR",
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
			args:    args{bs: &block.BlockSummary{Hash: b.Hash[:62]}},
			wantErr: true,
		},
		{
			name: "Test_Chain_Block_Not_Available_ERR",
			fields: func() fields {
				sCh := *sc.Chain
				conf := *sCh.Config
				conf.NumReplicators = 1
				sCh.Config = &conf

				return fields{
					Chain:          &sCh,
					BlockChannel:   sc.BlockChannel,
					RoundChannel:   sc.RoundChannel,
					BlockCache:     sc.BlockCache,
					BlockTxnCache:  sc.BlockTxnCache,
					SharderStats:   sc.SharderStats,
					BlockSyncStats: sc.BlockSyncStats,
					TieringStats:   sc.TieringStats,
				}
			}(),
			args:    args{bs: &block.BlockSummary{Hash: b.Hash[:62]}},
			wantErr: true,
		},
		{
			name: "Test_Chain_Block_From_Cache_OK",
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
			args:    args{bs: &block.BlockSummary{Hash: cacheB.Hash}},
			want:    cacheB,
			wantErr: false,
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
			got, err := sc.GetBlockBySummary(tt.args.ctx, tt.args.bs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockBySummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != nil && !reflect.DeepEqual(got.Hash, tt.want.Hash) {
				t.Errorf("GetBlockBySummary() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockFromHash(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.HashBlock()

	sharder.GetSharderChain().AddBlock(b)

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
			fields:  fields{Chain: sharder.GetSharderChain().Chain},
			args:    args{hash: b.Hash},
			want:    b,
			wantErr: false,
		},
		{
			name:    "Test_Chain_GetBlockFromHash_Unknown_Hash_OK",
			fields:  fields{Chain: sharder.GetSharderChain().Chain},
			args:    args{hash: encryption.Hash("another data")[:62]},
			wantErr: true,
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

	invB := block.NewBlock("", 2)
	invB.Hash = encryption.Hash("data")[:62]

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
		{
			name:    "Test_Chain_StoreBlockSummaryFromBlock_Invalid_Key_Size_ERR",
			args:    args{ctx: common.GetRootContext(), b: invB},
			wantErr: true,
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
			if err := sc.StoreBlockSummaryFromBlock(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("StoreBlockSummaryFromBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_StoreBlockSummary(t *testing.T) {
	t.Parallel()

	bs := datastore.GetEntityMetadata("block_summary").Instance().(*block.BlockSummary)
	bs.Hash = encryption.Hash("data")

	invBS := datastore.GetEntityMetadata("block_summary").Instance().(*block.BlockSummary)
	invBS.Hash = encryption.Hash("data")[:62]

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
		{
			name:    "TestChain_StoreBlockSummary_Invalid_Hash_Size_ERR",
			args:    args{ctx: common.GetRootContext(), bs: invBS},
			wantErr: true,
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
			if err := sc.StoreBlockSummary(tt.args.ctx, tt.args.bs); (err != nil) != tt.wantErr {
				t.Errorf("StoreBlockSummary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockSummaries_GetEntityMetadata(t *testing.T) {
	t.Parallel()

	type fields struct {
		IDField      datastore.IDField
		BSummaryList []*block.BlockSummary
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "Test_BlockSummaries_GetEntityMetadata_OK",
			want: datastore.GetEntityMetadata("block_summaries"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bs := &sharder.BlockSummaries{
				IDField:      tt.fields.IDField,
				BSummaryList: tt.fields.BSummaryList,
			}
			if got := bs.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockSummary(t *testing.T) {
	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")
	bs := b.GetSummary()
	if err := bs.GetEntityMetadata().GetStore().Write(common.GetRootContext(), bs); err != nil {
		t.Fatal()
	}

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
		ctx  context.Context
		hash string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.BlockSummary
		wantErr bool
	}{
		{
			name:    "Test_Chain_GetBlockSummary_OK",
			args:    args{ctx: nil, hash: encryption.Hash("data")},
			wantErr: false,
		},
		{
			name: "Test_Chain_GetBlockSummary_ERR",
			args: func() args {
				h := encryption.Hash("data")
				h = "!" + h
				return args{ctx: nil, hash: h}
			}(),
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
			_, err := sc.GetBlockSummary(tt.args.ctx, tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockSummary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_StoreMagicBlockMapFromBlock(t *testing.T) {
	sc := sharder.GetSharderChain()
	mbm := block.MagicBlockMapProvider().(*block.MagicBlockMap)
	mbm.SetKey("1")

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
		ctx context.Context
		mbm *block.MagicBlockMap
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_Chain_StoreMagicBlockMapFromBlock_OK",
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
				ctx: common.GetRootContext(),
				mbm: mbm,
			},
			wantErr: false,
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
			if err := sc.StoreMagicBlockMapFromBlock(tt.args.ctx, tt.args.mbm); (err != nil) != tt.wantErr {
				t.Errorf("StoreMagicBlockMapFromBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_GetMagicBlockMap(t *testing.T) {
	mbm := &block.MagicBlockMap{}
	mbm.Hash = encryption.Hash("mbm data")
	mbm.ID = "1"
	if err := mbm.GetEntityMetadata().GetStore().Write(common.GetRootContext(), mbm); err != nil {
		t.Fatal(err)
	}

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
		ctx              context.Context
		magicBlockNumber string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.MagicBlockMap
		wantErr bool
	}{
		{
			name:    "Test_Chain_GetMagicBlockMap_OK",
			args:    args{ctx: common.GetRootContext(), magicBlockNumber: mbm.ID},
			want:    mbm,
			wantErr: false,
		},
		{
			name:    "Test_Chain_GetMagicBlockMap_ERR",
			args:    args{ctx: common.GetRootContext(), magicBlockNumber: "!"},
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
			got, err := sc.GetMagicBlockMap(tt.args.ctx, tt.args.magicBlockNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMagicBlockMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMagicBlockMap() got = %v, want %v", got, tt.want)
			}
		})
	}
}
