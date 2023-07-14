package cmd

import (
	"fmt"
	"sync"
	"testing"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/rest"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/benchmark"
	"github.com/0chain/common/core/util"
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
	qsc cstate.TimedQueryStateContextI
}

func (ch *chainer) GetQueryStateContext() cstate.TimedQueryStateContextI {
	return ch.qsc
}

func (ch *chainer) SetQueryStateContext(qsc cstate.TimedQueryStateContextI) {
	ch.qsc = qsc
}

func (ch *chainer) GetStateContextI() cstate.StateContextI {
	return ch.qsc.(cstate.StateContextI)
}

func runSuites(
	suites []benchmark.TestSuite,
	mpt *util.MerklePatriciaTrie,
	root util.Key,
	data *benchmark.BenchData,
) []suiteResults {
	var results []suiteResults
	var wg sync.WaitGroup

	_, readOnlyBalances := getBalances(
		&transaction.Transaction{},
		extractMpt(mpt, root),
		data,
	)
	timedBalance := cstate.NewTimedQueryStateContext(readOnlyBalances, func() common.Timestamp {
		return data.Now
	})
	restSetup := &rest.RestHandler{
		QueryChainer: &chainer{
			qsc: timedBalance,
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
				suiteResult = runReadOnlySuite(suite, mpt, root, data, timedBalance)
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
	_ *util.MerklePatriciaTrie,
	_ util.Key,
	_ *benchmark.BenchData,
	balances cstate.TimedQueryStateContext,
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
	data *benchmark.BenchData,
) []benchmarkResults {
	var benchmarkResult []benchmarkResults
	var wg sync.WaitGroup
	scAddresses := []string{
		minersc.ADDRESS,
		storagesc.ADDRESS,
		faucetsc.ADDRESS,
		zcnsc.ADDRESS,
	}
	clientsMap := make(map[string]struct{}, len(data.Clients))
	for _, c := range append(data.Clients, scAddresses...) {
		clientsMap[c] = struct{}{}
	}

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
			var runCount int
			result := testing.Benchmark(func(b *testing.B) {
				b.StopTimer()
				var prevMptHashRoot string
				_ = prevMptHashRoot
				for i := 0; i < b.N; i++ {
					cloneMPT := util.CloneMPT(mpt)
					_, balances := getBalances(
						bm.Transaction(),
						extractMpt(cloneMPT, root),
						data,
					)
					timedBalance := cstate.NewTimedQueryStateContext(balances, func() common.Timestamp {
						return data.Now
					})

					// do the balances checking only once, otherwise it would slow down the tests too much
					var totalBalanceBefore currency.Coin
					if i == 0 {
						// get client balances and all delegate pools' balances before running the test
						// compare it
						for _, c := range append(data.Clients, scAddresses...) {
							bal, err := timedBalance.GetClientBalance(c)
							if err != nil {
								log.Fatal(err)
							}
							totalBalanceBefore += bal
						}
					}

					b.StartTimer()
					err = bm.Run(timedBalance, b)
					b.StopTimer()
					// data.Clients is subset of all clients, so we need to check if there are
					// any unknown clients that minted to or transferred to
					unknownMintTransferClients := make(map[string]struct{})
					if err == nil {
						ms := timedBalance.GetMints()
						for _, m := range ms {
							if _, ok := clientsMap[m.ToClientID]; !ok {
								unknownMintTransferClients[m.ToClientID] = struct{}{}

							}
						}

						for _, tt := range timedBalance.GetTransfers() {
							if _, ok := clientsMap[tt.ToClientID]; !ok {
								unknownMintTransferClients[tt.ToClientID] = struct{}{}
							}
						}

						for c := range unknownMintTransferClients {
							bl, err := balances.GetClientBalance(c)
							if err != nil {
								log.Fatal(err)
							}
							totalBalanceBefore += bl
						}

						mockUpdateState(bm.Name(), bm.Transaction(), balances)
					}
					runCount++
					currMptHashRoot := util.ToHex(timedBalance.GetState().GetRoot())
					if i > 0 && currMptHashRoot != prevMptHashRoot {
						log.Println("MPT state root mismatch detected! benchmark test name:", bm.Name())
						log.Println("Run:", i, "Previous MPT state root:", prevMptHashRoot, "Current MPT state root:", currMptHashRoot)
						err = fmt.Errorf("MPT hash root mismatch detected: running same function resulted in different MPT states")
						b.FailNow()
					} else {
						prevMptHashRoot = currMptHashRoot
					}

					if i == 0 {
						// get balances after mints
						unknownAddresses := make([]string, 0, len(unknownMintTransferClients))
						for c := range unknownMintTransferClients {
							unknownAddresses = append(unknownAddresses, c)
						}
						var totalBalanceAfter currency.Coin
						for _, c := range append(append(data.Clients, scAddresses...),
							unknownAddresses...) {
							bal, err := timedBalance.GetClientBalance(c)
							if err != nil {
								log.Fatal(err)
							}
							totalBalanceAfter += bal
						}

						// get total mints
						var mintTokens currency.Coin
						for _, m := range timedBalance.GetMints() {
							mintTokens += m.Amount
						}

						if totalBalanceBefore != totalBalanceAfter-mintTokens {
							log.Fatal(fmt.Sprintf("name:%s\ntokens mint or burned unexpected\nbefore:%v\nafter:-minted:%v\nminted:%v\n",
								bm.Name(),
								totalBalanceBefore,
								totalBalanceAfter-mintTokens, mintTokens))

						}
					}
				}
			})
			log.Println(bm.Name(), "run count is:", runCount)
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
