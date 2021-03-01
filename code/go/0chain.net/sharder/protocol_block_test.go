package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"context"
	"encoding/hex"
	"go.uber.org/zap"
	"net/url"
	"reflect"
	"testing"
)

func TestChain_ViewChange(t *testing.T) {
	sc := GetSharderChain()
	b := block.NewBlock("", 1)

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
			name: "Test_Chain_ViewChange_OK",
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
			args:    args{ctx: common.GetRootContext(), b: b},
			wantErr: false,
		},
		{
			name: "Test_Chain_ViewChange_OK2",
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
			args: func() args {
				b := block.NewBlock("", 1)
				b.MagicBlock = &block.MagicBlock{}
				b.MagicBlock.Miners = node.NewPool(node.NodeTypeMiner)
				b.MagicBlock.Sharders = node.NewPool(node.NodeTypeSharder)
				pk, _, err := encryption.GenerateKeys()
				if err != nil {
					t.Fatal(err)
				}
				pkB, err := hex.DecodeString(pk)
				if err != nil {
					t.Error(err)
				}
				nc := map[interface{}]interface{}{
					"type":       int8(1),
					"public_ip":  "",
					"n2n_ip":     "",
					"port":       123,
					"public_key": pk,
					"id":         encryption.Hash(pkB),
				}
				n, err := node.NewNode(nc)
				if err != nil {
					t.Fatal(err)
				}
				b.Miners.NodesMap = map[string]*node.Node{
					"node": n,
				}

				return args{ctx: common.GetRootContext(), b: b}
			}(),
			wantErr: false,
		},
		{
			name: "Test_Chain_ViewChange_ERR",
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
			args: func() args {
				b := block.NewBlock("", 1)
				b.MagicBlock = &block.MagicBlock{}

				return args{ctx: common.GetRootContext(), b: b}
			}(),
			wantErr: true,
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
			if err := sc.ViewChange(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("ViewChange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_AfterFetch(t *testing.T) {
	sc := GetSharderChain()
	sc.LatestFinalizedBlock = block.NewBlock("", 1)
	b := block.NewBlock("", 2)

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
			name: "Test_Chain_AfterFetch_OK",
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
			args:    args{ctx: common.GetRootContext(), b: b},
			wantErr: false,
		},
		{
			name: "Test_Chain_AfterFetch_ERR",
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
			args: func() args {
				b := block.NewBlock("", 2)
				b.LatestFinalizedMagicBlockRound = 1

				return args{ctx: common.GetRootContext(), b: b}
			}(),
			wantErr: true,
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
			if err := sc.AfterFetch(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("AfterFetch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_processBlock(t *testing.T) {
	sc := GetSharderChain()

	sc.LatestFinalizedBlock = block.NewBlock("", 0)

	// case 1
	b := block.NewBlock("", 1)

	// case 2
	var (
		r2 = round.NewRound(2)
		h2 = encryption.Hash("data")
		b2 = block.NewBlock("", r2.Number)
	)
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	if err := sc.MagicBlockStorage.Put(mb, r2.Number); err != nil {
		t.Fatal(err)
	}

	sign, err := encryption.Sign(prK, h2)
	if err != nil {
		t.Fatal(err)
	}
	b2.VerificationTickets = []*block.VerificationTicket{
		&block.VerificationTicket{
			VerifierID: encryption.Hash(pbK),
			Signature:  sign,
		},
	}

	b2.Hash = h2

	// case 3
	var (
		r3 = round.NewRound(3)
		b3 = block.NewBlock("", r3.Number)
	)
	pbK, prK, err = encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb3, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	if err := sc.MagicBlockStorage.Put(mb3, r3.Number); err != nil {
		t.Fatal(err)
	}

	b3.MagicBlock = mb
	mn := mb3.Miners.NodesMap[encryption.Hash(pbK)]
	node.RegisterNode(mn)
	b3.MinerID = mn.ID

	b3.VerificationTickets = []*block.VerificationTicket{
		&block.VerificationTicket{
			VerifierID: encryption.Hash(pbK),
		},
	}
	b3.PrevBlock = b2
	b3.Hash = b3.ComputeHash()
	sign, err = encryption.Sign(prK, b3.Hash)
	if err != nil {
		t.Fatal(err)
	}
	b3.VerificationTickets[0].Signature = sign

	b3.Signature, err = encryption.Sign(prK, b3.Hash)
	if err != nil {
		t.Fatal(err)
	}

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
	}{
		{
			name: "Test_Chain_processBlock_No_Tickets_OK",
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
			args: args{ctx: common.GetRootContext(), b: b},
		},
		{
			name: "Test_Chain_processBlock_OK",
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
			args: func() args {
				b := block.NewBlock("", 1)
				b.VerificationTickets = []*block.VerificationTicket{
					&block.VerificationTicket{},
				}

				return args{ctx: common.GetRootContext(), b: b}
			}(),
		},
		{
			name: "Test_Chain_processBlock_Err_Block_Validation",
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
			args: args{ctx: common.GetRootContext(), b: b2},
		},
		{
			name: "Test_Chain_processBlock_No_Err",
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
			args: args{ctx: common.GetRootContext(), b: b3},
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

			sc.processBlock(tt.args.ctx, tt.args.b)
		})
	}
}

func TestChain_syncBlockSummary(t *testing.T) {
	sc := GetSharderChain()
	r := round.NewRound(2000)
	r.BlockHash = encryption.Hash("some data")

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
		ctx        context.Context
		r          *round.Round
		roundRange int64
		scan       HealthCheckScan
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.BlockSummary
	}{
		{
			name: "Test_Chain_syncBlockSummary_OK",
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
				ctx:        common.GetRootContext(),
				r:          r,
				roundRange: 2000,
				scan:       ProximityScan,
			},
			want: nil,
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
			if got := sc.syncBlockSummary(tt.args.ctx, tt.args.r, tt.args.roundRange, tt.args.scan); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("syncBlockSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_requestBlock(t *testing.T) {
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
	type args struct {
		ctx context.Context
		r   *round.Round
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		{
			name: "Test_chain_requestBlock_OK",
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
				r:   round.NewRound(1),
			},
			want: nil,
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
			if got := sc.requestBlock(tt.args.ctx, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("requestBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_syncBlock(t *testing.T) {
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
	type args struct {
		ctx      context.Context
		r        *round.Round
		canShard bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		{
			name: "TestChain_syncBlock_OK",
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
				ctx:      common.GetRootContext(),
				r:        round.NewRound(1),
				canShard: false,
			},
			want: nil,
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
			if got := sc.syncBlock(tt.args.ctx, tt.args.r, tt.args.canShard); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("syncBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_storeRoundSummaries(t *testing.T) {
	sc := GetSharderChain()
	rs := &RoundSummaries{
		RSummaryList: []*round.Round{
			round.NewRound(8000),
			nil,
		},
	}

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
		rs  *RoundSummaries
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_storeRoundSummaries_OK",
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
				rs:  rs,
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

			sc.storeRoundSummaries(tt.args.ctx, tt.args.rs)
		})
	}
}

func TestChain_storeBlockSummaries(t *testing.T) {
	sc := GetSharderChain()
	bs := &BlockSummaries{
		BSummaryList: []*block.BlockSummary{
			block.NewBlock("", 1).GetSummary(),
			nil,
		},
	}

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
		bs  *BlockSummaries
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_storeBlockSummaries_OK",
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
				bs:  bs,
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

			sc.storeBlockSummaries(tt.args.ctx, tt.args.bs)
		})
	}
}

func TestChain_storeRoundSummary(t *testing.T) {
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
	type args struct {
		ctx context.Context
		r   *round.Round
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_storeRoundSummary_OK",
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
				r:   round.NewRound(1),
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

			sc.storeRoundSummary(tt.args.ctx, tt.args.r)
		})
	}
}

func TestChain_storeBlockSummary(t *testing.T) {
	sc := GetSharderChain()
	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

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
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_storeBlockSummary_OK",
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
				bs:  b.GetSummary(),
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

			sc.storeBlockSummary(tt.args.ctx, tt.args.bs)
		})
	}
}

func TestChain_storeBlock(t *testing.T) {
	sc := GetSharderChain()
	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

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
			name: "Test_Chain_storeBlock_OK",
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
				b:   b,
			},
			wantErr: false,
		},
		{
			name: "Test_Chain_storeBlock_ERR",
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
			args: func() args {
				b := block.NewBlock("", 3000)
				b.Hash = encryption.Hash("data")[:62]

				return args{
					ctx: common.GetRootContext(),
					b:   b,
				}
			}(),
			wantErr: true,
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
			if err := sc.storeBlock(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("storeBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_requestForRound(t *testing.T) {
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
	type args struct {
		ctx    context.Context
		params *url.Values
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *round.Round
	}{
		{
			name: "Test_Chain_requestForRound_OK",
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
			args: args{ctx: common.GetRootContext(), params: &url.Values{}},
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
			if got := sc.requestForRound(tt.args.ctx, tt.args.params); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("requestForRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_isValidRound(t *testing.T) {
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
		r *round.Round
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test_Chain_isValidRound_TRUE",
			args: func() args {
				r := round.NewRound(1)
				r.BlockHash = encryption.Hash("data")
				return args{r: r}
			}(),
			want: true,
		},
		{
			name: "Test_Chain_isValidRound_Nil_Round_FALSE",
			args: args{r: nil},
			want: false,
		},
		{
			name: "Test_Chain_isValidRound_Neg_Num_FALSE",
			args: args{r: round.NewRound(-1)},
			want: false,
		},
		{
			name: "Test_Chain_isValidRound_Nil_Hash_FALSE",
			args: args{r: round.NewRound(1)},
			want: false,
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
			if got := sc.isValidRound(tt.args.r); got != tt.want {
				t.Errorf("isValidRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_UpdatePendingBlock(t *testing.T) {
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
		ctx  context.Context
		b    *block.Block
		txns []datastore.Entity
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: " Test_Chain_UpdatePendingBlock_OK", // empty func
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

			sc.UpdatePendingBlock(tt.args.ctx, tt.args.b, tt.args.txns)
		})
	}
}

func TestChain_NotarizedBlockFetched(t *testing.T) {
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
	}{
		{
			name: "Test_Chain_NotarizedBlockFetched_OK", // empty func
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

			sc.NotarizedBlockFetched(tt.args.ctx, tt.args.b)
		})
	}
}

func TestChain_UpdateFinalizedBlock(t *testing.T) {
	sc := GetSharderChain()

	var (
		b = block.NewBlock("", -1)
		h = encryption.Hash("data")

		lfb = b
	)

	b.Hash = h[:62]
	b.Txns = append(b.Txns, &transaction.Transaction{})
	debugTxn := &transaction.Transaction{}
	debugTxn.TransactionData = "debug"
	b.Txns = append(b.Txns, debugTxn)

	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	b.MagicBlock = mb

	// sc.AddRound(r)
	sc.LatestFinalizedBlock = lfb

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
	}{
		{
			name: "Test_Chain_UpdateFinalizedBlock_OK",
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
				b:   b,
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
			sc.UpdateFinalizedBlock(tt.args.ctx, tt.args.b)
		})
	}
}

func TestChain_pullRelatedMagicBlock_ERR(t *testing.T) {
	sc := GetSharderChain()
	logging.Logger = zap.NewNop()

	b := block.NewBlock("", 1)
	b.LatestFinalizedMagicBlockRound = 1
	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	mb.Sharders.AddNode(node.Self.Underlying())

	b.MagicBlock = mb

	if err := sc.MagicBlockStorage.Put(mb, 1); err != nil {
		t.Fatal(err)
	}
	sc.LatestFinalizedMagicBlock = b

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
			name: "TestChain_pullRelatedMagicBlock_ERR",
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
				b:   b,
			},
			wantErr: true,
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
			if err := sc.pullRelatedMagicBlock(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("pullRelatedMagicBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_storeBlockTransactions(t *testing.T) {
	sc := GetSharderChain()

	b := block.NewBlock("", 1)
	b.Txns = []*transaction.Transaction{
		&transaction.Transaction{},
	}

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
			name: "Test_Chain_storeBlockTransactions_OK",
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
				b:   b,
			},
			wantErr: false,
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
			if err := sc.storeBlockTransactions(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("storeBlockTransactions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
