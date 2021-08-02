package magmasc

import (
	"sort"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
)

type (
	// consumersSorted represents slice of Consumer sorted in alphabetic order by ID.
	// consumersSorted allows O(logN) access.
	consumersSorted struct {
		Sorted []*bmp.Consumer `json:"sorted"`
	}
)

func (m *consumersSorted) add(consumer *bmp.Consumer) (int, bool) {
	if m.Sorted == nil {
		m.Sorted = make([]*bmp.Consumer, 0)
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, consumer)
		return 0, true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ExtID >= consumer.ExtID
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, consumer)
		return idx, true // appended
	}
	if m.Sorted[idx].ExtID == consumer.ExtID { // the same
		m.Sorted[idx] = consumer // replace
		return idx, false        // already have
	}

	// insert
	left, right := m.Sorted[:idx], append([]*bmp.Consumer{consumer}, m.Sorted[idx:]...)
	m.Sorted = append(left, right...)

	return idx, true // inserted
}

func (m *consumersSorted) get(id datastore.Key) (*bmp.Consumer, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

func (m *consumersSorted) getByHost(host string) (*bmp.Consumer, bool) {
	for _, item := range m.Sorted {
		if item.Host == host {
			return item, true // found
		}
	}

	return nil, false // not found
}

func (m *consumersSorted) getByIndex(idx int) (*bmp.Consumer, bool) {
	if idx < len(m.Sorted) {
		return m.Sorted[idx], true
	}

	return nil, false // not found
}

func (m *consumersSorted) getIndex(id datastore.Key) (int, bool) {
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

func (m *consumersSorted) getSorted() (sorted []*bmp.Consumer) {
	if m.Sorted != nil {
		sorted = make([]*bmp.Consumer, len(m.Sorted))
		copy(sorted, m.Sorted)
	}

	return sorted
}

func (m *consumersSorted) remove(id datastore.Key) bool {
	idx, found := m.getIndex(id)
	if found {
		m.removeByIndex(idx)
	}

	return found
}

func (m *consumersSorted) removeByIndex(idx int) *bmp.Consumer {
	consumer := *m.Sorted[idx] // copy consumer
	m.Sorted = append(m.Sorted[:idx], m.Sorted[idx+1:]...)

	return &consumer
}

func (m *consumersSorted) setSorted(sorted []*bmp.Consumer) {
	if sorted == nil {
		m.Sorted = nil
	} else {
		m.Sorted = make([]*bmp.Consumer, len(sorted))
		copy(m.Sorted, sorted)
	}
}

func (m *consumersSorted) update(consumer *bmp.Consumer) bool {
	idx, found := m.getIndex(consumer.ExtID)
	if found {
		m.Sorted[idx] = consumer // replace if found
	}

	return found
}
