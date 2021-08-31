package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
)

type BenchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
		*GlobalNode,
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
	gn, err := getGlobalNode(balances)
	if err != nil {
		panic(err)
	}
	_, err = bt.endpoint(&bt.txn, bt.input, gn, balances)
	if err != nil {
		panic(err)
	}
}

func BenchmarkTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuit {
	//var now = common.Timestamp(viper.GetInt64(sc.Now))
	var msc = MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	var tests = []BenchTest{
		{
			name:     "miner.add_miner",
			endpoint: msc.AddMiner,
			txn:      transaction.Transaction{},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{},
			}).Encode(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{bk.MinerTrans, testsI}
}
