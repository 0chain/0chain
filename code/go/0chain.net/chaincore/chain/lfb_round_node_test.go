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
	err := c.StoreLFBRound(round, blockHash)
	require.NoError(t, err)

	// Verify that the LFB round was stored correctly
	nd, err := c.stateDB.GetNode(LFBRoundKey)
	require.NoError(t, err)
	lfbr := &LfbRound{}
	_, err = lfbr.UnmarshalMsg(nd.Encode())
	require.NoError(t, err)
	require.Equal(t, round, lfbr.Round)
	require.Equal(t, blockHash, lfbr.Hash)
}
