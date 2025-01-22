package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"0chain.net/core/config"
	"0chain.net/smartcontract/dbs/postgresql"

	"0chain.net/smartcontract/dbs/event"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	ebk "0chain.net/smartcontract/dbs/benchmark"
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
	test      benchmark.BenchTestI
	result    testing.BenchmarkResult
	timings   map[string]time.Duration
	numEvents int
	error
}

type suiteResults struct {
	name    string
	results []benchmarkResults
}

type chainer struct {
	qsc cstate.TimedQueryStateContextI
}

func (ch *chainer) GetStateContext() cstate.StateContextI {
	return nil
}

func (ch *chainer) GetQueryStateContext() cstate.TimedQueryStateContextI {
	return ch.qsc
}

func (ch *chainer) SetQueryStateContext(qsc cstate.TimedQueryStateContextI) {
	ch.qsc = qsc
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

	var eventMap = make(map[string][]event.Event)
	for _, suite := range suites {
		log.Println("starting suite ==>", suite.Source)
		wg.Add(1)
		go func(suite benchmark.TestSuite, wg *sync.WaitGroup) {
			defer wg.Done()
			var suiteResult []benchmarkResults
			if suite.ReadOnly {
				suiteResult = runReadOnlySuite(suite, timedBalance)
			} else {
				var events map[string][]event.Event
				suiteResult, events = runSuite(suite, mpt, root, data)
				for key, value := range events {
					if len(value) > 0 {
						eventMap[key] = value
					}
				}
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
	data.EventDb.Close()

	if viper.GetString(benchmark.OptionsSmartContractEventFile) != "" {
		err := writeEvents(viper.GetString(benchmark.OptionsSmartContractEventFile), eventMap)
		if err != nil {
			log.Fatal("error writing out events: " + err.Error())
		}
	}

	if viper.GetBool(benchmark.OptionsEventDatabaseBenchmarks) {
		if viper.GetString(benchmark.OptionsSmartContractEventFile) != viper.GetString(benchmark.OptionsEventDatabaseEventFile) {
			var err error
			eventMap, err = readEventDbTests(viper.GetString(benchmark.OptionsEventDatabaseEventFile))
			if err != nil {
				log.Fatal(fmt.Sprintf("error reading event db benchmarks file %s: %v",
					viper.GetString(benchmark.OptionsEventDatabaseEventFile), err))
			}
		}
		timer := time.Now()
		log.Println("\nstarting benchmark tests")
		suiteResult := runEventDatabaseSuite(ebk.GetBenchmarkTestSuite(eventMap, benchmark.EventDatabase), data.EventDb)
		results = append(results, suiteResults{
			name:    benchmark.SourceNames[benchmark.EventDatabase],
			results: suiteResult,
		})

		log.Println("\nstarting benchmark event tests")
		suiteResultEvents := runEventDatabaseSuite(ebk.GetBenchmarkTestSuite(eventMap, benchmark.EventDatabaseEvents), data.EventDb)
		results = append(results, suiteResults{
			name:    benchmark.SourceNames[benchmark.EventDatabaseEvents],
			results: suiteResultEvents,
		})

		log.Println("\nstarting benchmark aggregate tests")
		suiteResultAggregates := runEventDatabaseSuite(ebk.GetBenchmarkTestSuite(eventMap, benchmark.EventDatabaseAggregates), data.EventDb)
		results = append(results, suiteResults{
			name:    benchmark.SourceNames[benchmark.EventDatabaseAggregates],
			results: suiteResultAggregates,
		})

		log.Println("finished benchmark tests, took:", time.Since(timer))
	}
	return results
}

func runReadOnlySuite(
	suite benchmark.TestSuite,
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
) ([]benchmarkResults, map[string][]event.Event) {
	var benchmarkResult []benchmarkResults
	var benchmarkEvents = make(map[string][]event.Event)
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
			var balances cstate.StateContextI
			result := testing.Benchmark(func(b *testing.B) {
				b.StopTimer()
				var prevMptHashRoot string
				_ = prevMptHashRoot
				for i := 0; i < b.N; i++ {
					cloneMPT := util.CloneMPT(mpt)
					_, balances = getBalances(
						bm.Transaction(),
						extractMpt(cloneMPT, root),
						data,
					)
					timedBalance := cstate.NewTimedQueryStateContext(balances, func() common.Timestamp {
						return data.Now
					})

					// do the balances checking only once, otherwise it would slow down the tests too much
					var totalBalanceBefore currency.Coin
					if viper.GetBool(benchmark.OptionVerifyBurnedTokens) {
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
					}

					b.StartTimer()
					err = bm.Run(timedBalance, b)
					b.StopTimer()

					unknownMintTransferClients := make(map[string]struct{})
					if viper.GetBool(benchmark.OptionVerifyBurnedTokens) {
						// data.Clients is subset of all clients, so we need to check if there are
						// any unknown clients that minted to or transferred to
						unknownMintTransferClients := make(map[string]struct{})
						if err == nil {
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

					if viper.GetBool(benchmark.OptionVerifyBurnedTokens) {
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

							if totalBalanceBefore != totalBalanceAfter {
								log.Fatal(fmt.Sprintf("name:%s\ntokens mint or burned unexpected\nbefore:%v\nafter:-minted:%v\n",
									bm.Name(),
									totalBalanceBefore,
									totalBalanceAfter))
							}
						}
					}
				}
				benchmarkEvents[bm.Name()] = balances.GetEvents()
			})

			var resTimings map[string]time.Duration
			if wt, ok := bm.(benchmark.WithTimings); ok && len(wt.Timings()) > 0 {
				resTimings = wt.Timings()
			}

			benchmarkResult = append(
				benchmarkResult,
				benchmarkResults{
					test:      bm,
					result:    result,
					error:     err,
					timings:   resTimings,
					numEvents: len(benchmarkEvents[bm.Name()]),
				},
			)

			log.Println("test", bm.Name(), "done. took:", time.Since(timer), "run count is:", runCount)
		}(bm, &wg)
	}
	wg.Wait()
	return benchmarkResult, benchmarkEvents
}

func writeEvents(filename string, events map[string][]event.Event) error {
	if filename == "" {
		return nil
	}
	data, err := json.MarshalIndent(events, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func runEventDatabaseSuite(
	suite benchmark.TestSuite,
	edb *event.EventDb,
) []benchmarkResults {
	var benchmarkResult []benchmarkResults
	const dummyChainId = ""
	const dummyPort = 1
	config.InitConfigurationGlobal(
		edb.Config().Host,
		dummyChainId,
		dummyPort,
		event.NewTestConfig(edb.Settings()),
	)
	//var wg sync.WaitGroup
	pdb, err := postgresql.NewPostgresDB(edb.Config())
	if err != nil {
		log.Fatal("creating parent postgres db:", err)
	}

	// Add edb partitions
	_ = edb.AddPermanentPartitions(0)

	for _, bm := range suite.Benchmarks {
		//wg.Add(1)
		//go func(bm benchmark.BenchTestI, wg *sync.WaitGroup) {
		//defer wg.Done()
		log.Println("edb bench test starting... bm name:", bm.Name())
		timer := time.Now()
		var err error
		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err = runEventDatabaseBenchmark(b, edb, pdb, bm, i)
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
		log.Println("edb test", bm.Name(), "done. took:", time.Since(timer))
		//}(bm, &wg)
	}
	//wg.Wait()
	return benchmarkResult
}

func CleanDbName(name string, index int) string {
	cleanName := strings.Replace("event_benchmark_"+name, ".", "_", -1) + "_" + strconv.Itoa(index)
	cleanName = strings.Replace(cleanName, "-", "_", -1)
	cleanName = strings.ToLower(cleanName)
	return cleanName
}

func runEventDatabaseBenchmark(
	b *testing.B,
	edb *event.EventDb,
	pdb *postgresql.PostgresDB,
	bm benchmark.BenchTestI,
	index int,
) (err error) {
	b.StopTimer()
	cleanName := CleanDbName(bm.Name(), index)
	cloneEdb, err := edb.Clone(cleanName, pdb)
	if err != nil {
		fmt.Println("error cloning event database: " + err.Error())
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic running events", r)
		}
		cloneEdb.Close()
		deleteError := pdb.Drop(cleanName)
		if deleteError != nil {
			log.Println("error deleting event database: " + deleteError.Error())
		}
	}()
	balances := cstate.NewStateContext(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil, nil, cloneEdb,
	)
	timedBalance := cstate.NewTimedQueryStateContext(balances, func() common.Timestamp {
		return 0
	})
	b.StartTimer()
	err = bm.Run(timedBalance, b)
	b.StopTimer()
	return err
}
