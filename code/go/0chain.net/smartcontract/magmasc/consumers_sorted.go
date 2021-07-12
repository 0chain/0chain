package magmasc

import (
	"sort"
	"sync"
)

type (
	// consumersSorted represents slice of Consumer sorted in alphabetic order by ID.
	// consumersSorted allows O(logN) access.
	consumersSorted struct {
		Sorted []*Consumer `json:"sorted"`
		mux    sync.RWMutex
	}
)

func (m *consumersSorted) add(consumer *Consumer) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.Sorted == nil {
		m.Sorted = make([]*Consumer, 0)
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, consumer)
		return true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ID >= consumer.ID
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, consumer)
		return true // appended
	}
	if m.Sorted[idx].ID == consumer.ID { // the same
		return false // already have
	}

	// insert
	left, right := m.Sorted[:idx], append([]*Consumer{consumer}, m.Sorted[idx:]...)
	m.Sorted = append(left, right...)

	return true // inserted
}

func (m *consumersSorted) get(id string) (*Consumer, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	m.mux.RLock()
	consumer := m.Sorted[idx]
	m.mux.RUnlock()

	return consumer, true // found
}

func (m *consumersSorted) getIndex(id string) (int, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if m.Sorted == nil {
		m.Sorted = make([]*Consumer, 0)
	}

	size := len(m.Sorted)
	if size > 0 {
		idx := sort.Search(size, func(idx int) bool {
			return m.Sorted[idx].ID >= id
		})
		if idx < size && m.Sorted[idx].ID == id {
			return idx, true // found
		}
	}

	return -1, false // not found
}

func (m *consumersSorted) remove(id string) bool {
	idx, found := m.getIndex(id)
	if found {
		m.removeByIndex(idx)
	}

	return found
}

func (m *consumersSorted) removeByIndex(idx int) *Consumer {
	m.mux.Lock()
	defer m.mux.Unlock()

	consumer := *m.Sorted[idx] // copy consumer
	m.Sorted = append(m.Sorted[:idx], m.Sorted[idx+1:]...)

	return &consumer
}

func (m *consumersSorted) update(consumer *Consumer) bool {
	idx, found := m.getIndex(consumer.ID)
	if found {
		m.mux.Lock()
		m.Sorted[idx] = consumer // replace if found
		m.mux.Unlock()
	}

	return found
}
