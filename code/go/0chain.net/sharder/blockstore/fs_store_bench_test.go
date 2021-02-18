package blockstore

import (
	"testing"

	"0chain.net/chaincore/chain"
	"0chain.net/core/encryption"
)

func BenchmarkFSBlockStore_getFileWithoutExtension(t *testing.B) {
	serverChain := chain.NewChainFromConfig()
	serverChain.RoundRange = 1
	chain.SetServerChain(serverChain)

	var (
		fbs       = makeTestFSBlockStore("test/bench/fsblockstore")
		h         = encryption.Hash("data")
		r   int64 = 1
	)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		fbs.getFileWithoutExtension(h, r)
	}
}
