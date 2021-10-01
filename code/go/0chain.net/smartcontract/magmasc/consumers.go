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
	// Consumers represents sorted list of consumers.
	Consumers struct {
		Sorted []*zmc.Consumer
	}
)

func (m *Consumers) add(scID string, item *zmc.Consumer, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "consumer invalid value").Wrap(errNilPointerValue)
	}
	if got, _ := sci.GetTrieNode(nodeUID(scID, consumerType, item.ExtID)); got != nil {
		return errors.New(errCodeInternal, "consumer already registered: "+item.ExtID)
	}

	return m.write(scID, item, db, sci)
}

func (m *Consumers) copy() *Consumers {
	list := Consumers{}
	if m.Sorted != nil {
		list.Sorted = make([]*zmc.Consumer, len(m.Sorted))
		copy(list.Sorted, m.Sorted)
	}

	return &list
}

func (m *Consumers) del(id string, db *gorocksdb.TransactionDB) (*zmc.Consumer, error) {
	if idx, found := m.getIndex(id); found {
		return m.delByIndex(idx, db)
	}

	return nil, errors.New(errCodeInternal, "value not present")
}

func (m *Consumers) delByIndex(idx int, db *gorocksdb.TransactionDB) (*zmc.Consumer, error) {
	if idx >= len(m.Sorted) {
		return nil, errors.New(errCodeInternal, "index out of range")
	}

	list := m.copy()
	item := *list.Sorted[idx] // get copy of item
	list.Sorted = append(list.Sorted[:idx], list.Sorted[idx+1:]...)

	blob, err := json.Marshal(list.Sorted)
	if err != nil {
		return nil, errors.Wrap(errCodeInternal, "encode consumers list failed", err)
	}

	tx := store.GetTransaction(db)
	if err = tx.Conn.Put([]byte(AllConsumersKey), blob); err != nil {
		return nil, errors.Wrap(errCodeInternal, "insert consumers list failed", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(errCodeInternal, "commit changes failed", err)
	}

	m.Sorted = list.Sorted

	return &item, nil
}

func (m *Consumers) hasEqual(item *zmc.Consumer) bool {
	if got, found := m.get(item.ExtID); !found || !reflect.DeepEqual(got, item) {
		return false // not found or not equal
	}

	return true // found and equal
}

func (m *Consumers) get(id string) (*zmc.Consumer, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

func (m *Consumers) getByHost(host string) (*zmc.Consumer, bool) {
	for _, item := range m.Sorted {
		if item.Host == host {
			return item, true // found
		}
	}

	return nil, false // not found
}

func (m *Consumers) getByIndex(idx int) (*zmc.Consumer, bool) {
	if idx < len(m.Sorted) {
		return m.Sorted[idx], true
	}

	return nil, false // not found
}

func (m *Consumers) getIndex(id string) (int, bool) {
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

func (m *Consumers) put(item *zmc.Consumer) (int, bool) {
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

	left, right := m.Sorted[:idx], append([]*zmc.Consumer{item}, m.Sorted[idx:]...) // insert
	m.Sorted = append(left, right...)

	return idx, true // inserted
}

func (m *Consumers) write(scID string, item *zmc.Consumer, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "consumer invalid value").Wrap(errNilPointerValue)
	}
	if _, err := sci.InsertTrieNode(nodeUID(scID, consumerType, item.ExtID), item); err != nil {
		return errors.Wrap(errCodeInternal, "insert consumer failed", err)
	}

	var list *Consumers
	if !m.hasEqual(item) { // check if an equal item already added
		got, found := m.getByHost(item.Host)
		if found && item.ID != got.ID { // check if a host already registered
			return errors.New(errCodeInternal, "consumer host already registered: "+item.Host)
		}

		list = m.copy()
		list.put(item) // add or replace
		blob, err := json.Marshal(list.Sorted)
		if err != nil {
			return errors.Wrap(errCodeInternal, "encode consumers list failed", err)
		}

		tx := store.GetTransaction(db)
		if err = tx.Conn.Put([]byte(AllConsumersKey), blob); err != nil {
			return errors.Wrap(errCodeInternal, "insert consumers list failed", err)
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

// consumersFetch extracts all consumers stored in memory data store with given id.
func consumersFetch(id string, db *gorocksdb.TransactionDB) (*Consumers, error) {
	list := &Consumers{}

	tx := store.GetTransaction(db)
	buf, err := tx.Conn.Get(tx.ReadOptions, []byte(id))
	if err != nil {
		return list, errors.Wrap(errCodeInternal, "get consumers list failed", err)
	}
	defer buf.Free()

	blob := buf.Data()
	if blob != nil {
		if err = json.Unmarshal(blob, &list.Sorted); err != nil {
			return list, errors.Wrap(errCodeInternal, "decode consumers list failed", err)
		}
	}

	return list, tx.Commit()
}
