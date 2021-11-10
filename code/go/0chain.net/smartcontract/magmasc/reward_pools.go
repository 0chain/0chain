package magmasc

import (
	"encoding/json"
	"sort"

	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/util"
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

var (
	// Ensure rewardPools implements interface.
	_ util.Serializable = (*rewardPools)(nil)
)

// Encode implements util.Serializable.
func (m *rewardPools) Encode() []byte {
	blob, _ := json.Marshal(m.Sorted)
	return blob
}

// Decode implements util.Serializable.
func (m *rewardPools) Decode(blob []byte) error {
	return json.Unmarshal(blob, &m.Sorted)
}

func (m *rewardPools) del(item *tokenPool) (*tokenPool, error) {
	if idx, found := m.getIndex(item.Id); found {
		return m.delByIndex(idx)
	}

	return nil, errors.New(zmc.ErrCodeInternal, "value not present")
}

func (m *rewardPools) delByIndex(idx int) (*tokenPool, error) {
	if idx >= len(m.Sorted) {
		return nil, errors.New(zmc.ErrCodeInternal, "index out of range")
	}

	item := *m.Sorted[idx] // get copy of item
	m.Sorted = append(m.Sorted[:idx], m.Sorted[idx+1:]...)

	return &item, nil
}

func (m *rewardPools) get(id string) (*tokenPool, bool) {
	idx, found := m.getIndex(id)
	if !found {
		return nil, false // not found
	}

	return m.Sorted[idx], true // found
}

func (m *rewardPools) getIndex(id string) (int, bool) {
	for idx, item := range m.Sorted {
		if item.Id == id {
			return idx, true // found
		}
	}

	return -1, false // not found
}

func (m *rewardPools) put(item *tokenPool) (int, bool) {
	if item == nil {
		return 0, false
	}

	size := len(m.Sorted)
	if size == 0 || item.GetExpiredAt().GetSeconds() == 0 {
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

// rewardsPoolsFetch extracts all token pools stored in memory data store with given id.
func rewardPoolsFetch(sci chain.StateContextI) (*rewardPools, error) {
	pools := &rewardPools{}

	blob, err := sci.GetTrieNode(allRewardPoolsKey)
	if err != nil && !errors.Is(err, util.ErrValueNotPresent) {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "fetch list failed", err)
	} else if errors.Is(err, util.ErrValueNotPresent) {
		return pools, nil
	}

	if err := pools.Decode(blob.Encode()); err != nil {
		return nil, errors.Wrap(zmc.ErrCodeInternal, "decode list failed", err)
	}
	return pools, nil
}
