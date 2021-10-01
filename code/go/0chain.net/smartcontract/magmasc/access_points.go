package magmasc

import (
	"encoding/json"
	"reflect"
	"sort"

	"github.com/0chain/gorocksdb"
	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	store "0chain.net/core/ememorystore"
)

type (
	// AccessPoints represents sorted list of access points.
	AccessPoints struct {
		Sorted []*zmc.AccessPoint
	}
)

func (m *AccessPoints) add(scID string, item *zmc.AccessPoint, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "access point invalid value").Wrap(errNilPointerValue)
	}
	if got, _ := sci.GetTrieNode(nodeUID(scID, accessPointType, item.ID)); got != nil {
		return errors.New(errCodeInternal, "access point already registered: "+item.ID)
	}

	return m.write(scID, item, db, sci)
}

func (m *AccessPoints) copy() *AccessPoints {
	list := AccessPoints{}
	if m.Sorted != nil {
		list.Sorted = make([]*zmc.AccessPoint, len(m.Sorted))
		copy(list.Sorted, m.Sorted)
	}

	return &list
}

func (m *AccessPoints) del(id string, db *gorocksdb.TransactionDB) (*zmc.AccessPoint, error) {
	if idx, found := m.getIndex(id); found {
		return m.delByIndex(idx, db)
	}

	return nil, errors.New(errCodeInternal, "value not present")
}

func (m *AccessPoints) delByIndex(idx int, db *gorocksdb.TransactionDB) (*zmc.AccessPoint, error) {
	if idx >= len(m.Sorted) {
		return nil, errors.New(errCodeInternal, "index out of range")
	}

	list := m.copy()
	item := *list.Sorted[idx] // get copy of item
	list.Sorted = append(list.Sorted[:idx], list.Sorted[idx+1:]...)

	blob, err := json.Marshal(list.Sorted)
	if err != nil {
		return nil, errors.Wrap(errCodeInternal, "encode access points list failed", err)
	}

	tx := store.GetTransaction(db)
	if err = tx.Conn.Put([]byte(AllAccessPointsKey), blob); err != nil {
		return nil, errors.Wrap(errCodeInternal, "insert access points list failed", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(errCodeInternal, "commit changes failed", err)
	}

	m.Sorted = list.Sorted

	return &item, nil
}

func (m *AccessPoints) hasEqual(item *zmc.AccessPoint) bool {
	if got, found := m.get(item.ID); !found || !reflect.DeepEqual(got, item) {
		return false // not found or not equal
	}

	return true // found and equal
}

func (m *AccessPoints) get(id string) (*zmc.AccessPoint, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

//nolint:unused
func (m *AccessPoints) getByIndex(idx int) (*zmc.AccessPoint, bool) {
	if idx < len(m.Sorted) {
		return m.Sorted[idx], true
	}

	return nil, false // not found
}

func (m *AccessPoints) getIndex(id string) (int, bool) {
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

func (m *AccessPoints) put(item *zmc.AccessPoint) (int, bool) {
	if item == nil {
		return 0, false
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, item)
		return 0, true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ID >= item.ID
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, item)
		return idx, true // appended
	}
	if m.Sorted[idx].ID == item.ID { // the same
		m.Sorted[idx] = item // replace
		return idx, false    // already have
	}

	left, right := m.Sorted[:idx], append([]*zmc.AccessPoint{item}, m.Sorted[idx:]...) // insert
	m.Sorted = append(left, right...)

	return idx, true // inserted
}

func (m *AccessPoints) write(scID string, item *zmc.AccessPoint, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "access point invalid value").Wrap(errNilPointerValue)
	}
	if _, err := sci.InsertTrieNode(nodeUID(scID, accessPointType, item.ID), item); err != nil {
		return errors.Wrap(errCodeInternal, "insert access point failed", err)
	}

	var list *AccessPoints
	if !m.hasEqual(item) { // check if an equal item already added
		list = m.copy()
		list.put(item) // add or replace
		blob, err := json.Marshal(list.Sorted)
		if err != nil {
			return errors.Wrap(errCodeInternal, "encode access points list failed", err)
		}

		tx := store.GetTransaction(db)
		if err = tx.Conn.Put([]byte(AllAccessPointsKey), blob); err != nil {
			return errors.Wrap(errCodeInternal, "insert access points list failed", err)
		}
		if err = tx.Commit(); err != nil {
			return errors.Wrap(errCodeInternal, "commit changes failed", err)
		}
	}
	if list != nil {
		m.Sorted = list.Sorted
	}

	return nil
}

// accessPointsFetch extracts all access points stored in memory data store with given id.
func accessPointsFetch(id string, db *gorocksdb.TransactionDB) (*AccessPoints, error) {
	list := &AccessPoints{}

	tx := store.GetTransaction(db)
	buf, err := tx.Conn.Get(tx.ReadOptions, []byte(id))
	if err != nil {
		return list, errors.Wrap(errCodeInternal, "get access points  list failed", err)
	}
	defer buf.Free()

	blob := buf.Data()
	if blob != nil {
		if err = json.Unmarshal(blob, &list.Sorted); err != nil {
			return list, errors.Wrap(errCodeInternal, "decode access points  list failed", err)
		}
	}

	return list, tx.Commit()
}
