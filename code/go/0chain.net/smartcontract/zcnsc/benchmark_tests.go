package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
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

func (bt benchTest) Run(state cstate.StateContextI, b *testing.B) error {
	b.Logf("Running test '%s' from ZCNSC Bridge", bt.name)
	_, err := bt.endpoint(bt.Transaction(), bt.input, state)
	return err
}

func BenchmarkTests(data benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	sc := createSmartContract()

	return createTestSuite(
		[]benchTest{
			{
				name:     benchmark.Zcn + AddAuthorizerFunc,
				endpoint: sc.AddAuthorizer,
				txn:      createTransaction(data.Clients[addingAuthorizer], data.PublicKeys[addingAuthorizer]),
				input:    createAuthorizer(data.PublicKeys[addingAuthorizer], addingAuthorizer),
			},
			{
				name:     benchmark.Zcn + DeleteAuthorizerFunc,
				endpoint: sc.DeleteAuthorizer,
				txn:      createTransaction(data.Clients[removableAuthorizer], data.PublicKeys[removableAuthorizer]),
				input:    nil,
			},
			{
				name:     benchmark.Zcn + BurnFunc,
				endpoint: sc.Burn,
				txn:      createRandomBurnTransaction(data.Clients, data.PublicKeys),
				input:    createBurnPayload(),
			},
			{
				name:     benchmark.Zcn + MintFunc,
				endpoint: sc.Mint,
				txn:      createRandomTransaction(data.Clients, data.PublicKeys),
				input:    createMintPayload(),
			},
		},
	)
}

func createMintPayload() []byte {
	nonce = nonce + 1
	payload := MintPayload{
		EthereumTxnID:     "0xc8285f5304b1B7aAB09a7d26721D6F585448D0ed",
		Amount:            1,
		Nonce:             nonce,
		Signatures:        nil, // TODO: fill
		ReceivingClientID: "",  // TODO: fill
	}
	return payload.Encode()
}

func createBurnPayload() []byte {
	nonce = nonce + 1
	payload := BurnPayload{
		Nonce:           nonce,
		EthereumAddress: "0xc8285f5304b1B7aAB09a7d26721D6F585448D0ed",
	}
	return payload.Encode()
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

func createRandomBurnTransaction(clients, publicKey []string) *transaction.Transaction {
	index := randomIndex(len(clients))
	return createBurnTransaction(clients[index], publicKey[index])
}

func createBurnTransaction(clientId, publicKey string) *transaction.Transaction {
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("mock transaction hash"),
		},
		ClientID:   clientId,
		PublicKey:  publicKey,
		ToClientID: config.SmartContractConfig.GetString(benchmark.BurnAddress),
		Value:      3000,
	}
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
