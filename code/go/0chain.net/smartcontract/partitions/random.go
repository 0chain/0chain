package partitions

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"

	"0chain.net/core/util"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"

	"0chain.net/chaincore/chain/state"
)

const notFound = -1

//msgp:ignore randomSelector
//go:generate msgp -io=false -tests=false -unexported=true -v

type randomSelector struct {
	Name          string                  `json:"name"`
	PartitionSize int                     `json:"partition_size"`
	NumPartitions int                     `json:"num_partitions"`
	Partitions    []*partition            `json:"-" msg:"-"`
	Callback      ChangePartitionCallback `json:"-" msg:"-"`
}

func newRandomSelector(
	name string,
	size int,
	callback ChangePartitionCallback,
) (*randomSelector, error) {
	// TODO: limit the name length
	return &randomSelector{
		Name:          name,
		PartitionSize: size,
		Callback:      callback,
	}, nil
}

func PartitionKey(name string, index int) datastore.Key {
	return name + encryption.Hash(":partition:"+strconv.Itoa(index))
}

func (rs *randomSelector) partitionKey(index int) datastore.Key {
	return PartitionKey(rs.Name, index)
}

func (rs *randomSelector) SetCallback(callback ChangePartitionCallback) {
	rs.Callback = callback
}

func (rs *randomSelector) Add(state state.StateContextI, item PartitionItem) (int, error) {
	var part *partition
	var err error
	if len(rs.Partitions) > 0 {
		part, err = rs.getPartition(state, len(rs.Partitions)-1)
		if err != nil {
			return 0, err
		}
	}
	if len(rs.Partitions) == 0 || part.length() >= rs.PartitionSize {
		part = rs.addPartition()
	}
	if err := part.add(item); err != nil {
		return rs.NumPartitions - 1, err
	}
	return len(rs.Partitions) - 1, nil
}

func (rs *randomSelector) addRawItem(state state.StateContextI, item item) (int, error) {
	var part *partition
	var err error
	if len(rs.Partitions) > 0 {
		part, err = rs.getPartition(state, len(rs.Partitions)-1)
		if err != nil {
			return 0, err
		}
	}
	if len(rs.Partitions) == 0 || part.length() >= rs.PartitionSize {
		part = rs.addPartition()
	}
	if err := part.addRaw(item); err != nil {
		return 0, err
	}
	return len(rs.Partitions) - 1, nil
}

func (rs *randomSelector) RemoveItem(
	state state.StateContextI,
	id string,
	index int,
) error {
	part, err := rs.getPartition(state, index)
	if err != nil {
		return err
	}

	err = part.remove(id)
	if err != nil {
		return err
	}

	lastPart, err := rs.getPartition(state, len(rs.Partitions)-1)
	if err != nil {
		return err
	}

	if index == rs.NumPartitions-1 {
		if lastPart.length() == 0 {
			if err := rs.deleteTail(state); err != nil {
				return err
			}
		}
		return nil
	}

	replace := lastPart.cutTail()
	if replace == nil {
		return fmt.Errorf("empty last partitions, currpt data")
	}
	if err := part.addRaw(*replace); err != nil {
		return err
	}
	if rs.Callback != nil {
		err = rs.Callback(replace.ID, replace.Data, len(rs.Partitions)-1, index, state)
		if err != nil {
			return err
		}
	}

	if lastPart.length() == 0 {
		if err := rs.deleteTail(state); err != nil {
			return err
		}
	}

	return nil
}

func (rs *randomSelector) AddRand(
	state state.StateContextI,
	item PartitionItem,
	r *rand.Rand,
) (int, error) {
	if rs.NumPartitions == 0 {
		return rs.Add(state, item)
	}
	index := r.Intn(rs.NumPartitions)
	if index == rs.NumPartitions-1 {
		return rs.Add(state, item)
	}

	partition, err := rs.getPartition(state, index)
	if err != nil {
		return -1, err
	}
	moving := partition.cutTail()
	if moving == nil {
		return -1, fmt.Errorf("empty partitions, corrupt data")
	}
	if err := partition.add(item); err != nil {
		return 0, err
	}

	movedTo, err := rs.addRawItem(state, *moving)
	if err != nil {
		return -1, err
	}
	if rs.Callback != nil {
		err = rs.Callback(moving.ID, moving.Data, index, movedTo, state)
		if err != nil {
			return -1, err
		}
	}

	return index, nil
}

func (rs *randomSelector) GetRandomItems(state state.StateContextI, r *rand.Rand, vs interface{}) error {
	if rs.NumPartitions == 0 {
		return errors.New("empty list, no items to return")
	}
	index := r.Intn(rs.NumPartitions)

	part, err := rs.getPartition(state, index)
	if err != nil {
		return err
	}

	its, err := part.itemRange(0, part.length())
	if err != nil {
		return err
	}

	rtv := make([]item, 0, rs.PartitionSize)
	rtv = append(rtv, its...)

	if index == rs.NumPartitions-1 && len(rtv) < rs.PartitionSize && rs.NumPartitions > 1 {
		secondLast, err := rs.getPartition(state, index-1)
		if err != nil {
			return err
		}
		want := rs.PartitionSize - len(rtv)
		if secondLast.length() < want {
			return fmt.Errorf("second last part too small %d instead of %d",
				secondLast.length(), rs.NumPartitions)
		}
		its, err := secondLast.itemRange(secondLast.length()-want, secondLast.length())
		if err != nil {
			return err
		}

		rtv = append(rtv, its...)
	}

	return setPartitionItems(rtv, vs)
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

func (rs *randomSelector) addPartition() *partition {
	newPartition := &partition{
		Key: rs.partitionKey(rs.NumPartitions),
	}

	rs.Partitions = append(rs.Partitions, newPartition)
	rs.NumPartitions++
	return newPartition
}

func (rs *randomSelector) deleteTail(balances state.StateContextI) error {
	_, err := balances.DeleteTrieNode(rs.partitionKey(len(rs.Partitions) - 1))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
	}
	rs.Partitions = rs.Partitions[:len(rs.Partitions)-1]
	rs.NumPartitions--
	return nil
}

func (rs *randomSelector) Size(state state.StateContextI) (int, error) {
	if rs.NumPartitions == 0 {
		return 0, nil
	}
	lastPartition, err := rs.getPartition(state, rs.NumPartitions-1)
	if err != nil {
		return 0, err
	}

	return (rs.NumPartitions-1)*rs.PartitionSize + lastPartition.length(), nil
}

func (rs *randomSelector) Save(balances state.StateContextI) error {
	var numPartitions = 0
	for i, partition := range rs.Partitions {
		if partition == nil {
			continue
		}
		if partition.length() == 0 {
			_, err := balances.DeleteTrieNode(rs.partitionKey(i))
			if err != nil {
				if err != util.ErrValueNotPresent {
					return err
				}
			}
			continue
		}
		if partition.changed() {
			err := partition.save(balances)
			if err != nil {
				return err
			}
		}
		numPartitions++
	}

	rs.NumPartitions = numPartitions

	_, err := balances.InsertTrieNode(rs.Name, rs)
	if err != nil {
		return err
	}
	return nil
}

func (rs *randomSelector) getPartition(state state.StateContextI, i int) (*partition, error) {
	if i >= len(rs.Partitions) {
		return nil, fmt.Errorf("partition id %v greater than number of partitions %v", i, len(rs.Partitions))
	}
	if rs.Partitions[i] != nil {
		return rs.Partitions[i], nil
	}

	part := &partition{}
	err := part.load(state, rs.partitionKey(i))
	if err != nil {
		return nil, err
	}
	rs.Partitions[i] = part
	return part, nil
}

func (rs *randomSelector) MarshalMsg(o []byte) ([]byte, error) {
	d := randomSelectorDecode(*rs)
	return d.MarshalMsg(o)
}

func (rs *randomSelector) UnmarshalMsg(b []byte) ([]byte, error) {
	d := &randomSelectorDecode{}
	o, err := d.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	*rs = randomSelector(*d)

	rs.Partitions = make([]*partition, d.NumPartitions)
	return o, nil
}

func (rs *randomSelector) Msgsize() int {
	d := randomSelectorDecode(*rs)
	return d.Msgsize()
}

type randomSelectorDecode randomSelector

func (rs *randomSelector) UpdateItem(
	state state.StateContextI,
	partIndex int,
	it PartitionItem,
) error {

	partition, err := rs.getPartition(state, partIndex)
	if err != nil {
		return err
	}

	return partition.update(it)
}

func (rs *randomSelector) GetItem(
	state state.StateContextI,
	partIndex int,
	id string,
	v PartitionItem,
) error {

	pt, err := rs.getPartition(state, partIndex)
	if err != nil {
		return err
	}

	item, ok := pt.find(id)
	if !ok {
		return errors.New("item not present")
	}

	_, err = v.UnmarshalMsg(item.Data)
	return err
}
