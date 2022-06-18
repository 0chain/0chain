package partitions

import (
	"errors"
	"fmt"

	"0chain.net/core/common"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

var (
	ErrPartitionItemAlreadyExist = errors.New("item already exists")
)

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

func (p *partition) save(state state.StateContextI) error {
	_, err := state.InsertTrieNode(p.Key, p)
	return err
}

func (p *partition) load(state state.StateContextI, key datastore.Key) error {
	err := state.GetTrieNode(key, p)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
	}

	p.Key = key
	return nil
}

func (p *partition) add(it PartitionItem) error {
	for _, bi := range p.Items {
		if bi.ID == it.GetID() {
			return ErrPartitionItemAlreadyExist
		}
	}

	v, err := it.MarshalMsg(nil)
	if err != nil {
		return err
	}
	p.Items = append(p.Items, item{ID: it.GetID(), Data: v})
	p.Changed = true
	return nil
}

func (p *partition) addRaw(it item) error {
	for _, v := range p.Items {
		if v.ID == it.ID {
			return errors.New("item already exists")
		}
	}

	p.Items = append(p.Items, it)
	p.Changed = true
	return nil
}

func (p *partition) update(it PartitionItem) error {
	for i := 0; i < p.length(); i++ {
		if p.Items[i].ID == it.GetID() {
			v, err := it.MarshalMsg(nil)
			if err != nil {
				return err
			}

			p.Items[i] = item{ID: it.GetID(), Data: v}
			p.Changed = true
			return nil
		}
	}
	return errors.New("item not found")
}

func (p *partition) remove(id string) error {
	if len(p.Items) == 0 {
		return fmt.Errorf("searching empty partition")
	}
	index := p.findIndex(id)
	if index == notFound {
		return fmt.Errorf("cannot findIndex id %v in partition", id)
	}
	p.Items[index] = p.Items[len(p.Items)-1]
	p.Items = p.Items[:len(p.Items)-1]
	p.Changed = true
	return nil
}

func (p *partition) cutTail() *item {
	if len(p.Items) == 0 {
		return nil
	}

	tail := p.Items[len(p.Items)-1]
	p.Items = p.Items[:len(p.Items)-1]
	p.Changed = true
	return &tail
}

func (p *partition) length() int {
	return len(p.Items)
}

func (p *partition) changed() bool {
	return p.Changed
}

func (p *partition) itemRange(start, end int) ([]item, error) {
	if start > end || end > len(p.Items) {
		return nil, fmt.Errorf("invalid index, start:%v, end:%v, len:%v", start, end, len(p.Items))
	}

	return p.Items[start:end], nil
}

func (p *partition) find(id string) (item, bool) {
	for _, v := range p.Items {
		if v.ID == id {
			return v, true
		}
	}

	return item{}, false
}

func (p *partition) findIndex(id string) int {
	for i, item := range p.Items {
		if item.ID == id {
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
