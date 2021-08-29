package cmd

import (
	"fmt"
	"sync"
	"testing"

	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/storagesc"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
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
		mpt, root, clients, keys, blobbers, allocations := setUpMpt(vi, "db")
		benchmarks := storagesc.BenchmarkTests(vi, clients, keys, blobbers, allocations)
		type results struct {
			test   benchmark.BenchTest
			result testing.BenchmarkResult
		}
		benchmarkResult := []results{}
		var wg sync.WaitGroup
		for _, bm := range benchmarks {
			wg.Add(1)
			go func(bm benchmark.BenchTest, wg *sync.WaitGroup) {
				defer wg.Done()
				result := testing.Benchmark(func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						_, balances := getBalances(bm.Name, &bm.Txn, root, mpt)
						b.StartTimer()
						_, err := bm.Endpoint(&bm.Txn, bm.Input, balances)
						require.NoError(b, err)
					}
				})
				benchmarkResult = append(
					benchmarkResult,
					results{
						test:   bm,
						result: result,
					},
				)
				fmt.Println("test", bm.Name, "done")
			}(bm, &wg)
		}
		wg.Wait()

		fmt.Printf("name, ms\n")
		for _, result := range benchmarkResult {
			fmt.Printf(
				"%s,%f\n",
				result.test.Name,
				float64(result.result.T.Milliseconds())/float64(result.result.N),
			)
		}
	},
}
