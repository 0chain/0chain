package minersc

import (
	"fmt"
	"testing"

	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func TestDeleteMinersIDs(t *testing.T) {
	state := newTestBalances()
	for i := 0; i < 10; i++ {
		err := saveDeleteNodeID(state, spenum.Miner, fmt.Sprintf("m%d", i))
		require.NoError(t, err)
	}

	ids, err := getDeleteNodeIDs(state, DeleteMinersKey)
	require.NoError(t, err)
	require.Len(t, ids, 10)
	for i := 0; i < 10; i++ {
		require.Equal(t, fmt.Sprintf("m%d", i), ids[i])
	}
}

func TestDeleteShardersIDs(t *testing.T) {
	state := newTestBalances()
	for i := 0; i < 10; i++ {
		err := saveDeleteNodeID(state, spenum.Sharder, fmt.Sprintf("s%d", i))
		require.NoError(t, err)
	}

	ids, err := getDeleteNodeIDs(state, DeleteShardersKey)
	require.NoError(t, err)
	require.Len(t, ids, 10)
	for i := 0; i < 10; i++ {
		require.Equal(t, fmt.Sprintf("s%d", i), ids[i])
	}

	// save exist delete miner ID, should not return error
	err = saveDeleteNodeID(state, spenum.Sharder, "s0")
	require.NoError(t, err)
}

func TestRemoveIDs(t *testing.T) {
	tt := []struct {
		name   string
		init   NodeIDs
		remove NodeIDs
		expect NodeIDs
	}{
		{
			name:   "remove first 1",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"1"},
			expect: NodeIDs{"2", "3", "4", "5", "6", "7", "8"},
		},
		{
			name:   "remove first 2",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"1", "2"},
			expect: NodeIDs{"3", "4", "5", "6", "7", "8"},
		},
		{
			name:   "remove random 2",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"1", "4"},
			expect: NodeIDs{"2", "3", "5", "6", "7", "8"},
		},
		{
			name:   "remove middle 3",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"3", "4", "5"},
			expect: NodeIDs{"1", "2", "6", "7", "8"},
		},
		{
			name:   "remove last 1",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"8"},
			expect: NodeIDs{"1", "2", "3", "4", "5", "6", "7"},
		},
		{
			name:   "remove last 3",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"6", "7", "8"},
			expect: NodeIDs{"1", "2", "3", "4", "5"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, removeIDs(tc.init, tc.remove))
		})
	}
}

func TestDeleteNodesOnViewChange(t *testing.T) {
	s := newTestBalances()
	var (
		allMinerNodeIDs NodeIDs
	)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("m%d", i)
		allMinerNodeIDs = append(allMinerNodeIDs, id)
		mn := NewMinerNode()
		mn.ID = id
		_, err := s.InsertTrieNode(provider.GetKey(id), mn)
		require.NoError(t, err)
	}

	_, err := s.InsertTrieNode(AllMinersKey, &allMinerNodeIDs)
	require.NoError(t, err)

	deleteNodeIDs := []string{"m0", "m1"}

	// set delete nodes
	for _, id := range deleteNodeIDs {
		err = saveDeleteNodeID(s, spenum.Miner, id)
		require.NoError(t, err)
		err = saveDeleteNodeID(s, spenum.Miner, id)
		require.NoError(t, err)
	}

	err = deleteNodesOnViewChange(s, spenum.Miner)
	require.NoError(t, err)

	// assert that delete node have been removed
	for _, id := range deleteNodeIDs {
		var n MinerNode
		err = s.GetTrieNode(provider.GetKey(id), &n)
		require.Equal(t, util.ErrValueNotPresent, err)
	}

	// assert the delete node key list is empty
	var nids NodeIDs
	err = s.GetTrieNode(DeleteMinersKey, &nids)
	require.NoError(t, err)
	require.Equal(t, 0, len(nids))

	// assert the nodes have been removed from ALL node key list
	aids, err := getNodeIDs(s, AllMinersKey)
	require.NoError(t, err)
	aMap := make(map[string]struct{}, len(aids))
	for _, id := range aids {
		aMap[id] = struct{}{}
	}
	for _, id := range deleteNodeIDs {
		_, ok := aMap[id]
		require.False(t, ok)
	}
}
