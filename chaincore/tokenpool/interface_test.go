package tokenpool

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/0chain/common/core/currency"

	"github.com/stretchr/testify/assert"
)

func TestTokenPoolTransferResponse_Encode(t *testing.T) {
	t.Parallel()

	tp := TokenPoolTransferResponse{
		TxnHash:    "txn hash",
		FromPool:   "from pool",
		ToPool:     "to pool",
		Value:      5,
		FromClient: "from cient",
		ToClient:   "to client",
	}
	blob, err := json.Marshal(&tp)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		TxnHash    string
		FromPool   string
		ToPool     string
		Value      currency.Coin
		FromClient string
		ToClient   string
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields(tp),
			want:   blob,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &TokenPoolTransferResponse{
				TxnHash:    tt.fields.TxnHash,
				FromPool:   tt.fields.FromPool,
				ToPool:     tt.fields.ToPool,
				Value:      tt.fields.Value,
				FromClient: tt.fields.FromClient,
				ToClient:   tt.fields.ToClient,
			}
			if got := p.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenPoolTransferResponse_Decode(t *testing.T) {
	t.Parallel()

	tp := TokenPoolTransferResponse{
		TxnHash:    "txn hash",
		FromPool:   "from pool",
		ToPool:     "to pool",
		Value:      5,
		FromClient: "from cient",
		ToClient:   "to client",
	}
	blob, err := json.Marshal(&tp)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		TxnHash    string
		FromPool   string
		ToPool     string
		Value      currency.Coin
		FromClient string
		ToClient   string
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *TokenPoolTransferResponse
	}{
		{
			name:    "OK",
			args:    args{input: blob},
			wantErr: false,
			want:    &tp,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &TokenPoolTransferResponse{
				TxnHash:    tt.fields.TxnHash,
				FromPool:   tt.fields.FromPool,
				ToPool:     tt.fields.ToPool,
				Value:      tt.fields.Value,
				FromClient: tt.fields.FromClient,
				ToClient:   tt.fields.ToClient,
			}
			if err := p.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, p)
		})
	}
}
