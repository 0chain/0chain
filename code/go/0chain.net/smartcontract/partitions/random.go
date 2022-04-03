package partitions

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"

	"0chain.net/core/util"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"

	"0chain.net/chaincore/chain/state"
)

const notFound = -1

//msgp:ignore randomSelector
//go:generate msgp -io=false -tests=false -unexported=true -v
type ItemType int

const (
	ItemString ItemType = iota
	ItemValidator
)

//------------------------------------------------------------------------------

type randomSelector struct {
	Name          string                  `json:"name"`
	PartitionSize int                     `json:"partition_size"`
	NumPartitions int                     `json:"num_partitions"`
	Partitions    []PartitionItemList     `json:"-" msg:"-"`
	Callback      ChangePartitionCallback `json:"-" msg:"-"`
	ItemType      ItemType                `json:"item_type"` // todo think of something better
}

func NewRandomSelector(
	name string,
	size int,
	callback ChangePartitionCallback,
	itemType ItemType,
) RandPartition {
	return &randomSelector{
		Name:          name,
		PartitionSize: size,
		Callback:      callback,
		ItemType:      itemType,
	}
}

func PartitionKey(name string, index int) datastore.Key {
	return datastore.Key(name + encryption.Hash(":partition:"+strconv.Itoa(index)))
}

func (rs *randomSelector) partitionKey(index int) datastore.Key {
	return PartitionKey(rs.Name, index)
}

func (rs *randomSelector) SetCallback(callback ChangePartitionCallback) {
	rs.Callback = callback
}

func (rs *randomSelector) Add(
	item PartitionItem,
	balances state.StateContextI,
) (int, error) {
	var part PartitionItemList
	var err error
	if len(rs.Partitions) > 0 {
		part, err = rs.getPartition(len(rs.Partitions)-1, balances)
		if err != nil {
			return 0, err
		}
	}
	if len(rs.Partitions) == 0 || part.length() >= rs.PartitionSize {
		part = rs.addPartition()
	}
	part.add(item)
	return len(rs.Partitions) - 1, nil
}

func (rs *randomSelector) Remove(
	item PartitionItem,
	index int,
	balances state.StateContextI,
) error {
	part, err := rs.getPartition(index, balances)
	if err != nil {
		return err
	}

	err = part.remove(item)
	if err != nil {
		return err
	}

	lastPart, err := rs.getPartition(len(rs.Partitions)-1, balances)
	if err != nil {
		return err
	}

	if index == rs.NumPartitions-1 {
		if lastPart.length() == 0 {
			if err := rs.deleteTail(balances); err != nil {
				return err
			}
		}
		return nil
	}

	replacment := lastPart.cutTail()
	if replacment == nil {
		fmt.Errorf("empty last partitions, currpt data")
	}
	part.add(replacment)
	if rs.Callback != nil {
		err = rs.Callback(replacment, len(rs.Partitions)-1, index, balances)
		if err != nil {
			return err
		}
	}

	if lastPart.length() == 0 {
		if err := rs.deleteTail(balances); err != nil {
			return err
		}
	}

	return nil
}

func (rs *randomSelector) AddRand(
	item PartitionItem,
	r *rand.Rand,
	balances state.StateContextI,
) (int, error) {
	if rs.NumPartitions == 0 {
		return rs.Add(item, balances)
	}
	index := r.Intn(rs.NumPartitions)
	if index == rs.NumPartitions-1 {
		return rs.Add(item, balances)
	}

	partition, err := rs.getPartition(index, balances)
	if err != nil {
		return -1, err
	}
	moving := partition.cutTail()
	if moving == nil {
		fmt.Errorf("empty partitions, currpt data")
	}
	partition.add(item)

	movedTo, err := rs.Add(moving, balances)
	if err != nil {
		return -1, err
	}
	if rs.Callback != nil {
		err = rs.Callback(moving, index, movedTo, balances)
		if err != nil {
			return -1, err
		}
	}

	return index, nil
}

func (rs *randomSelector) GetRandomSlice(
	r *rand.Rand,
	balances state.StateContextI,
) ([]PartitionItem, error) {
	if rs.NumPartitions == 0 {
		return nil, errors.New("Empty list, no items to return")
	}
	index := r.Intn(rs.NumPartitions)

	var rtv []PartitionItem
	partition, err := rs.getPartition(index, balances)
	if err != nil {
		return nil, err
	}
	rtv = append(rtv, partition.itemRange(0, partition.length())...)
	if index == rs.NumPartitions-1 && len(rtv) < rs.PartitionSize && rs.NumPartitions > 1 {
		secondLast, err := rs.getPartition(index-1, balances)
		if err != nil {
			return nil, err
		}
		want := rs.PartitionSize - len(rtv)
		if secondLast.length() < want {
			return nil, fmt.Errorf("second last partition too small %d instead of %d",
				secondLast.length(), rs.NumPartitions)
		}
		rtv = append(rtv, partition.itemRange(secondLast.length()-1, partition.length())...)
	}

	return rtv, nil
}

func (rs *randomSelector) addPartition() PartitionItemList {
	var newPartition PartitionItemList
	if rs.ItemType == ItemString {
		newPartition = &itemList{
			Key: rs.partitionKey(rs.NumPartitions),
		}
	} else {
		newPartition = &validatorItemList{
			Key: rs.partitionKey(rs.NumPartitions),
		}
	}
	rs.Partitions = append(rs.Partitions, newPartition)
	rs.NumPartitions++
	return newPartition
}

func (rs *randomSelector) deleteTail(balances state.StateContextI) error {
	err := balances.DeleteTrieNode(rs.partitionKey(len(rs.Partitions) - 1))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
	}
	rs.Partitions = rs.Partitions[:len(rs.Partitions)-1]
	rs.NumPartitions--
	return nil
}

func (rs *randomSelector) Size(balances state.StateContextI) (int, error) {
	if rs.NumPartitions == 0 {
		return 0, nil
	}
	lastPartition, err := rs.getPartition(rs.NumPartitions-1, balances)
	if err != nil {
		return 0, err
	}

	return (rs.NumPartitions-1)*rs.PartitionSize + lastPartition.length(), nil
}

func (rs *randomSelector) Save(balances state.StateContextI) error {
	var numPartitions = 0
	for i, partition := range rs.Partitions {
		if partition != nil && partition.changed() {
			if partition.length() > 0 {
				err := partition.save(balances)
				if err != nil {
					return err
				}
				numPartitions++
			} else {
				err := balances.DeleteTrieNode(rs.partitionKey(i))
				if err != nil {
					if err != util.ErrValueNotPresent {
						return err
					}
				}
			}
		}
	}
	rs.NumPartitions = numPartitions

	err := balances.InsertTrieNode(rs.Name, rs)
	if err != nil {
		return err
	}
	return nil
}

func (rs *randomSelector) getPartition(
	i int, balances state.StateContextI,
) (PartitionItemList, error) {
	if i >= len(rs.Partitions) {
		return nil, fmt.Errorf("partition id %v grater than numbr of partitions %v", i, len(rs.Partitions))
	}
	if rs.Partitions[i] != nil {
		return rs.Partitions[i], nil
	}
	var part PartitionItemList
	if rs.ItemType == ItemString {
		part = &itemList{}
	} else {
		part = &validatorItemList{}
	}
	err := part.get(rs.partitionKey(i), balances)
	if err != nil {
		return nil, err
	}
	rs.Partitions[i] = part
	return part, nil
}

func GetRandomSelector(
	key datastore.Key,
	balances state.StateContextI,
) (RandPartition, error) {
	var rs *randomSelector
	raw, err := balances.GetTrieNode(key, rs)
	if err != nil {
		return nil, err

	}
	var ok bool
	if rs, ok = raw.(*randomSelector); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}
	return rs, nil
}

func (rs *randomSelector) Encode() []byte {
	var b, err = json.Marshal(rs)
	if err != nil {
		panic(err)
	}
	return b
}

func (rs *randomSelector) Decode(b []byte) error {
	err := json.Unmarshal(b, rs)
	rs.Partitions = make([]PartitionItemList, rs.NumPartitions)
	return err
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

	rs.Partitions = make([]PartitionItemList, d.NumPartitions)
	return o, nil
}

func (rs *randomSelector) Msgsize() int {
	d := randomSelectorDecode(*rs)
	return d.Msgsize()
}

type randomSelectorDecode randomSelector
