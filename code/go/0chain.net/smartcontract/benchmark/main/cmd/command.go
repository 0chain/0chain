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
		var b testing.B
		var vi = GetViper(&b, "testdata/benchmark.yaml")
		mpt, root, clients, keys, blobbers, allocations := setUpMpt(&b, vi)
		benchmarks := storagesc.BenchmarkTests(vi, clients, keys, blobbers, allocations)
		type results struct {
			test   benchmark.BenchTest
			result testing.BenchmarkResult
		}
		benchmarkResult := []results{}
		var wg sync.WaitGroup
		for _, bm := range benchmarks {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				result := testing.Benchmark(func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						_, balances := getBalances(b, bm.Name, &bm.Txn, root, mpt)
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
			}(&wg)
		}
		wg.Wait()

		fmt.Printf("name, ms, #tests")
		for _, result := range benchmarkResult {
			fmt.Printf(
				"%s,%d,%d",
				result.test.Name,
				result.result.T.Milliseconds(),
				result.result.N,
			)
		}
		fmt.Println(benchmarkResult)
	},
}
