package sharder

import (
	"reflect"
	"testing"
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

func TestCycleCounters_transfer(t *testing.T) {
	t.Parallel()

	var (
		curr = BlockCounters{}
		prev = BlockCounters{}
	)

	curr.init()
	prev.init()

	type fields struct {
		ScanMode HealthCheckScan
		current  BlockCounters
		previous BlockCounters
	}
	tests := []struct {
		name   string
		fields fields
		want   CycleCounters
	}{
		{
			name: "TestCycleCounters_transfer_OK",
			fields: fields{
				current:  curr,
				previous: prev,
			},
			want: CycleCounters{
				current:  curr,
				previous: curr,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cc := &CycleCounters{
				ScanMode: tt.fields.ScanMode,
				current:  tt.fields.current,
				previous: tt.fields.previous,
			}

			cc.transfer()

			if !reflect.DeepEqual(cc, &tt.want) {
				t.Errorf("transfer() got = %v, want = %v", cc, tt.want)
			}
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
				roundEdge:  5, // random chosen number
				roundRange: 0,
			},
			want: RangeBounds{
				roundLow:   5,
				roundHigh:  5,
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

func TestSyncStats_getCycleControl(t *testing.T) {
	t.Parallel()

	var (
		syncStats       = SyncStats{}
		deepScanCC      = CycleControl{ScanMode: DeepScan}
		proximityScanCC = CycleControl{ScanMode: ProximityScan}
	)

	syncStats.cycle[DeepScan] = deepScanCC
	syncStats.cycle[ProximityScan] = proximityScanCC

	type fields struct {
		cycle [2]CycleControl
	}
	type args struct {
		scanMode HealthCheckScan
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      *CycleControl
		wantPanic bool
	}{
		{
			name:   "Test_SyncStats_getCycleControl_Deep_Scan_OK",
			fields: fields{cycle: syncStats.cycle},
			args:   args{scanMode: DeepScan},
			want:   &deepScanCC,
		},
		{
			name:   "Test_SyncStats_getCycleControl_Proximity_Scan_OK",
			fields: fields{cycle: syncStats.cycle},
			args:   args{scanMode: ProximityScan},
			want:   &proximityScanCC,
		},
		{
			name:      "Test_SyncStats_getCycleControl_PANIC",
			args:      args{scanMode: 2}, // it will panic if args is > 1
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

			bss := &SyncStats{
				cycle: tt.fields.cycle,
			}

			if got := bss.getCycleControl(tt.args.scanMode); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCycleControl() = %v, want %v", got, tt.want)
			}
		})
	}
}
