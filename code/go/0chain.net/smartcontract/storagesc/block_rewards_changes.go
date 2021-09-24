package storagesc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/storagesc/blockrewards"
)

var (
	blockRewardChangesKey = datastore.Key(ADDRESS + encryption.Hash("block_reward_changes"))
)

type blockRewardChange struct {
	Round  int64                    `json:"round"`
	Change blockrewards.BlockReward `json:"changes"`
}

type blockRewardChanges struct {
	Changes []blockRewardChange `json:"changes"`
}

func updateBlockRewardSettingsList(before, after *blockrewards.BlockReward, balances cstate.StateContextI) {
	if *before == *after {
		return
	}

}

func (qt *blockRewardChanges) Encode() []byte {
	var b, err = json.Marshal(qt)
	if err != nil {
		panic(err)
	}
	return b
}

func (qt *blockRewardChanges) Decode(p []byte) error {
	return json.Unmarshal(p, qt)
}

func (brc *blockRewardChanges) startBlockRewardChanges(balances cstate.StateContextI) error {
	if len(brc.Changes) > 0 {
		return nil
	}

	conf, err := (&StorageSmartContract{}).setupConfig(balances)
	if err != nil {
		return err
	}

	brc.Changes = append(brc.Changes, blockRewardChange{
		Round:  balances.GetBlock().Round,
		Change: *conf.BlockReward,
	})
	return nil
}

func getBlockRewardChanges(balances cstate.StateContextI) (*blockRewardChanges, error) {
	var val util.Serializable
	var qtl blockRewardChanges
	val, err := balances.GetTrieNode(blockRewardChangesKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		err = qtl.startBlockRewardChanges(balances)
		return &qtl, err
	}

	err = qtl.Decode(val.Encode())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	if len(qtl.Changes) == 0 {
		err = qtl.startBlockRewardChanges(balances)
	}
	if err != nil {
		return nil, err
	}
	return &qtl, nil
}
