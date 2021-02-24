package sharder_test

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"context"
	"reflect"
	"testing"

	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/sharder"
)

func TestRoundSummariesProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "Test_RoundSummariesProvider_OK",
			want: &sharder.RoundSummaries{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sharder.RoundSummariesProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundSummariesProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRoundSummaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *sharder.RoundSummaries
	}{
		{
			name: "Test_NewRoundSummaries_OK",
			want: datastore.GetEntityMetadata("round_summaries").Instance().(*sharder.RoundSummaries),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sharder.NewRoundSummaries(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRoundSummaries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundSummaries_GetEntityMetadata(t *testing.T) {
	t.Parallel()

	type fields struct {
		IDField      datastore.IDField
		RSummaryList []*round.Round
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "Test_RoundSummaries_GetEntityMetadata_OK",
			want: datastore.GetEntityMetadata("round_summaries"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rs := &sharder.RoundSummaries{
				IDField:      tt.fields.IDField,
				RSummaryList: tt.fields.RSummaryList,
			}
			if got := rs.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewHealthyRound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *sharder.HealthyRound
	}{
		{
			name: "Test_NewHealthyRound_OK",
			want: datastore.GetEntityMetadata("healthy_round").Instance().(*sharder.HealthyRound),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sharder.NewHealthyRound(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHealthyRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthyRoundProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "Test_HealthyRoundProvider_OK",
			want: &sharder.HealthyRound{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sharder.HealthyRoundProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HealthyRoundProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthyRound_GetEntityMetadata(t *testing.T) {
	t.Parallel()

	type fields struct {
		IDField datastore.IDField
		Number  int64
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "Test_HealthyRound_GetEntityMetadata_OK",
			want: datastore.GetEntityMetadata("healthy_round"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &sharder.HealthyRound{
				IDField: tt.fields.IDField,
				Number:  tt.fields.Number,
			}
			if got := hr.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestHealthyRound_GetKey(t *testing.T) {
	t.Parallel()

	type fields struct {
		IDField datastore.IDField
		Number  int64
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name: "Test_HealthyRound_GetKey_OK",
			want: datastore.ToKey(datastore.GetEntityMetadata("healthy_round").GetName()),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hr := &sharder.HealthyRound{
				IDField: tt.fields.IDField,
				Number:  tt.fields.Number,
			}
			if got := hr.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSharderRoundFactory_CreateRoundF(t *testing.T) {
	t.Parallel()

	type args struct {
		roundNum int64
	}
	tests := []struct {
		name string
		args args
		want round.RoundI
	}{
		{
			name: "Test_SharderRoundFactory_CreateRoundF_OK",
			args: args{roundNum: 1},
			want: round.NewRound(1),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mrf := sharder.SharderRoundFactory{}
			if got := mrf.CreateRoundF(tt.args.roundNum); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateRoundF() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_StoreRound(t *testing.T) {
	t.Parallel()

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
		r   *round.Round
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_Chain_StoreRound_OK",
			args:    args{ctx: common.GetRootContext(), r: round.NewRound(1)},
			wantErr: false,
		},
		{
			name:    "Test_Chain_StoreRound_Write_ERR",
			args:    args{ctx: common.GetRootContext(), r: round.NewRound(-1)},
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
			if err := sc.StoreRound(tt.args.ctx, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("StoreRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_GetMostRecentRoundFromDB(t *testing.T) {
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
		name    string
		fields  fields
		args    args
		want    *round.Round
		wantErr bool
	}{
		{
			name:    "TestChain_GetMostRecentRoundFromDB_OK",
			args:    args{ctx: common.GetRootContext()},
			want:    datastore.GetEntityMetadata("round").Instance().(*round.Round),
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
			got, err := sc.GetMostRecentRoundFromDB(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMostRecentRoundFromDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMostRecentRoundFromDB() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_ReadHealthyRound(t *testing.T) {
	hr := datastore.GetEntity("healthy_round").(*sharder.HealthyRound)
	hr.ID = "healthy_round"

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
		name    string
		fields  fields
		args    args
		want    *sharder.HealthyRound
		wantErr bool
	}{
		{
			name:    "Test_Chain_ReadHealthyRound_JSON_Input_ERR",
			args:    args{ctx: common.GetRootContext()},
			want:    hr,
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
			got, err := sc.ReadHealthyRound(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadHealthyRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadHealthyRound() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_WriteHealthyRound(t *testing.T) {
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
		hr  *sharder.HealthyRound
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_Chain_WriteHealthyRound_OK",
			args:    args{ctx: common.GetRootContext()},
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
			if err := sc.WriteHealthyRound(tt.args.ctx, tt.args.hr); (err != nil) != tt.wantErr {
				t.Errorf("WriteHealthyRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
