package multisigsc

import (
	"encoding/json"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	bk "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

type BenchTest struct {
	name     string
	endpoint string
	txn      *transaction.Transaction
	input    []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: bt.txn.Hash,
		},
		ClientID:     bt.txn.ClientID,
		ToClientID:   bt.txn.ToClientID,
		ValueZCN:     bt.txn.ValueZCN,
		CreationDate: bt.txn.CreationDate,
	}
}

func (bt BenchTest) Run(balances cstate.StateContextI, _ *testing.B) error {
	var msc = MultiSigSmartContract{
		SmartContract: sci.NewSC(Address),
	}
	msc.setSC(msc.SmartContract, &smartcontract.BCContext{})
	var err error
	switch bt.endpoint {
	case RegisterFuncName:
		_, err = msc.register(
			bt.txn.ClientID,
			bt.input,
			balances,
		)
	case VoteFuncName:
		_, err = msc.vote(
			bt.txn.Hash,
			bt.txn.ClientID,
			balances.GetBlock().CreationDate,
			bt.input,
			balances,
		)
	default:
		panic("unknown endpoint: " + bt.endpoint)
	}

	return err
}

func BenchmarkTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuite {
	var tests = []BenchTest{
		{
			name:     "multi_sig." + RegisterFuncName,
			endpoint: RegisterFuncName,
			txn: &transaction.Transaction{
				ClientID: data.Clients[len(data.Clients)-1],
			},
			input: func() []byte {
				wallet := &Wallet{
					ClientID:           data.Clients[len(data.Clients)-1],
					SignatureScheme:    viper.GetString(bk.InternalSignatureScheme),
					PublicKey:          data.PublicKeys[len(data.PublicKeys)-1],
					SignerThresholdIDs: data.Clients[:MaxSigners],
					SignerPublicKeys:   data.PublicKeys[:MaxSigners],
					NumRequired:        MaxSigners,
				}
				return wallet.Encode()
			}(),
		},
		{
			name:     "multi_sig." + VoteFuncName,
			endpoint: VoteFuncName,
			txn: &transaction.Transaction{
				ClientID: data.Clients[0],
				HashIDField: datastore.HashIDField{
					Hash: "my hash",
				},
			},
			input: func() []byte {
				st := &state.SignedTransfer{
					Transfer: state.Transfer{
						ClientID:   data.Clients[0],
						ToClientID: data.Clients[1],
						Amount:     1,
					},
					SchemeName: viper.GetString(bk.InternalSignatureScheme),
					PublicKey:  data.PublicKeys[0],
				}
				_ = sigScheme.SetPublicKey(data.PublicKeys[0])
				sigScheme.SetPrivateKey(data.PrivateKeys[0])
				signature, _ := sigScheme.Sign(encryption.Hash(st.Transfer.Encode()))
				bytes, _ := json.Marshal(&Vote{
					Transfer:  st.Transfer,
					Signature: signature,
				})
				return bytes
			}(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.MultiSig,
		Benchmarks: testsI,
	}
}
