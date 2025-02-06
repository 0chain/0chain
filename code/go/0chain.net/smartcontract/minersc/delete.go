package minersc

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"
)

var deleteNodeKeyMap = map[spenum.Provider]string{
	spenum.Miner:   DeleteMinersKey,
	spenum.Sharder: DeleteShardersKey,
}

var registerNodeKeyMap = map[spenum.Provider]string{
	spenum.Miner:   RegisterMinersKey,
	spenum.Sharder: RegisterShardersKey,
}

var allNodeKeyMap = map[spenum.Provider]string{
	spenum.Miner:   AllMinersKey,
	spenum.Sharder: AllShardersKey,
}

func deleteNodesOnViewChange(state state.StateContextI, pType spenum.Provider) error {
	var (
		ids NodeIDs
		err error
	)

	dKey, ok := deleteNodeKeyMap[pType]
	if !ok {
		return fmt.Errorf("get delete node key failed, invalid node type: %s", pType)
	}

	allKey, ok := allNodeKeyMap[pType]
	if !ok {
		return fmt.Errorf("get all node key failed, invalid node type: %s", pType)
	}

	ids, err = getDeleteNodeIDs(state, dKey)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	// reset delete node ids list
	if err := resetDeleteNodeIDs(state, dKey); err != nil {
		return err
	}

	allNodeIDs, err := getNodeIDs(state, allKey)
	if err != nil {
		return err
	}

	allNodeIDs = removeIDs(allNodeIDs, ids)
	_, err = state.InsertTrieNode(allKey, &allNodeIDs)
	if err != nil {
		return err
	}

	for _, id := range ids {
		_, err := state.DeleteTrieNode(provider.GetKey(id))
		if err != nil {
			return err
		}
	}
	return nil
}

// remove items b from a
func removeIDs(a, b NodeIDs) NodeIDs {
	if len(b) == 0 {
		return a
	}

	toDeleteMap := make(map[string]struct{}, len(b))
	for _, id := range b {
		toDeleteMap[id] = struct{}{}
	}

	var j int
	for _, id := range a {
		if _, ok := toDeleteMap[id]; !ok {
			a[j] = id
			j++
		}
	}
	a = a[:j]
	return a
}

func getDeleteNodeIDs(state state.StateContextI, key string) (NodeIDs, error) {
	ids, err := getNodeIDs(state, key)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
	}

	return ids, nil
}

func resetDeleteNodeIDs(state state.StateContextI, key string) error {
	_, err := state.InsertTrieNode(key, &NodeIDs{})
	return err
}
