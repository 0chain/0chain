package partitions

import (
	"math/rand"

	"0chain.net/chaincore/chain/state"
)

type PartitionItem interface {
	Name() string
}

type ChangePartitionCallback = func(PartitionItem, int, int, state.StateContextI) error

type Partition interface {
	Add(PartitionItem, state.StateContextI) (int, error)
	Remove(PartitionItem, int, state.StateContextI) error

	SetCallback(ChangePartitionCallback)
	Size(state.StateContextI) (int, error)
	Save(state.StateContextI) error
}

type RandPartition interface {
	Partition
	GetRandomSlice(*rand.Rand, state.StateContextI) ([]string, error)
}
