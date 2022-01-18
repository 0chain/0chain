package zcnsc

import (
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

	authToDelete := authorizers[0]
	indexOfNewAuth := len(authorizers)

	return createTestSuite(
		[]benchTest{
			{
				name:     benchmark.Zcn + AddAuthorizerFunc,
				endpoint: sc.AddAuthorizer,
				txn:      createTransaction(data.Clients[indexOfNewAuth], data.PublicKeys[indexOfNewAuth]),
				input:    createAuthorizerPayload(data, indexOfNewAuth),
			},
			{
				name:     benchmark.Zcn + DeleteAuthorizerFunc,
				endpoint: sc.DeleteAuthorizer,
				txn:      createTransaction(authToDelete.ID, authToDelete.PublicKey),
				input:    nil,
			},
			{
				name:     benchmark.Zcn + BurnFunc,
				endpoint: sc.Burn,
				txn:      createRandomBurnTransaction(data.Clients, data.PublicKeys),
				input:    createBurnPayloadForZCNSCBurn(),
			},
			{
				name:     benchmark.Zcn + MintFunc + ".1Confirmation",
				endpoint: sc.Mint,
				txn:      createRandomTransaction(),
				input:    createMintPayloadForZCNSCMint(scheme, data, 0, 1),
			},
			{
				name:     benchmark.Zcn + MintFunc + ".10Confirmation",
				endpoint: sc.Mint,
				txn:      createRandomTransaction(),
				input:    createMintPayloadForZCNSCMint(scheme, data, 1, 10),
			},
			{
				name:     benchmark.Zcn + MintFunc + "100Confirmation",
				endpoint: sc.Mint,
				txn:      createRandomTransaction(),
				input:    createMintPayloadForZCNSCMint(scheme, data, 10, 110),
			},
		},
	)
}

func createMintPayloadForZCNSCMint(scheme benchmark.SignatureScheme, data benchmark.BenchData, from, to int) []byte {
	var sigs []*AuthorizerSignature

	client := data.Clients[1]
	lim := len(authorizers)

	for i := from; i < to && i < lim; i++ {

		auth := authorizers[i]

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
			ID:        auth.ID,
			Signature: pb.Signature,
		}

		err = pb.verifySignature(auth.PublicKey)
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

func createRandomTransaction() *transaction.Transaction {
	index := randomIndex(len(authorizers))
	auth := authorizers[index]
	return createTransaction(auth.ID, auth.PublicKey)
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
