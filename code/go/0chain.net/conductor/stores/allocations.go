package stores

import (
	"errors"
	"sync"

	"0chain.net/conductor/types"
)

type (
	Allocation = types.Allocation

	AllocationStore struct {
		data []Allocation
		lock sync.RWMutex
	}
)

var allocationStore AllocationStore

func init() {
	allocationStore = AllocationStore{}
	allocationStore.lock = sync.RWMutex{}
	allocationStore.data = make([]Allocation, 0)
}

func GetAllocationStore() *AllocationStore {
	return &allocationStore
}

func (s *AllocationStore) Add(allocation Allocation) {
	_ = append(allocationStore.data, allocation)
}

func (s *AllocationStore) GetLatest() (Allocation, error) {
	if len(allocationStore.data) == 0 {
		return Allocation{}, errors.New("no allocations")
	}
	return allocationStore.data[len(allocationStore.data)-1], nil
}
