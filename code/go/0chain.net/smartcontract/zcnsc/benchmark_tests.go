package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/benchmark"
	"github.com/stretchr/testify/require"
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
				txn:      createTransaction(),
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

// TODO: complete transaction
func createTransaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func randomIndex(num int) int {
	return 0
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
