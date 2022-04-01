package partitions

import (
	"math/rand"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type PartitionItem interface {
	util.Serializable
	Name() string
	Data() string

	// Copy creates copy of a current PartitionItem implementation.
	Copy() PartitionItem
}

type PartitionItemList interface {
	util.Serializable
	add(it PartitionItem)
	set(idx int, item PartitionItem) error
	remove(item PartitionItem) error
	cutTail() PartitionItem
	changed() bool
	length() int
	itemRange(start, end int) []PartitionItem
	save(balances state.StateContextI) error
	get(key datastore.Key, balances state.StateContextI) error
	getByIndex(int) (PartitionItem, error)
}

type ChangePartitionCallback = func(PartitionItem, int, int, state.StateContextI) error

type Partition interface {
	util.Serializable
	Add(PartitionItem, state.StateContextI) (int, error)
	Remove(PartitionItem, int, state.StateContextI) error

	SetCallback(ChangePartitionCallback)
	Size(state.StateContextI) (int, error)
	Save(state.StateContextI) error

	// Shuffle swaps PartitionItem with provided firstItemIdx and firstPartitionIdx with
	// randomly selected PartitionItem, and saves the resulted Partition in the provided state.StateContextI.
	//
	// Partition section and Item index of the second PartitionItem's should be chosen randomly
	// using the provided source.
	Shuffle(firstItemIdx, firstPartitionIdx int, r *rand.Rand, balances state.StateContextI) error
}

type RandPartition interface {
	Partition
	AddRand(PartitionItem, *rand.Rand, state.StateContextI) (int, error)
	GetRandomSlice(*rand.Rand, state.StateContextI) ([]PartitionItem, error)
}
