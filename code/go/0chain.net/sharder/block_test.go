package sharder_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	dmocks "0chain.net/core/datastore/mocks"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/sharder"
	"0chain.net/sharder/blockstore"
	bsmocks "0chain.net/sharder/blockstore/mocks"
)

func init() {
	store := dmocks.NewStoreMock()
	sharder.SetupBlockSummaries()
	block.SetupBlockSummaryEntity(store)
	block.SetupEntity(store)
	block.SetupMagicBlockMapEntity(store)
	blockstore.SetupStore(bsmocks.NewBlockStoreMock())
	round.SetupEntity(store)
	common.SetupRootContext(context.TODO())
}

const (
	roundDataDir    = "round"
	blockDataDir    = "block"
	roundSummaryDir = "roundSummary"
	blockSummaryDir = "blockSummary"
)

func initDBs(t *testing.T) (closeAndClear func()) {
	dbDir, err := ioutil.TempDir("", "dbs")
	require.NoError(t, err)

	blockDir := filepath.Join(dbDir, blockDataDir)
	require.NoError(t, err)
	err = os.MkdirAll(blockDir, 0700)
	require.NoError(t, err)

	roundDir := filepath.Join(dbDir, roundDataDir)
	err = os.MkdirAll(roundDir, 0700)
	require.NoError(t, err)

	rsDir := filepath.Join(dbDir, roundSummaryDir)
	err = os.MkdirAll(rsDir, 0700)
	require.NoError(t, err)

	bsDir := filepath.Join(dbDir, blockSummaryDir)
	err = os.MkdirAll(bsDir, 0700)
	require.NoError(t, err)

	rDB, err := ememorystore.CreateDB(roundDir)
	require.NoError(t, err)

	ememorystore.AddPool(round.Provider().GetEntityMetadata().GetDB(), rDB)

	bDB, err := ememorystore.CreateDB(blockDir)
	require.NoError(t, err)

	ememorystore.AddPool(block.Provider().GetEntityMetadata().GetDB(), bDB)

	rsDB, err := ememorystore.CreateDB(rsDir)
	require.NoError(t, err)

	ememorystore.AddPool("roundsummarydb", rsDB)

	bsDB, err := ememorystore.CreateDB(bsDir)
	require.NoError(t, err)

	ememorystore.AddPool(block.BlockSummaryProvider().GetEntityMetadata().GetDB(), bsDB)

	closeAndClear = func() {
		rDB.Close()
		bDB.Close()
		rsDB.Close()
		bsDB.Close()

		err = os.RemoveAll(dbDir)
		require.NoError(t, err)
	}

	return
}

func makeTestChain(t *testing.T) *sharder.Chain {
	ch, ok := chain.Provider().(*chain.Chain)
	if !ok {
		t.Fatal("types missmatching")
	}
	conf := chain.NewConfigImpl(&chain.ConfigData{BlockSize: 1024})
	ch.ChainConfig = conf
	ch.Initialize()
	sharder.SetupSharderChain(ch)
	chain.SetServerChain(ch)
	return sharder.GetSharderChain()
}

func TestNewBlockSummaries(t *testing.T) {
	want, ok := datastore.GetEntityMetadata("block_summaries").Instance().(*sharder.BlockSummaries)
	if !ok {
		t.Fatal("types missmatching")
	}

	tests := []struct {
		name string
		want *sharder.BlockSummaries
	}{
		{
			name: "Test_NewBlockSummaries_OK",
			want: want,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := sharder.NewBlockSummaries(); !reflect.DeepEqual(got, tt.want) {
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
			want: &sharder.BlockSummaries{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := sharder.BlockSummariesProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BlockSummariesProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockBySummary(t *testing.T) {
	b := block.NewBlock("", 1)
	b.HashBlock()

	makeTestChain(t)
	chain.GetServerChain().AddBlock(b)

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
			name:    "Test_Chain_GetBlockBySummary_OK",
			fields:  fields{Chain: sharder.GetSharderChain().Chain},
			args:    args{bs: &block.BlockSummary{Hash: b.Hash}},
			want:    b,
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
	b := block.NewBlock("", 1)
	b.HashBlock()
	makeTestChain(t)
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
	}
	for _, tt := range tests {
		tt := tt
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
	cl := initDBs(t)
	defer cl()

	ctx := context.WithValue(context.TODO(), node.SelfNodeKey, node.Self)

	b := block.NewBlock("", 1)
	b.HashBlock()

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
			args:    args{ctx: ctx, b: b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
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
			if err := sc.StoreBlockSummaryFromBlock(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("StoreBlockSummaryFromBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_StoreBlockSummary(t *testing.T) {
	cl := initDBs(t)
	defer cl()

	bs := datastore.GetEntityMetadata("block_summary").Instance().(*block.BlockSummary)
	bs.Hash = encryption.Hash("data")

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
	}
	for _, tt := range tests {
		tt := tt
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
			if err := sc.StoreBlockSummary(tt.args.ctx, tt.args.bs); (err != nil) != tt.wantErr {
				t.Errorf("StoreBlockSummary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
