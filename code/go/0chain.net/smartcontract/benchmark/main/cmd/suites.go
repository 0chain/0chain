package cmd

import (
	"fmt"
	"log"
	"sync"
	"testing"

	"github.com/spf13/pflag"

	"0chain.net/core/util"
	"0chain.net/smartcontract/benchmark"
)

type benchmarkResults struct {
	test   benchmark.BenchTestI
	result testing.BenchmarkResult
}

type suiteResults struct {
	name    string
	results []benchmarkResults
}

func getTestSuites(data benchmark.BenchData, flags *pflag.FlagSet) []benchmark.TestSuit {
	var (
		err     error
		suits   []benchmark.TestSuit
		bkNames []string
	)
	if flags.Changed("tests") {
		bkNames, err = flags.GetStringSlice("tests")
		if err != nil {
			log.Fatal(err)
		}
	}
	if bkNames == nil {
		for _, bks := range benchmarkSources {
			suits = append(suits, bks(data, &BLS0ChainScheme{}))
		}
		return suits
	}
	for _, name := range bkNames {
		if code, ok := benchmark.BenchmarkSourceCode[name]; ok {
			suits = append(suits, benchmarkSources[code](data, &BLS0ChainScheme{}))
		} else {
			log.Fatal(fmt.Errorf("Invalid test source %s", name))
		}
	}
	return suits
}

func runSuites(
	suites []benchmark.TestSuit,
	verbose bool,
	mpt *util.MerklePatriciaTrie,
	root util.Key,
	data benchmark.BenchData,
) []suiteResults {
	var results []suiteResults
	var wg sync.WaitGroup
	for _, suite := range suites {
		wg.Add(1)
		go func(suite benchmark.TestSuit, wg *sync.WaitGroup) {
			defer wg.Done()
			results = append(results, suiteResults{
				name:    benchmark.BenchmarkSourceNames[suite.Source],
				results: runSuite(suite, verbose, mpt, root, data),
			})
		}(suite, &wg)
	}
	wg.Wait()
	return results
}

func runSuite(
	suite benchmark.TestSuit,
	verbose bool,
	mpt *util.MerklePatriciaTrie,
	root util.Key,
	data benchmark.BenchData,
) []benchmarkResults {
	benchmarkResult := []benchmarkResults{}
	var wg sync.WaitGroup
	for _, bm := range suite.Benchmarks {
		wg.Add(1)
		go func(bm benchmark.BenchTestI, wg *sync.WaitGroup) {
			defer wg.Done()
			result := testing.Benchmark(func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					_, balances := getBalances(
						bm.Transaction(),
						extractMpt(mpt, root),
						data,
					)
					b.StartTimer()
					bm.Run(balances)
				}
			})
			benchmarkResult = append(
				benchmarkResult,
				benchmarkResults{
					test:   bm,
					result: result,
				},
			)
			if verbose {
				fmt.Println("test", bm.Name(), "done")
			}
		}(bm, &wg)
	}
	wg.Wait()
	return benchmarkResult
}
