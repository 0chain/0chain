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
	// Users represents sorted list of users.
	Users struct {
		Sorted []*zmc.User
	}
)

func (m *Users) add(scID string, item *zmc.User, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(zmc.ErrCodeInternal, "user invalid value").Wrap(zmc.ErrNilPointerValue)
	}
	if got, _ := sci.GetTrieNode(nodeUID(scID, userType, item.Id)); got != nil {
		return errors.New(zmc.ErrCodeInternal, "user already registered: "+item.Id)
	}
	return m.write(scID, item, db, sci)
}

func (m *Users) copy() *Users {
	list := Users{}
	if m.Sorted != nil {
		list.Sorted = make([]*zmc.User, len(m.Sorted))
		copy(list.Sorted, m.Sorted)
	}

	return &list
}

func (m *Users) del(id string, db *gorocksdb.TransactionDB) (*zmc.User, error) {
	if idx, found := m.getIndex(id); found {
		return m.delByIndex(idx, db)
	}

	return nil, errors.New(zmc.ErrCodeInternal, "value not present")
}

func (m *Users) delByIndex(idx int, db *gorocksdb.TransactionDB) (*zmc.User, error) {
	if idx >= len(m.Sorted) || idx < 0 {
		return nil, errors.New(zmc.ErrCodeInternal, "index out of range")
	}

	list := m.copy()
	item := *list.Sorted[idx] // get copy of item
	list.Sorted = append(list.Sorted[:idx], list.Sorted[idx+1:]...)

	blob, err := json.Marshal(list.Sorted)
	if err != nil {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "encode users list failed", err)
	}

	tx := store.GetTransaction(db)
	if err = tx.Conn.Put([]byte(allUsersKey), blob); err != nil {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "insert users list failed", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "commit changes failed", err)
	}

	m.Sorted = list.Sorted

	return &item, nil
}

func (m *Users) hasEqual(item *zmc.User) bool {
	if got, found := m.get(item.Id); !found || !reflect.DeepEqual(got, item) {
		return false // not found or not equal
	}

	return true // found and equal
}

func (m *Users) get(id string) (*zmc.User, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

func (m *Users) getByConsumer(consumerID string) (*zmc.User, bool) {
	for _, item := range m.Sorted {
		if item.ConsumerId == consumerID {
			return item, true // found
		}
	}

	return nil, false // not found
}

func (m *Users) getByIndex(idx int) (*zmc.User, bool) {
	if idx < len(m.Sorted) && idx >= 0 {
		return m.Sorted[idx], true
	}

	return nil, false // not found
}

func (m *Users) getIndex(id string) (int, bool) {
	size := len(m.Sorted)
	if size > 0 {
		idx := sort.Search(size, func(idx int) bool {
			return m.Sorted[idx].Id >= id
		})
		if idx < size && m.Sorted[idx].Id == id {
			return idx, true // found
		}
	}

	return -1, false // not found
}

func (m *Users) put(item *zmc.User) (int, bool) {
	if item == nil {
		return 0, false
	}

	size := len(m.Sorted)
	if size == 0 {
		m.Sorted = append(m.Sorted, item)
		return 0, true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].Id >= item.Id
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, item)
		return idx, true // appended
	}
	if m.Sorted[idx].Id == item.Id { // the same
		m.Sorted[idx] = item // replace
		return idx, false    // already have
	}

	left, right := m.Sorted[:idx], append([]*zmc.User{item}, m.Sorted[idx:]...) // insert
	m.Sorted = append(left, right...)

	return idx, true // inserted
}

func (m *Users) write(scID string, item *zmc.User, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(zmc.ErrCodeInternal, "user invalid value").Wrap(zmc.ErrNilPointerValue)
	}
	if _, err := sci.InsertTrieNode(nodeUID(scID, userType, item.Id), item); err != nil {
		return errors.Wrap(zmc.ErrCodeInternal, "insert user failed", err)
	}

	var list *Users
	if !m.hasEqual(item) { // check if an equal item already added
		got, found := m.getByConsumer(item.ConsumerId)
		if found && item.Id != got.Id { // check if a consumer already registered
			return errors.New(zmc.ErrCodeInternal, "user's consumer already registered: "+item.ConsumerId)
		}

		list = m.copy()
		list.put(item) // add or replace
		blob, err := json.Marshal(list.Sorted)
		if err != nil {
			return errors.Wrap(zmc.ErrCodeInternal, "encode users list failed", err)
		}

		tx := store.GetTransaction(db)
		if err = tx.Conn.Put([]byte(allUsersKey), blob); err != nil {
			return errors.Wrap(zmc.ErrCodeInternal, "insert users list failed", err)
		}
		if err = tx.Commit(); err != nil {
			return errors.Wrap(zmc.ErrCodeInternal, "commit changes failed", err)
		}
	}
	if list != nil {
		m.Sorted = list.Sorted
	}

	return nil
}

// usersFetch extracts all users stored in memory data store with given id.
func usersFetch(id string, db *gorocksdb.TransactionDB) (*Users, error) {
	list := &Users{}

	tx := store.GetTransaction(db)
	buf, err := tx.Conn.Get(tx.ReadOptions, []byte(id))
	if err != nil {
		return list, errors.Wrap(zmc.ErrCodeInternal, "get users list failed", err)
	}
	defer buf.Free()

	blob := buf.Data()
	if blob != nil {
		if err = json.Unmarshal(blob, &list.Sorted); err != nil {
			return list, errors.Wrap(zmc.ErrCodeInternal, "decode users list failed", err)
		}
	}

	return list, tx.Commit()
}
