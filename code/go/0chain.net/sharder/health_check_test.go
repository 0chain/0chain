package sharder

import (
	"0chain.net/core/encryption"
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
	sc.BlockSyncStats.getCycleControl(ProximityScan).counters.current.ElapsedSeconds = 1

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

func TestChain_healthCheck(t *testing.T) {
	sc := GetSharderChain()

	// case 1
	r2 := round.NewRound(2)
	r2.BlockHash = encryption.Hash("data round 2")
	b2 := block.NewBlock("", 3)
	b2.Hash = r2.BlockHash
	bs2 := b2.GetSummary()
	if err := sc.StoreRound(common.GetRootContext(), r2); err != nil {
		t.Fatal(err)
	}
	if err := bs2.GetEntityMetadata().GetStore().Write(common.GetRootContext(), bs2); err != nil {
		t.Fatal(err)
	}

	// case 2
	r3 := round.NewRound(3)
	r3.BlockHash = encryption.Hash("data round 3")
	b3 := block.NewBlock("", 3)
	b3.Hash = r3.BlockHash
	bs3 := b3.GetSummary()
	if err := sc.StoreRound(common.GetRootContext(), r3); err != nil {
		t.Fatal(err)
	}
	if err := sc.storeBlock(common.GetRootContext(), b3); err != nil {
		t.Fatal(err)
	}
	if err := bs3.GetEntityMetadata().GetStore().Write(common.GetRootContext(), bs3); err != nil {
		t.Fatal(err)
	}

	// case 3
	r4 := round.NewRound(4)
	r4.BlockHash = encryption.Hash("data round 4")
	b4 := block.NewBlock("", 4)
	b4.Hash = r3.BlockHash
	if err := sc.StoreRound(common.GetRootContext(), r4); err != nil {
		t.Fatal(err)
	}
	if err := sc.storeBlock(common.GetRootContext(), b4); err != nil {
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
		ctx      context.Context
		rNum     int64
		scanMode HealthCheckScan
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantBlockStatus BlockHealthCheckStatus
	}{
		{
			name: "TestChain_healthCheck_HealthCheck_Success",
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
			args:            args{ctx: common.GetRootContext(), rNum: 3, scanMode: ProximityScan},
			wantBlockStatus: HealthCheckSuccess,
		},
		{
			name: "TestChain_healthCheck_Health_Check_Failure",
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
			args:            args{ctx: common.GetRootContext(), rNum: 1, scanMode: ProximityScan},
			wantBlockStatus: HealthCheckFailure,
		},
		{
			name: "TestChain_healthCheck_HealthCheck_No_Block_Failure",
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
			args:            args{ctx: common.GetRootContext(), rNum: 2, scanMode: ProximityScan},
			wantBlockStatus: HealthCheckFailure,
		},
		{
			name: "TestChain_healthCheck_HealthCheck_No_Block_Summary_Failure",
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
			args:            args{ctx: common.GetRootContext(), rNum: 4, scanMode: ProximityScan},
			wantBlockStatus: HealthCheckFailure,
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

			cc := &sc.BlockSyncStats.getCycleControl(ProximityScan).counters.current
			fail := cc.HealthCheckFailure
			succ := cc.HealthCheckSuccess

			sc.healthCheck(tt.args.ctx, tt.args.rNum, tt.args.scanMode)

			if fail == cc.HealthCheckFailure && tt.wantBlockStatus == HealthCheckFailure {
				t.Error("expected failure, but got success")
			}
			if succ == cc.HealthCheckSuccess && tt.wantBlockStatus == HealthCheckSuccess {
				t.Error("expected success, but got failure")
			}
		})
	}
}

func TestChain_setCycleBounds(t *testing.T) {
	sc := GetSharderChain()
	sc.LatestFinalizedBlock = block.NewBlock("", 0)
	sc.HCCycleScan[ProximityScan].Window = 1

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
		in0      context.Context
		scanMode HealthCheckScan
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_Chain_setCycleBounds_OK",
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
				in0:      common.GetRootContext(),
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

			sc.setCycleBounds(tt.args.in0, tt.args.scanMode)
		})
	}
}
