package cmd

import (
	"fmt"
	"log"
	"sort"
	"time"

	"0chain.net/smartcontract/multisigsc"

	"0chain.net/smartcontract/vestingsc"

	"0chain.net/smartcontract/interestpoolsc"

	"0chain.net/smartcontract/faucetsc"

	"0chain.net/chaincore/node"

	"0chain.net/core/logging"

	"0chain.net/smartcontract/storagesc"

	"0chain.net/smartcontract/minersc"

	"github.com/spf13/viper"

	bk "0chain.net/smartcontract/benchmark"
	"github.com/spf13/cobra"
)

var benchmarkSources = map[bk.BenchmarkSource]func(data bk.BenchData, sigScheme bk.SignatureScheme) bk.TestSuit{
	bk.Storage:          storagesc.BenchmarkTests,
	bk.StorageRest:      storagesc.BenchmarkRestTests,
	bk.Miner:            minersc.BenchmarkTests,
	bk.MinerRest:        minersc.BenchmarkRestTests,
	bk.Faucet:           faucetsc.BenchmarkTests,
	bk.FaucetRest:       faucetsc.BenchmarkRestTests,
	bk.InterestPool:     interestpoolsc.BenchmarkTests,
	bk.InterestPoolRest: interestpoolsc.BenchmarkRestTests,
	bk.Vesting:          vestingsc.BenchmarkTests,
	bk.MultiSig:         multisigsc.BenchmarkTests,
}

func init() {
	logging.InitLogging("testing")
	node.Self = &node.SelfNode{
		Node: &node.Node{},
	}
	rootCmd.PersistentFlags().Bool("verbose", true, "show updates")
	rootCmd.PersistentFlags().StringSlice("tests", nil, "list of tests to show, nil show all")
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
		var err error
		verbose := true
		if cmd.Flags().Changed("verbose") {
			verbose, err = cmd.Flags().GetBool("verbose")
			if err != nil {
				log.Fatal(err)
			}
		}
		printSimSettings(verbose)

		mpt, root, data := setUpMpt("db", verbose)
		if verbose {
			log.Println("finished setting up blockchain")
		}
		suites := getTestSuites(data, cmd.Flags())
		results := runSuites(suites, verbose, mpt, root, data)

		printResults(results, verbose)

	},
}

func printResults(results []suiteResults, verbose bool) {
	const (
		colourReset  = "\033[0m"
		colourRed    = "\033[31m"
		colourGreen  = "\033[32m"
		colourYellow = "\033[33m"
		colourPurple = "\033[35m"
	)

	var (
		colour string
		bad    = viper.GetDuration(bk.Bad)
		worry  = viper.GetDuration(bk.Worry)
		good   = viper.GetDuration(bk.Satisfactory)
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
			if verbose {
				fmt.Printf(
					"%s%s,%f%s%s\n",
					colour,
					bkResult.test.Name(),
					takenMs,
					colourReset,
					"ms",
				)
			} else {
				fmt.Printf(
					"%s,%f\n",
					bkResult.test.Name(),
					takenMs,
				)
			}

		}
	}

}

func printSimSettings(verbose bool) {
	if verbose {
		println("simulator settings")
		println("num clients", viper.GetInt(bk.NumClients))
		println("num miners", viper.GetInt(bk.NumMiners))
		println("num sharders", viper.GetInt(bk.NumSharders))
		println("num blobbers", viper.GetInt(bk.NumBlobbers))
		println("num allocations", viper.GetInt(bk.NumAllocations))
		println()
	}
}
