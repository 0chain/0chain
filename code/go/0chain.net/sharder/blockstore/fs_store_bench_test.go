package blockstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/chain"
	"0chain.net/core/encryption"
)

func BenchmarkFSBlockStore_getFileWithoutExtension(t *testing.B) {
	serverChain := chain.NewChainFromConfig()
	conf := serverChain.ChainConfig.(*chain.ConfigImpl)
	conf.ConfDataForTest().RoundRange = 1

	chain.SetServerChain(serverChain)

	currDir, err := os.Getwd()
	require.NoError(t, err)

	storeDir := filepath.Join(currDir, "tmp")
	fbs := NewFSBlockStore(storeDir, &minioClientMock{})
	defer func() {
		err := os.RemoveAll(filepath.Join(currDir, "tmp"))
		require.NoError(t, err)
	}()

	var (
		h       = encryption.Hash("data")
		r int64 = 1
	)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		fbs.getFileWithoutExtension(h, r)
	}
}
