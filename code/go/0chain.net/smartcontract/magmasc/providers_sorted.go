package magmasc

import (
	"sort"
	"sync"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
)

type (
	// providersSorted represents slice of Provider sorted in alphabetic order by ID.
	// providersSorted allows O(logN) access.
	providersSorted struct {
		Sorted []*bmp.Provider `json:"sorted"`
		mux    sync.RWMutex
	}
)

func (m *providersSorted) add(provider *bmp.Provider) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.Sorted == nil {
		m.Sorted = make([]*bmp.Provider, 0)
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, provider)
		return true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ExtID >= provider.ExtID
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, provider)
		return true // appended
	}
	if m.Sorted[idx].ExtID == provider.ExtID { // the same
		m.Sorted[idx] = provider // replace
		return false             // already have
	}

	// insert
	left, right := m.Sorted[:idx], append([]*bmp.Provider{provider}, m.Sorted[idx:]...)
	m.Sorted = append(left, right...)

	return true // inserted
}

func (m *providersSorted) get(id datastore.Key) (*bmp.Provider, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	m.mux.RLock()
	provider := m.Sorted[idx]
	m.mux.RUnlock()

	return provider, true // found
}

func (m *providersSorted) getIndex(id datastore.Key) (int, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if m.Sorted == nil {
		m.Sorted = make([]*bmp.Provider, 0)
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

func (m *providersSorted) remove(id datastore.Key) bool {
	idx, found := m.getIndex(id)
	if found {
		m.removeByIndex(idx)
	}

	return found
}

func (m *providersSorted) removeByIndex(idx int) *bmp.Provider {
	m.mux.Lock()
	defer m.mux.Unlock()

	provider := *m.Sorted[idx] // copy provider
	m.Sorted = append(m.Sorted[:idx], m.Sorted[idx+1:]...)

	return &provider
}

func (m *providersSorted) update(provider *bmp.Provider) bool {
	idx, found := m.getIndex(provider.ExtID)
	if found {
		m.mux.Lock()
		m.Sorted[idx] = provider // replace if found
		m.mux.Unlock()
	}

	return found
}
