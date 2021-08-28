package benchmark

import (
	"fmt"
	"testing"

	"0chain.net/smartcontract/storagesc"
	"github.com/stretchr/testify/require"
)

func main() {
	var b testing.B
	var vi = getViper(&b, "testdata/benchmark.yaml")
	mpt, root, clients, keys, blobbers, allocations := setUpMpt(&b, vi)
	allocations = allocations
	blobbers = blobbers
	benchmarks := storagesc.BenchmarkTests(vi, clients, keys, blobbers, allocations)
	benchmarkResult := []testing.BenchmarkResult{}
	for _, bm := range benchmarks {
		benchmarkResult = append(
			benchmarkResult,
			testing.Benchmark(func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					_, balances := getBalances(b, bm.Name, &bm.Txn, root, mpt)
					b.StartTimer()
					_, err := bm.Endpoint(&bm.Txn, bm.Input, balances)
					require.NoError(b, err)
				}
			}))
	}
	fmt.Println(benchmarkResult)
}
