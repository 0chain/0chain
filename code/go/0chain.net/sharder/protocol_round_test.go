package sharder

import (
	"context"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

type roundMock struct {
	number int64
}

func (r roundMock) GetRoundNumber() int64 {
	return r.number
}

func (r roundMock) GetRandomSeed() int64 { panic("implement me") }

func (r roundMock) SetRandomSeed(seed int64, minersNum int) { panic("implement me") }

func (r roundMock) HasRandomSeed() bool { panic("implement me") }

func (r roundMock) GetTimeoutCount() int { panic("implement me") }

func (r roundMock) SetTimeoutCount(tc int) bool { panic("implement me") }

func (r roundMock) SetRandomSeedForNotarizedBlock(seed int64, minersNum int) { panic("implement me") }

func (r roundMock) IsRanksComputed() bool { panic("implement me") }

func (r roundMock) GetMinerRank(miner *node.Node) int { panic("implement me") }

func (r roundMock) GetMinersByRank(miners *node.Pool) []*node.Node { panic("implement me") }

func (r roundMock) AddProposedBlock(b *block.Block) (*block.Block, bool) { panic("implement me") }

func (r roundMock) GetProposedBlocks() []*block.Block { panic("implement me") }

func (r roundMock) AddNotarizedBlock(b *block.Block) (*block.Block, bool) {
	if len(b.Hash) != 64 {
		return nil, false
	}

	return nil, true
}

func (r roundMock) GetNotarizedBlocks() []*block.Block {
	return []*block.Block{
		block.NewBlock("", 1),
		block.NewBlock("", 1),
	}
}

func (r roundMock) GetHeaviestNotarizedBlock() *block.Block { panic("implement me") }

func (r roundMock) GetBestRankedNotarizedBlock() *block.Block { panic("implement me") }

func (r roundMock) Finalize(b *block.Block) { panic("implement me") }

func (r roundMock) IsFinalizing() bool {
	return true
}

func (r roundMock) SetFinalizing() bool { panic("implement me") }

func (r roundMock) IsFinalized() bool {
	return true
}

func (r roundMock) Clear() { panic("implement me") }

func (r roundMock) GetState() int { panic("implement me") }

func (r roundMock) SetState(state int) { panic("implement me") }

func (r roundMock) AddVRFShare(share *round.VRFShare, threshold int) bool { panic("implement me") }

func (r roundMock) GetVRFShares() map[string]*round.VRFShare { panic("implement me") }

var _ round.RoundI = (*roundMock)(nil)

func TestChain_AddNotarizedBlock(t *testing.T) {
	sc := GetSharderChain()
	sc.BlocksToSharder = chain.FINALIZED

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

	r := roundMock{number: 1}

	sc.AddRound(r)
	sc.AddRound(roundMock{number: 0})

	n := node.NewPool(1)

	sc.LatestFinalizedMagicBlock = &block.Block{
		MagicBlock: &block.MagicBlock{
			Miners:   n,
			Sharders: n,
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
		r   round.RoundI
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test_Chain_AddNotarizedBlock_TRUE",
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
				r:   r,
				b:   b,
			},
			want: true,
		},
		{
			name: "Test_Chain_AddNotarizedBlock_FALSE",
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
				b.Hash = encryption.Hash("data")[:62]

				return args{
					ctx: common.GetRootContext(),
					r:   roundMock{},
					b:   b,
				}
			}(),
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
			if got := sc.AddNotarizedBlock(tt.args.ctx, tt.args.r, tt.args.b); got != tt.want {
				t.Errorf("AddNotarizedBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}
