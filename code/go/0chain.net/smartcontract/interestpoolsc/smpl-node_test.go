package interestpoolsc

import (
	"reflect"
	"testing"

	"0chain.net/chaincore/state"
)

func TestSimpleGlobalNode_Encode(t *testing.T) {
	type fields struct {
		MaxMint     state.Balance
		TotalMinted state.Balance
		MinLock     state.Balance
		APR         float64
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "empty ok",
			fields: fields{},
			want: []byte{
				123, 34, 109, 97, 120, 95, 109, 105, 110, 116, 34, 58, 48, 44, 34, 116, 111,
				116, 97, 108, 95, 109, 105, 110, 116, 101, 100, 34, 58, 48,
				44, 34, 109, 105, 110, 95, 108, 111, 99, 107, 34, 58, 48, 44, 34,
				97, 112, 114, 34, 58, 48, 125},
		},
		{
			name: "full ok",
			fields: fields{
				MaxMint:     10,
				TotalMinted: 15,
				MinLock:     15,
				APR:         25,
			},
			want: []byte{
				123, 34, 109, 97, 120, 95, 109, 105, 110, 116, 34, 58, 49, 48, 44,
				34, 116, 111, 116, 97, 108, 95, 109, 105, 110, 116, 101, 100, 34, 58,
				49, 53, 44, 34, 109, 105, 110, 95, 108, 111, 99, 107, 34, 58, 49, 53, 44,
				34, 97, 112, 114, 34, 58, 50, 53, 125},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sgn := &SimpleGlobalNode{
				MaxMint:     tt.fields.MaxMint,
				TotalMinted: tt.fields.TotalMinted,
				MinLock:     tt.fields.MinLock,
				APR:         tt.fields.APR,
			}
			if got := sgn.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimpleGlobalNode_Decode(t *testing.T) {
	type fields struct {
		MaxMint     state.Balance
		TotalMinted state.Balance
		MinLock     state.Balance
		APR         float64
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
				123, 34, 109, 97, 120, 95, 109, 105, 110, 116, 34, 58, 49, 48, 44,
				34, 116, 111, 116, 97, 108, 95, 109, 105, 110, 116, 101, 100, 34, 58,
				49, 53, 44, 34, 109, 105, 110, 95, 108, 111, 99, 107, 34, 58, 49, 53, 44,
				34, 97, 112, 114, 34, 58, 50, 53, 125}},
			wantErr: false,
			full:    true,
		},
		{
			name: "empty ok",
			args: args{input: []byte{
				123, 34, 109, 97, 120, 95, 109, 105, 110, 116, 34, 58, 48, 44, 34, 116, 111,
				116, 97, 108, 95, 109, 105, 110, 116, 101, 100, 34, 58, 48,
				44, 34, 109, 105, 110, 95, 108, 111, 99, 107, 34, 58, 48, 44, 34,
				97, 112, 114, 34, 58, 48, 125}},
			wantErr: false,
		},
		{
			name:    "error unmarshal",
			args:    args{input: []byte{66}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sgn := &SimpleGlobalNode{
				MaxMint:     tt.fields.MaxMint,
				TotalMinted: tt.fields.TotalMinted,
				MinLock:     tt.fields.MinLock,
				APR:         tt.fields.APR,
			}
			if err := sgn.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.full {
				if sgn.MaxMint != 10 || sgn.MinLock != 15 ||
					sgn.TotalMinted != 15 || sgn.APR != 25 {
					t.Errorf("wrong decoded data")
				}
			}
		})
	}
}
