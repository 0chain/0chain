package cmd

import (
	"fmt"
	"log"
	"sort"

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
	bk.Storage: storagesc.BenchmarkTests,
	bk.Miner:   minersc.BenchmarkTests,
	bk.Faucet:  faucetsc.BenchmarkTests,
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
		printSimSettings()

		mpt, root, data := setUpMpt("db")
		suites := getTestSuites(data, cmd.Flags())
		results := runSuites(suites, verbose, mpt, root, data)

		printResults(results)

	},
}

func printResults(results []suiteResults) {
	fmt.Println("\nResults")
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].name < results[j].name
	})
	for _, suiteResult := range results {
		sort.SliceStable(suiteResult.results, func(i, j int) bool {
			return suiteResult.results[i].test.Name() < suiteResult.results[j].test.Name()
		})
	}
	for _, suiteResult := range results {
		fmt.Printf("\nbenchmark suite " + suiteResult.name + "\n")
		fmt.Printf("name, ms\n")
		for _, bkResult := range suiteResult.results {
			fmt.Printf(
				"%s,%f\n",
				bkResult.test.Name(),
				float64(bkResult.result.T.Milliseconds())/float64(bkResult.result.N),
			)
		}
	}

}

func printSimSettings() {
	println("simulator settings")
	println("num clients", viper.GetInt(bk.NumClients))
	println("num miners", viper.GetInt(bk.NumMiners))
	println("num sharders", viper.GetInt(bk.NumSharders))
	println("num blobbers", viper.GetInt(bk.NumBlobbers))
	println("num allocations", viper.GetInt(bk.NumAllocations))
	println()
}
