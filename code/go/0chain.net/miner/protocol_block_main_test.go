package miner

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"github.com/0chain/common/core/logging"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestTxnIterInfo_checkForCurrent(t *testing.T) {
	type fields struct {
		pastTxns    []datastore.Entity
		futureTxns  map[datastore.Key]*clientNonceTxns
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
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[0]}}},
				currentTxns: nil,
			},
			args: args{txs2[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[0]}}},
				currentTxns: nil,
			},
		}, {
			name: "test_with_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[4], txs1[5], txs1[6]}}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[4], txs1[5], txs1[6]}}},
				currentTxns: nil,
			},
		}, {
			name: "test_with_next_and_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[1], txs1[6], txs1[7], txs1[8]}}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {1, []*transaction.Transaction{txs1[6], txs1[7], txs1[8]}}},
				currentTxns: []*transaction.Transaction{txs1[1]},
			},
		}, {
			name: "test_with_next_two_and_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[1], txs1[4], txs1[8]}}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {2, []*transaction.Transaction{txs1[8]}}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4]},
			},
		}, {
			name: "test_with_next_three_and_no_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[1], txs1[4], txs1[6]}}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    nil,
				futureTxns:  map[string]*clientNonceTxns{"1": {3, []*transaction.Transaction{}}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4], txs1[6]},
			},
		}, {
			name: "test_with_next_three_and_2_similar_no_gap",
			fields: fields{
				pastTxns:    nil,
				futureTxns:  map[datastore.Key]*clientNonceTxns{"1": {0, []*transaction.Transaction{txs1[1], txs1[3], txs1[4], txs1[6]}}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    []datastore.Entity{txs1[3]},
				futureTxns:  map[string]*clientNonceTxns{"1": {3, []*transaction.Transaction{}}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4], txs1[6]},
			},
		}, {
			name: "test_with_next_three_and_2_similar_and_gap",
			fields: fields{
				pastTxns: nil,
				futureTxns: map[datastore.Key]*clientNonceTxns{"1": {0,
					[]*transaction.Transaction{txs1[1], txs1[3], txs1[4], txs1[6], txs1[8]}}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    []datastore.Entity{txs1[3]},
				futureTxns:  map[string]*clientNonceTxns{"1": {3, []*transaction.Transaction{txs1[8]}}},
				currentTxns: []*transaction.Transaction{txs1[1], txs1[4], txs1[6]},
			},
		}, {
			name: "test_full",
			fields: fields{
				pastTxns: nil,
				futureTxns: map[datastore.Key]*clientNonceTxns{"1": {0,
					[]*transaction.Transaction{txs1[1], txs1[2], txs1[3], txs1[4], txs1[5], txs1[6], txs1[7], txs1[8]}}},
				currentTxns: nil,
			},
			args: args{txs1[0]},
			want: fields{
				pastTxns:    []datastore.Entity{txs1[2], txs1[3], txs1[5]},
				futureTxns:  map[string]*clientNonceTxns{"1": {5, []*transaction.Transaction{}}},
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
			require.EqualValues(t, tt.want.futureTxns, tii.futureTxns)
			require.Equal(t, tt.want.currentTxns, tii.currentTxns)
		})
	}
}

func initDefaultPool() error {
	mr, err := miniredis.Run()
	if err != nil {
		return err
	}

	memorystore.DefaultPool = &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", mr.Addr())
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}

	return nil
}

func TestChain_deletingTxns(t *testing.T) {

	transaction.SetTxnTimeout(int64(3 * time.Minute))
	txs1 := []*transaction.Transaction{
		{Nonce: 0},
		{Nonce: 1},
		{Nonce: 2},
		{Nonce: 3},
		{Nonce: 4},
	}

	tests := []struct {
		name string
		txns []*transaction.Transaction
	}{
		{
			name: "test_sample",
			txns: []*transaction.Transaction{txs1[0], txs1[1]},
		},
	}

	// var err error

	require.NoError(t, initDefaultPool())

	logging.InitLogging("testing", "")

	n1 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7071, Status: node.NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n2 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7072, Status: node.NodeStatusActive}
	n2.ID = "5fbb6924c222e96df6c491dfc4a542e1bbfc75d821bcca992544899d62121b55"
	n3 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7073, Status: node.NodeStatusActive}
	n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	node.Self = &node.SelfNode{}
	node.Self.Node = n1

	setupSelfNodeKeys()

	np := node.NewPool(node.NodeTypeMiner)
	np.AddNode(n1)
	np.AddNode(n2)
	np.AddNode(n3)

	mb := block.NewMagicBlock()
	mb.Miners = np

	c := chain.Provider().(*chain.Chain)
	c.ID = datastore.ToKey(config.GetServerChainID())
	c.SetMagicBlock(mb)
	chain.SetServerChain(c)
	SetupMinerChain(c)
	mc := GetMinerChain()
	mc.SetMagicBlock(mb)
	SetupM2MSenders()

	setupTempRocksDBDir()
	common.SetupRootContext(node.GetNodeContext())
	config.SetServerChainID(config.GetMainChainID())
	transaction.SetupEntity(memorystore.GetStorageProvider())
	client.SetupEntity(memorystore.GetStorageProvider())
	chain.SetupEntity(memorystore.GetStorageProvider(), "")

	memorystore.AddPool("txndb", memorystore.DefaultPool)
	memorystore.AddPool("clientdb", memorystore.DefaultPool)

	sigScheme := encryption.GetSignatureScheme("bls0chain")
	require.NoError(t, sigScheme.GenerateKeys())

	var cl *client.Client
	cl = client.NewClient(client.SignatureScheme(encryption.SignatureSchemeBls0chain))
	cl.EntityCollection = &datastore.EntityCollection{CollectionName: "collection.cli", CollectionSize: 60000000000, CollectionDuration: time.Minute}
	require.NoError(t, cl.SetPublicKey(sigScheme.GetPublicKey()))

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// storing txns
			for _, txn := range tt.txns {
				txn.CreationDate = common.Now()
				txn.PublicKey = cl.PublicKey
				txn.ClientID = cl.ID
				txn.Hash = txn.ComputeHash()

				txnData, err := json.Marshal(struct {
					name string
				}{
					name: "test",
				})
				require.NoError(t, err)
				txn.TransactionData = string(txnData)

				sig, err := txn.Sign(sigScheme)
				require.NoError(t, err)

				txn.Signature = sig

				_, err = transaction.PutTransaction(ctx, txn)
				require.NoError(t, err)

			}

			// verifying that txns exist
			for _, txn := range tt.txns {
				r, err := http.NewRequest("POST", "/api/v1/transactions?hash="+txn.Hash, nil)
				require.NoError(t, err)

				_, err = transaction.GetTransaction(ctx, r)
				require.NoError(t, err)
			}

			// deleting txns
			var txnsEntity []datastore.Entity
			for _, txn := range tt.txns {
				txnsEntity = append(txnsEntity, txn)
			}
			require.NoError(t, mc.deleteTxns(txnsEntity))

			// checking if txns are deleted
			for _, txn := range tt.txns {
				r, err := http.NewRequest("POST", "/api/v1/transactions?hash="+txn.Hash, nil)
				require.NoError(t, err)

				_, err = transaction.GetTransaction(ctx, r)
				if err != nil && strings.HasPrefix(err.Error(), "entity_not_found: txn not found") {
					t.Log("txn deleted")
				} else {
					t.Error("txn not deleted")
				}
			}
		})
	}

}
