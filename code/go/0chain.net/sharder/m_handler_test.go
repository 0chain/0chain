package sharder_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/sharder"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestLatestFinalizedBlockHandler(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

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
			sc := &sharder.MockChainer{}
			sc.On("GetLatestFinalizedBlock").Return(b)
			sc.On("GetBlockChannel").Return(make(chan *block.Block, 1))
			sc.On("ForceFinalizeRound()")

			got, err := sharder.LatestFinalizedBlockHandler(sc)(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("LatestFinalizedBlockHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, got, tt.want)
		})
	}
}

func TestChain_AcceptMessage(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

	makeTestChain(t).AddBlock(b)

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
				Chain: sharder.GetSharderChain().Chain,
			},
			args: args{entityName: "block", entityID: b.Hash},
			want: false,
		},
		{
			name: "Test_Chain_AcceptMessage_Not_Existing_Block_TRUE",
			fields: fields{
				Chain: sharder.GetSharderChain().Chain,
			},
			args: args{entityName: "block", entityID: encryption.Hash("Test_Chain_AcceptMessage_Not_Existing_Block_TRUE")},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				blockChannel:   tt.fields.BlockChannel,
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

func TestNotarizedBlockHandler(t *testing.T) {
	t.Parallel()

	var (
		ctx = common.GetRootContext()
		b   = block.NewBlock("", 1)
	)

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		setLFB  bool
		wantErr bool
	}{
		{
			name:    "Test_NotarizedBlockHandler_From_Latest_Finalized_Block_TRUE",
			args:    args{ctx: ctx, entity: b},
			want:    true,
			setLFB:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &sharder.MockChainer{}
			sc.On("GetLatestFinalizedBlock").Return(b)
			sc.On("GetBlock", mock.Anything, mock.Anything).Return(b, nil)

			got, err := sharder.NotarizedBlockHandler(sc)(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("NotarizedBlockHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NotarizedBlockHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotarizedBlockKickHandler(t *testing.T) {
	t.Parallel()

	var (
		ctx = common.GetRootContext()
		b   = block.NewBlock("", 1)
	)

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		setLFB  bool
		wantErr bool
	}{
		{
			name:    "Test_NotarizedBlockKickHandler_From_Latest_Finalized_Block_TRUE",
			args:    args{ctx: ctx, entity: b},
			want:    true,
			setLFB:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &sharder.MockChainer{}
			sc.On("GetLatestFinalizedBlock").Return(b)
			sc.On("GetBlockChannel").Return(make(chan *block.Block, 1))
			sc.On("ForceFinalizeRound()")

			got, err := sharder.NotarizedBlockKickHandler(sc)(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("NotarizedBlockKickHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NotarizedBlockKickHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}
