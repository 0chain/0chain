package blockrewards

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

const (
	storageScAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

var (
	QualifyingTotalsPerBlockKey = datastore.Key(storageScAddress + encryption.Hash("qualifying_totals_per_block"))
	ConfigKey                   = datastore.Key(storageScAddress + ":configurations")
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
	Round    int64 `json:"round"` // todo probably remove after debug
	Capacity int64 `json:"capacity"`
	Used     int64 `json:"used"`
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

func (qtl *QualifyingTotalsList) Initialise() {
	qtl.Totals = []QualifyingTotals{
		{},
	}
}

func (qtl *QualifyingTotalsList) GetCapacity(round int64) int64 {
	return qtl.Totals[round].Capacity
}

func (qtl *QualifyingTotalsList) GetUsed(round int64) int64 {
	return qtl.Totals[round].Used
}

func (qtl *QualifyingTotalsList) Save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(QualifyingTotalsPerBlockKey, qtl)
	return err
}

func GetQualifyingTotalsList(
	_ int64,
	balances cstate.StateContextI,
) (*QualifyingTotalsList, error) {
	var val util.Serializable
	var qtl QualifyingTotalsList
	val, err := balances.GetTrieNode(QualifyingTotalsPerBlockKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		qtl.Initialise()
		return &qtl, nil
	}

	err = qtl.Decode(val.Encode())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	if len(qtl.Totals) == 0 {
		qtl.Initialise()
	}
	return &qtl, nil
}
