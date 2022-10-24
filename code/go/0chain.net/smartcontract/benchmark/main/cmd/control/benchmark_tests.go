package control

import (
	"strconv"
	"testing"

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
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (bt BenchTest) Run(balances cstate.TimedQueryStateContext, _ *testing.B) error {
	err := bt.endpoint(balances)
	return err
}

func BenchmarkTests(
	_ bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var tests = []BenchTest{
		{
			name:     "control.access_array." + strconv.Itoa(viper.GetInt(bk.ControlM)),
			endpoint: controlArray,
		},
		{
			name:     "control.access_individual." + strconv.Itoa(viper.GetInt(bk.ControlN)),
			endpoint: controlIndividual,
		},
		{
			name:     "control.update_array." + strconv.Itoa(viper.GetInt(bk.ControlM)),
			endpoint: controlUpdateArray,
		},
		{
			name:     "control.update_individual." + strconv.Itoa(viper.GetInt(bk.ControlN)),
			endpoint: controlUpdateIndividual,
		},
		//{
		//	name:     "control.all_miners." + strconv.Itoa(viper.GetInt(bk.NumMiners)),
		//	endpoint: allMiners,
		//},
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
		err := balances.GetTrieNode(getControlNKey(i), &it)
		if err != nil {
			return err
		}
		itArray = append(itArray, it) //nolint
	}
	return nil
}

func controlUpdateIndividual(balances cstate.StateContextI) error {
	m := viper.GetInt(bk.ControlM)
	n := viper.GetInt(bk.ControlN)
	if m == 0 || n > m {
		return nil
	}

	for i := 0; i < n; i++ {
		var it item
		err := balances.GetTrieNode(getControlNKey(i), &it)
		if err != nil {
			return err
		}

		it.Field = 1
		_, err = balances.InsertTrieNode(getControlNKey(i), &it)
		if err != nil {
			return err
		}
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
	err := balances.GetTrieNode(controlMKey, &ia)
	if err != nil {
		return err
	}

	return nil
}

func controlUpdateArray(balances cstate.StateContextI) error {
	m := viper.GetInt(bk.ControlM)
	n := viper.GetInt(bk.ControlN)
	if m == 0 || n > m {
		return nil
	}

	var ia itemArray
	err := balances.GetTrieNode(controlMKey, &ia)
	if err != nil {
		return err
	}

	ia.Fields = append(ia.Fields, 1)
	_, err = balances.InsertTrieNode(controlMKey, &ia)
	if err != nil {
		return err
	}

	return nil
}

//func allMiners(balances cstate.StateContextI) error {
//	nodesList := &minersc.MinerNodes{}
//	err := balances.GetTrieNode(minersc.AllMinersKey, nodesList)
//	return err
//}
