package control

import (
	"fmt"
	"strconv"
	"testing"

	"0chain.net/smartcontract/minersc"

	"0chain.net/core/common"

	"github.com/spf13/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
)

type BenchTest struct {
	name     string
	endpoint func(
		cstate.StateContextI,
	) error
	txn   *transaction.Transaction
	input []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (bt BenchTest) Run(balances cstate.StateContextI, _ *testing.B) {
	err := bt.endpoint(balances)
	if err != nil {
		panic(err)
	}
}

func BenchmarkTests(
	_ bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var tests = []BenchTest{
		{
			name:     "control.array." + strconv.Itoa(viper.GetInt(bk.ControlM)),
			endpoint: controlArray,
		},
		{
			name:     "control.individual." + strconv.Itoa(viper.GetInt(bk.ControlN)),
			endpoint: controlIndividual,
		},
		{
			name:     "control.all_miners." + strconv.Itoa(viper.GetInt(bk.NumMiners)),
			endpoint: allMiners,
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.Storage,
		Benchmarks: testsI,
	}
}

func controlIndividual(balances cstate.StateContextI) error {
	m := viper.GetInt(bk.ControlM)
	n := viper.GetInt(bk.ControlN)
	if m == 0 || n > m {
		return nil
	}

	var itArray []item
	for i := 0; i < n; i++ {
		var it item
		val, err := balances.GetTrieNode(getControlNKey(i))
		if err != nil {
			return err
		}
		if err := it.Decode(val.Encode()); err != nil {
			return fmt.Errorf("%w: %s", common.ErrDecoding, err)
		}
		itArray = append(itArray, it)
	}
	return nil
}

func controlArray(balances cstate.StateContextI) error {
	m := viper.GetInt(bk.ControlM)
	n := viper.GetInt(bk.ControlN)
	if m == 0 || n > m {
		return nil
	}

	var ia itemArray
	val, err := balances.GetTrieNode(controlMKey)
	if err != nil {
		return err
	}
	if err := ia.Decode(val.Encode()); err != nil {
		return fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return nil
}

func allMiners(balances cstate.StateContextI) error {
	nodesBytes, err := balances.GetTrieNode(minersc.AllMinersKey)
	if err != nil {
		return err
	}

	nodesList := &minersc.MinerNodes{}
	if err = nodesList.Decode(nodesBytes.Encode()); err != nil {
		return err
	}
	return nil
}
