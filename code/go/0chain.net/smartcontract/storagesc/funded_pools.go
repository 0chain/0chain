package storagesc

import (
	"encoding/json"

	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"github.com/0chain/errors"
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
	_, err = balances.InsertTrieNode(fundedPoolsKey(ssc.ID, clientId), pools)
	return nil
}

func (ssc *StorageSmartContract) isFundedPool(
	clientId, poolId string,
	balances cstate.StateContextI,
) (bool, error) {
	pools, err := ssc.getFundedPools(clientId, balances)
	if err != nil {
		return false, errors.Wrap(err, "error getting funded pools")
	}

	for _, id := range *pools {
		fmt.Println("id: ", id)
		fmt.Println("poolId: ", poolId)
		if id == poolId {
			return true, nil
		}
	}
	return false, nil
}

func (fp *fundedPools) Decode(p []byte) error {
	return json.Unmarshal(p, fp)
}

// getReadPoolBytes of a client
func (ssc *StorageSmartContract) getFundedPoolsBytes(
	clientID datastore.Key,
	balances cstate.StateContextI,
) ([]byte, error) {
	var val util.Serializable
	val, err := balances.GetTrieNode(fundedPoolsKey(ssc.ID, clientID))
	if err != nil {
		return nil, err
	}
	return val.Encode(), nil
}

// getReadPool of current client
func (ssc *StorageSmartContract) getFundedPools(
	clientID datastore.Key,
	balances cstate.StateContextI,
) (*fundedPools, error) {
	var poolb []byte
	var err error
	fp := new(fundedPools)
	if poolb, err = ssc.getFundedPoolsBytes(clientID, balances); err != nil {
		if !errors.Is(err, util.ErrValueNotPresent) {
			return nil, err
		}
		return fp, nil
	}
	err = fp.Decode(poolb)
	if err != nil {
		return nil, errors.Wrap(err, common.ErrDecoding.Error())
	}
	return fp, nil
}
