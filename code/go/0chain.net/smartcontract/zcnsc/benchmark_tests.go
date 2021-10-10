package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
	"testing"
)

type benchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
		cstate.StateContextI,
	) (string, error)
	txn   *transaction.Transaction
	input []byte
}

func (bt benchTest) Name() string {
	return bt.name
}

func (bt benchTest) Transaction() *transaction.Transaction {
	return bt.txn
}

func (bt benchTest) Run(state cstate.StateContextI, b *testing.B) {
	_, err := bt.endpoint(bt.Transaction(), bt.input, state)
	require.NoError(b, err)
}

func BenchmarkTests(data benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	sc := createSmartContract()

	return createTestSuite(
		[]benchTest{
			{
				name:     benchmark.Zcn + AddAuthorizerFunc,
				endpoint: sc.AddAuthorizer,
				txn:      createTransaction(data.Clients, data.PublicKeys),
				input:    createAuthorizer(data.PublicKeys),
			},
		},
	)
}

func createAuthorizer(publicKey []string) []byte {
	index := randomIndex(len(publicKey))
	node := authorizerNodeArg{
		PublicKey: publicKey[index],
		URL:       "http://localhost:303" + strconv.Itoa(index),
	}
	return node.Encode()
}

func createTransaction(clients, publicKey []string) *transaction.Transaction {
	index := randomIndex(len(clients))
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("mock transaction hash"),
		},
		ClientID:   clients[index],
		PublicKey:  publicKey[index],
		ToClientID: ADDRESS,
		Value:      3000,
	}
}

func randomIndex(max int) int {
	return rand.Intn(max)
}

func createTestSuite(restTests []benchTest) benchmark.TestSuite {
	var tests []benchmark.BenchTestI

	for _, test := range restTests {
		tests = append(tests, test)
	}

	return benchmark.TestSuite{
		Source:     benchmark.ZCNSCBridge,
		Benchmarks: tests,
	}
}
