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
	allocationStore.data = append(allocationStore.data, allocation)

	//log.Println("Added allocation: ", allocation)
	//
	//latestLocalAllocation, err := allocationStore.GetLatest()
	//if err != nil {
	//	log.Println("Error getting latest allocation: ", err)
	//	return
	//}
	//
	//log.Println("Latest allocation: ", latestLocalAllocation)
}

func (s *AllocationStore) GetLatest() (Allocation, error) {
	if len(allocationStore.data) == 0 {
		return Allocation{}, errors.New("no allocations")
	}
	return allocationStore.data[len(allocationStore.data)-1], nil
}
