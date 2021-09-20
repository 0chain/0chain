package blockrewards

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

const (
	storagScAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

var (
	QualifyingTotalsKey         = datastore.Key(storagScAddress + encryption.Hash("qualifying_totals"))
	QualifyingTotalsPerBlockKey = datastore.Key(storagScAddress + encryption.Hash("qualifying_totals_per_block"))
	ConfigKey                   = datastore.Key(storagScAddress + ":configurations")
)

type BlockReward struct {
	BlockReward           state.Balance `json:"block_reward"`
	QualifyingStake       state.Balance `json:"qualifying_stake"`
	SharderWeight         float64       `json:"sharder_weight"`
	MinerWeight           float64       `json:"miner_weight"`
	BlobberCapacityWeight float64       `json:"blobber_capacity_weight"`
	BlobberUsageWeight    float64       `json:"blobber_usage_weight"`
}

func (br *BlockReward) SetWeightsFromRatio(sharderRatio, minerRatio, bCapcacityRatio, bUsageRatio float64) {
	total := sharderRatio + minerRatio + bCapcacityRatio + bUsageRatio
	if total == 0 {
		br.SharderWeight = 0
		br.MinerWeight = 0
		br.BlobberCapacityWeight = 0
		br.BlobberUsageWeight = 0
	} else {
		br.SharderWeight = sharderRatio / total
		br.MinerWeight = minerRatio / total
		br.BlobberCapacityWeight = bCapcacityRatio / total
		br.BlobberUsageWeight = bUsageRatio / total
	}
}

type QualifyingTotals struct {
	Round              int64        `json:"round"` // todo probably remove after debug
	Capacity           int64        `json:"capacity"`
	Used               int64        `json:"used"`
	LastSettingsChange int64        `json:"last_settings_change"`
	SettingsChange     *BlockReward `json:"settings_change"`
}

func (qt *QualifyingTotals) Encode() []byte {
	var b, err = json.Marshal(qt)
	if err != nil {
		panic(err)
	}
	return b
}

func (qt *QualifyingTotals) Decode(p []byte) error {
	return json.Unmarshal(p, qt)
}

type QualifyingTotalsList struct {
	Totals []QualifyingTotals `json:"totals"`
}

func NewQualifyingTotalsList() *QualifyingTotalsList {
	return &QualifyingTotalsList{make([]QualifyingTotals, 1024)}
}

func (qtl *QualifyingTotalsList) Encode() []byte {
	var b, err = json.Marshal(qtl)
	if err != nil {
		panic(err)
	}
	return b
}

func (qtl *QualifyingTotalsList) Decode(p []byte) error {
	return json.Unmarshal(p, qtl)
}

func (qtl *QualifyingTotalsList) HasBlockRewardsSettingsChanged(balances cstate.StateContextI) (*BlockReward, bool, error) {
	val, err := balances.GetTrieNode(ConfigKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, false, err
		}
		if balances.GetBlock().Round > 1 {
			return nil, false, nil
		}
		const pfx = "smart_contracts.storagesc."
		br := BlockReward{
			BlockReward:     state.Balance(viper.GetFloat64(pfx+"block_reward.block_reward") * 1e10),
			QualifyingStake: state.Balance(viper.GetFloat64(pfx+"block_reward.qualifying_stake") * 1e10),
		}
		br.SetWeightsFromRatio(
			viper.GetFloat64(pfx+"block_reward.sharder_ratio"),
			viper.GetFloat64(pfx+"block_reward.miner_ratio"),
			viper.GetFloat64(pfx+"block_reward.blobber_capacity_ratio"),
			viper.GetFloat64(pfx+"block_reward.blobber_usage_ratio"),
		)
		return &br, true, nil
	}
	b, err := json.Marshal(val)
	if err != nil {
		return nil, false, err
	}
	var conf = struct {
		BlockReward *BlockReward `json:"block_reward"`
	}{}
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return nil, false, err
	}
	if len(qtl.Totals) == 0 {
		return conf.BlockReward, true, nil
	}

	lastSettings := qtl.Totals[len(qtl.Totals)-1].LastSettingsChange
	settings := qtl.Totals[lastSettings].SettingsChange

	if settings.BlockReward != conf.BlockReward.BlockReward ||
		settings.QualifyingStake != conf.BlockReward.QualifyingStake ||
		settings.BlobberUsageWeight != conf.BlockReward.BlobberUsageWeight ||
		settings.BlobberCapacityWeight != conf.BlockReward.BlobberCapacityWeight ||
		settings.MinerWeight != conf.BlockReward.MinerWeight ||
		settings.SharderWeight != conf.BlockReward.SharderWeight {
		return conf.BlockReward, true, nil
	}
	return nil, false, nil
}

func (qtl *QualifyingTotalsList) Save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(QualifyingTotalsPerBlockKey, qtl)
	return err
}

func GetQualifyingTotalsList(balances cstate.StateContextI) (*QualifyingTotalsList, error) {
	var val util.Serializable
	val, err := balances.GetTrieNode(QualifyingTotalsPerBlockKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return NewQualifyingTotalsList(), nil
	}

	qtl := NewQualifyingTotalsList()
	err = qtl.Decode(val.Encode())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return qtl, nil
}
