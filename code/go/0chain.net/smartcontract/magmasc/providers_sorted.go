package magmasc

import (
	"sort"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
)

type (
	// providersSorted represents slice of Provider sorted in alphabetic order by ID.
	// providersSorted allows O(logN) access.
	providersSorted struct {
		Sorted []*bmp.Provider `json:"sorted"`
	}
)

func (m *providersSorted) add(provider *bmp.Provider) (int, bool) {
	if m.Sorted == nil {
		m.Sorted = make([]*bmp.Provider, 0)
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, provider)
		return 0, true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ExtID >= provider.ExtID
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, provider)
		return idx, true // appended
	}
	if m.Sorted[idx].ExtID == provider.ExtID { // the same
		m.Sorted[idx] = provider // replace
		return idx, false        // already have
	}

	// insert
	left, right := m.Sorted[:idx], append([]*bmp.Provider{provider}, m.Sorted[idx:]...)
	m.Sorted = append(left, right...)

	return idx, true // inserted
}

func (m *providersSorted) get(id datastore.Key) (*bmp.Provider, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

func (m *providersSorted) getByHost(host string) (*bmp.Provider, bool) {
	for _, item := range m.Sorted {
		if item.Host == host {
			return item, true // found
		}
	}

	return nil, false // not found
}

func (m *providersSorted) getByIndex(idx int) (*bmp.Provider, bool) {
	if idx < len(m.Sorted) {
		return m.Sorted[idx], true
	}

	return nil, false // not found
}

func (m *providersSorted) getIndex(id datastore.Key) (int, bool) {
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

func (m *providersSorted) getSorted() (sorted []*bmp.Provider) {
	if m.Sorted != nil {
		sorted = make([]*bmp.Provider, len(m.Sorted))
		copy(sorted, m.Sorted)
	}

	return sorted
}

func (m *providersSorted) remove(id datastore.Key) bool {
	idx, found := m.getIndex(id)
	if found {
		m.removeByIndex(idx)
	}

	return found
}

func (m *providersSorted) removeByIndex(idx int) *bmp.Provider {
	provider := *m.Sorted[idx] // copy provider
	m.Sorted = append(m.Sorted[:idx], m.Sorted[idx+1:]...)

	return &provider
}

func (m *providersSorted) setSorted(sorted []*bmp.Provider) {
	if sorted == nil {
		m.Sorted = nil
	} else {
		m.Sorted = make([]*bmp.Provider, len(sorted))
		copy(m.Sorted, sorted)
	}
}

func (m *providersSorted) update(provider *bmp.Provider) bool {
	idx, found := m.getIndex(provider.ExtID)
	if found {
		m.Sorted[idx] = provider // replace if found
	}

	return found
}
