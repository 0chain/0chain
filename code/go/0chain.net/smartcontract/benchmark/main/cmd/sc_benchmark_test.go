package cmd

import (
	"testing"

	"0chain.net/smartcontract/storagesc"

	"github.com/stretchr/testify/require"
)

func BenchmarkExecute(b *testing.B) {
	var vi = GetViper(b, "testdata/benchmark.yaml")

	mpt, root, clients, keys, blobbers, allocations := setUpMpt(b, vi)
	benchmarks := storagesc.BenchmarkTests(vi, clients, keys, blobbers, allocations)
	for _, bm := range benchmarks {
		b.Run(bm.Name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				_, balances := getBalances(b, bm.Name, &bm.Txn, root, mpt)
				b.StartTimer()
				_, err := bm.Endpoint(&bm.Txn, bm.Input, balances)
				require.NoError(b, err)
			}
		})
	}
}
