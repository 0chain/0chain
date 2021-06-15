package zcnsc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMockStateContext_GetTrieNode(t *testing.T) {
}

func TestGlobalNode_Decode(t *testing.T) {
}

func TestShouldSaveGlobalNode(t *testing.T) {
	node := CreateSmartContractGlobalNode()
	balances := CreateMockStateContext()
	err := node.save(balances)
	require.NoError(t, err, "must save the global node in state")
}
