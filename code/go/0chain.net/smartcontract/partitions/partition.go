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

type ChangePositionHandler func(
	OrderedPartitionItem, PartitionId, PartitionId, state.StateContextI,
) error

type OrderedPartition interface {
	Change(OrderedPartitionItem, PartitionId, state.StateContextI) error
	OnChangePosition(ChangePositionHandler)
}

type LeagueTable interface {
	OrderedPartition
}

type ItemToPartition func(PartitionItem) (string, error)
type RangeToPartitionList func(PartitionRange) ([]string, error)
type Validator func(PartitionRange, string) bool

type PartitionRange interface {
}

type PartitionIterator interface {
	Start(PartitionRange) error
	Next() string
}

type CrossPartition interface {
	Add(PartitionItem, state.StateContextI) error
	Remove(PartitionItem, state.StateContextI) error
	Change(PartitionItem, PartitionItem, state.StateContextI) error
	GetItems(PartitionRange, int, state.StateContextI) ([]string, error)
	SetPartitionHandler(ItemToPartition)
	SetPartitionIterator(PartitionIterator)
	SetItemValidator(Validator)
}
