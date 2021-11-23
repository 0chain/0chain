package interestpoolsc

import (
	"0chain.net/core/datastore"
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
		OwnerId     datastore.Key
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
				97, 112, 114, 34, 58, 48, 44, 34, 111, 119, 110, 101, 114, 95, 105, 100, 34, 58, 34, 34, 125},
		},
		{
			name: "full ok",
			fields: fields{
				MaxMint:     10,
				TotalMinted: 15,
				MinLock:     15,
				APR:         25,
				OwnerId:     "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
			},
			want: []byte{
				123, 34, 109, 97, 120, 95, 109, 105, 110, 116, 34, 58, 49, 48, 44, 34, 116, 111, 116, 97,
				108, 95, 109, 105, 110, 116, 101, 100, 34, 58, 49, 53, 44, 34, 109, 105, 110, 95, 108, 111,
				99, 107, 34, 58, 49, 53, 44, 34, 97, 112, 114, 34, 58, 50, 53, 44, 34, 111, 119, 110, 101,
				114, 95, 105, 100, 34, 58, 34, 49, 55, 52, 54, 98, 48, 54, 98, 98, 48, 57, 102, 53, 53, 101,
				101, 48, 49, 98, 51, 51, 98, 53, 101, 50, 101, 48, 53, 53, 100, 54, 99, 99, 55, 97, 57, 48,
				48, 99, 98, 53, 55, 99, 48, 97, 51, 97, 53, 101, 97, 97, 98, 98, 56, 97, 48, 101, 55, 55,
				52, 53, 56, 48, 50, 34, 125}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sgn := &SimpleGlobalNode{
				MaxMint:     tt.fields.MaxMint,
				TotalMinted: tt.fields.TotalMinted,
				MinLock:     tt.fields.MinLock,
				APR:         tt.fields.APR,
				OwnerId:     tt.fields.OwnerId,
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
		OwnerId     datastore.Key
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
				123, 34, 109, 97, 120, 95, 109, 105, 110, 116, 34, 58, 49, 48, 44, 34, 116, 111,
				116, 97, 108, 95, 109, 105, 110, 116, 101, 100, 34, 58, 49, 53, 44, 34, 109, 105,
				110, 95, 108, 111, 99, 107, 34, 58, 49, 53, 44, 34, 97, 112, 114, 34, 58, 50, 53,
				44, 34, 111, 119, 110, 101, 114, 95, 105, 100, 34, 58, 34, 49, 55, 52, 54, 98, 48,
				54, 98, 98, 48, 57, 102, 53, 53, 101, 101, 48, 49, 98, 51, 51, 98, 53, 101, 50, 101,
				48, 53, 53, 100, 54, 99, 99, 55, 97, 57, 48, 48, 99, 98, 53, 55, 99, 48, 97, 51, 97,
				53, 101, 97, 97, 98, 98, 56, 97, 48, 101, 55, 55, 52, 53, 56, 48, 50, 34, 125}},
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
				OwnerId:     tt.fields.OwnerId,
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
