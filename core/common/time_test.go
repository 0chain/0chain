package common

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestTimeToString(t *testing.T) {
	t.Parallel()

	ts := Now()

	type args struct {
		ts Timestamp
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_TimeToString_OK",
			args: args{ts: ts},
			want: strconv.FormatInt(int64(ts), 10),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := TimeToString(tt.args.ts); got != tt.want {
				t.Errorf("TimeToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimestamp_Duration(t *testing.T) {
	t.Parallel()

	ts := Now()

	tests := []struct {
		name string
		t    Timestamp
		want time.Duration
	}{
		{
			name: "Test_Timestamp_Duration_OK",
			t:    ts,
			want: time.Second * time.Duration(ts),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.t.Duration(); got != tt.want {
				t.Errorf("Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithinTime(t *testing.T) {
	t.Parallel()

	type args struct {
		o       int64
		ts      int64
		seconds int64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_WithinTime_OK",
			args: args{
				o:       0,
				ts:      0,
				seconds: 0,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := WithinTime(tt.args.o, tt.args.ts, tt.args.seconds); got != tt.want {
				t.Errorf("WithinTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSleepOrDone(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx   context.Context
		sleep time.Duration
	}
	tests := []struct {
		name     string
		args     args
		wantDone bool
	}{
		{
			name:     "Test_SleepOrDone_FALSE",
			args:     args{ctx: context.TODO(), sleep: time.Millisecond},
			wantDone: false,
		},
		{
			name:     "Test_SleepOrDone_TRUE",
			args:     args{ctx: context.TODO(), sleep: time.Millisecond},
			wantDone: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.args.ctx
			var cancel context.CancelFunc
			if tt.wantDone {
				ctx, cancel = context.WithCancel(tt.args.ctx)
				cancel()
			}

			if gotDone := SleepOrDone(ctx, tt.args.sleep); gotDone != tt.wantDone {
				t.Errorf("SleepOrDone() = %v, want %v", gotDone, tt.wantDone)
			}
		})
	}
}

func TestToTime(t *testing.T) {
	t.Parallel()

	ts := Now()

	type args struct {
		ts Timestamp
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			name: "TestToTime_OK",
			args: args{ts: ts},
			want: time.Unix(int64(ts), 0),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToTime(tt.args.ts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithin(t *testing.T) {
	t.Parallel()

	type args struct {
		ts      int64
		seconds int64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_TestWithin_OK",
			args: args{
				ts:      0,
				seconds: 0,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Within(tt.args.ts, tt.args.seconds); got != tt.want {
				t.Errorf("WithinTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
