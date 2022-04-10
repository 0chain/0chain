package partitions

import (
	"errors"
	"fmt"
	"runtime/debug"

	"0chain.net/core/common"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

// item represent the partition item
type item struct {
	ID   string
	Data []byte
}

type partition struct {
	Key     string `json:"-" msg:"-"`
	Items   []item `json:"items"`
	Changed bool   `json:"-" msg:"-"`
}

func (il *partition) save(state state.StateContextI) error {
	_, err := state.InsertTrieNode(il.Key, il)
	return err
}

func (il *partition) load(state state.StateContextI, key datastore.Key) error {
	err := state.GetTrieNode(key, il)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il.Key = key
	}

	return nil
}

func (il *partition) add(it PartitionItem) error {
	for _, bi := range il.Items {
		if bi.ID == it.GetID() {
			return errors.New("item already exists")
		}
	}

	v, err := it.MarshalMsg(nil)
	if err != nil {
		return err
	}
	il.Items = append(il.Items, item{ID: it.GetID(), Data: v})
	il.Changed = true
	return nil
}

func (il *partition) addRaw(it item) error {
	for _, v := range il.Items {
		if v.ID == it.ID {
			return errors.New("item already exists")
		}
	}

	il.Items = append(il.Items, it)
	il.Changed = true
	return nil
}

func (il *partition) update(it PartitionItem) error {
	for i := 0; i < il.length(); i++ {
		if il.Items[i].ID == it.GetID() {
			v, err := it.MarshalMsg(nil)
			if err != nil {
				return err
			}

			il.Items[i] = item{ID: it.GetID(), Data: v}
			il.Changed = true
			return nil
		}
	}
	return errors.New("item not found")
}

func (il *partition) remove(item PartitionItem) error {
	if len(il.Items) == 0 {
		return fmt.Errorf("searching empty partition")
	}
	index := il.findIndex(item)
	if index == notFound {
		return fmt.Errorf("cannot findIndex item %v in partition", item)
	}
	il.Items[index] = il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return nil
}

func (il *partition) cutTail() *item {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *partition) length() int {
	return len(il.Items)
}

func (il *partition) changed() bool {
	return il.Changed
}

func (il *partition) itemRange(start, end int) ([]item, error) {
	if start > end || end > len(il.Items) {
		debug.PrintStack()
		return nil, fmt.Errorf("invalid index, start:%v, end:%v, len:%v", start, end, len(il.Items))
	}

	return il.Items[start:end], nil
}

func (il *partition) find(id string) (item, bool) {
	for _, v := range il.Items {
		if v.ID == id {
			return v, true
		}
	}

	return item{}, false
}

func (il *partition) findIndex(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.ID == searchItem.GetID() {
			return i
		}
	}
	return notFound
}

//go:generate msgp -io=false -tests=false -unexported=true -v

type PartitionLocation struct {
	Location  int
	Timestamp common.Timestamp
}

func NewPartitionLocation(location int, timestamp common.Timestamp) *PartitionLocation {
	pl := new(PartitionLocation)
	pl.Location = location
	pl.Timestamp = timestamp

	return pl
}
