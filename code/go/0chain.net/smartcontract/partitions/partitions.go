package partitions

import (
	"math/rand"

	"0chain.net/core/util"

	"0chain.net/chaincore/chain/state"
)

// APIs

type Partitions struct {
	rp *randomSelector
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
		return &Partitions{rp: &rs}, nil
	case util.ErrValueNotPresent:
		rs, err := newRandomSelector(name, partitionSize, nil)
		if err != nil {
			return nil, err
		}

		pt := &Partitions{rp: rs}
		if err := pt.rp.Save(state); err != nil {
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

	return &Partitions{rp: &rs}, nil
}

// AddItem adds a partition item to parititons
func (p *Partitions) AddItem(state state.StateContextI, item PartitionItem) (int, error) {
	return p.rp.Add(state, item)
}

// Save saves the partitions data into state
func (p *Partitions) Save(state state.StateContextI) error {
	return p.rp.Save(state)
}

// GetItem returns partition item of given partition index and id
func (p *Partitions) GetItem(state state.StateContextI, partIndex int, id string, v PartitionItem) error {
	return p.rp.GetItem(state, partIndex, id, v)
}

// UpdateItem updates item on given partition index
func (p *Partitions) UpdateItem(state state.StateContextI, partIndex int, item PartitionItem) error {
	return p.rp.UpdateItem(state, partIndex, item)
}

// Size returns the total item number in partitions
func (p *Partitions) Size(state state.StateContextI) (int, error) {
	return p.rp.Size(state)
}

// GetRandomItems returns items of partition size number from random partition,
// if the last partition is not full, it will try to get and fill it with its partition
// of index - 1.
func (p *Partitions) GetRandomItems(state state.StateContextI, r *rand.Rand, v interface{}) error {
	return p.rp.GetRandomItems(state, r, v)
}

type ChangePartitionCallback = func(string, []byte, int, int, state.StateContextI) error

// Partitioner interface wraps the methods for processing a partition
//type Partitioner interface {
//	util.MPTSerializable
//	Add(state.StateContextI, PartitionItem) (int, error)
//	Remove(state.StateContextI, PartitionItem, int) error
//
//	SetCallback(ChangePartitionCallback)
//	Size(state.StateContextI) (int, error)
//	Save(state.StateContextI) error
//	UpdateItem(state state.StateContextI, partIndex int, it PartitionItem) error
//	GetItem(state state.StateContextI, partIndex int, id string, v PartitionItem) error
//}

// randPartitioner wraps partitioner interface and methods for
// adding and getting items randomly
//type randPartitioner interface {
//	Partitioner
//	AddRand(state.StateContextI, PartitionItem, *rand.Rand) (int, error)
//	GetRandomItems(state.StateContextI, *rand.Rand, interface{}) error
//}
