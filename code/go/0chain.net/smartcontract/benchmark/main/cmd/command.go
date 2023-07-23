package cmd

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"github.com/spf13/pflag"

	"0chain.net/chaincore/node"
	bk "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"github.com/0chain/common/core/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfigPath = "testdata/benchmark.yaml"
)

func init() {
	logging.InitLogging("testing", "")
	node.Self = &node.SelfNode{
		Node: node.Provider(),
	}

	pflag.String("config", defaultConfigPath, "path to config")
	pflag.String("load", "", "path to load")

	pflag.StringSlice("tests", nil, "comma delimited list of test suites")
	pflag.Bool("verbose", true, "verbose")
	pflag.StringSlice("omit", nil, "comma delimited list of tests to ommit")

	//	pflag.Parse()
	//err := viper.BindPFlags(pflag.CommandLine)

	_ = viper.BindPFlag("config", pflag.Lookup("config"))
	_ = viper.BindPFlag("load", pflag.Lookup("load"))

	_ = viper.BindPFlag(bk.OptionTestSuites, pflag.Lookup("tests"))
	_ = viper.BindEnv(bk.OptionTestSuites, "TESTS")
	_ = viper.BindPFlag(bk.OptionOmittedTests, pflag.Lookup("omit"))
	_ = viper.BindEnv(bk.OptionOmittedTests, "OMIT")
	_ = viper.BindPFlag(bk.OptionVerbose, pflag.Lookup("verbose"))
	_ = viper.BindEnv(bk.OptionVerbose, "VERBOSE")

	impl := chain.NewConfigImpl(&chain.ConfigData{})
	config.Configuration().ChainConfig = impl

	viper.AutomaticEnv()
}

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Benchmark 0chain smart-contract",
	Long:  `Benchmark 0chain smart-contract`,
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in benchmark function", r)
			}
		}()
		totalTimer := time.Now()
		// path to config file can only come from command line options

		loadPath := viper.GetString("load")
		log.Println("load path", loadPath)
		configPath := viper.GetString("config")
		if loadPath != "" {
			configPath = path.Join(loadPath, "benchmark.yaml")
		}
		log.Println("config path", configPath)

		GetViper(loadPath)
		log.PrintSimSettings()
		common.SetupRootContext(context.Background())

		tests, omittedTests := suitesOmits()
		log.Println("read in command line options")

		executor := common.NewWithContextFunc(viper.GetInt(bk.OptionsLoadConcurrency))
		mpt, root, data := getMpt(loadPath, configPath, executor)
		log.Println("finished setting up blockchain", "root", string(root))

		savePath := viper.GetString(bk.OptionSavePath)
		if len(savePath) > 0 && loadPath != savePath {
			if err := viper.WriteConfigAs(path.Join(savePath, "benchmark.yaml")); err != nil {
				log.Fatal("cannot copy config file to", savePath)
			}
		}
		testsTimer := time.Now()
		suites := getTestSuites(data, tests, omittedTests)
		results := runSuites(suites, mpt, root, data)
		log.Println()
		log.Println("tests took", time.Since(testsTimer))
		log.Println("benchmark took", time.Since(totalTimer))
		printTimings(results)
		printResults(results)
	},
}

func printTimings(results []suiteResults) {
	fmt.Println()
	fmt.Println("Timings")
	for _, r := range results {
		for _, br := range r.results {
			if br.timings != nil {
				fmt.Printf(
					"%v:\n",
					br.test.Name(),
				)

				var entries []struct {
					name string
					dur  time.Duration
				}

				for k, v := range br.timings {
					entries = append(entries, struct {
						name string
						dur  time.Duration
					}{
						name: k,
						dur:  v,
					})
				}
				sort.Slice(entries, func(i, j int) bool {
					return entries[i].dur < entries[j].dur
				})
				for _, e := range entries {
					fmt.Printf(
						"%v: %v\n", e.name, e.dur,
					)
				}
			}
		}
	}
}

func printResults(results []suiteResults) {
	const (
		colourReset  = "\033[0m"
		colourRed    = "\033[31m"
		colourGreen  = "\033[32m"
		colourYellow = "\033[33m"
		colourPurple = "\033[35m"
	)

	var (
		verbose = log.GetVerbose()
		colour  string
		bad     = viper.GetDuration(bk.Bad)
		worry   = viper.GetDuration(bk.Worry)
		good    = viper.GetDuration(bk.Satisfactory)
	)

	if verbose {
		fmt.Println("\nResults")
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].name < results[j].name
	})
	for _, suiteResult := range results {
		sort.SliceStable(suiteResult.results, func(i, j int) bool {
			return suiteResult.results[i].test.Name() < suiteResult.results[j].test.Name()
		})
	}
	mapResults := make(map[string][]benchmarkResults)
	for _, suiteResult := range results {
		mapResults[suiteResult.name] = suiteResult.results
		if verbose {
			fmt.Printf("\nbenchmark suite " + suiteResult.name + "\n")
		}
		for _, bkResult := range suiteResult.results {
			takenMs := float64(bkResult.result.T.Milliseconds()) / float64(bkResult.result.N)
			takenDuration := time.Duration(takenMs * float64(time.Millisecond))
			if !verbose || !viper.GetBool(bk.Colour) {
				colour = colourReset
			} else if takenDuration >= bad {
				colour = colourRed
			} else if takenDuration > worry {
				colour = colourPurple
			} else if takenDuration > good {
				colour = colourYellow
			} else {
				colour = colourGreen
			}

			statusColor := colourGreen
			status := "OK"
			errMessage := ""
			err := bkResult.error

			if err != nil {
				status = "FAILED"
				errMessage = " - " + err.Error()
				statusColor = colourRed
			}

			if verbose {
				fmt.Printf(
					"%s%s,%f%s%s %s%s%s%s\n",
					colour,
					bkResult.test.Name(),
					takenMs,
					colourReset,
					"ms",
					statusColor,
					status,
					errMessage,
					colourReset,
				)
			} else {
				fmt.Printf(
					"%s,%f %s%s%s%s\n",
					bkResult.test.Name(),
					takenMs,
					statusColor,
					status,
					errMessage,
					colourReset,
				)
			}
		}
	}

	if viper.GetBool(bk.OptionsEventDatabaseBenchmarks) && viper.GetBool(bk.EventDbEnabled) &&
		viper.GetString(bk.OptionsSmartContractEventFile) == viper.GetString(bk.OptionsEventDatabaseEventFile) {
		fmt.Printf("\nCombined smartcontract and event processing times")
		fmt.Printf("\n%s,%s,%s,%s,%s,%s,%s\n", "name", "sc/ms", "events/ms", "num events", "ms/event", "event", "aggreates")
		for i, edbResult := range mapResults[bk.SourceNames[bk.EventDatabase]] {
			name := edbResult.test.Name()
			edbEventsResult := mapResults[bk.SourceNames[bk.EventDatabaseEvents]][i]
			edbEventsAggregates := mapResults[bk.SourceNames[bk.EventDatabaseAggregates]][i]
			splitName := strings.Split(name, ".")
			if len(splitName) != 2 {
				log.Println("bad name", name, "should be exactly one period.")
			}
			for _, smartContractRestult := range mapResults[splitName[0]] {
				if smartContractRestult.test.Name() == name {
					takenSC := float64(smartContractRestult.result.T.Milliseconds()) / float64(smartContractRestult.result.N)
					takenEdb := float64(edbResult.result.T.Milliseconds()) / float64(edbResult.result.N)
					takenEvents := float64(edbEventsResult.result.T.Milliseconds()) / float64(edbEventsResult.result.N)
					takenAggregates := float64(edbEventsAggregates.result.T.Milliseconds()) / float64(edbEventsAggregates.result.N)
					fmt.Printf("%s,%f,%f,%d,%f,%f,%f\n",
						name,
						takenSC,
						takenEdb,
						smartContractRestult.numEvents,
						takenEdb/float64(smartContractRestult.numEvents),
						takenEvents,
						takenAggregates,
					)
				}
			}

		}
	}
}
