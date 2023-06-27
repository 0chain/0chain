package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"0chain.net/smartcontract/benchmark/main/cmd/control"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"

	"0chain.net/core/common"

	bk "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"

	"0chain.net/smartcontract/benchmark/main/cmd/log"
)

var benchmarkSources = map[bk.Source]func(data bk.BenchData, sigScheme bk.SignatureScheme) bk.TestSuite{
	bk.Storage:         storagesc.BenchmarkTests,
	bk.StorageRest:     storagesc.BenchmarkRestTests,
	bk.Miner:           minersc.BenchmarkTests,
	bk.MinerRest:       minersc.BenchmarkRestTests,
	bk.Faucet:          faucetsc.BenchmarkTests,
	bk.FaucetRest:      faucetsc.BenchmarkRestTests,
	bk.Vesting:         vestingsc.BenchmarkTests,
	bk.VestingRest:     vestingsc.BenchmarkRestTests,
	bk.MultiSig:        multisigsc.BenchmarkTests,
	bk.ZCNSCBridge:     zcnsc.BenchmarkTests,
	bk.ZCNSCBridgeRest: zcnsc.BenchmarkRestTests,
	bk.Control:         control.BenchmarkTests,
}

/* piers remove
func getBenchmarkSources() (map[bk.Source]func(data bk.BenchData, sigScheme bk.SignatureScheme) bk.TestSuite, error) {
	_, ok := benchmarkSources[bk.EventDatabase]
	if ok {
		return benchmarkSources, nil
	}
	inTests := viper.GetString(bk.OptionsEventDatabaseEventFile)
	if inTests == "" {
		return benchmarkSources, nil
	}
	testMap, err := readEdbTests(inTests)
	if err != nil {
		return nil, err
	}
	benchmarkSources[bk.EventDatabase] = event.GetBenchmarkTests(testMap)
	return benchmarkSources, nil
}
*/

func readEdbTests(filename string) (map[string][]event.Event, error) {
	file, _ := ioutil.ReadFile(filename)
	var data map[string][]event.Event
	err := json.Unmarshal(file, &data)
	return data, err
}

func suitesOmits() ([]string, []string) {
	verbose := viper.GetBool(bk.OptionVerbose)
	log.SetVerbose(verbose)

	testSuites := viper.GetStringSlice(bk.OptionTestSuites)
	for i := 0; i < len(testSuites); i++ {
		testSuites[i] = strings.TrimSpace(testSuites[i])
	}

	omit := viper.GetStringSlice(bk.OptionOmittedTests)
	for i := 0; i < len(omit); i++ {
		omit[i] = strings.TrimSpace(omit[i])
	}
	return testSuites, omit
}

func getTestSuites(
	data *bk.BenchData,
	bkNames, omit []string,
) []bk.TestSuite {
	var suites []bk.TestSuite
	if len(bkNames) == 0 {
		for _, bks := range benchmarkSources {
			suite := bks(*data, &BLS0ChainScheme{})
			suite.RemoveBenchmarks(omit)
			suites = append(suites, suite)
		}
		return suites
	}

	common.ConfigRateLimits()

	for _, name := range bkNames {
		if code, ok := bk.SourceCode[name]; ok {
			suite := benchmarkSources[code](*data, &BLS0ChainScheme{})
			suite.RemoveBenchmarks(omit)
			suites = append(suites, suite)
		} else {
			log.Fatal(fmt.Errorf("Invalid test source %s", name))
		}
	}
	return suites
}
