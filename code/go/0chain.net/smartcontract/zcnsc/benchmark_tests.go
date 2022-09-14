package zcnsc

import (
	"log"
	"math/rand"
	"strconv"
	"testing"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/spf13/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
)

const (
	owner = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"
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

func (bt benchTest) Run(state cstate.TimedQueryStateContext, b *testing.B) error {
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
				name:     benchmark.ZcnSc + UpdateGlobalConfigFunc,
				endpoint: sc.UpdateGlobalConfig,
				txn:      createTransaction(owner, ""),
				input: (&smartcontract.StringMap{
					Fields: map[string]string{
						MinMintAmount:      "2",
						MinBurnAmount:      "3",
						MinStakeAmount:     "1",
						MinLockAmount:      "4",
						MinAuthorizers:     "17",
						PercentAuthorizers: "73",
						MaxFee:             "800",
						BurnAddress:        "7000000000000000000000000000000000000000000000000000000000000000",
					},
				}).Encode(),
			},
			{
				name:     benchmark.ZcnSc + UpdateAuthorizerConfigFunc,
				endpoint: sc.UpdateAuthorizerConfig,
				txn:      createTransaction(data.Clients[0], data.PublicKeys[0]),
				input: (&AuthorizerNode{
					ID:        data.Clients[0],
					PublicKey: data.PublicKeys[0],
					URL:       "http://localhost:3030",
					Config: &AuthorizerConfig{
						Fee: currency.Coin(viper.GetInt(benchmark.ZcnMaxFee) / 2),
					},
				}).Encode(),
			},
			{
				name:     benchmark.ZcnSc + UpdateAuthorizerStakePoolFunc,
				endpoint: sc.UpdateAuthorizerStakePool,
				txn:      createTransaction(data.Clients[0], data.PublicKeys[0]),
				input: (&UpdateAuthorizerStakePoolPayload{
					StakePoolSettings: stakepool.Settings{
						DelegateWallet:     data.Clients[0],
						MinStake:           currency.Coin(1.1 * 1e10),
						MaxStake:           currency.Coin(103 * 1e10),
						MaxNumDelegates:    7,
						ServiceChargeRatio: 0.17,
					},
				}).Encode(),
			},
			{
				name:     benchmark.ZcnSc + CollectRewardsFunc,
				endpoint: sc.CollectRewards,
				txn:      createTransaction(data.Clients[0], data.PublicKeys[0]),
				input: (&stakepool.CollectRewardRequest{
					ProviderType: spenum.Authorizer,
					PoolId:       getMockAuthoriserStakePoolId(data.Clients[0], 0),
				}).Encode(),
			},
			{
				name:     benchmark.ZcnSc + AddToDelegatePoolFunc,
				endpoint: sc.AddToDelegatePool,
				txn:      createTransaction(data.Clients[0], data.PublicKeys[0]),
				input: (&stakePoolRequest{
					AuthorizerID: data.Clients[0],
				}).encode(),
			},
			{
				name:     benchmark.ZcnSc + DeleteFromDelegatePoolFunc,
				endpoint: sc.DeleteFromDelegatePool,
				txn:      createTransaction(data.Clients[0], data.PublicKeys[0]),
				input: (&stakePoolRequest{
					PoolID:       getMockAuthoriserStakePoolId(data.Clients[0], 0),
					AuthorizerID: data.Clients[0],
				}).encode(),
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
	payload := &BurnPayload{
		EthereumAddress: "0xc8285f5304b1B7aAB09a7d26721D6F585448D0ed",
	}

	return payload.Encode()
}

func createAuthorizerPayload(data benchmark.BenchData, index int) []byte {
	an := &AddAuthorizerPayload{
		PublicKey:         data.PublicKeys[index],
		URL:               "http://localhost:303" + strconv.Itoa(index),
		StakePoolSettings: getMockStakePoolSettings(data.Clients[index]),
	}
	ap, err := an.Encode()
	if err != nil {
		log.Fatal(err)
	}
	return ap
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
		ToClientID:   config.SmartContractConfig.GetString(benchmark.ZcnBurnAddress),
		Value:        3000,
		CreationDate: common.Now(),
	}
}

func createTransaction(clientId, publicKey string) *transaction.Transaction {
	creationTimeRaw := viper.GetInt64(benchmark.MptCreationTime)
	creationTime := common.Now()
	if creationTimeRaw != 0 {
		creationTime = common.Timestamp(creationTimeRaw)
	}
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("mock transaction hash"),
		},
		ClientID:     clientId,
		PublicKey:    publicKey,
		ToClientID:   ADDRESS,
		Value:        3000,
		CreationDate: creationTime,
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
