package cmd

import (
	"fmt"
	"sort"
	"time"

	"0chain.net/smartcontract/benchmark/main/cmd/control"

	"0chain.net/chaincore/node"
	"0chain.net/core/logging"
	bk "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var benchmarkSources = map[bk.Source]func(data bk.BenchData, sigScheme bk.SignatureScheme) bk.TestSuite{
	bk.Storage:          storagesc.BenchmarkTests,
	bk.StorageRest:      storagesc.BenchmarkRestTests,
	bk.Miner:            minersc.BenchmarkTests,
	bk.MinerRest:        minersc.BenchmarkRestTests,
	bk.Faucet:           faucetsc.BenchmarkTests,
	bk.FaucetRest:       faucetsc.BenchmarkRestTests,
	bk.InterestPool:     interestpoolsc.BenchmarkTests,
	bk.InterestPoolRest: interestpoolsc.BenchmarkRestTests,
	bk.Vesting:          vestingsc.BenchmarkTests,
	bk.VestingRest:      vestingsc.BenchmarkRestTests,
	bk.MultiSig:         multisigsc.BenchmarkTests,
	bk.ZCNSCBridge:      zcnsc.BenchmarkTests,
	bk.ZCNSCBridgeRest:  zcnsc.BenchmarkRestTests,
	bk.Control:          control.BenchmarkTests,
}

func init() {
	logging.InitLogging("testing")
	node.Self = &node.SelfNode{
		Node: &node.Node{},
	}
	rootCmd.PersistentFlags().Bool("verbose", true, "show updates")
	rootCmd.PersistentFlags().StringSlice("tests", nil, "list of tests to show, nil show all")
	rootCmd.PersistentFlags().StringSlice("omit", nil, "list endpoints to omit")
}

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Benchmark 0chain smart-contract",
	Long:  `Benchmark 0chain smart-contract`,
	Run: func(cmd *cobra.Command, args []string) {
		GetViper("testdata/benchmark.yaml")
		log.PrintSimSettings()

		tests, omittedTests := setupOptions(cmd.Flags())
		log.Println("read in command line options")

		mpt, root, data := setUpMpt("db")
		log.Println("finished setting up blockchain")

		suites := getTestSuites(data, tests, omittedTests)
		results := runSuites(suites, mpt, root, data)

		printResults(results)
	},
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
			err := bkResult.test.Error()

			if err != "" {
				status = "FAILED"
				errMessage = " - " + err
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
