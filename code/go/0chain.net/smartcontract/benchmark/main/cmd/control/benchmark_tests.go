package control

import (
	"fmt"
	"strconv"
	"testing"

	"0chain.net/smartcontract/minersc"

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

func (bt BenchTest) Run(balances cstate.StateContextI, _ *testing.B) error {
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
		var it *item
		raw, err := balances.GetTrieNode(getControlNKey(i), it)
		var ok bool
		if it, ok = raw.(*item); !ok {
			return fmt.Errorf("unexpected node type")
		}
		if err != nil {
			return err
		}
		itArray = append(itArray, *it)
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
		var it *item
		raw, err := balances.GetTrieNode(getControlNKey(i), it)
		if err != nil {
			return err
		}
		var ok bool
		if it, ok = raw.(*item); !ok {
			return fmt.Errorf("unexpected node type")
		}
		it.Field = 1
		err = balances.InsertTrieNode(getControlNKey(i), it)
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

	var ia *itemArray
	raw, err := balances.GetTrieNode(controlMKey, ia)
	if err != nil {
		return err
	}
	var ok bool
	if ia, ok = raw.(*itemArray); !ok {
		return fmt.Errorf("unexpected node type")
	}
	return nil
}

func controlUpdateArray(balances cstate.StateContextI) error {
	m := viper.GetInt(bk.ControlM)
	n := viper.GetInt(bk.ControlN)
	if m == 0 || n > m {
		return nil
	}

	var ia *itemArray
	raw, err := balances.GetTrieNode(controlMKey, ia)
	if err != nil {
		return err
	}
	var ok bool
	if ia, ok = raw.(*itemArray); !ok {
		return fmt.Errorf("unexpected node type")
	}
	ia.Fields = append(ia.Fields, 1)
	err = balances.InsertTrieNode(controlMKey, ia)
	if err != nil {
		return err
	}

	return nil
}

func allMiners(balances cstate.StateContextI) error {
	nodesList := &minersc.MinerNodes{}
	raw, err := balances.GetTrieNode(minersc.AllMinersKey, nodesList)
	var ok bool
	if nodesList, ok = raw.(*minersc.MinerNodes); !ok {
		return fmt.Errorf("unexpected node type")
	}
	return err
}
