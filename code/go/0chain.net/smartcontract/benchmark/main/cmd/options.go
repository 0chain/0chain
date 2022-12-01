package cmd

import (
	"fmt"
	"strings"

	"0chain.net/core/common"

	bk "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"

	"0chain.net/smartcontract/benchmark/main/cmd/log"
)

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
	data bk.BenchData,
	bkNames, omit []string,
) []bk.TestSuite {
	var name string
	defer func() {
		if r := recover(); r != nil {
			log.Println("piers Recovered benchmark in getTestSuites", r, "test name", name)
		}
	}()
	var suites []bk.TestSuite
	if len(bkNames) == 0 {
		for _, bks := range benchmarkSources {
			suite := bks(data, &BLS0ChainScheme{})
			suite.RemoveBenchmarks(omit)
			suites = append(suites, suite)
		}
		return suites
	}

	common.ConfigRateLimits()
	log.Println("piers bkNames", bkNames)
	for _, name = range bkNames {
		if code, ok := bk.SourceCode[name]; ok {
			log.Println("piers loading test suite", name)
			suite := benchmarkSources[code](data, &BLS0ChainScheme{})
			log.Println("piers loaded test suite", name, "about to omit", omit)
			suite.RemoveBenchmarks(omit)
			log.Println("piers removed omitted from test suite", name)
			suites = append(suites, suite)
		} else {
			log.Fatal(fmt.Errorf("Invalid test source %s", name))
		}
	}
	return suites
}
