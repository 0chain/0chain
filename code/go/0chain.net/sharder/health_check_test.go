package sharder

import (
	"context"
	"reflect"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
)

func TestHealthCheckScan_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		e         HealthCheckScan
		want      string
		wantPanic bool
	}{
		{
			name: "Test_HealthCheckScan_String_Deep_OK",
			e:    0,
			want: "Deep.....",
		},
		{
			name: "Test_HealthCheckScan_String_Proximity_OK",
			e:    1,
			want: "Proximity",
		},
		{
			name:      "Test_HealthCheckScan_String_Proximity_PANIC",
			e:         2, // e > 1 will panic
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic, but it is not")
					}
				}()
			}

			if got := tt.e.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_HealthCheckWorker(t *testing.T) {
	sc := GetSharderChain()
	sc.HCCycleScan[ProximityScan].Settle = time.Nanosecond
	sc.HCCycleScan[ProximityScan].ReportStatus = time.Nanosecond * 2
	sc.HCCycleScan[ProximityScan].RepeatInterval = time.Nanosecond
	cc := sc.BlockSyncStats.getCycleControl(ProximityScan)
	cc.bounds.highRound = 2
	cc.bounds.lowRound = 1

	sc.AddRound(roundMock{number: 1})

	ctx, cancel := context.WithTimeout(common.GetRootContext(), time.Second)

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
		scanMode HealthCheckScan
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_HealthCheckWorker_OK",
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
				ctx:      ctx,
				scanMode: ProximityScan,
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

			sc.LatestFinalizedBlock = block.NewBlock("", 1)

			go func() {
				time.Sleep(time.Millisecond * 200)
				cancel()
			}()

			sc.HealthCheckWorker(tt.args.ctx, tt.args.scanMode)
		})
	}
}

func TestChain_HealthCheckWorker_Disabled_In_Configs(t *testing.T) {
	sc := GetSharderChain()
	sc.HCCycleScan[ProximityScan].Enabled = false
	sc.HCCycleScan[ProximityScan].ReportStatus = time.Nanosecond

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
		scanMode HealthCheckScan
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_HealthCheckWorker_OK",
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
				scanMode: ProximityScan,
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

			sc.LatestFinalizedBlock = block.NewBlock("", 0)

			go sc.HealthCheckWorker(tt.args.ctx, tt.args.scanMode)
			time.Sleep(time.Millisecond * 200)
		})
	}
}

func TestGetRangeBounds(t *testing.T) {
	t.Parallel()

	type args struct {
		roundEdge  int64
		roundRange int64
	}
	tests := []struct {
		name string
		args args
		want RangeBounds
	}{
		{
			name: "Test_GetRangeBounds_OK",
			args: args{
				roundEdge:  5, // random chosen number
				roundRange: 1,
			},
			want: RangeBounds{
				roundLow:   5,
				roundHigh:  6,
				roundRange: 2,
			},
		},
		{
			name: "Test_GetRangeBounds_OK2",
			args: args{
				roundEdge:  5,
				roundRange: 0,
			},
			want: RangeBounds{
				roundLow:   5,
				roundHigh:  5,
				roundRange: 1,
			},
		},
		{
			name: "Test_GetRangeBounds_OK3",
			args: args{
				roundEdge:  0,
				roundRange: 0,
			},
			want: RangeBounds{
				roundLow:   1,
				roundHigh:  1,
				roundRange: 1,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := GetRangeBounds(tt.args.roundEdge, tt.args.roundRange); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRangeBounds() = %v, want %v", got, tt.want)
			}
		})
	}
}
