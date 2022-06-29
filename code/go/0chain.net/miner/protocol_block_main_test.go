package miner

import (
	"context"
	"net/http"
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
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	// "github.com/go-redis/redis"
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

func setupClientEntity() {
	em := datastore.EntityMetadataImpl{
		Name:     "client",
		DB:       "clientdb",
		Store:    memorystore.GetStorageProvider(),
		Provider: client.Provider,
	}
	// clientEntityMetadata = &em
	datastore.RegisterEntityMetadata("client", &em)

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

	txs1 := []*transaction.Transaction{
		{ClientID: "1", Nonce: 0, TransactionData: "this better work!"},
		{ClientID: "1", Nonce: 1, TransactionData: ""},
		{ClientID: "1", Nonce: 2, TransactionData: ""},
		{ClientID: "1", Nonce: 3, TransactionData: ""},
		{ClientID: "1", Nonce: 4, TransactionData: ""},
	}

	type fields struct {
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
				txns: []datastore.Entity{txs1[0], txs1[1]},
			},
			arg:  txs1[4],
			want: []datastore.Entity{txs1[0], txs1[1]},
		},
	}

	// memorystore.AddPool("txndb", memorystore.DefaultPool)
	err := initDefaultPool()
	if err != nil {
		panic(err)
	}
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
	// setupClientEntity()
	client.SetupEntity(memorystore.GetStorageProvider())
	chain.SetupEntity(memorystore.GetStorageProvider(), "")

	memorystore.AddPool("txndb", memorystore.DefaultPool)
	memorystore.AddPool("clientdb", memorystore.DefaultPool)

	sigScheme := encryption.GetSignatureScheme("bls0chain")
	err = sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}

	// cl := &client.Client{}
	var cl *client.Client
	cl = client.NewClient(client.SignatureScheme(encryption.SignatureSchemeBls0chain))
	cl.EntityCollection = &datastore.EntityCollection{CollectionName: "collection.cli", CollectionSize: 60000000000, CollectionDuration: time.Minute}
	err = cl.SetPublicKey(sigScheme.GetPublicKey())
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	_, err = client.PutClient(ctx, cl)

	cl.IDField = datastore.IDField{ID: cl.ID}
	cl.ID = "1"

	if err != nil {
		panic(err)
	}

	err = client.PutClientCache(cl)
	if err != nil {
		panic(err)
	}

	mc.RegisterClient()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// storing txns
			for _, txn := range tt.fields.txns {
				txn.(*transaction.Transaction).CreationDate = common.Now()
				txn.(*transaction.Transaction).PublicKey = cl.PublicKey
				txn.(*transaction.Transaction).Hash = txn.(*transaction.Transaction).ComputeHash()

				sig, err := txn.(*transaction.Transaction).Sign(sigScheme)
				if err != nil {
					panic(err)
				}

				txn.(*transaction.Transaction).Signature = sig

				if err != nil {
					panic(err)
				}

				_, err = transaction.PutTransaction(ctx, txn)
				if err != nil {
					panic(err)
				}

			}

			// getting txns
			thsh := tt.fields.txns[0].(*transaction.Transaction).Hash
			r, err := http.NewRequest("POST", "/api/v1/transactions?hash="+thsh, nil)
			if err != nil {
				panic(err)
			}
			_, err = transaction.GetTransaction(ctx, r)
			if err != nil {
				panic(err)
			}

			// deleting txns
			mc.deleteTxns(tt.fields.txns)

			// checking if txns are deleted
			_, err = transaction.GetTransaction(ctx, r)
			if err != nil {
				println(err.Error())
			} else {
				panic(err)
			}
		})

	}

}
