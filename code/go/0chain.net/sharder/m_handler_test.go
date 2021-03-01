package sharder_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/sharder"
)

func TestLatestFinalizedBlockHandler(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")
	sc := sharder.GetSharderChain()
	sc.LatestFinalizedBlock = b

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

			sc := sharder.GetSharderChain()
			sc.LatestFinalizedBlock = b

			got, err := sharder.LatestFinalizedBlockHandler(tt.args.ctx, tt.args.r)
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

func TestFinalizedBlockHandler(t *testing.T) {
	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")
	b2 := block.NewBlock("", 3)
	b2.Hash = encryption.Hash("data")[:62]
	b3 := block.NewBlock("", 4)
	b3.Hash = encryption.Hash("another data")

	lfb := block.NewBlock("", 2)
	sc := sharder.GetSharderChain()
	sc.LatestFinalizedBlock = lfb

	sc.AddBlock(b3)

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name          string
		args          args
		want          interface{}
		wantErr       bool
		wantChSending bool
	}{
		{
			name:    "Test_FinalizedBlockHandler_OK",
			args:    args{ctx: nil, entity: b},
			want:    true,
			wantErr: false,
		},
		{
			name:    "Test_FinalizedBlockHandler_Not_A_Block_Entity_ERR",
			args:    args{ctx: nil, entity: round.NewRound(1)},
			wantErr: true,
		},
		{
			name:          "Test_FinalizedBlockHandler_New_Block_OK",
			args:          args{ctx: nil, entity: b2},
			want:          true,
			wantChSending: true,
			wantErr:       false,
		},
		{
			name:    "Test_FinalizedBlockHandler_Existing_Block_OK",
			args:    args{ctx: nil, entity: b3},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantChSending {
				bCh := sharder.GetSharderChain().GetBlockChannel()

				go func() {
					<-bCh
				}()
			}

			got, err := sharder.FinalizedBlockHandler(tt.args.ctx, tt.args.entity)

			if (err != nil) != tt.wantErr {
				t.Errorf("FinalizedBlockHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FinalizedBlockHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotarizedBlockKickHandler(t *testing.T) {
	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")
	b2 := block.NewBlock("", 3)
	b2.Hash = encryption.Hash("data")[:62]
	b3 := block.NewBlock("", 3)
	b3.Hash = encryption.Hash("another data")

	lfb := block.NewBlock("", 2)
	sc := sharder.GetSharderChain()
	sc.LatestFinalizedBlock = lfb

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name          string
		args          args
		want          interface{}
		wantErr       bool
		wantChSending bool
	}{
		{
			name:    "Test_NotarizedBlockKickHandler_OK",
			args:    args{ctx: nil, entity: b},
			want:    true,
			wantErr: false,
		},
		{
			name:    "Test_NotarizedBlockKickHandler_Not_A_Block_Entity_ERR",
			args:    args{ctx: nil, entity: round.NewRound(1)},
			wantErr: true,
		},
		{
			name:          "Test_NotarizedBlockKickHandler_New_Block_OK",
			args:          args{ctx: nil, entity: b2},
			want:          true,
			wantChSending: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantChSending {
				bCh := sharder.GetSharderChain().GetBlockChannel()

				go func() {
					<-bCh
				}()
			}

			got, err := sharder.NotarizedBlockKickHandler(tt.args.ctx, tt.args.entity)

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
