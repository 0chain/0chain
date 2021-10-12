package partitions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"0chain.net/core/util"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

const notFound = -1

//------------------------------------------------------------------------------

type item string

func (ri item) Name() string {
	return string(ri)
}

func ItemFromString(name string) PartitionItem {
	return item(name)
}

//------------------------------------------------------------------------------

type itemList struct {
	Items   []PartitionItem `json:"partition"`
	changed bool            `json:"-"`
}

func (il *itemList) Encode() []byte {
	var b, err = json.Marshal(il)
	if err != nil {
		panic(err)
	}
	return b
}

func (il *itemList) Decode(b []byte) error {
	return json.Unmarshal(b, il)
}

func (il *itemList) add(item PartitionItem) {
	il.Items = append(il.Items, item)
	il.changed = true
}

func (il *itemList) remove(item PartitionItem) error {
	if len(il.Items) == 0 {
		return fmt.Errorf("searching empty partition")
	}
	index := il.find(item)
	if index == notFound {
		return fmt.Errorf("cannot find item %v in partition", item)
	}
	il.Items[index] = il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.changed = true
	return nil
}

func (il *itemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.changed = true
	return tail
}

func (il *itemList) find(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.Name() == searchItem.Name() {
			return i
		}
	}
	return notFound
}

//------------------------------------------------------------------------------

type randomSelector struct {
	Name          datastore.Key           `json:"name"`
	PartitionSize int                     `json:"size"`
	NumPartitions int                     `json:"num_partitions"`
	partitions    []*itemList             `json:"-"`
	Callback      ChangePartitionCallback `json:"-"`
}

func NewRandomSelector(
	name string,
	size int,
	callback ChangePartitionCallback,
) RandPartition {
	return &randomSelector{
		Name:          name,
		PartitionSize: size,
		Callback:      callback,
	}
}

func NewPopulatedRandomSelector(
	name string,
	size int,
	callback ChangePartitionCallback,
	data []PartitionItem,
) RandPartition {
	rs := &randomSelector{
		Name:          name,
		PartitionSize: size,
		Callback:      callback,
	}

	for i := 0; i < len(data)/size; i++ {
		partition := itemList{
			Items:   data[size*i : size*i+1],
			changed: true,
		}
		rs.partitions = append(rs.partitions, &partition)
		rs.NumPartitions++
	}
	if len(data)%size > 0 {
		partition := itemList{
			Items:   data[rs.NumPartitions*size:],
			changed: true,
		}
		rs.partitions = append(rs.partitions, &partition)
		rs.NumPartitions++
	}

	return rs
}

func (rs *randomSelector) partitionKey(index int) datastore.Key {
	return datastore.Key(rs.Name + encryption.Hash(":partition:"+strconv.Itoa(index)))
}

func (rs *randomSelector) SetCallback(callback ChangePartitionCallback) {
	rs.Callback = callback
}

func (rs *randomSelector) Add(
	item PartitionItem,
	balances state.StateContextI,
) (int, error) {
	var part *itemList
	var err error
	if len(rs.partitions) > 0 {
		part, err = rs.getPartition(len(rs.partitions)-1, balances)
		if err != nil {
			return 0, err
		}
	}
	if len(rs.partitions) == 0 || len(part.Items) >= rs.PartitionSize {
		part = rs.addPartition()
	}
	part.add(item)
	return len(rs.partitions) - 1, nil
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

	lastPart, err := rs.getPartition(len(rs.partitions)-1, balances)
	if err != nil {
		return err
	}

	if index == rs.NumPartitions-1 {
		if len(lastPart.Items) == 0 {
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
	err = rs.Callback(replacment, len(rs.partitions)-1, index, balances)
	if err != nil {
		return err
	}

	if len(lastPart.Items) == 0 {
		if err := rs.deleteTail(balances); err != nil {
			return err
		}
	}

	return nil
}

func (rs *randomSelector) GetRandomSlice(
	seed int64,
	balances state.StateContextI,
) ([]PartitionItem, error) {
	rand.Seed(seed)
	var rtv []PartitionItem
	index := rand.Intn(rs.NumPartitions)
	partition, err := rs.getPartition(index, balances)
	if err != nil {
		return nil, err
	}
	rtv = append(rtv, partition.Items...)
	if index == rs.NumPartitions-1 && len(rtv) < rs.PartitionSize && rs.NumPartitions > 1 {
		secondLast, err := rs.getPartition(index-1, balances)
		if err != nil {
			return nil, err
		}
		want := rs.PartitionSize - len(rtv)
		if len(secondLast.Items) < want {
			return nil, fmt.Errorf("second last partition too small %d instead of %d",
				len(secondLast.Items), rs.NumPartitions)
		}
		rtv = append(rtv, secondLast.Items[len(secondLast.Items)-want:]...)
	}

	return rtv, nil
}

func (rs *randomSelector) addPartition() *itemList {
	var newPartition itemList
	rs.partitions = append(rs.partitions, &newPartition)
	rs.NumPartitions++
	return &newPartition
}

func (rs *randomSelector) deleteTail(balances state.StateContextI) error {
	_, err := balances.DeleteTrieNode(rs.partitionKey(len(rs.partitions) - 1))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
	}
	rs.partitions = rs.partitions[:len(rs.partitions)-1]
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

	return (rs.NumPartitions-1)*rs.PartitionSize + len(lastPartition.Items), nil
}

func (rs *randomSelector) Save(balances state.StateContextI) error {
	var numPartitions = 0
	for i, division := range rs.partitions {
		if division.changed {
			if len(division.Items) > 0 {
				_, err := balances.InsertTrieNode(rs.partitionKey(i), division)
				if err != nil {
					return err
				}
				numPartitions++
			} else {
				_, err := balances.DeleteTrieNode(rs.partitionKey(i))
				if err != nil {
					if err != util.ErrValueNotPresent {
						return err
					}
				}
			}
		}
	}
	rs.NumPartitions = numPartitions

	_, err := balances.InsertTrieNode(rs.Name, rs)
	if err != nil {
		return err
	}
	return nil
}

func (rs *randomSelector) getPartition(i int, balances state.StateContextI) (*itemList, error) {
	if i >= len(rs.partitions) {
		return nil, fmt.Errorf("partition id %v grater than numbr of partitions %v", i, len(rs.partitions))
	}
	if rs.partitions[i] != nil {
		return rs.partitions[i], nil
	}
	var part itemList
	val, err := balances.GetTrieNode(rs.partitionKey(i))
	if err != nil {
		return nil, err
	}
	if err := part.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	rs.partitions[i] = &part
	return &part, nil
}

func GetRandomSelector(
	key datastore.Key,
	balances state.StateContextI,
) (RandPartition, error) {
	var rs randomSelector
	val, err := balances.GetTrieNode(key)
	if err != nil {
		return nil, err

	}
	if err := rs.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return &rs, nil
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
	rs.partitions = make([]*itemList, rs.NumPartitions, rs.NumPartitions)
	return err
}
