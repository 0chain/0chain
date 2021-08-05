package magmasc

import (
	"encoding/json"
	"sort"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	store "0chain.net/core/ememorystore"
)

type (
	// Providers represents sorted list of providers.
	Providers struct {
		Sorted []*bmp.Provider
	}
)

func (m *Providers) add(scID string, item *bmp.Provider, db *store.Connection, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "provider invalid value").Wrap(errNilPointerValue)
	}
	if _, found := m.getIndex(item.ExtID); found {
		return errors.New(errCodeInternal, "provider already registered: "+item.ExtID)
	}
	if _, found := m.getByHost(item.Host); found {
		return errors.New(errCodeInternal, "provider host already registered: "+item.Host)
	}

	return m.write(scID, item, db, sci)
}

func (m *Providers) copy() (list Providers) {
	if m.Sorted != nil {
		list.Sorted = make([]*bmp.Provider, len(m.Sorted))
		copy(list.Sorted, m.Sorted)
	}

	return list
}

func (m *Providers) del(id string, db *store.Connection) (*bmp.Provider, error) {
	idx, found := m.getIndex(id)
	if found {
		return m.delByIndex(idx, db)
	}

	return nil, errors.New(errCodeInternal, "value not present")
}

func (m *Providers) delByIndex(idx int, db *store.Connection) (*bmp.Provider, error) {
	if idx >= len(m.Sorted) {
		return nil, errors.New(errCodeInternal, "index out of range")
	}

	list := m.copy()
	item := *list.Sorted[idx] // get copy of item
	list.Sorted = append(list.Sorted[:idx], list.Sorted[idx+1:]...)

	blob, err := json.Marshal(list.Sorted)
	if err != nil {
		return nil, errors.Wrap(errCodeInternal, "encode providers list failed", err)
	}
	if err = db.Conn.Put([]byte(AllProvidersKey), blob); err != nil {
		return nil, errors.Wrap(errCodeInternal, "insert providers list failed", err)
	}
	if err = db.Commit(); err != nil {
		return nil, errors.Wrap(errCodeInternal, "commit changes failed", err)
	}

	m.Sorted = list.Sorted

	return &item, nil
}

func (m *Providers) get(id string) (*bmp.Provider, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

func (m *Providers) getByHost(host string) (*bmp.Provider, bool) {
	for _, item := range m.Sorted {
		if item.Host == host {
			return item, true // found
		}
	}

	return nil, false // not found
}

func (m *Providers) getByIndex(idx int) (*bmp.Provider, bool) {
	if idx < len(m.Sorted) {
		return m.Sorted[idx], true
	}

	return nil, false // not found
}

func (m *Providers) getIndex(id string) (int, bool) {
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

func (m *Providers) put(item *bmp.Provider) (int, bool) {
	if item == nil {
		return 0, false
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, item)
		return 0, true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ExtID >= item.ExtID
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, item)
		return idx, true // appended
	}
	if m.Sorted[idx].ExtID == item.ExtID { // the same
		m.Sorted[idx] = item // replace
		return idx, false    // already have
	}

	// insert
	left, right := m.Sorted[:idx], append([]*bmp.Provider{item}, m.Sorted[idx:]...)
	m.Sorted = append(left, right...)

	return idx, true // inserted
}

func (m *Providers) write(scID string, item *bmp.Provider, db *store.Connection, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "provider invalid value").Wrap(errNilPointerValue)
	}

	list := m.copy()
	list.put(item) // add or replace

	blob, err := json.Marshal(list.Sorted)
	if err != nil || blob == nil {
		return errors.Wrap(errCodeInternal, "encode providers list failed", err)
	}
	if err = db.Conn.Put([]byte(AllProvidersKey), blob); err != nil {
		return errors.Wrap(errCodeInternal, "insert providers list failed", err)
	}
	if _, err = sci.InsertTrieNode(nodeUID(scID, item.ExtID, providerType), item); err != nil {
		_ = db.Conn.Rollback()
		return errors.Wrap(errCodeInternal, "insert provider failed", err)
	}
	if err = db.Commit(); err != nil {
		return errors.Wrap(errCodeInternal, "commit changes failed", err)
	}

	m.Sorted = list.Sorted

	return nil
}

// fetchProviders extracts all providers stored in memory data store with given id.
func fetchProviders(id string, db *store.Connection) (*Providers, error) {
	list := &Providers{}

	buf, err := db.Conn.Get(db.ReadOptions, []byte(id))
	if err != nil {
		return list, errors.Wrap(errCodeInternal, "get providers list failed", err)
	}
	defer buf.Free()

	blob := buf.Data()
	if blob != nil {
		if err = json.Unmarshal(blob, &list.Sorted); err != nil {
			return list, errors.Wrap(errCodeInternal, "decode providers list failed", err)
		}
	}

	return list, nil
}
