package partitions

import (
	"math/rand"

	"0chain.net/core/datastore"

	"0chain.net/core/util"

	"0chain.net/chaincore/chain/state"
)

type PartitionItem interface {
	util.Serializable
	Name() string
	Data() string
}

type PartitionItemList interface {
	util.Serializable
	add(it PartitionItem)
	remove(item PartitionItem) error
	cutTail() PartitionItem
	changed() bool
	length() int
	itemRange(start, end int) []PartitionItem
	save(balances state.StateContextI) error
	get(key datastore.Key, balances state.StateContextI) error
}

type ChangePartitionCallback = func(PartitionItem, int, int, state.StateContextI) error

type Partition interface {
	util.Serializable
	Add(PartitionItem, state.StateContextI) (int, error)
	Remove(PartitionItem, int, state.StateContextI) error

	SetCallback(ChangePartitionCallback)
	Size(state.StateContextI) (int, error)
	Save(state.StateContextI) error
}

type RandPartition interface {
	Partition
	GetRandomSlice(*rand.Rand, state.StateContextI) ([]PartitionItem, error)
}
