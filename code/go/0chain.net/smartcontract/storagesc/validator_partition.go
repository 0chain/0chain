package storagesc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

//go:generate msgp -v -io=false -tests=false -unexported=true

const allValidatorsPartitionSize = 50

func getValidatorsList(state state.StateContextI) (*partitions.Partitions, error) {
	validators, err := partitions.GetPartitions(state, ALL_VALIDATORS_KEY)
	if err != nil {
		return nil, err
	}
	validators.SetCallback(validatorCallback)
	return validators, nil
}

type ValidationPartitionNode struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func (vn *ValidationPartitionNode) GetID() string {
	return vn.Id
}

func validatorCallback(id string, data []byte, toPartition, _ int, sCtx state.StateContextI) error {
	replace := &ValidationNode{
		ID: id,
	}
	if err := sCtx.GetTrieNode(replace.GetKey(ADDRESS), replace); err != nil {
		return err
	}
	replace.PartitionPosition = toPartition
	if _, err := sCtx.InsertTrieNode(replace.GetKey(ADDRESS), replace); err != nil {
		return err
	}
	return nil
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, ALL_VALIDATORS_KEY, allValidatorsPartitionSize)
		return err
	})
}
