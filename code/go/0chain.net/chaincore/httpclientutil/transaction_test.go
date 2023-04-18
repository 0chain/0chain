package httpclientutil

import (
	"testing"

	"0chain.net/smartcontract/minersc"
	"github.com/stretchr/testify/require"
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
				Version:         "1.0",
				TransactionType: TxnTypeSmartContract,
				ClientID:        id,
				ToClientID:      minersc.ADDRESS,
				ChainID:         chainID,
				PublicKey:       pbKey,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewSmartContractTxn(tt.args.ID, tt.args.chainID, tt.args.pkey, minersc.ADDRESS)
			got.CreationDate = 0
			require.Equal(t, got, tt.want)
		})
	}
}
