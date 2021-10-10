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
	b.Logf("Running test '%s' from ZCNSC Bridge", bt.name)
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
				txn:      createRandomTransaction(data.Clients, data.PublicKeys),
				input:    createRandomAuthorizer(data.PublicKeys),
			},
			{
				name:     benchmark.Zcn + DeleteAuthorizerFunc,
				endpoint: sc.DeleteAuthorizer,
				txn:      createTransaction(data.Clients[commonClientId], data.PublicKeys[commonClientId]),
				input:    nil,
			},
		},
	)
}

func createRandomAuthorizer(publicKey []string) []byte {
	index := randomIndex(len(publicKey))
	return createAuthorizer(publicKey[index], index)
}

func createAuthorizer(publicKey string, index int) []byte {
	node := authorizerNodeArg{
		PublicKey: publicKey,
		URL:       "http://localhost:303" + strconv.Itoa(index),
	}
	return node.Encode()
}

func createRandomTransaction(clients, publicKey []string) *transaction.Transaction {
	index := randomIndex(len(clients))
	return createTransaction(clients[index], publicKey[index])
}

func createTransaction(clientId, publicKey string) *transaction.Transaction {
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("mock transaction hash"),
		},
		ClientID:   clientId,
		PublicKey:  publicKey,
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
