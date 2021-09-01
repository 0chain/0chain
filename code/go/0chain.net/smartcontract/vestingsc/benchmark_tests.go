package vestingsc

import (
	"encoding/json"
	"time"

	"github.com/spf13/viper"

	"0chain.net/core/common"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
)

type BenchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
		cstate.StateContextI,
	) (string, error)
	txn   transaction.Transaction
	input []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() transaction.Transaction {
	return bt.txn
}

func (bt BenchTest) Run(balances cstate.StateContextI) {
	_, err := bt.endpoint(&bt.txn, bt.input, balances)
	if err != nil {
		panic(err) // todo temporary, remove later
	}
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuit {
	var vsc = VestingSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	vsc.setSC(vsc.SmartContract, &smartcontract.BCContext{})
	var tests = []BenchTest{
		{
			name:     "vesting.add",
			endpoint: vsc.add,
			txn: transaction.Transaction{
				ClientID: data.Clients[0],
				Value:    int64(viper.GetFloat64(bk.VestingMinLock) * 1e10),
			},
			input: func() []byte {
				var dests destinations
				for i := 0; i < viper.GetInt(bk.VestingMaxDestinations); i++ {
					dests = append(dests, &destination{})
				}
				bytes, _ := json.Marshal(&addRequest{
					Description:  "my description",
					StartTime:    common.Timestamp(100),
					Duration:     time.Hour,
					Destinations: dests,
				})
				return bytes
			}(),
		}, /*
			{
				name:     "vesting.trigger",
				endpoint: vsc.trigger,
				txn:      transaction.Transaction{},
				input: func() []byte {
					bytes, _ := json.Marshal(&poolRequest{
						PoolID: "my pool",
					})
					return bytes
				}(),
			},*/
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{bk.Storage, testsI}
}
