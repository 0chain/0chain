package chain

import (
	"context"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
)

func Test_EstimateTransactionCost(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx    context.Context
		b      *block.Block
		bState util.MerklePatriciaTrieI
		txn    *transaction.Transaction
	}

	ch := NewChainFromConfig()

	clientState := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil, statecache.NewEmpty())
	bState := util.NewMerklePatriciaTrie(clientState.GetNodeDB(), 2, clientState.GetRoot(), statecache.NewEmpty())

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Test_EstimateTransferCost_TxnTypeSend",
			args: args{ctx: nil, b: block.NewBlock("", 1), bState: bState, txn: &transaction.Transaction{TransactionType: transaction.TxnTypeSend}},
			want: 10,
		},
		{
			name: "Test_EstimateTransferCost_TxnTypeData",
			args: args{ctx: nil, b: block.NewBlock("", 1), bState: util.NewMerklePatriciaTrie(clientState.GetNodeDB(), 2, clientState.GetRoot(), statecache.NewEmpty()), txn: &transaction.Transaction{TransactionType: transaction.TxnTypeData}},
			want: 0,
		},
		{
			name: "Test_EstimateTransferCost_TxnTypeLockIn",
			args: args{ctx: nil, b: block.NewBlock("", 1), bState: util.NewMerklePatriciaTrie(clientState.GetNodeDB(), 2, clientState.GetRoot(), statecache.NewEmpty()), txn: &transaction.Transaction{TransactionType: transaction.TxnTypeLockIn}},
			want: 0,
		},
		{
			name: "Test_EstimateTransferCost_TxnTypeStorageWrite",
			args: args{ctx: nil, b: block.NewBlock("", 1), bState: util.NewMerklePatriciaTrie(clientState.GetNodeDB(), 2, clientState.GetRoot(), statecache.NewEmpty()), txn: &transaction.Transaction{TransactionType: transaction.TxnTypeStorageWrite}},
			want: 0,
		},
		{
			name: "Test_EstimateTransferCost_TxnTypeStorageRead",
			args: args{ctx: nil, b: block.NewBlock("", 1), bState: util.NewMerklePatriciaTrie(clientState.GetNodeDB(), 2, clientState.GetRoot(), statecache.NewEmpty()), txn: &transaction.Transaction{TransactionType: transaction.TxnTypeStorageRead}},
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.args.b.ClientState = tt.args.bState
			if got, _ := ch.EstimateTransactionCost(tt.args.ctx, tt.args.b, tt.args.txn); got != tt.want {
				t.Errorf("EstimateTransactionCost() = %v, want %v", got, tt.want)
			}
		})
	}
}
