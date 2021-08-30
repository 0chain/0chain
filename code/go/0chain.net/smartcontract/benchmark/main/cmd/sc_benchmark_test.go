package cmd

import (
	"testing"

	"0chain.net/smartcontract/storagesc"
)

func BenchmarkExecute(b *testing.B) {
	var vi = GetViper("testdata/benchmark.yaml")

	mpt, root, clients, keys, blobbers, allocations := setUpMpt(vi, "testdata")
	benchmarks := storagesc.BenchmarkTests(vi, clients, keys, blobbers, allocations)
	for _, bm := range benchmarks {
		b.Run(bm.Name(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				_, balances := getBalances(bm.Transaction(), extractMpt(mpt, root))
				b.StartTimer()
				bm.Run(balances)
			}
		})
	}
}
