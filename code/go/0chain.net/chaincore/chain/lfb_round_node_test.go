package chain

import (
	"testing"

	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func TestStoreLFBRound(t *testing.T) {
	c := &Chain{stateDB: util.NewMemoryNodeDB()}
	round := int64(123)
	blockHash := "abc123"
	err := c.StoreLFBRound(round, 0, blockHash)
	require.NoError(t, err)

	// Verify that the LFB round was stored correctly
	nd, err := c.LoadLFBRound()
	require.NoError(t, err)
	require.Equal(t, round, nd.Round)
	require.Equal(t, blockHash, nd.Hash)
}
