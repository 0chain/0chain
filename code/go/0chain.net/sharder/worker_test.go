package sharder_test

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/sharder"
	"0chain.net/sharder/blockstore"
	"0chain.net/smartcontract/minersc"
)

var _ blockstore.BlockStore = (*blockStoreMock)(nil)

func TestSetupWorkers(t *testing.T) {
	ctx := common.GetRootContext()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestSetupWorkers_OK",
			args: args{ctx: ctx},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverChain := chain.GetServerChain()
			serverChain.SetupWorkers(ctx)
			sharder.SetupWorkers(ctx)
			time.Sleep(time.Millisecond * 500)
		})
	}
}

func TestChain_BlockWorker(t *testing.T) {
	sc := sharder.GetSharderChain()
	ctx, cancel := context.WithCancel(common.GetRootContext())

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
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_BlockWorker_OK",
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
			args: args{ctx: ctx},
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

			go sc.BlockWorker(tt.args.ctx)
			go func() {
				sc.GetBlockChannel() <- block.NewBlock("", 1)
			}()
			time.Sleep(time.Millisecond * 200)
			cancel()
		})
	}
}

func TestChain_RegisterSharderKeepWorker(t *testing.T) {
	sc := sharder.GetSharderChain()
	ctx, cancel := context.WithCancel(common.GetRootContext())

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
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_RegisterSharderKeepWorker_OK",
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
			args: args{ctx: ctx},
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

			go func() {
				phCh := sc.PhaseEvents()
				phE := chain.PhaseEvent{
					Phase: minersc.PhaseNode{
						Phase:        0,
						StartRound:   0,
						CurrentRound: 0,
						Restarts:     0,
					},
					Sharders: false,
				}

				phCh <- phE

				phE.Phase.StartRound = 1
				phCh <- phE

				phE.Phase.StartRound = 1
				phE.Phase.Phase = 1
				phCh <- phE

				cancel()
			}()

			sc.RegisterSharderKeepWorker(tt.args.ctx)
		})
	}
}

func TestChain_MinioWorker(t *testing.T) {
	sc := sharder.GetSharderChain()
	sc.CurrentRound = 6
	ctx, cancel := context.WithCancel(common.GetRootContext())

	r := round.NewRound(3)
	r.BlockHash = encryption.Hash("data")
	sc.AddRound(r)
	r2 := round.NewRound(4)
	r2.BlockHash = encryption.Hash("data")[:62] // with invalid hash
	r3 := round.NewRound(5)
	r3.BlockHash = encryption.Hash("data")[:62] // with invalid hash
	sc.AddRound(r3)
	r4 := round.NewRound(6)
	r4.BlockHash = encryption.Hash("another data")
	sc.AddRound(r4)

	if err := blockstore.GetStore().UploadToCloud(r.BlockHash, r.Number); err != nil {
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
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		minioEnabled bool
	}{
		{
			name: "Test_Chain_MinioWorker_OK",
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
			args:         args{ctx: ctx},
			minioEnabled: true,
		},
		{
			name: "Test_Chain_MinioWorker_Minio_Disabled_OK",
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
			args:         args{ctx: ctx},
			minioEnabled: false,
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

			if tt.minioEnabled {
				viper.Set("minio.enabled", true)
			} else {
				viper.Set("minio.enabled", false)
			}

			go sc.MinioWorker(tt.args.ctx)
			if tt.minioEnabled {
				time.Sleep(time.Second)
				cancel()
			}
			time.Sleep(time.Millisecond * 200)
		})
	}
}

func TestChain_SharderHealthCheck(t *testing.T) {
	sc := sharder.GetSharderChain()
	ctx, cancel := context.WithCancel(common.GetRootContext())

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
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_SharderHealthCheck_OK",
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
			args: args{ctx: ctx},
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

			cancel()
			sc.SharderHealthCheck(tt.args.ctx)
		})
	}
}
