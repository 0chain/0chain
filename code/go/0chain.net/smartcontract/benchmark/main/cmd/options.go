package cmd

import (
	"fmt"
	"strings"

	bk "0chain.net/smartcontract/benchmark"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"0chain.net/smartcontract/benchmark/main/cmd/log"
)

func setupOptions(flags *pflag.FlagSet) ([]string, []string) {
	var err error
	var verbose bool
	if flags.Changed("verbose") {
		verbose, err = flags.GetBool("verbose")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		verbose = viper.GetBool(bk.OptionVerbose)
	}
	log.SetVerbose(verbose)

	var testSuites []string
	if flags.Changed("tests") {
		testSuites, err = flags.GetStringSlice("tests")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		testSuites = viper.GetStringSlice(bk.OptionTestSuites)
	}
	for i := 0; i < len(testSuites); i++ {
		testSuites[i] = strings.TrimSpace(testSuites[i])
	}

	var omit []string
	if flags.Changed("omit") {
		omit, err = flags.GetStringSlice("omit")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		omit = viper.GetStringSlice(bk.OptionOmittedTests)
	}
	for i := 0; i < len(omit); i++ {
		omit[i] = strings.TrimSpace(omit[i])
	}
	return testSuites, omit
}

func getTestSuites(
	data bk.BenchData,
	bkNames, omit []string,
) []bk.TestSuite {
	var suites []bk.TestSuite
	if len(bkNames) == 0 {
		for _, bks := range benchmarkSources {
			suite := bks(data, &BLS0ChainScheme{})
			suite.RemoveBenchmarks(omit)
			suites = append(suites, suite)
		}
		return suites
	}
	for _, name := range bkNames {
		if code, ok := bk.SourceCode[name]; ok {
			suite := benchmarkSources[code](data, &BLS0ChainScheme{})
			suite.RemoveBenchmarks(omit)
			suites = append(suites, suite)
			//suites = append(suites, benchmarkSources[code](data, &BLS0ChainScheme{}))
		} else {
			log.Fatal(fmt.Errorf("Invalid test source %s", name))
		}
	}
	return suites
}
