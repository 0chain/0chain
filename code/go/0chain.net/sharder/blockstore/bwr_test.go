package blockstore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitBlockWhereRecord(t *testing.T) {
	workDir := "./rocks"
	err := os.Mkdir(workDir, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(workDir)
	cacheSize := uint64(100 * MB)

	require.NotPanics(t, func() {
		initBlockWhereRecord(cacheSize, "start", workDir)
	})

	require.NotNil(t, bmrDB)

	bwr := &blockWhereRecord{
		Hash:      "hash",
		Tiering:   DiskTier,
		BlockPath: "/path/to/block",
	}

	require.NoError(t, bwr.save())

	bwr, err = getBWR("hash")
	require.NoError(t, err)

	require.Equal(t, bwr.Tiering, DiskTier)
	require.Equal(t, bwr.BlockPath, "/path/to/block")
}
