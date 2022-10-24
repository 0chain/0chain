package storagesc

import "0chain.net/chaincore/chain/state"

var (
	initPartitionsFuncs []initPartitionFunc
)

type initPartitionFunc func(state.StateContextI) error

// InitPartitions create partitions if not exist
func InitPartitions(state state.StateContextI) error {
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
