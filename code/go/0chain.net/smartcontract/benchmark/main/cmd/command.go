package cmd

import (
	"fmt"
	"log"
	"sync"
	"testing"

	"0chain.net/smartcontract/minersc"

	"github.com/spf13/viper"

	"0chain.net/smartcontract/benchmark"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().Bool("verbose", true, "show updates")
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
		//benchmarksSC := storagesc.BenchmarkTests(data, &BLS0ChainScheme{})
		benchmarksMN := minersc.BenchmarkTests(data, &BLS0ChainScheme{})
		type results struct {
			test   benchmark.BenchTestI
			result testing.BenchmarkResult
		}
		benchmarkResult := []results{}

		var wg sync.WaitGroup
		for _, bm := range benchmarksMN {
			wg.Add(1)
			go func(bm benchmark.BenchTestI, wg *sync.WaitGroup) {
				defer wg.Done()
				result := testing.Benchmark(func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						_, balances := getBalances(bm.Transaction(), extractMpt(mpt, root))
						b.StartTimer()
						bm.Run(balances)
					}
				})
				benchmarkResult = append(
					benchmarkResult,
					results{
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

		fmt.Println("\nResults")
		fmt.Printf("name, ms\n")
		for _, result := range benchmarkResult {
			fmt.Printf(
				"%s,%f\n",
				result.test.Name(),
				float64(result.result.T.Milliseconds())/float64(result.result.N),
			)
		}
	},
}

func printSimSettings() {
	println("\n\nsimulator settings")
	println("num clients", viper.GetInt(benchmark.NumClients))
	println("num miners", viper.GetInt(benchmark.NumMiners))
	println("num sharders", viper.GetInt(benchmark.NumSharders))
	println("num blobbers", viper.GetInt(benchmark.NumBlobbers))
	println("num allocations", viper.GetInt(benchmark.NumAllocations))
}
