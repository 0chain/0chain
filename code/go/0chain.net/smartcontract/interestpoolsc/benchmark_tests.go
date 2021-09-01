package interestpoolsc

import (
	"0chain.net/core/common"
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
	txn      transaction.Transaction
	input    []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() transaction.Transaction {
	return bt.txn
}

func (bt BenchTest) Run(balances cstate.StateContextI) {
	var isc = InterestPoolSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	isc.setSC(isc.SmartContract, &smartcontract.BCContext{})
	un := isc.getUserNode(bt.txn.ClientID, balances)
	gn := isc.getGlobalNode(balances, bt.endpoint)
	var err error
	switch bt.endpoint {
	case "lock":
		_, err = isc.lock(&bt.txn, un, gn, bt.input, balances)
	case "unlock":
		_, err = isc.unlock(&bt.txn, un, gn, bt.input, balances)
	default:
		panic("unknown endpoint: " + bt.endpoint)
	}
	if err != nil {
		panic(err)
	}
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuit {
	var tests = []BenchTest{
		// todo updateVariables waiting for Pr 487
		{
			name:     "interest_pool.lock",
			endpoint: "lock",
			txn: transaction.Transaction{
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
			txn: transaction.Transaction{
				CreationDate: 2 * common.Timestamp(viper.GetDuration(bk.InterestPoolMinLockPeriod)),
				ClientID:     data.Clients[0],
			},
			input: (&poolStat{
				ID: getInterestPoolId(0),
			}).encode(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{bk.Faucet, testsI}
}
