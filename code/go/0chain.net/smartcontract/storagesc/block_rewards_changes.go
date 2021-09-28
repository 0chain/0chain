package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"

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

func (brc *blockRewardChanges) getLatestChange() (*blockRewardChange, int) {
	if len(brc.Changes) == 0 {
		return nil, 0
	}
	return &brc.Changes[len(brc.Changes)-1], len(brc.Changes) - 1
}

func (brc *blockRewardChanges) getPreviousChange(index int) (*blockRewardChange, int) {
	if index == 0 || index >= len(brc.Changes) {
		return nil, 0
	}
	return &brc.Changes[index-1], index - 1
}

func updateBlockRewardSettingsList(
	before, after blockrewards.BlockReward,
	conf *scConfig,
	balances cstate.StateContextI,
) error {
	if before == after {
		return nil
	}
	changes, err := getBlockRewardChanges(balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		changes = newBlockRewardChanges(conf)
		changes.save(balances)
	}
	changes.Changes = append(changes.Changes, blockRewardChange{
		Round:  balances.GetBlock().Round,
		Change: after,
	})
	logging.Logger.Info("piers7 updateBlockRewardSettingsList",
		zap.Int64("round", balances.GetBlock().Round),
		zap.Any("before", before),
		zap.Any("after", after),
		zap.Any("changes", changes),
	)
	_, err = balances.InsertTrieNode(blockRewardChangesKey, changes)
	return err
}

func (brc *blockRewardChanges) Encode() []byte {
	var b, err = json.Marshal(brc)
	if err != nil {
		panic(err)
	}
	return b
}

func (brc *blockRewardChanges) Decode(p []byte) error {
	return json.Unmarshal(p, brc)
}

func (brc *blockRewardChanges) save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(blockRewardChangesKey, brc)
	return err
}

func newBlockRewardChanges(
	conf *scConfig,
) *blockRewardChanges {
	return &blockRewardChanges{
		Changes: []blockRewardChange{
			{
				Round:  1,
				Change: *conf.BlockReward,
			},
		},
	}
}

func getBlockRewardChanges(balances cstate.StateContextI) (*blockRewardChanges, error) {
	var val util.Serializable
	var qtl blockRewardChanges
	val, err := balances.GetTrieNode(blockRewardChangesKey)
	if err != nil {
		return nil, err
	}

	err = qtl.Decode(val.Encode())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	if len(qtl.Changes) == 0 {
		return nil, errors.New("getBlockRewardChanges, empty changes list")
	}
	if err != nil {
		return nil, err
	}
	return &qtl, nil
}
