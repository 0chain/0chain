package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs/postgresql"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/event"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/viper"
	ebk "0chain.net/smartcontract/dbs/benchmark"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/rest"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

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
						eventMap[suite.Name+"_"+key] = value
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
	var evt2 event.Event
	res2 := data.EventDb.Store.Get().Model(&event.Event{}).First(&evt2)
	res2 = res2

	data.EventDb.Close()
	sqlDB, err := data.EventDb.Get().DB()
	// Close
	err = sqlDB.Close()

	err = writeEvents(viper.GetString(benchmark.OptionsSmartContractEventFile), eventMap)
	if err != nil {
		log.Fatal("error writing out events: " + err.Error())
	}

	if viper.GetBool(benchmark.OptionsEventDatabaseBenchmarks) {
		if viper.GetString(benchmark.OptionsSmartContractEventFile) != viper.GetString(benchmark.OptionsEventDatabaseEventFile) {
			eventMap, err = readEventDbTests(viper.GetString(benchmark.OptionsEventDatabaseEventFile))
			if err != nil {
				log.Fatal(fmt.Sprintf("error reading event db benchmarks file %s: %v",
					viper.GetString(benchmark.OptionsEventDatabaseEventFile), err))
			}
		}
		suiteResult := runEventDatabaseSuite(ebk.GetBenchmarkTestSuite(eventMap), data.EventDb)
		results = append(results, suiteResults{
			name:    benchmark.SourceNames[benchmark.EventDatabase],
			results: suiteResult,
		})
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
					b.StartTimer()
					err = bm.Run(timedBalance, b)
					b.StopTimer()
					if err != nil {
						mockUpdateState(bm.Transaction(), balances)
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
				}
				events := balances.GetEvents()
				name := bm.Name()
				events = events
				name = name
				benchmarkEvents[bm.Name()] = balances.GetEvents()
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
	return ioutil.WriteFile(filename, data, 0644)
}

func runEventDatabaseSuite(
	suite benchmark.TestSuite,
	edb *event.EventDb,
) []benchmarkResults {
	var benchmarkResult []benchmarkResults
	config.InitConfigurationGlobal(
		edb.Config().Host,
		"piers' port",
		123,
		event.NewTestConfig(edb.Settings()),
	)
	var wg sync.WaitGroup
	pdb, err := postgresql.NewPostgresDB(edb.Config())
	if err != nil {
		log.Fatal("creating parent postgres db:", err)
	}

	for _, bm := range suite.Benchmarks {
		wg.Add(1)
		go func(bm benchmark.BenchTestI, wg *sync.WaitGroup) {
			defer wg.Done()
			timer := time.Now()
			//log.Println("starting", bm.Name())
			var err error
			result := testing.Benchmark(func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					//fmt.Println("in for loop piers", bm.Name(), "index", i)
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
			log.Println("test", bm.Name(), "done. took:", time.Since(timer))
		}(bm, &wg)
	}
	wg.Wait()
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
		cloneEdb,
	)
	timedBalance := cstate.NewTimedQueryStateContext(balances, func() common.Timestamp {
		return 0
	})
	b.StartTimer()
	err = bm.Run(timedBalance, b)
	b.StopTimer()
	return err
}
