package cmd

import (
	"testing"

	"0chain.net/smartcontract/storagesc"
)

func BenchmarkExecute(b *testing.B) {
	GetViper("testdata/benchmark.yaml")

	mpt, root, data := setUpMpt("testdata")
	benchmarks := storagesc.BenchmarkTests(data, &BLS0ChainScheme{})
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
