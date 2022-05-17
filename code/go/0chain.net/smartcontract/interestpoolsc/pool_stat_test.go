package interestpoolsc

import (
	"reflect"
	"testing"
	"time"

	"0chain.net/pkg/currency"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

func Test_poolStats_addStat(t *testing.T) {
	type fields struct {
		Stats []*poolStat
	}
	type args struct {
		p *poolStat
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "ok",
			fields: fields{Stats: []*poolStat{}},
			args: args{
				p: testPoolState(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &poolStats{
				Stats: tt.fields.Stats,
			}
			ps.addStat(tt.args.p)
			if ps.Stats[0] != tt.args.p {
				t.Errorf("wrong pool state added")
			}
		})
	}
}

func Test_poolStats_encode(t *testing.T) {
	type fields struct {
		Stats []*poolStat
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "encode empty",
			fields: fields{Stats: []*poolStat{}},
			want: []byte{
				123, 34, 115, 116, 97, 116, 115, 34, 58, 91, 93, 125,
			},
		},
		{
			name: "encode empty",
			fields: fields{Stats: []*poolStat{&poolStat{
				ID:     "owner",
				Locked: true,
				APR:    10,
			}}},
			want: []byte{123, 34, 115, 116, 97, 116, 115, 34, 58, 91, 123, 34, 112, 111,
				111, 108, 95, 105, 100, 34, 58, 34, 111, 119, 110, 101, 114, 34, 44, 34, 115, 116,
				97, 114, 116, 95, 116, 105, 109, 101, 34, 58, 48, 44, 34, 100, 117, 114, 97, 116,
				105, 111, 110, 34, 58, 48, 44, 34, 116, 105, 109, 101, 95, 108, 101, 102, 116, 34,
				58, 48, 44, 34, 108, 111, 99, 107, 101, 100, 34, 58, 116, 114, 117, 101, 44, 34, 97,
				112, 114, 34, 58, 49, 48, 44, 34, 116, 111, 107, 101, 110, 115, 95, 101, 97, 114, 110, 101,
				100, 34, 58, 48, 44, 34, 98, 97, 108, 97, 110, 99, 101, 34, 58, 48, 125, 93, 125},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &poolStats{
				Stats: tt.fields.Stats,
			}
			if got := ps.encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_poolStats_decode(t *testing.T) {
	type fields struct {
		Stats []*poolStat
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		full    bool
	}{
		{
			name:   "empty ok",
			fields: fields{Stats: []*poolStat{}},
			args: args{
				input: []byte{
					123, 34, 115, 116, 97, 116, 115, 34, 58, 91, 93, 125,
				},
			},
			wantErr: false,
		},
		{
			name:   "full ok",
			fields: fields{Stats: []*poolStat{}},
			args: args{
				input: []byte{123, 34, 115, 116, 97, 116, 115, 34, 58, 91, 123, 34, 112, 111,
					111, 108, 95, 105, 100, 34, 58, 34, 111, 119, 110, 101, 114, 34, 44, 34, 115, 116,
					97, 114, 116, 95, 116, 105, 109, 101, 34, 58, 48, 44, 34, 100, 117, 114, 97, 116,
					105, 111, 110, 34, 58, 48, 44, 34, 116, 105, 109, 101, 95, 108, 101, 102, 116, 34,
					58, 48, 44, 34, 108, 111, 99, 107, 101, 100, 34, 58, 116, 114, 117, 101, 44, 34, 97,
					112, 114, 34, 58, 49, 48, 44, 34, 116, 111, 107, 101, 110, 115, 95, 101, 97, 114, 110, 101,
					100, 34, 58, 48, 44, 34, 98, 97, 108, 97, 110, 99, 101, 34, 58, 48, 125, 93, 125},
			},
			wantErr: false,
			full:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &poolStats{
				Stats: tt.fields.Stats,
			}
			if err := ps.decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.full {
				ps0 := ps.Stats[0]
				if ps0.ID != "owner" || !ps0.Locked && ps0.APR != 10 {
					t.Errorf("wrong data decoded")
				}
			}
		})
	}
}

func Test_poolStat_encode(t *testing.T) {
	type fields struct {
		ID           datastore.Key
		StartTime    common.Timestamp
		Duartion     time.Duration
		TimeLeft     time.Duration
		Locked       bool
		APR          float64
		TokensEarned currency.Coin
		Balance      currency.Coin
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "full ok",
			fields: fields{
				Balance:      10,
				TokensEarned: 10,
			},
			want: []byte{
				123, 34, 112, 111, 111, 108, 95, 105, 100, 34, 58, 34, 34, 44, 34, 115, 116,
				97, 114, 116, 95, 116, 105, 109, 101, 34, 58, 48, 44, 34, 100, 117, 114, 97,
				116, 105, 111, 110, 34, 58, 48, 44, 34, 116, 105, 109, 101, 95, 108, 101, 102,
				116, 34, 58, 48, 44, 34, 108, 111, 99, 107, 101, 100, 34, 58, 102, 97, 108, 115,
				101, 44, 34, 97, 112, 114, 34, 58, 48, 44, 34, 116, 111, 107, 101, 110, 115, 95,
				101, 97, 114, 110, 101, 100, 34, 58, 49, 48, 44, 34, 98, 97, 108, 97, 110, 99, 101,
				34, 58, 49, 48, 125,
			},
		},
		{
			name:   "empty ok",
			fields: fields{},
			want: []byte{
				123, 34, 112, 111, 111, 108, 95, 105, 100, 34, 58, 34, 34, 44, 34, 115, 116,
				97, 114, 116, 95, 116, 105, 109, 101, 34, 58, 48, 44, 34, 100, 117, 114, 97,
				116, 105, 111, 110, 34, 58, 48, 44, 34, 116, 105, 109, 101, 95, 108, 101, 102,
				116, 34, 58, 48, 44, 34, 108, 111, 99, 107, 101, 100, 34, 58, 102, 97, 108, 115,
				101, 44, 34, 97, 112, 114, 34, 58, 48, 44, 34, 116, 111, 107, 101, 110, 115, 95,
				101, 97, 114, 110, 101, 100, 34, 58, 48, 44, 34, 98, 97, 108, 97, 110, 99, 101,
				34, 58, 48, 125,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &poolStat{
				ID:           tt.fields.ID,
				StartTime:    tt.fields.StartTime,
				Duartion:     tt.fields.Duartion,
				TimeLeft:     tt.fields.TimeLeft,
				Locked:       tt.fields.Locked,
				APR:          tt.fields.APR,
				TokensEarned: tt.fields.TokensEarned,
				Balance:      tt.fields.Balance,
			}
			if got := ps.encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_poolStat_decode(t *testing.T) {
	type fields struct {
		ID           datastore.Key
		StartTime    common.Timestamp
		Duartion     time.Duration
		TimeLeft     time.Duration
		Locked       bool
		APR          float64
		TokensEarned currency.Coin
		Balance      currency.Coin
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		full    bool
	}{
		{
			name: "full ok",
			args: args{input: []byte{
				123, 34, 112, 111, 111, 108, 95, 105, 100, 34, 58, 34, 34, 44, 34, 115, 116,
				97, 114, 116, 95, 116, 105, 109, 101, 34, 58, 48, 44, 34, 100, 117, 114, 97,
				116, 105, 111, 110, 34, 58, 48, 44, 34, 116, 105, 109, 101, 95, 108, 101, 102,
				116, 34, 58, 48, 44, 34, 108, 111, 99, 107, 101, 100, 34, 58, 102, 97, 108, 115,
				101, 44, 34, 97, 112, 114, 34, 58, 48, 44, 34, 116, 111, 107, 101, 110, 115, 95,
				101, 97, 114, 110, 101, 100, 34, 58, 49, 48, 44, 34, 98, 97, 108, 97, 110, 99, 101,
				34, 58, 49, 48, 125,
			}},
			wantErr: false,
			full:    true,
		},
		{
			name: "full ok",
			args: args{input: []byte{
				123, 34, 112, 111, 111, 108, 95, 105, 100, 34, 58, 34, 34, 44, 34, 115, 116,
				97, 114, 116, 95, 116, 105, 109, 101, 34, 58, 48, 44, 34, 100, 117, 114, 97,
				116, 105, 111, 110, 34, 58, 48, 44, 34, 116, 105, 109, 101, 95, 108, 101, 102,
				116, 34, 58, 48, 44, 34, 108, 111, 99, 107, 101, 100, 34, 58, 102, 97, 108, 115,
				101, 44, 34, 97, 112, 114, 34, 58, 48, 44, 34, 116, 111, 107, 101, 110, 115, 95,
				101, 97, 114, 110, 101, 100, 34, 58, 48, 44, 34, 98, 97, 108, 97, 110, 99, 101,
				34, 58, 48, 125,
			}},
			wantErr: false,
		},
		{
			name:    "full ok",
			args:    args{input: []byte{66}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &poolStat{
				ID:           tt.fields.ID,
				StartTime:    tt.fields.StartTime,
				Duartion:     tt.fields.Duartion,
				TimeLeft:     tt.fields.TimeLeft,
				Locked:       tt.fields.Locked,
				APR:          tt.fields.APR,
				TokensEarned: tt.fields.TokensEarned,
				Balance:      tt.fields.Balance,
			}
			if err := ps.decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("decode() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.full && ps.Balance != 10 && ps.TokensEarned != 10 {
				t.Errorf("wrong decoded data")
			}
		})
	}
}
