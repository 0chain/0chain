package partitions

import (
	"fmt"
	"math/rand"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"

	"0chain.net/chaincore/chain/state"
)

const (
	errItemNotFoundCode = "item not found"
	errItemExistCode    = "item already exist"
)

// APIs

type Partitions struct {
	rs        *randomSelector
	toRemove  []string
	toAdd     []idIndex
	locations map[string]int
}

type idIndex struct {
	ID  string
	Idx int
}

type PartitionItem interface {
	util.MPTSerializableSize
	GetID() string
}

// CreateIfNotExists creates a partition if not exist
// It's a common patten to call this to create partitions on
// top-level partitions when project start
func CreateIfNotExists(state state.StateContextI, name string, partitionSize int) (*Partitions, error) {
	rs := randomSelector{}
	err := state.GetTrieNode(name, &rs)
	switch err {
	case nil:
		return &Partitions{rs: &rs}, nil
	case util.ErrValueNotPresent:
		rs, err := newRandomSelector(name, partitionSize, nil)
		if err != nil {
			return nil, err
		}

		pt := &Partitions{rs: rs}
		if err := pt.rs.Save(state); err != nil {
			return nil, err
		}

		return pt, nil
	default:
		return nil, err
	}
}

// GetPartitions returns partitions of given name
func GetPartitions(state state.StateContextI, name string) (*Partitions, error) {
	rs := randomSelector{}
	if err := state.GetTrieNode(name, &rs); err != nil {
		return nil, err
	}

	return &Partitions{rs: &rs}, nil
}

// ErrItemNotFound checks if error is common.Error and code is 'item not found'
func ErrItemNotFound(err error) bool {
	cErr, ok := err.(*common.Error)
	if !ok {
		return false
	}

	return cErr.Code == errItemNotFoundCode
}

// ErrItemExist checks if error is common.Error and code is 'item already exist'
func ErrItemExist(err error) bool {
	cErr, ok := err.(*common.Error)
	if !ok {
		return false
	}

	return cErr.Code == errItemExistCode
}

// GetName returns the partitions name
func (p *Partitions) GetName() string {
	return p.rs.Name
}

// AddItem adds a partition item to parititons
func (p *Partitions) AddItem(state state.StateContextI, item PartitionItem) error {
	// duplicate item checking
	_, ok, err := p.getItemPartIndex(state, item.GetID())
	if err != nil {
		return err
	}

	if ok {
		return common.NewError(errItemExistCode, item.GetID())
	}

	idx, err := p.rs.Add(state, item)
	if err != nil {
		return err
	}
	p.toAdd = append(p.toAdd, idIndex{
		ID:  item.GetID(),
		Idx: idx,
	})

	p.loadLocations(idx)
	return nil
}

func (p *Partitions) loadLocations(idx int) {
	if p.locations == nil {
		p.locations = make(map[string]int)
	}
	if idx < 0 {
		return
	}

	part := p.rs.Partitions[idx]
	for _, it := range part.Items {
		kid := p.getLocKey(it.ID)
		if _, ok := p.locations[kid]; ok {
			return
		}

		p.locations[kid] = idx
	}
}

// Save saves the partitions data into state
func (p *Partitions) Save(state state.StateContextI) error {
	if err := p.rs.Save(state); err != nil {
		return err
	}

	for _, k := range p.toRemove {
		if err := p.removeItemLoc(state, k); err != nil {
			return err
		}
	}

	p.toRemove = p.toRemove[:0]

	for _, k := range p.toAdd {
		if err := p.saveItemLoc(state, k.ID, k.Idx); err != nil {
			return err
		}
	}

	p.toAdd = p.toAdd[:0]
	return nil
}

// GetItem returns partition item of given partition index and id
func (p *Partitions) GetItem(state state.StateContextI, id string, v PartitionItem) error {
	loc, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, id)
	}

	if err := p.rs.GetItem(state, loc, id, v); err != nil {
		return err
	}

	p.loadLocations(loc)

	return nil
}

// UpdateItem updates item on given partition index
func (p *Partitions) UpdateItem(state state.StateContextI, item PartitionItem) error {
	loc, ok, err := p.getItemPartIndex(state, item.GetID())
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, item.GetID())
	}

	if err := p.rs.UpdateItem(state, loc, item); err != nil {
		return err
	}

	p.loadLocations(loc)
	return nil
}

// Size returns the total item number in partitions
func (p *Partitions) Size(state state.StateContextI) (int, error) {
	return p.rs.Size(state)
}

// Num returns the number of partitions
func (p *Partitions) Num() int {
	return p.rs.NumPartitions
}

// RemoveItem removes the partition item from given partIndex in Partitions
func (p *Partitions) RemoveItem(state state.StateContextI, id string) error {
	loc, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, id)
	}

	if err := p.rs.RemoveItem(state, id, loc); err != nil {
		return err
	}

	p.toRemove = append(p.toRemove, id)

	p.loadLocations(loc)
	delete(p.locations, p.getLocKey(id))
	return nil
}

// RemoveItems removes items by keys from the same partition of partIndex
func (p *Partitions) RemoveItems(state state.StateContextI, partIndex int, keys []string) error {
	for _, k := range keys {
		if err := p.rs.RemoveItem(state, k, partIndex); err != nil {
			return err
		}
		p.toRemove = append(p.toRemove, k)
		delete(p.locations, p.getLocKey(k))
	}

	return nil
}

// GetRandomItems returns items of partition size number from random partition,
// if the last partition is not full, it will try to get and fill it with its partition
// of index - 1.
func (p *Partitions) GetRandomItems(state state.StateContextI, r *rand.Rand, v interface{}) error {
	return p.rs.GetRandomItems(state, r, v)
}

type RandMatchFunc func(key string, data []byte) bool

func (p *Partitions) UpdateRandomItems(state state.StateContextI, r *rand.Rand, randN int, f func(string, []byte) ([]byte, error)) error {
	return p.rs.UpdateRandomItems(state, r, randN, f)
}

// Foreach loads all partitions and iterate through one by one
// break whenever the callback function returns error
// for the callback function, if it returns bytes different from the input data, then the
// changes of items will be saved to partitions.
func (p *Partitions) Foreach(state state.StateContextI, f func(key string, data []byte, partIndex int) ([]byte, bool, error)) error {
	return p.rs.foreach(state, f)
}

func (p *Partitions) Exist(state state.StateContextI, id string) (bool, error) {
	_, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return false, err
	}

	return ok, nil
}

type ChangePartitionCallback = func(string, []byte, int, int, state.StateContextI) error

func (p *Partitions) getLocKey(id string) datastore.Key {
	return encryption.Hash(fmt.Sprintf("%s:%s", p.rs.Name, id))
}

func (p *Partitions) getItemPartIndex(state state.StateContextI, id string) (int, bool, error) {
	var pl location

	kid := p.getLocKey(id)
	loc, ok := p.locations[kid]
	if ok {
		return loc, true, nil
	}

	if err := state.GetTrieNode(kid, &pl); err != nil {
		if err == util.ErrValueNotPresent {
			return -1, false, nil
		}

		return -1, false, err
	}

	return pl.Location, true, nil
}

func (p *Partitions) saveItemLoc(state state.StateContextI, id string, partIndex int) error {
	_, err := state.InsertTrieNode(p.getLocKey(id), &location{Location: partIndex})
	return err
}

func (p *Partitions) removeItemLoc(state state.StateContextI, id string) error {
	_, err := state.DeleteTrieNode(p.getLocKey(id))
	return err
}

func (p *Partitions) Update(state state.StateContextI, key string, f func(data []byte) ([]byte, error)) error {
	l, ok, err := p.getItemPartIndex(state, key)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, key)
	}

	part, err := p.rs.getPartition(state, l)
	if err != nil {
		return err
	}

	v, idx, ok := part.find(key)
	if !ok {
		return common.NewError(errItemNotFoundCode, key)
	}

	nData, err := f(v.Data)
	if err != nil {
		return err
	}
	v.Data = nData
	part.Items[idx] = v
	part.Changed = true

	p.loadLocations(l)
	return nil
}
