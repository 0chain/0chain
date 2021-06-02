package httpclientutil

import (
	"reflect"
	"testing"
)

func TestNewTransactionEntity(t *testing.T) {
	t.Parallel()

	var (
		id      = "id"
		chainID = "chain id"
		pbKey   = "public key"
	)

	type args struct {
		ID      string
		chainID string
		pkey    string
	}
	tests := []struct {
		name string
		args args
		want *Transaction
	}{
		{
			name: "OK",
			args: args{
				ID:      id,
				chainID: chainID,
				pkey:    pbKey,
			},
			want: &Transaction{
				Version:   "1.0",
				ClientID:  id,
				ChainID:   chainID,
				PublicKey: pbKey,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewTransactionEntity(tt.args.ID, tt.args.chainID, tt.args.pkey)
			got.CreationDate = 0
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTransactionEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}
