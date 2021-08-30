package cmd

import (
	"fmt"
	"sync"
	"testing"

	"github.com/spf13/viper"

	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/storagesc"
	"github.com/spf13/cobra"
)

func init() {

}

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Benchmark 0chain smart-contract",
	Long:  `Benchmark 0chain smart-contract`,
	Run: func(cmd *cobra.Command, args []string) {
		var vi = GetViper("testdata/benchmark.yaml")
		printSimSettings(vi)

		mpt, root, clients, keys, blobbers, allocations := setUpMpt(vi, "db")
		benchmarks := storagesc.BenchmarkTests(vi, clients, keys, blobbers, allocations)
		type results struct {
			test   benchmark.BenchTestI
			result testing.BenchmarkResult
		}
		benchmarkResult := []results{}

		var wg sync.WaitGroup
		for _, bm := range benchmarks {
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
				fmt.Println("test", bm.Name(), "done")
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

func printSimSettings(vi *viper.Viper) {
	println("\n\nsimulator settings")
	println("num clients", vi.GetInt(benchmark.NumClients))
	println("num miners", vi.GetInt(benchmark.NumMiners))
	println("num sharders", vi.GetInt(benchmark.NumSharders))
	println("num blobbers", vi.GetInt(benchmark.NumBlobbers))
	println("num allocations", vi.GetInt(benchmark.NumAllocations))
}
