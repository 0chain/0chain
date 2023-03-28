package storagesc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

//go:generate msgp -v -io=false -tests=false -unexported=true

const allValidatorsPartitionSize = 50

func getValidatorsList(state state.StateContextI) (*partitions.Partitions, error) {
	return partitions.GetPartitions(state, ALL_VALIDATORS_KEY)
}

type ValidationPartitionNode struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func (vn *ValidationPartitionNode) GetID() string {
	return vn.Id
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, ALL_VALIDATORS_KEY, allValidatorsPartitionSize)
		return err
	})
}
