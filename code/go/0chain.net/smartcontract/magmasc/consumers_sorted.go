package magmasc

import (
	"sort"
	"sync"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
)

type (
	// consumersSorted represents slice of Consumer sorted in alphabetic order by ID.
	// consumersSorted allows O(logN) access.
	consumersSorted struct {
		Sorted []*bmp.Consumer `json:"sorted"`
		mux    sync.RWMutex
	}
)

func (m *consumersSorted) add(consumer *bmp.Consumer) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.Sorted == nil {
		m.Sorted = make([]*bmp.Consumer, 0)
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, consumer)
		return true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ExtID >= consumer.ExtID
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, consumer)
		return true // appended
	}
	if m.Sorted[idx].ExtID == consumer.ExtID { // the same
		m.Sorted[idx] = consumer // replace
		return false             // already have
	}

	// insert
	left, right := m.Sorted[:idx], append([]*bmp.Consumer{consumer}, m.Sorted[idx:]...)
	m.Sorted = append(left, right...)

	return true // inserted
}

func (m *consumersSorted) get(id datastore.Key) (*bmp.Consumer, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	m.mux.RLock()
	consumer := m.Sorted[idx]
	m.mux.RUnlock()

	return consumer, true // found
}

func (m *consumersSorted) getIndex(id datastore.Key) (int, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if m.Sorted == nil {
		m.Sorted = make([]*bmp.Consumer, 0)
	}

	size := len(m.Sorted)
	if size > 0 {
		idx := sort.Search(size, func(idx int) bool {
			return m.Sorted[idx].ExtID >= id
		})
		if idx < size && m.Sorted[idx].ExtID == id {
			return idx, true // found
		}
	}

	return -1, false // not found
}

func (m *consumersSorted) remove(id datastore.Key) bool {
	idx, found := m.getIndex(id)
	if found {
		m.removeByIndex(idx)
	}

	return found
}

func (m *consumersSorted) removeByIndex(idx int) *bmp.Consumer {
	m.mux.Lock()
	defer m.mux.Unlock()

	consumer := *m.Sorted[idx] // copy consumer
	m.Sorted = append(m.Sorted[:idx], m.Sorted[idx+1:]...)

	return &consumer
}

func (m *consumersSorted) update(consumer *bmp.Consumer) bool {
	idx, found := m.getIndex(consumer.ExtID)
	if found {
		m.mux.Lock()
		m.Sorted[idx] = consumer // replace if found
		m.mux.Unlock()
	}

	return found
}
