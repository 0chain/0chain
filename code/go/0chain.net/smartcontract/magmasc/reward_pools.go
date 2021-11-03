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
	// rewardPools a list of token pool implementation.
	// The list of reward pools sorted by expiration by value.
	// Those that expire earlier at the top of the list.
	// Those that no expires placed at the end of the list.
	rewardPools struct {
		Sorted []*tokenPool
	}
)

func (m *rewardPools) add(scID string, item *tokenPool, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(zmc.ErrCodeInternal, "reward pool invalid value").Wrap(zmc.ErrNilPointerValue)
	}
	if got, _ := sci.GetTrieNode(nodeUID(scID, rewardTokenPool, item.Id)); got != nil {
		return errors.New(zmc.ErrCodeInternal, "reward pool already registered")
	}

	return m.write(scID, item, db, sci)
}

func (m *rewardPools) copy() *rewardPools {
	list := &rewardPools{}
	if m.Sorted != nil {
		list.Sorted = make([]*tokenPool, len(m.Sorted))
		copy(list.Sorted, m.Sorted)
	}

	return list
}

func (m *rewardPools) del(scID string, item *tokenPool, db *gorocksdb.TransactionDB, sci chain.StateContextI) (*tokenPool, error) {
	if _, err := sci.DeleteTrieNode(nodeUID(scID, rewardTokenPool, item.Id)); err != nil {
		return nil, errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "delete reward pool failed", err)
	}
	if idx, found := m.getIndex(item.Id); found {
		return m.delByIndex(idx, db)
	}

	return nil, errors.New(zmc.ErrCodeInternal, "value not present")
}

func (m *rewardPools) delByIndex(idx int, db *gorocksdb.TransactionDB) (*tokenPool, error) {
	if idx >= len(m.Sorted) {
		return nil, errors.New(zmc.ErrCodeInternal, "index out of range")
	}

	list := m.copy()
	item := *list.Sorted[idx] // get copy of item
	list.Sorted = append(list.Sorted[:idx], list.Sorted[idx+1:]...)

	blob, err := json.Marshal(list.Sorted)
	if err != nil {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "encode reward pools list failed", err)
	}

	tx := store.GetTransaction(db)
	if err = tx.Conn.Put([]byte(allRewardPoolsKey), blob); err != nil {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "insert reward pools list failed", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "commit changes failed", err)
	}

	m.Sorted = list.Sorted

	return &item, nil
}

func (m *rewardPools) hasEqual(item *tokenPool) bool {
	if got, found := m.get(item.Id); !found || !reflect.DeepEqual(got, item) {
		return false // not found or not equal
	}

	return true // found and equal
}

func (m *rewardPools) get(id string) (*tokenPool, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

func (m *rewardPools) getByIndex(idx int) (*tokenPool, bool) {
	if idx < len(m.Sorted) {
		return m.Sorted[idx], true
	}

	return nil, false // not found
}

func (m *rewardPools) getIndex(id string) (int, bool) {
	size := len(m.Sorted)
	if size > 0 {
		for idx, item := range m.Sorted {
			if item.Id == id {
				return idx, true // found
			}
		}
	}

	return -1, false // not found
}

func (m *rewardPools) put(item *tokenPool) (int, bool) {
	if item == nil {
		return 0, false
	}

	size := len(m.Sorted)
	if size == 0 || item.ExpiredAt.Seconds == 0 {
		m.Sorted = append(m.Sorted, item)
		return size, true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return m.Sorted[idx].ExpiredAt.Seconds >= item.ExpiredAt.Seconds
	})
	if idx == size { // out of bounds
		m.Sorted = append(m.Sorted, item)
		return idx, true // appended
	}

	left, right := m.Sorted[:idx], append([]*tokenPool{item}, m.Sorted[idx:]...) // insert
	m.Sorted = append(left, right...)

	return idx, true // inserted
}

func (m *rewardPools) write(scID string, item *tokenPool, db *gorocksdb.TransactionDB, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(zmc.ErrCodeInternal, "reward pool invalid value").Wrap(zmc.ErrNilPointerValue)
	}
	if _, err := sci.InsertTrieNode(nodeUID(scID, rewardTokenPool, item.Id), item); err != nil {
		return errors.Wrap(zmc.ErrCodeInternal, "insert reward pool failed", err)
	}

	var list *rewardPools
	if !m.hasEqual(item) { // check if an equal item already added
		list = m.copy()
		list.put(item) // add or replace
		blob, err := json.Marshal(list.Sorted)
		if err != nil {
			return errors.Wrap(zmc.ErrCodeInternal, "encode reward pools list failed", err)
		}

		tx := store.GetTransaction(db)
		if err = tx.Conn.Put([]byte(allRewardPoolsKey), blob); err != nil {
			return errors.Wrap(zmc.ErrCodeInternal, "insert reward pools list failed", err)
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

// rewardsPoolsFetch extracts all token pools stored in memory data store with given id.
func rewardPoolsFetch(id string, db *gorocksdb.TransactionDB) (*rewardPools, error) {
	pools := &rewardPools{}
	tx := store.GetTransaction(db)

	buf, err := tx.Conn.Get(tx.ReadOptions, []byte(id))
	if err != nil {
		return pools, errors.Wrap(zmc.ErrCodeInternal, "get reward pools list failed", err)
	}
	defer buf.Free()

	blob := buf.Data()
	if blob != nil {
		if err = json.Unmarshal(blob, &pools.Sorted); err != nil {
			return pools, errors.Wrap(zmc.ErrCodeInternal, "decode reward pools list failed", err)
		}
	}

	return pools, nil
}
