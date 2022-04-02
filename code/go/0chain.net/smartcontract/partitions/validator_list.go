package partitions

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -v -io=false -tests=false -unexported=true

func NewPopulatedValidatorSelector(
	name string,
	size int,
	data []ValidationNode,
) RandPartition {
	rs := &randomSelector{
		Name:          name,
		PartitionSize: size,
		ItemType:      ItemValidator,
	}

	for i := 0; i < len(data)/size; i++ {
		partition := validatorItemList{
			Key:     rs.partitionKey(i),
			Items:   data[size*i : size*(i+1)],
			Changed: true,
		}
		rs.Partitions = append(rs.Partitions, &partition)
		rs.NumPartitions++
	}
	if len(data)%size > 0 {
		partition := validatorItemList{
			Key:     rs.partitionKey(rs.NumPartitions),
			Items:   data[rs.NumPartitions*size:],
			Changed: true,
		}
		rs.Partitions = append(rs.Partitions, &partition)
		rs.NumPartitions++
	}

	return rs
}

//------------------------------------------------------------------------------

type ValidationNode struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func (vn *ValidationNode) Encode() []byte {
	var b, err = json.Marshal(vn)
	if err != nil {
		panic(err)
	}
	return b
}

func (vn *ValidationNode) Decode(b []byte) error {
	return json.Unmarshal(b, vn)
}

func (vn *ValidationNode) Data() string {
	return vn.Url
}

func (vn *ValidationNode) Name() string {
	return vn.Id
}

//------------------------------------------------------------------------------

type validatorItemList struct {
	Key     string           `json:"-" msg:"-"`
	Items   []ValidationNode `json:"items"`
	Changed bool             `json:"-" msg:"-"`
}

func (il *validatorItemList) Encode() []byte {
	var b, err = json.Marshal(il)
	if err != nil {
		panic(err)
	}
	return b
}

func (il *validatorItemList) Decode(b []byte) error {
	return json.Unmarshal(b, il)
}

func (il *validatorItemList) save(balances state.StateContextI) error {
	_, err := balances.InsertTrieNode(il.Key, il)
	return err
}

func (il *validatorItemList) get(key datastore.Key, balances state.StateContextI) error {
	err := balances.GetTrieNode(key, il)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il.Key = key
		return nil
	}
	return nil
}

func (il *validatorItemList) add(it PartitionItem) error {
	for _, bi := range il.Items {
		if bi.Name() == it.Name() {
			return errors.New("blobber item already exists")
		}
	}

	il.Items = append(il.Items, ValidationNode{
		Id:  it.Name(),
		Url: string(it.Data()),
	})
	il.Changed = true
	return nil
}

func (il *validatorItemList) update(it PartitionItem) error {
	val, ok := it.(*ValidationNode)
	if !ok {
		return errors.New("invalid item")
	}

	for i := 0; i < il.length(); i++ {
		if il.Items[i].Name() == it.Name() {
			newItem := *val
			il.Items[i] = newItem
			il.Changed = true
			return nil
		}
	}
	return errors.New("item not found")
}

func (il *validatorItemList) remove(item PartitionItem) error {
	if len(il.Items) == 0 {
		return fmt.Errorf("searching empty partition")
	}
	index := il.find(item)
	if index == notFound {
		return fmt.Errorf("cannot find item %v in partition", item)
	}
	il.Items[index] = il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return nil
}

func (il *validatorItemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *validatorItemList) length() int {
	return len(il.Items)
}

func (il *validatorItemList) changed() bool {
	return il.Changed
}

func (il *validatorItemList) itemRange(start, end int) []PartitionItem {
	if start > end || end > len(il.Items) {
		return nil
	}

	var rtv []PartitionItem
	for i := start; i < end; i++ {
		rtv = append(rtv, &il.Items[i])
	}
	return rtv
}

func (il *validatorItemList) find(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.Name() == searchItem.Name() {
			return i
		}
	}
	return notFound
}

//------------------------------------------------------------------------------
