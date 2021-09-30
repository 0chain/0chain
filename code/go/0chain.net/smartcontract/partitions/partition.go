package partitions

import (
	"0chain.net/chaincore/chain/state"
)

type PartitionItem interface {
	Name() string
}

type PartitionLocation interface {
	PartitionId() int
	Position() int
}

type OrderedPartitionItem interface {
	PartitionItem
	GraterThan(item PartitionItem) bool
}

type Partition interface {
	Add(PartitionItem, state.StateContextI) error
	Remove(PartitionLocation, state.StateContextI) error
}

type changePositionHandler func(
	OrderedPartitionItem,
	PartitionLocation,
	PartitionLocation,
	state.StateContextI,
) error

type OrderedPartition interface {
	Change(OrderedPartitionItem, PartitionLocation, state.StateContextI) error
	OnChangePosition(changePositionHandler)
}

type LeagueTable interface {
	OrderedPartition
}
