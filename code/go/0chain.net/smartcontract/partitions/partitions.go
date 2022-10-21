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
	rs *randomSelector
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

	if err := p.saveItemLoc(state, item.GetID(), idx); err != nil {
		return err
	}

	return nil
}

// Save saves the partitions data into state
func (p *Partitions) Save(state state.StateContextI) error {
	return p.rs.Save(state)
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

	return p.rs.GetItem(state, loc, id, v)
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

	return p.rs.UpdateItem(state, loc, item)
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

	return p.removeItemLoc(state, id)
}

// GetRandomItems returns items of partition size number from random partition,
// if the last partition is not full, it will try to get and fill it with its partition
// of index - 1.
func (p *Partitions) GetRandomItems(state state.StateContextI, r *rand.Rand, v interface{}) error {
	return p.rs.GetRandomItems(state, r, v)
}

// Foreach loads all partitions and iterate through one by one
// break whenever the callback function returns error
func (p *Partitions) Foreach(state state.StateContextI, f func(key string, data []byte) error) error {
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
	if err := state.GetTrieNode(p.getLocKey(id), &pl); err != nil {
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
