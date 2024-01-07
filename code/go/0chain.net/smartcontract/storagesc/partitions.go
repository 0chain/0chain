package storagesc

import (
	"0chain.net/smartcontract/common"
)

var (
	initPartitionsFuncs = []initPartitionFunc{}
)

type initPartitionFunc func(common.StateContextI) error

// InitPartitions create partitions if not exist
func InitPartitions(state common.StateContextI) error {
	for _, f := range initPartitionsFuncs {
		if err := f(state); err != nil {
			return err
		}
	}

	return nil
}

func regInitPartsFunc(f initPartitionFunc) {
	initPartitionsFuncs = append(initPartitionsFuncs, f)
}
