package zcnsc

import (
	"github.com/spf13/viper"
	"math/rand"
	"strconv"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
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

// Mint testing stages:
// Create BurnTicketPayload
// Get wallet of a random authorizer
// Get private key from wallet
// Sign payload using authorizer private key
// Collect N signatures
// Send it to mint endpoint

func BenchmarkTests(data benchmark.BenchData, scheme benchmark.SignatureScheme) benchmark.TestSuite {
	sc := createSmartContract()

	indexOfNewAuth := len(data.Clients) - 1

	return createTestSuite(
		[]benchTest{
			{
				name:     benchmark.ZcnSc + AddAuthorizerFunc,
				endpoint: sc.AddAuthorizer,
				txn:      createTransaction(data.Clients[indexOfNewAuth], data.PublicKeys[indexOfNewAuth]),
				input:    createAuthorizerPayload(data, indexOfNewAuth),
			},
			{
				name:     benchmark.ZcnSc + DeleteAuthorizerFunc,
				endpoint: sc.DeleteAuthorizer,
				txn:      createTransaction(data.Clients[0], data.PublicKeys[0]),
				input:    nil,
			},
			{
				name:     benchmark.ZcnSc + BurnFunc,
				endpoint: sc.Burn,
				txn:      createRandomBurnTransaction(data.Clients, data.PublicKeys),
				input:    createBurnPayloadForZCNSCBurn(),
			},
			{
				name:     benchmark.ZcnSc + MintFunc + strconv.Itoa(viper.GetInt(benchmark.NumAuthorizers)) + "Confirmations",
				endpoint: sc.Mint,
				txn:      createRandomTransaction(data.Clients[0], data.PublicKeys[0]),
				input:    createMintPayloadForZCNSCMint(scheme, data),
			},
			{
				name:     benchmark.ZcnSc + MintFunc + "100Confirmation",
				endpoint: sc.Mint,
				txn:      createRandomTransaction(data.Clients[0], data.PublicKeys[0]),
				input:    createMintPayloadForZCNSCMint(scheme, data),
			},
		},
	)
}

func createMintPayloadForZCNSCMint(scheme benchmark.SignatureScheme, data benchmark.BenchData) []byte {
	var sigs []*AuthorizerSignature

	client := data.Clients[1]

	for i := 0; i < viper.GetInt(benchmark.NumAuthorizers); i++ {
		pb := &proofOfBurn{
			TxnID:             encryption.Hash(strconv.Itoa(i)),
			Amount:            100,
			ReceivingClientID: client,
			Nonce:             0,
			Scheme:            scheme,
		}

		err := pb.sign(data.PrivateKeys[i])
		if err != nil {
			panic(err)
		}

		sig := &AuthorizerSignature{
			ID:        data.Clients[i],
			Signature: pb.Signature,
		}

		err = pb.verifySignature(data.PublicKeys[i])
		if err != nil {
			panic(err)
		}

		sigs = append(sigs, sig)
	}

	// mintNonce = mintNonce + 1
	payload := &MintPayload{
		EthereumTxnID:     "0xc8285f5304b1B7aAB09a7d26721D6F585448D0ed",
		Amount:            1,
		Nonce:             mintNonce + 1,
		Signatures:        sigs,
		ReceivingClientID: client,
	}

	return payload.Encode()
}

func createBurnPayloadForZCNSCBurn() []byte {
	burnNonce = burnNonce + 1
	payload := &BurnPayload{
		Nonce:           burnNonce,
		EthereumAddress: "0xc8285f5304b1B7aAB09a7d26721D6F585448D0ed",
	}

	return payload.Encode()
}

func createAuthorizerPayload(data benchmark.BenchData, index int) []byte {
	an := &authorizerNodeArg{
		PublicKey: data.PublicKeys[index],
		URL:       "http://localhost:303" + strconv.Itoa(index),
	}

	return an.Encode()
}

func createRandomTransaction(id, publicKey string) *transaction.Transaction {
	return createTransaction(id, publicKey)
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
		ClientID:     clientId,
		PublicKey:    publicKey,
		ToClientID:   config.SmartContractConfig.GetString(benchmark.BurnAddress),
		Value:        3000,
		CreationDate: common.Now(),
	}
}

func createTransaction(clientId, publicKey string) *transaction.Transaction {
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("mock transaction hash"),
		},
		ClientID:     clientId,
		PublicKey:    publicKey,
		ToClientID:   ADDRESS,
		Value:        3000,
		CreationDate: common.Now(),
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
