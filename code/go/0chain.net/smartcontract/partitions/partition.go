package partitions

import (
	"0chain.net/chaincore/chain/state"
)

type PartitionItem interface {
	Name() string
}

type ChangePartitionCallback = func(PartitionItem, int, int, state.StateContextI) error

type Partition interface {
	Add(PartitionItem, state.StateContextI) (int, error)
	Remove(PartitionItem, int, state.StateContextI) error
}

type RandomisingPartition interface {
	Partition
	GetRandomPartition(int64, state.StateContextI) ([]PartitionItem, error)
}
