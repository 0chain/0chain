package metric

import (
	"container/ring"
	"fmt"
	"reflect"
	"testing"
	"time"

	"0chain.net/chaincore/round"
)

func TestFormattedTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	m := round.Info{TimeStamp: &now}

	type args struct {
		metric Metric
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_FormattedTime_OK",
			args: args{metric: &m},
			want: fmt.Sprintf("%02d:%02d", now.Minute(), now.Second()),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FormattedTime(tt.args.metric); got != tt.want {
				t.Errorf("FormattedTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPowerMetrics(t *testing.T) {
	t.Parallel()

	var (
		bufferLen = 2
		power     = 2
		buffer    = make([]*ring.Ring, bufferLen)
	)
	for idx := 0; idx < bufferLen; idx++ {
		buffer[idx] = ring.New(power)
	}

	type args struct {
		power     int
		bufferLen int
	}
	tests := []struct {
		name string
		args args
		want *PowerMetrics
	}{
		{
			name: "Test_NewPowerMetrics_OK",
			args: args{power: power, bufferLen: bufferLen},
			want: &PowerMetrics{
				power:       power,
				bufferLen:   bufferLen,
				powerBuffer: buffer,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewPowerMetrics(tt.args.power, tt.args.bufferLen); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPowerMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPowerMetrics_Collect(t *testing.T) {
	t.Parallel()

	pm := NewPowerMetrics(2, 2)

	now := time.Now()

	type fields struct {
		power        int
		bufferLen    int
		powerBuffer  []*ring.Ring
		CurrentValue Metric
	}
	type args struct {
		data Metric
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_PowerMetrics_Collect_OK",
			fields: fields{
				power:        pm.power,
				bufferLen:    pm.bufferLen,
				powerBuffer:  pm.powerBuffer,
				CurrentValue: pm.CurrentValue,
			},
			args: args{data: &round.Info{TimeStamp: &now}},
		},
		{
			name: "Test_PowerMetrics_Collect_OK2",
			fields: fields{
				power:        pm.power,
				bufferLen:    pm.bufferLen,
				powerBuffer:  pm.powerBuffer,
				CurrentValue: pm.CurrentValue,
			},
			args: args{data: &round.Info{TimeStamp: &now, Number: 3}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pm := &PowerMetrics{
				power:        tt.fields.power,
				bufferLen:    tt.fields.bufferLen,
				powerBuffer:  tt.fields.powerBuffer,
				CurrentValue: tt.fields.CurrentValue,
			}

			pm.Collect(tt.args.data)
		})
	}
}

func TestPowerMetrics_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	m := round.Info{TimeStamp: &now}

	pm := NewPowerMetrics(2, 2)
	pm.CurrentValue = &m
	pm.Collect(&m)

	wantPM := NewPowerMetrics(2, 2)
	wantPM.CurrentValue = &m
	wantPM.Collect(&m)

	values := make([]Metric, (wantPM.power)*(wantPM.bufferLen)+1)
	values[0] = wantPM.CurrentValue
	index := 1

	arr := make([]Metric, wantPM.power)
	var arrIdx = len(arr) - 1
	for idx := 0; idx < wantPM.bufferLen; idx++ {
		r := wantPM.powerBuffer[idx]
		r.Do(func(val interface{}) {
			if val != nil {
				arr[arrIdx] = val.(Metric)
				arrIdx--
			}
		})

		for i := arrIdx + 1; i < len(arr); i++ {
			values[index] = arr[i]
			index++
		}
		arrIdx = len(arr) - 1
	}

	type fields struct {
		power        int
		bufferLen    int
		powerBuffer  []*ring.Ring
		CurrentValue Metric
	}
	tests := []struct {
		name   string
		fields fields
		want   []Metric
	}{
		{
			name: "Test_PowerMetrics_GetAll_OK",
			fields: fields{
				power:        pm.power,
				bufferLen:    pm.bufferLen,
				powerBuffer:  pm.powerBuffer,
				CurrentValue: pm.CurrentValue,
			},
			want: values[:index],
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pm := &PowerMetrics{
				power:        tt.fields.power,
				bufferLen:    tt.fields.bufferLen,
				powerBuffer:  tt.fields.powerBuffer,
				CurrentValue: tt.fields.CurrentValue,
			}
			if got := pm.GetAll(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAll() = %v, want %v", got, tt.want)
			}
		})
	}
}
