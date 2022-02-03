package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"fmt"
)

func fundedPoolsKey(scKey, clientID string) datastore.Key {
	return datastore.Key(scKey + ":fundedpools:" + clientID)
}

type fundedPools []string

func (fp *fundedPools) Encode() []byte {
	var b, err = json.Marshal(fp)
	if err != nil {
		panic(err) // must never happens
	}
	return b
}

func (ssc *StorageSmartContract) addToFundedPools(
	clientId, poolId string,
	balances cstate.StateContextI,
) error {
	pools, err := ssc.getFundedPools(clientId, balances)
	if err != nil {
		return fmt.Errorf("error getting funded pools: %v", err)
	}
	*pools = append(*pools, poolId)
	err = balances.InsertTrieNode(fundedPoolsKey(ssc.ID, clientId), pools)
	return nil
}

func (ssc *StorageSmartContract) isFundedPool(
	clientId, poolId string,
	balances cstate.StateContextI,
) (bool, error) {
	pools, err := ssc.getFundedPools(clientId, balances)
	if err != nil {
		return false, fmt.Errorf("error getting funded pools: %v", err)
	}
	for _, id := range *pools {
		if id == poolId {
			return true, nil
		}
	}
	return false, nil
}

func (fp *fundedPools) Decode(p []byte) error {
	return json.Unmarshal(p, fp)
}

// getReadPool of current client
func (ssc *StorageSmartContract) getFundedPools(clientID datastore.Key, balances cstate.StateContextI) (*fundedPools, error) {
	var err error
	fp := new(fundedPools)

	var raw util.Serializable
	raw, err = balances.GetTrieNode(fundedPoolsKey(ssc.ID, clientID), fp)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return fp, nil
	}
	var ok bool
	if fp, ok = raw.(*fundedPools); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}
	return fp, nil
}
