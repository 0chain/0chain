package partitions

import (
	"0chain.net/chaincore/chain/state"
)

type PartitionItem interface {
	Name() string
}

type PartitionId int

const NoPartition PartitionId = -1

type OrderedPartitionItem interface {
	PartitionItem
	GraterThan(item PartitionItem) bool
}

type Partition interface {
	Add(PartitionItem, state.StateContextI) error
	Remove(string, PartitionId, state.StateContextI) error
}

type changePositionHandler func(
	OrderedPartitionItem, PartitionId, PartitionId, state.StateContextI,
) error

type OrderedPartition interface {
	Change(OrderedPartitionItem, PartitionId, state.StateContextI) error
	OnChangePosition(changePositionHandler)
}

type LeagueTable interface {
	OrderedPartition
}
