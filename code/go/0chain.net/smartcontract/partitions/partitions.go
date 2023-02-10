package partitions

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"

	"0chain.net/chaincore/chain/state"
)

const (
	errItemNotFoundCode = "item not found"
	errItemExistCode    = "item already exist"
)

const notFoundIndex = -1

//msgp:ignore Partitions idIndex
//go:generate msgp -io=false -tests=false -unexported=true -v

type Partitions struct {
	Name          string       `json:"name"`
	PartitionSize int          `json:"partition_size"`
	NumPartitions int          `json:"num_partitions"`
	Partitions    []*partition `json:"-" msg:"-"`

	toAdd     []idIndex      `json:"-" msg:"-"`
	locations map[string]int `json:"-" msg:"-"`
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
	p := Partitions{}
	err := state.GetTrieNode(name, &p)
	switch err {
	case nil:
		return &p, nil
	case util.ErrValueNotPresent:
		p, err := newPartitions(name, partitionSize)
		if err != nil {
			return nil, err
		}

		if err := p.Save(state); err != nil {
			return nil, err
		}

		return p, nil
	default:
		return nil, err
	}
}

// GetPartitions returns partitions of given name
func GetPartitions(state state.StateContextI, name string) (*Partitions, error) {
	p := Partitions{}
	if err := state.GetTrieNode(name, &p); err != nil {
		return nil, err
	}

	return &p, nil
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

func newPartitions(name string, size int) (*Partitions, error) {
	// TODO: limit the name length
	return &Partitions{
		Name:          name,
		PartitionSize: size,
	}, nil
}

func partitionKey(name string, index int) datastore.Key {
	return name + encryption.Hash(":partition:"+strconv.Itoa(index))
}

func (p *Partitions) partitionKey(index int) datastore.Key {
	return partitionKey(p.Name, index)
}

func (p *Partitions) Add(state state.StateContextI, item PartitionItem) error {
	// duplicate item checking
	_, ok, err := p.getItemPartIndex(state, item.GetID())
	if err != nil {
		return err
	}

	if ok {
		return common.NewError(errItemExistCode, item.GetID())
	}

	idx, err := p.add(state, item)
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

func (p *Partitions) add(state state.StateContextI, item PartitionItem) (int, error) {
	var (
		part     *partition
		err      error
		partsNum = p.partitionsNum()
	)

	if partsNum > 0 {
		part, err = p.getPartition(state, partsNum-1)
		if err != nil {
			logging.Logger.Debug("partition add - failed to get last partition", zap.Error(err))
			return 0, err
		}
	}

	if partsNum == 0 || part.length() >= p.PartitionSize {
		part = p.addPartition()
	}

	if err := part.add(item); err != nil {
		return 0, err
	}

	return p.partitionsNum() - 1, nil
}

func (p *Partitions) Get(state state.StateContextI, id string, v PartitionItem) error {
	loc, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, id)
	}

	if err := p.get(state, loc, id, v); err != nil {
		return err
	}

	p.loadLocations(loc)
	return nil
}

func (p *Partitions) get(state state.StateContextI, partIndex int, id string, v PartitionItem) error {
	pt, err := p.getPartition(state, partIndex)
	if err != nil {
		return err
	}

	item, _, ok := pt.find(id)
	if !ok {
		return errors.New("item not present")
	}

	_, err = v.UnmarshalMsg(item.Data)
	return err
}

func (p *Partitions) UpdateItem(state state.StateContextI, it PartitionItem) error {
	loc, ok, err := p.getItemPartIndex(state, it.GetID())
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, it.GetID())
	}

	if err := p.updateItem(state, loc, it); err != nil {
		return err
	}

	p.loadLocations(loc)
	return nil
}

func (p *Partitions) updateItem(
	state state.StateContextI,
	partIndex int,
	it PartitionItem,
) error {
	part, err := p.getPartition(state, partIndex)
	if err != nil {
		return err
	}

	return part.update(it)
}

func (p *Partitions) Update(state state.StateContextI, key string, f func(data []byte) ([]byte, error)) error {
	l, ok, err := p.getItemPartIndex(state, key)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, key)
	}

	part, err := p.getPartition(state, l)
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

func (p *Partitions) Remove(state state.StateContextI, id string) error {
	loc, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(errItemNotFoundCode, id)
	}

	if err := p.removeItem(state, id, loc); err != nil {
		return err
	}

	p.loadLocations(loc)
	delete(p.locations, p.getLocKey(id))

	return p.removeItemLoc(state, id)
}

func (p *Partitions) removeItem(
	state state.StateContextI,
	id string,
	index int,
) error {
	part, err := p.getPartition(state, index)
	if err != nil {
		return err
	}

	err = part.remove(id)
	if err != nil {
		return err
	}

	lastPart, err := p.getPartition(state, p.partitionsNum()-1)
	if err != nil {
		logging.Logger.Error("load last partition failed",
			zap.Error(err),
			zap.Int("part number", p.partitionsNum()))
		return fmt.Errorf("load last partition failed: %v", err)
	}

	if index == p.partitionsNum()-1 {
		if lastPart.length() == 0 {
			if err := p.deleteTail(state); err != nil {
				return err
			}
		}
		return nil
	}

	replace := lastPart.cutTail()
	if replace == nil {
		logging.Logger.Error("empty last partition - should not happen!!",
			zap.Int("part index", p.NumPartitions-1),
			zap.Int("part num", p.NumPartitions))

		return fmt.Errorf("empty last partitions, currpt data")
	}
	if err := part.addRaw(*replace); err != nil {
		return err
	}

	// update the location of the replaced item, it moved from the last partition to the current one
	if err := p.saveItemLoc(state, replace.ID, index); err != nil {
		return err
	}

	if lastPart.length() == 0 {
		if err := p.deleteTail(state); err != nil {
			return err
		}
	}

	return nil
}

func (p *Partitions) GetRandomItems(state state.StateContextI, r *rand.Rand, vs interface{}) error {
	if p.partitionsNum() == 0 {
		return errors.New("empty list, no items to return")
	}
	index := r.Intn(p.partitionsNum())

	part, err := p.getPartition(state, index)
	if err != nil {
		return err
	}

	its, err := part.itemRange(0, part.length())
	if err != nil {
		return err
	}

	rtv := make([]item, 0, p.PartitionSize)
	rtv = append(rtv, its...)

	if index == p.partitionsNum()-1 && len(rtv) < p.PartitionSize && p.partitionsNum() > 1 {
		secondLast, err := p.getPartition(state, index-1)
		if err != nil {
			return err
		}
		want := p.PartitionSize - len(rtv)
		if secondLast.length() < want {
			return fmt.Errorf("second last part too small %d instead of %d",
				secondLast.length(), p.partitionsNum())
		}
		its, err := secondLast.itemRange(secondLast.length()-want, secondLast.length())
		if err != nil {
			return err
		}

		rtv = append(rtv, its...)
	}

	return setPartitionItems(rtv, vs)
}

func (p *Partitions) Size(state state.StateContextI) (int, error) {
	if p.partitionsNum() == 0 {
		return 0, nil
	}

	lastPart, err := p.getPartition(state, p.partitionsNum()-1)
	if err != nil {
		return 0, err
	}

	return (p.partitionsNum()-1)*p.PartitionSize + lastPart.length(), nil
}

func (p *Partitions) Exist(state state.StateContextI, id string) (bool, error) {
	_, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (p *Partitions) Save(state state.StateContextI) error {
	for _, part := range p.Partitions {
		if part != nil && part.changed() {
			err := part.save(state)
			if err != nil {
				return err
			}
		}
	}

	_, err := state.InsertTrieNode(p.Name, p)
	if err != nil {
		return err
	}

	for _, k := range p.toAdd {
		if err := p.saveItemLoc(state, k.ID, k.Idx); err != nil {
			return err
		}
	}

	p.toAdd = p.toAdd[:0]
	return nil
}

func (p *Partitions) foreach(state state.StateContextI, f func(string, []byte, int) ([]byte, bool, error)) error {
	for i := 0; i < p.partitionsNum(); i++ {
		part, err := p.getPartition(state, i)
		if err != nil {
			return fmt.Errorf("could not get partition: name:%s, index: %d", p.Name, i)
		}

		for i, v := range part.Items {
			ret, bk, err := f(v.ID, v.Data, i)
			if err != nil {
				return err
			}
			if !bytes.Equal(ret, v.Data) {
				v.Data = ret
				part.Items[i] = v
				part.Changed = true
			}

			if bk {
				return nil
			}
		}
	}

	return nil
}

func setPartitionItems(rtv []item, vs interface{}) error {
	// slice type
	vst := reflect.TypeOf(vs)
	if vst.Kind() != reflect.Ptr {
		return errors.New("invalid return value type, it must be a pointer of slice")
	}

	// element type - slice
	vts := vst.Elem()
	if vts.Kind() != reflect.Slice {
		return errors.New("invalid return value type, it must be a pointer of slice")
	}

	// item type
	vt := vts.Elem()

	// create a new item slice
	rv := reflect.MakeSlice(vts, len(rtv), len(rtv))

	for i, v := range rtv {
		// create new item instance and assert PartitionItem interface
		pi, ok := reflect.New(vt).Interface().(PartitionItem)
		if !ok {
			return errors.New("invalid value type, the item does not meet PartitionItem interface")
		}

		// decode data
		if _, err := pi.UnmarshalMsg(v.Data); err != nil {
			return err
		}

		// set to slice
		rv.Index(i).Set(reflect.ValueOf(pi).Elem())
	}

	// set slice back to v param
	reflect.ValueOf(vs).Elem().Set(rv)
	return nil
}

func (p *Partitions) addPartition() *partition {
	newPartition := &partition{
		Key: p.partitionKey(p.partitionsNum()),
	}

	p.Partitions = append(p.Partitions, newPartition)
	p.NumPartitions++
	return newPartition
}

func (p *Partitions) deleteTail(balances state.StateContextI) error {
	_, err := balances.DeleteTrieNode(p.partitionKey(p.partitionsNum() - 1))
	if err != nil {
		logging.Logger.Debug("partition delete tail failed",
			zap.Error(err),
			zap.Int("partition num", p.partitionsNum()))
		return err
	}
	p.Partitions = p.Partitions[:p.partitionsNum()-1]
	p.NumPartitions--
	return nil
}

func (p *Partitions) getPartition(state state.StateContextI, i int) (*partition, error) {
	if i >= p.partitionsNum() {
		return nil, fmt.Errorf("partition id %v greater than number of partitions %v", i, p.partitionsNum())
	}
	if p.Partitions[i] != nil {
		return p.Partitions[i], nil
	}

	part := &partition{}
	err := part.load(state, p.partitionKey(i))
	if err != nil {
		logging.Logger.Error("partition load failed",
			zap.Error(err),
			zap.Int("index", i),
			zap.Int("partition num", p.partitionsNum()),
			zap.String("partition key", p.partitionKey(i)))
		return nil, err
	}
	p.Partitions[i] = part
	return part, nil
}

func (p *Partitions) partitionsNum() int {
	// assert the partitions number match
	if p.NumPartitions != len(p.Partitions) {
		logging.Logger.DPanic(fmt.Sprintf("number of partitions mismatch, numPartitions: %d, len(partitions): %d",
			p.NumPartitions, len(p.Partitions)))
	}
	return p.NumPartitions
}

func (p *Partitions) MarshalMsg(o []byte) ([]byte, error) {
	d := partitionsDecode(*p)
	return d.MarshalMsg(o)
}

func (p *Partitions) UnmarshalMsg(b []byte) ([]byte, error) {
	d := &partitionsDecode{}
	o, err := d.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	*p = Partitions(*d)

	p.Partitions = make([]*partition, d.NumPartitions)
	return o, nil
}

func (p *Partitions) Msgsize() int {
	d := partitionsDecode(*p)
	return d.Msgsize()
}

type partitionsDecode Partitions
