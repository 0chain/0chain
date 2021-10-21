package interestpoolsc

import (
	"testing"

	sc "0chain.net/smartcontract"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/spf13/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
)

type BenchTest struct {
	name     string
	endpoint string
	txn      *transaction.Transaction
	input    []byte
	error    string
}

func (bt BenchTest) Error() string {
	return bt.error
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
		Value:        bt.txn.Value,
		CreationDate: bt.txn.CreationDate,
	}
}

func (bt BenchTest) Run(balances cstate.StateContextI, _ *testing.B) error {
	var isc = InterestPoolSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	isc.setSC(isc.SmartContract, &smartcontract.BCContext{})
	un := isc.getUserNode(bt.txn.ClientID, balances)
	gn := isc.getGlobalNode(balances, bt.endpoint)
	var err error
	switch bt.endpoint {
	case "lock":
		_, err = isc.lock(bt.Transaction(), un, gn, bt.input, balances)
	case "unlock":
		_, err = isc.unlock(bt.Transaction(), un, gn, bt.input, balances)
	case "updateVariables":
		_, err = isc.updateVariables(bt.Transaction(), gn, bt.input, balances)
	default:
		panic("unknown endpoint: " + bt.endpoint)
	}

	return err
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var tests = []BenchTest{
		{
			name:     "interest_pool.lock",
			endpoint: "lock",
			txn: &transaction.Transaction{
				Value:      int64(viper.GetFloat64(bk.InterestPoolMinLock) * 1e10),
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: (&newPoolRequest{
				Duration: viper.GetDuration(bk.InterestPoolMinLockPeriod),
			}).encode(),
		},
		{
			name:     "interest_pool.unlock",
			endpoint: "unlock",
			txn: &transaction.Transaction{
				CreationDate: 2 * common.Timestamp(viper.GetDuration(bk.InterestPoolMinLockPeriod)),
				ClientID:     data.Clients[0],
			},
			input: (&poolStat{
				ID: getInterestPoolId(0),
			}).encode(),
		},
		{
			name:     "interest_pool.updateVariables",
			endpoint: "updateVariables",
			txn: &transaction.Transaction{
				ClientID: owner,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					Settings[MinLock]:       "1",
					Settings[Apr]:           "0.2",
					Settings[MinLockPeriod]: "3m",
					Settings[MaxMint]:       "5",
				},
			}).Encode(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.InterestPool,
		Benchmarks: testsI,
	}
}
