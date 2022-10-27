package cmd

import (
	"fmt"
	"path"
	"sort"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"github.com/spf13/pflag"

	"0chain.net/smartcontract/zcnsc"

	"0chain.net/smartcontract/benchmark/main/cmd/control"

	"0chain.net/chaincore/node"
	bk "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"github.com/0chain/common/core/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfigPath = "testdata/benchmark.yaml"
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
					"\n%v:\n",
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
	for _, suiteResult := range results {
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
}
