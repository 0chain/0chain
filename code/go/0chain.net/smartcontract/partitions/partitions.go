package partitions

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/sortedmap"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"

	"0chain.net/chaincore/chain/state"
)

const (
	ErrItemNotFoundCode = "item not found"
	errItemExistCode    = "item already exist"
)

const notFoundIndex = -1

//msgp:ignore Partitions idIndex
//go:generate msgp -io=false -tests=false -unexported=true -v

type Partitions struct {
	Name          string     `json:"name"`
	PartitionSize int        `json:"partition_size"`
	Last          *partition `json:"last"`

	Partitions map[int]*partition `json:"-" msg:"-"`
	locations  map[string]int     `msg:"-"`
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

	return cErr.Code == ErrItemNotFoundCode
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
		Last: &partition{
			Key: partitionKey(name, 0), // partition index starts from 1
		},
		Partitions: map[int]*partition{},
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

	if err := p.add(state, item); err != nil {
		return err
	}

	return nil
}

func (p *Partitions) add(state state.StateContextI, item PartitionItem) error {
	_, _, ok := p.Last.find(item.GetID())
	if ok {
		return common.NewError(errItemExistCode, item.GetID())
	}

	// check if Last is full
	if p.Last.length() == p.PartitionSize {
		if err := p.pack(state); err != nil {
			return fmt.Errorf("could not pack partition: %v", err)
		}
	}

	if err := p.Last.add(item); err != nil {
		return fmt.Errorf("could not save item to partition: %v", err)
	}

	return nil
}

func (p *Partitions) pack(state state.StateContextI) error {
	// separate the Last partition from the partitions and create a new empty partition
	if err := p.Last.save(state); err != nil {
		return err
	}

	// save item locations
	loc := p.Last.Loc
	for _, it := range p.Last.Items {
		if err := p.saveItemLoc(state, it.ID, loc); err != nil {
			return err
		}
	}
	p.Partitions[p.Last.Loc] = p.Last
	loc++

	//p.PrevLoc++
	p.Last = &partition{
		Key: partitionKey(p.Name, loc),
		Loc: loc,
	}
	return nil
}

func (p *Partitions) Get(state state.StateContextI, id string, v PartitionItem) error {
	it, _, ok := p.Last.find(id)
	if ok {
		_, err := v.UnmarshalMsg(it.Data)
		return err
	}

	loc, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(ErrItemNotFoundCode, id)
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
	_, _, ok := p.Last.find(it.GetID())
	if ok {
		return p.Last.update(it)
	}

	loc, ok, err := p.getItemPartIndex(state, it.GetID())
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(ErrItemNotFoundCode, it.GetID())
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
	v, idx, ok := p.Last.find(key)
	if ok {
		nData, err := f(v.Data)
		if err != nil {
			return err
		}

		v.Data = nData
		p.Last.Items[idx] = v
		return nil
	}

	l, ok, err := p.getItemPartIndex(state, key)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(ErrItemNotFoundCode, key)
	}

	part, err := p.getPartition(state, l)
	if err != nil {
		return err
	}

	v, idx, ok = part.find(key)
	if !ok {
		return common.NewError(ErrItemNotFoundCode, key)
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
	_, idx, ok := p.Last.find(id)
	if ok {
		return p.removeFromLast(state, idx)
	}

	loc, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return err
	}

	if !ok {
		return common.NewError(ErrItemNotFoundCode, id)
	}

	if err := p.removeItem(state, id, loc); err != nil {
		return err
	}

	p.loadLocations(loc)
	return p.removeItemLoc(state, id)
}

func (p *Partitions) removeFromLast(state state.StateContextI, idx int) error {
	p.Last.Items[idx] = p.Last.Items[len(p.Last.Items)-1]
	p.Last.Items = p.Last.Items[:len(p.Last.Items)-1]
	if p.Last.length() > 0 {
		return nil
	}

	// last is empty, reload from previous partition
	return p.loadLastFromPrev(state)
}

func (p *Partitions) loadLastFromPrev(state state.StateContextI) error {
	// load previous partition
	if p.Last.Loc == 0 {
		// no previous partition
		return nil
	}

	b := state.GetBlock()
	prev, err := p.getPartition(state, p.Last.Loc-1)
	if err != nil {
		logging.Logger.Error("could not get previous partition",
			zap.String("name", p.Name),
			zap.Int("loc", p.Last.Loc-1),
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.String("state root", util.ToHex(state.GetState().GetRoot())),
			zap.String("txn", state.GetTransaction().Hash),
			zap.Error(err))
		return fmt.Errorf("could not get previous partition: %d, %v", p.Last.Loc-1, err)
	}
	p.Last = prev

	// remove all prev locations
	for _, it := range prev.Items {
		if err := p.removeItemLoc(state, it.ID); err != nil {
			return err
		}
	}

	// delete prev
	_, err = state.DeleteTrieNode(prev.Key)
	if err != nil {
		return fmt.Errorf("could not remove prev partition: %s, err: %v", prev.Key, err)
	}

	delete(p.Partitions, p.Last.Loc)
	return nil
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

	if index == p.Last.Loc {
		return nil
	}

	replace := p.Last.cutTail()
	if replace == nil {
		logging.Logger.Error("empty last partition - should not happen!!",
			zap.Int("last loc", p.Last.Loc))

		return fmt.Errorf("empty last partitions, currpt data")
	}

	if err := part.addRaw(*replace); err != nil {
		return err
	}

	// update the location of the replaced item, it moved from the last partition to the current one
	if err := p.saveItemLoc(state, replace.ID, index); err != nil {
		return err
	}

	if p.Last.length() > 0 {
		return nil
	}

	// p.Last is empty, load items from previous partition
	return p.loadLastFromPrev(state)
}

func (p *Partitions) GetRandomItems(state state.StateContextI, r *rand.Rand, vs interface{}, options ...string) error {
	logging.Logger.Debug("jayash GetRandomItems",
		zap.Any("options", options),
		zap.Any("p", p))

	if p.Last.length() == 0 {
		return errors.New("empty list, no items to return")
	}

	var index int
	if p.Last.Loc > 0 {
		index = r.Intn(p.Last.Loc + 1)
	}
	part, err := p.getPartition(state, index)
	if err != nil {
		return err
	}

	its, err := part.itemRange(0, part.length())
	if err != nil {
		return err
	}

	return setPartitionItems(its, vs)
}

func (p *Partitions) Size(state state.StateContextI) (int, error) {
	if p.Last.length() == 0 {
		return 0, nil
	}

	return p.Last.Loc*p.PartitionSize + p.Last.length(), nil
}

func (p *Partitions) Exist(state state.StateContextI, id string) (bool, error) {
	_, _, ok := p.Last.find(id)
	if ok {
		return true, nil
	}

	_, ok, err := p.getItemPartIndex(state, id)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (p *Partitions) Save(state state.StateContextI) error {
	keys := sortedmap.NewFromMap(p.Partitions).GetKeys()
	for _, k := range keys {
		part := p.Partitions[k]
		if part.changed() {
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
	logging.Logger.Debug("save partitions",
		zap.String("name", p.Name),
		zap.Int("loc", p.Last.Loc),
		zap.Int("items", len(p.Last.Items)))
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

func (p *Partitions) getPartition(state state.StateContextI, i int) (*partition, error) {
	if i > p.Last.Loc {
		return nil, fmt.Errorf("partition id %d overflow %d", i, p.Last.Loc)
	}

	if i == p.Last.Loc {
		return p.Last, nil
	}

	part, ok := p.Partitions[i]
	if ok {
		return part, nil
	}

	part = &partition{}
	err := part.load(state, p.partitionKey(i))
	if err != nil {
		logging.Logger.Error("partition load failed",
			zap.Error(err),
			zap.Int("index", i),
			zap.Int("last loc", p.Last.Loc),
			zap.String("partition key", p.partitionKey(i)))
		return nil, err
	}
	p.Partitions[i] = part
	return part, nil
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

	p.Last.Key = partitionKey(p.Name, d.Last.Loc)
	p.Partitions = make(map[int]*partition)
	return o, nil
}

func (p *Partitions) Msgsize() int {
	d := partitionsDecode(*p)
	return d.Msgsize()
}

type partitionsDecode Partitions
