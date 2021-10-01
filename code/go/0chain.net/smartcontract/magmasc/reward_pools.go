package magmasc

import (
	"encoding/json"

	"github.com/0chain/gosdk/zmagmacore/errors"
	"github.com/0chain/gosdk/zmagmacore/time"

	chain "0chain.net/chaincore/chain/state"
	store "0chain.net/core/ememorystore"
)

type (
	// rewardPools a list of token pool implementation.
	// Contents a list of token pools by mapped keys:
	// 	TokenPool.PayeeID -> TokenPool.ID -> tokenPool
	rewardPools struct {
		List map[string]map[string]*tokenPool
	}
)

func (m *rewardPools) add(scID string, item *tokenPool, db *store.Connection, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "token pool invalid value").Wrap(errNilPointerValue)
	}
	if got, _ := sci.GetTrieNode(nodeUID(scID, rewardTokenPool, item.ID)); got != nil {
		return errors.New(errCodeInternal, "token pool already registered")
	}

	return m.write(scID, item, db, sci)
}

func (m *rewardPools) copy() *rewardPools {
	pools := rewardPools{List: make(map[string]map[string]*tokenPool)}
	for pid, items := range m.List {
		for id, item := range items {
			if pools.List[pid] == nil {
				pools.List[pid] = make(map[string]*tokenPool)
			}
			pools.List[pid][id] = item
		}
	}

	return &pools
}

func (m *rewardPools) del(payeeID, poolID string, db *store.Connection) (*tokenPool, error) {
	var pools *rewardPools

	pool, found := m.List[payeeID][poolID]
	if found {
		if pool.ExpireAt > time.Now() {
			return nil, errors.New(errCodeInternal, "token pool is not expired yet")
		}

		pools = m.copy()
		delete(pools.List[payeeID], poolID)

		blob, err := json.Marshal(pools.List)
		if err != nil {
			return nil, errors.Wrap(errCodeInternal, "encode pools list failed", err)
		}
		if err = db.Conn.Put([]byte(allRewardPoolsKey), blob); err != nil {
			return nil, errors.Wrap(errCodeInternal, "insert pools list failed", err)
		}
		if err = db.Commit(); err != nil {
			return nil, errors.Wrap(errCodeInternal, "commit changes failed", err)
		}
	}
	if pools != nil {
		m.List = pools.List
	}

	return pool, nil
}

//nolint:unused
func (m *rewardPools) get(pid, id string) (*tokenPool, bool) {
	pool, found := m.List[pid][id]
	return pool, found
}

func (m *rewardPools) put(item *tokenPool) {
	if m.List == nil {
		m.List = make(map[string]map[string]*tokenPool)
	}
	if m.List[item.PayeeID] == nil {
		m.List[item.PayeeID] = make(map[string]*tokenPool)
	}

	m.List[item.PayeeID][item.ID] = item
}

func (m *rewardPools) write(scID string, item *tokenPool, db *store.Connection, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "token pool invalid value").Wrap(errNilPointerValue)
	}

	var pools *rewardPools
	if _, found := m.List[item.PayeeID][item.ID]; !found { // check if item already added
		pools = m.copy()
		pools.put(item)

		blob, err := json.Marshal(pools.List)
		if err != nil {
			return errors.Wrap(errCodeInternal, "encode pools list failed", err)
		}
		if err = db.Conn.Put([]byte(allRewardPoolsKey), blob); err != nil {
			return errors.Wrap(errCodeInternal, "insert pools list failed", err)
		}
	}
	if _, err := sci.InsertTrieNode(nodeUID(scID, rewardTokenPool, item.ID), item); err != nil {
		return errors.Wrap(errCodeInternal, "insert token pool failed", err)
	}
	if err := db.Commit(); err != nil {
		return errors.Wrap(errCodeInternal, "commit changes failed", err)
	}
	if pools != nil {
		m.List = pools.List
	}

	return nil
}

// rewardPoolsFetch extracts all token pools stored in memory data store with given id.
func rewardPoolsFetch(id string, db *store.Connection) (*rewardPools, error) {
	pools := &rewardPools{List: make(map[string]map[string]*tokenPool)}

	buf, err := db.Conn.Get(db.ReadOptions, []byte(id))
	if err != nil {
		return pools, errors.Wrap(errCodeInternal, "get token pools list failed", err)
	}
	defer buf.Free()

	blob := buf.Data()
	if blob != nil {
		if err = json.Unmarshal(blob, &pools.List); err != nil {
			return pools, errors.Wrap(errCodeInternal, "decode token pools list failed", err)
		}
	}

	return pools, nil
}
