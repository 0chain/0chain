package cmd

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/rest"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"fmt"
	"sync"
	"testing"
	"time"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

	"0chain.net/core/util"
	"0chain.net/smartcontract/benchmark"
)

type benchmarkResults struct {
	test    benchmark.BenchTestI
	result  testing.BenchmarkResult
	timings map[string]time.Duration
	error
}

type suiteResults struct {
	name    string
	results []benchmarkResults
}

type chainer struct {
	qsc cstate.QueryStateContextI
}

func (ch *chainer) GetQueryStateContext() cstate.QueryStateContextI {
	return ch.qsc
}

func (ch *chainer) SetQueryStateContext(qsc cstate.QueryStateContextI) {
	ch.qsc = qsc
}

func runSuites(
	suites []benchmark.TestSuite,
	mpt *util.MerklePatriciaTrie,
	root util.Key,
	data benchmark.BenchData,
) []suiteResults {
	var results []suiteResults
	var wg sync.WaitGroup

	_, readOnlyBalances := getBalances(
		&transaction.Transaction{},
		extractMpt(mpt, root),
		data,
	)
	restSetup := &rest.RestHandler{
		QueryChainer: &chainer{
			qsc: readOnlyBalances,
		},
	}
	faucetsc.SetupRestHandler(restSetup)
	minersc.SetupRestHandler(restSetup)
	storagesc.SetupRestHandler(restSetup)
	vestingsc.SetupRestHandler(restSetup)
	zcnsc.SetupRestHandler(restSetup)

	for _, suite := range suites {
		log.Println("starting suite ==>", suite.Source)
		wg.Add(1)
		go func(suite benchmark.TestSuite, wg *sync.WaitGroup) {
			defer wg.Done()
			var suiteResult []benchmarkResults
			if suite.ReadOnly {
				suiteResult = runReadOnlySuite(suite, mpt, root, data, readOnlyBalances)
			} else {
				suiteResult = runSuite(suite, mpt, root, data)
			}
			if suiteResult == nil {
				return
			}
			results = append(results, suiteResults{
				name:    benchmark.SourceNames[suite.Source],
				results: suiteResult,
			})
		}(suite, &wg)
	}
	wg.Wait()
	return results
}

func runReadOnlySuite(
	suite benchmark.TestSuite,
	mpt *util.MerklePatriciaTrie,
	root util.Key,
	data benchmark.BenchData,
	balances cstate.StateContextI,
) []benchmarkResults {
	if !viper.GetBool(benchmark.EventDbEnabled) || balances.GetEventDB() == nil {
		log.Println("event database not enabled, skipping ", suite.Source.String())
		return nil
	}

	var benchmarkResult []benchmarkResults
	var wg sync.WaitGroup
	for _, bm := range suite.Benchmarks {
		wg.Add(1)
		go func(bm benchmark.BenchTestI, wg *sync.WaitGroup) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered in benchmark test", bm.Name(), "message", r)
				}
			}()
			timer := time.Now()
			log.Println("starting", bm.Name())
			var err error
			result := testing.Benchmark(func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					err = bm.Run(balances, b)
				}
			})
			benchmarkResult = append(
				benchmarkResult,
				benchmarkResults{
					test:   bm,
					result: result,
					error:  err,
				},
			)
			log.Println("test", bm.Name(), "done. took:", time.Since(timer))
		}(bm, &wg)
	}
	wg.Wait()
	return benchmarkResult
}

func runSuite(
	suite benchmark.TestSuite,
	mpt *util.MerklePatriciaTrie,
	root util.Key,
	data benchmark.BenchData,
) []benchmarkResults {
	var benchmarkResult []benchmarkResults
	var wg sync.WaitGroup

	for _, bm := range suite.Benchmarks {
		wg.Add(1)
		go func(bm benchmark.BenchTestI, wg *sync.WaitGroup) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered in benchmark test", bm.Name(), "message", r)
				}
			}()
			timer := time.Now()
			log.Println("starting", bm.Name())
			var err error

			result := testing.Benchmark(func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					_, balances := getBalances(
						bm.Transaction(),
						extractMpt(mpt, root),
						data,
					)
					b.StartTimer()
					err = bm.Run(balances, b)
					if err != nil {
						mockUpdateState(bm.Transaction(), balances)
					}
				}
			})
			var resTimings map[string]time.Duration
			if wt, ok := bm.(benchmark.WithTimings); ok && len(wt.Timings()) > 0 {
				resTimings = wt.Timings()
			}

			benchmarkResult = append(
				benchmarkResult,
				benchmarkResults{
					test:    bm,
					result:  result,
					error:   err,
					timings: resTimings,
				},
			)

			log.Println("test", bm.Name(), "done. took:", time.Since(timer))
		}(bm, &wg)
	}
	wg.Wait()
	return benchmarkResult
}
