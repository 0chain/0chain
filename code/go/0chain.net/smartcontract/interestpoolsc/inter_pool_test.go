package interestpoolsc

import (
	"reflect"
	"testing"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/tokenpool"
)

const (
	emptyInterestpoolJson = "{\"pool\":{\"pool\":{\"id\":\"\",\"balance\":0},\"lock\":null},\"apr\":0,\"tokens_earned\":0}"

	wrongEmptyInterestpoolJson = "{\"pool\":null,\"lock\":null},\"apr\":0,\"tokens_earned\":0}"
)

func Test_newInterestPool(t *testing.T) {
	tests := []struct {
		name string
		want *interestPool
	}{
		{
			name: "new interest pool",
			want: &interestPool{
				ZcnLockingPool: &tokenpool.ZcnLockingPool{TokenLockInterface: &TokenLock{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newInterestPool(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newInterestPool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_interestPool_encode(t *testing.T) {
	type fields struct {
		ZcnLockingPool *tokenpool.ZcnLockingPool
		APR            float64
		TokensEarned   currency.Coin
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "interest pool encode",
			fields: fields{
				ZcnLockingPool: &tokenpool.ZcnLockingPool{},
				APR:            0,
				TokensEarned:   0,
			},
			want: []byte(emptyInterestpoolJson),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &interestPool{
				ZcnLockingPool: tt.fields.ZcnLockingPool,
				APR:            tt.fields.APR,
				TokensEarned:   tt.fields.TokensEarned,
			}
			if got := ip.encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_interestPool_decode(t *testing.T) {
	type fields struct {
		ZcnLockingPool *tokenpool.ZcnLockingPool
		APR            float64
		TokensEarned   currency.Coin
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "decode ok",
			fields: fields{
				ZcnLockingPool: &tokenpool.ZcnLockingPool{},
				APR:            0,
				TokensEarned:   0,
			},
			args:    args{input: []byte(emptyInterestpoolJson)},
			wantErr: false,
		},
		{
			name: "decode false",
			fields: fields{
				ZcnLockingPool: &tokenpool.ZcnLockingPool{},
				APR:            0,
				TokensEarned:   0,
			},
			args:    args{input: []byte(wrongEmptyInterestpoolJson)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &interestPool{
				ZcnLockingPool: tt.fields.ZcnLockingPool,
				APR:            tt.fields.APR,
				TokensEarned:   tt.fields.TokensEarned,
			}
			if err := ip.decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
