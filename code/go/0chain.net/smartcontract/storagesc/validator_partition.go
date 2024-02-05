package storagesc

import (
	"0chain.net/chaincore/chain/state"
	common2 "0chain.net/smartcontract/partitions"
	partitions_v1 "0chain.net/smartcontract/partitions_v_1"
	partitions_v2 "0chain.net/smartcontract/partitions_v_2"
)

//go:generate msgp -v -io=false -tests=false -unexported=true

const allValidatorsPartitionSize = 50

func getValidatorsList(balances state.StateContextI) (res common2.Partitions, err error) {
	actError := state.WithActivation(balances, "apollo", func() error {
		res, err = partitions_v1.GetPartitions(balances, ALL_VALIDATORS_KEY)
		return nil
	}, func() error {
		res, err = partitions_v2.GetPartitions(balances, ALL_VALIDATORS_KEY)
		return nil
	})
	if actError != nil {
		return nil, actError
	}

	return
}

type ValidationPartitionNode struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func (vn *ValidationPartitionNode) GetID() string {
	return vn.Id
}

func init() {
	regInitPartsFunc(func(balances state.StateContextI) error {
		return state.WithActivation(balances, "apollo", func() error {
			_, err := partitions_v1.CreateIfNotExists(balances, ALL_VALIDATORS_KEY, allValidatorsPartitionSize)
			return err
		}, func() error {
			_, err := partitions_v2.CreateIfNotExists(balances, ALL_VALIDATORS_KEY, allValidatorsPartitionSize)
			return err
		})
	})
}
