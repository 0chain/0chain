package miner

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
	// "github.com/stretchr/testify/require"
)

/*
func TestTxnIterInfo_checkForCurrent(t *testing.T) {
	type fields struct {
		pastTxns    []datastore.Entity
		futureTxns  map[datastore.Key][]*transaction.Transaction
		currentTxns []*transaction.Transaction
	}
	type args struct {
		txn *transaction.Transaction
	}
	txs1 := []*transaction.Transaction{
		{ClientID: "1", Fee: 0, Nonce: 0},
		{ClientID: "1", Fee: 5, Nonce: 1},
		{ClientID: "1", Fee: 6, Nonce: 1},
		{ClientID: "1", Fee: 3, Nonce: 1},
		{ClientID: "1", Fee: 5, Nonce: 2},
		{ClientID: "1", Fee: 3, Nonce: 2},
		{ClientID: "1", Fee: 0, Nonce: 3},
		{ClientID: "1", Fee: 1, Nonce: 4},
		{ClientID: "1", Fee: 0, Nonce: 5},
	}
	txs2 := []*transaction.Transaction{
		{ClientID: "2", Fee: 0, Nonce: 0},
		{ClientID: "2", Fee: 5, Nonce: 1},
		{ClientID: "2", Fee: 6, Nonce: 1},
		{ClientID: "2", Fee: 3, Nonce: 1},
		{ClientID: "2", Fee: 5, Nonce: 2},
		{ClientID: "2", Fee: 3, Nonce: 2},
		{ClientID: "2", Fee: 0, Nonce: 3},
		{ClientID: "2", Fee: 1, Nonce: 4},
		{ClientID: "2", Fee: 0, Nonce: 5},
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   fields
	}{
		{
			name: "test_for_empty_future",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  nil,
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  nil,
				currentTxns: nil,
			},
		}, {
			name: "test_for_empty_future_with_client",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[0]}},
				currentTxns: nil,
			},
			args: args{txs2[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[0]}},
				currentTxns: nil,
			},
		}, {
			name: "test_with_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[4], txs1[5], txs1[6]}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[4], txs1[5], txs1[6]}},
				currentTxns: nil,
			},
		}, {
			name: "test_with_next_and_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[1], txs1[6], txs1[7], txs1[8]}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[6], txs1[7], txs1[8]}},
				currentTxns: []*transaction.Transaction{txs1[1]},
			},
		}, {
			name: "test_with_next_two_and_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[1], txs1[4], txs1[8]}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[8]}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4]},
			},
		}, {
			name: "test_with_next_three_and_no_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[1], txs1[4], txs1[6]}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[string][]*transaction.Transaction{"1": {}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4], txs1[6]},
			},
		}, {
			name: "test_with_next_three_and_2_similar_no_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[1], txs1[3], txs1[4], txs1[6]}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    []datastore.Entity{txs1[3]},
				futureTxns:  map[string][]*transaction.Transaction{"1": {}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4], txs1[6]},
			},
		}, {
			name: "test_with_next_three_and_2_similar_and_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[1], txs1[3], txs1[4], txs1[6], txs1[8]}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    []datastore.Entity{txs1[3]},
				futureTxns:  map[string][]*transaction.Transaction{"1": {txs1[8]}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4], txs1[6]},
			},
		}, {
			name: "test_full",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key][]*transaction.Transaction{"1": {txs1[1], txs1[2], txs1[3], txs1[4], txs1[5], txs1[6], txs1[7], txs1[8]}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    []datastore.Entity{txs1[2], txs1[3], txs1[5]},
				futureTxns:  map[string][]*transaction.Transaction{"1": {}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4], txs1[6], txs1[7], txs1[8]},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tii := TxnIterInfo{
				invalidTxns: tt.fields.pastTxns,
				futureTxns:  tt.fields.futureTxns,
				currentTxns: tt.fields.currentTxns,
			}
			tii.checkForCurrent(tt.args.txn)
			require.Equal(t, tt.want.pastTxns, tii.pastTxns)
			require.Equal(t, tt.want.futureTxns, tii.futureTxns)
			require.Equal(t, tt.want.currentTxns, tii.currentTxns)
		})
	}
}

func TestTxnIterInfo_checkForInvalidTxns(t *testing.T) {
	type fields struct {
		pastTxns    []datastore.Entity
		txns  []*transaction.Transaction
	}
	txs1 := []*transaction.Transaction{
		{ClientID: "1", Nonce: 0},
		{ClientID: "1", Nonce: 1},
		{ClientID: "1", Nonce: 1},
		{ClientID: "1", Nonce: 1},
		{ClientID: "1", Nonce: 2},
	}
	txs2 := []*transaction.Transaction{
		{ClientID: "2", Nonce: 0},
		{ClientID: "2", Nonce: 1},
	}

	tests := []struct {
		name   string
		fields fields
		want   []datastore.Entity
	}{
		{
			name: "test_for_empty_pastTxns_and_no_txns",
			fields: fields{
				pastTxns:    nil,
				txns:  nil,
			},
			want: []datastore.Entity{},
		}, {
			name: "test_for_empty_pastTxns",
			fields: fields{
				pastTxns:    nil,
				txns:  []*transaction.Transaction{txs1[0], txs2[1]},
			},
			want: []datastore.Entity{},
		}, {
			name: "test_for_pastTxns_from_one_client",
			fields: fields{
				pastTxns:    []datastore.Entity{txs1[0]},
				txns:  []*transaction.Transaction{txs1[1], txs2[1]},
			},
			want: []datastore.Entity{txs1[0]},
		}, {
			name: "test_for_pastTxns_from_two_client",
			fields: fields{
				pastTxns:    []datastore.Entity{txs1[0], txs2[0]},
				txns:  []*transaction.Transaction{txs1[1], txs2[1]},
			},
			want: []datastore.Entity{txs1[0], txs2[0]},
		}, {
			name: "test_for_with_txns_and_pastTxns_from_different_clients",
			fields: fields{
				pastTxns:    []datastore.Entity{txs1[0]},
				txns:  []*transaction.Transaction{txs2[1]},
			},
			want: []datastore.Entity{},
		}, {
			name: "test_with_no_txns",
			fields: fields{
				pastTxns:    []datastore.Entity{txs1[0]},
				txns:  nil,
			},
			want: []datastore.Entity{},
		}, {
			name: "test_with_equal_nonce",
			fields: fields{
				pastTxns:    []datastore.Entity{txs1[1]},
				txns:  []*transaction.Transaction{txs1[2]},
			},
			want: []datastore.Entity{txs1[1]},
		}, {
			name: "test_with_multiple_clashes",
			fields: fields{
				pastTxns:    []datastore.Entity{txs1[0], txs1[1], txs1[2]},
				txns:  []*transaction.Transaction{txs1[3]},
			},
			want: []datastore.Entity{txs1[0], txs1[1], txs1[2]},
		}, {
			name: "test_for_pastTxns_with_larger_nonce",
			fields: fields{
				pastTxns:    []datastore.Entity{txs1[4]},
				txns:  []*transaction.Transaction{txs1[1]},
			},
			want: []datastore.Entity{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tii := TxnIterInfo{
				pastTxns: tt.fields.pastTxns,
			}
			invalidTxns:= tii.checkForInvalidTxns(tt.fields.txns)
			require.Equal(t, tt.want, invalidTxns)
		})
	}
}

*/

func setupTempRocksDBDir() func() {
	if err := os.MkdirAll("data/rocksdb/state", 0766); err != nil {
		panic(err)
	}

	return func() {
		if err := os.RemoveAll("data/rocksdb/state"); err != nil {
			panic(err)
		}

		if err := os.RemoveAll("log"); err != nil {
			panic(err)
		}
	}
}

func TestChain_deletingTxns(t *testing.T) {

	txs1 := []*transaction.Transaction{
		{ClientID: "1", Nonce: 0, TransactionData: "false"},
		{ClientID: "1", Nonce: 1, TransactionData: ""},
		{ClientID: "1", Nonce: 2, TransactionData: ""},
		{ClientID: "1", Nonce: 3, TransactionData: ""},
		{ClientID: "1", Nonce: 4, TransactionData: ""},
	}

	type fields struct {
		ctx  context.Context
		txns []datastore.Entity
	}

	tests := []struct {
		name   string
		fields fields
		arg    *transaction.Transaction
		want   []datastore.Entity
	}{
		{
			name: "test_sample",
			fields: fields{
				ctx:  context.Background(),
				txns: []datastore.Entity{txs1[0], txs1[1]},
			},
			arg:  txs1[4],
			want: []datastore.Entity{txs1[0], txs1[1]},
		},
	}

	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	redisClient.FlushDB()
	// memorystore.AddPool("txndb", memorystore.DefaultPool)

	mc := GetMinerChain()
	setupTempRocksDBDir()
	common.SetupRootContext(node.GetNodeContext())
	config.SetServerChainID(config.GetMainChainID())
	transaction.SetupEntity(memorystore.GetStorageProvider())
	client.SetupEntity(memorystore.GetStorageProvider())
	chain.SetupEntity(memorystore.GetStorageProvider(), "")

	sigScheme := encryption.GetSignatureScheme("bls0chain")
	err = sigScheme.GenerateKeys()
	if err != nil {
		println(err.Error())
		panic(err)
	}

	c := &client.Client{}
	err = c.SetPublicKey(sigScheme.GetPublicKey())
	if err != nil {
		println(err.Error())
		panic(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			println("Starting test: " + tt.name)
			// storing txns
			for _, txn := range tt.fields.txns {
				txn.(*transaction.Transaction).CreationDate = common.Now()
				txn.(*transaction.Transaction).PublicKey = c.PublicKey
				txn.(*transaction.Transaction).Hash = txn.(*transaction.Transaction).ComputeHash()

				sig, err := txn.(*transaction.Transaction).Sign(sigScheme)
				if err != nil {
					fmt.Printf("Error signing\n")
				}

				txn.(*transaction.Transaction).Signature = sig

				if err != nil {
					println(err.Error())
				}

				out, err := transaction.PutTransaction(tt.fields.ctx, txn)
				println("==> heyo")
				if err != nil {
					println(err)
				} else {
					if out != nil {
						println("t is not nil!!!")
						t := out.(*transaction.Transaction)
						println("Stored txn: " + t.ClientID)
					} else {
						println("t is nil!!!")
					}

				}

			}
			println("Stored txns")

			// deleting txns
			mc.deleteTxns(tt.fields.txns)
			println("Deleted txns")

			// checking if txns are deleted
			// memorystore.
		})

	}

}
