package tokenpool

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
)

func TestZcnPool_Encode(t *testing.T) {
	t.Parallel()

	zp := ZcnPool{
		TokenPool{
			ID:      "id",
			Balance: 5,
		},
	}
	blob, err := json.Marshal(&zp)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		TokenPool TokenPool
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				TokenPool: zp.TokenPool,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &ZcnPool{
				TokenPool: tt.fields.TokenPool,
			}
			if got := p.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZcnPool_Decode(t *testing.T) {
	t.Parallel()

	zp := ZcnPool{
		TokenPool{
			ID:      "id",
			Balance: 5,
		},
	}
	blob, err := json.Marshal(&zp)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		TokenPool TokenPool
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ZcnPool
		wantErr bool
	}{
		{
			name:    "OK",
			args:    args{input: blob},
			want:    &zp,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &ZcnPool{
				TokenPool: tt.fields.TokenPool,
			}
			if err := p.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, p)
		})
	}
}

func TestZcnPool_DigPool_Err(t *testing.T) {
	t.Parallel()

	txn := transaction.Transaction{}
	txn.Value = -1

	type fields struct {
		TokenPool TokenPool
	}
	type args struct {
		id  string
		txn *transaction.Transaction
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *state.Transfer
		want1   string
		wantErr bool
	}{
		{
			name:    "ERR",
			args:    args{txn: &txn},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &ZcnPool{
				TokenPool: tt.fields.TokenPool,
			}
			got, got1, err := p.DigPool(tt.args.id, tt.args.txn)
			if (err != nil) != tt.wantErr {
				t.Errorf("DigPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DigPool() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("DigPool() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestZcnPool_EmptyPool_Err(t *testing.T) {
	t.Parallel()

	type fields struct {
		TokenPool TokenPool
	}
	type args struct {
		fromClientID string
		toClientID   string
		entity       interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *state.Transfer
		want1   string
		wantErr bool
	}{
		{
			name:    "ERR",
			fields:  fields{TokenPool{Balance: 0}},
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &ZcnPool{
				TokenPool: tt.fields.TokenPool,
			}
			got, got1, err := p.EmptyPool(tt.args.fromClientID, tt.args.toClientID, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmptyPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EmptyPool() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("EmptyPool() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
