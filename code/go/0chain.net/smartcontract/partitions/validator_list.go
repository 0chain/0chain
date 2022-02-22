package partitions

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -v

func NewPopulatedValidatorSelector(
	name string,
	size int,
	data []ValidationNode,
) RandPartition {
	rs := &RandomSelector{
		Name:          name,
		PartitionSize: size,
		ItemType:      ItemValidator,
	}

	for i := 0; i < len(data)/size; i++ {
		partition := ValidatorItemList{
			Key:     rs.partitionKey(i),
			Items:   data[size*i : size*(i+1)],
			Changed: true,
		}
		rs.Partitions = append(rs.Partitions, &partition)
		rs.NumPartitions++
	}
	if len(data)%size > 0 {
		partition := ValidatorItemList{
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

type ValidatorItemList struct {
	Key     string           `json:"-" msg:"-"`
	Items   []ValidationNode `json:"items"`
	Changed bool             `json:"-" msg:"-"`
}

func (il *ValidatorItemList) Encode() []byte {
	var b, err = json.Marshal(il)
	if err != nil {
		panic(err)
	}
	return b
}

func (il *ValidatorItemList) Decode(b []byte) error {
	return json.Unmarshal(b, il)
}

func (il *ValidatorItemList) save(balances state.StateContextI) error {
	_, err := balances.InsertTrieNode(il.Key, il)
	return err
}

func (il *ValidatorItemList) get(key datastore.Key, balances state.StateContextI) error {
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

func (il *ValidatorItemList) add(it PartitionItem) {
	il.Items = append(il.Items, ValidationNode{
		Id:  it.Name(),
		Url: string(it.Data()),
	})
	il.Changed = true
}

func (il *ValidatorItemList) remove(item PartitionItem) error {
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

func (il *ValidatorItemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *ValidatorItemList) length() int {
	return len(il.Items)
}

func (il *ValidatorItemList) changed() bool {
	return il.Changed
}

func (il *ValidatorItemList) itemRange(start, end int) []PartitionItem {
	if start > end || end > len(il.Items) {
		return nil
	}

	var rtv []PartitionItem
	for i := start; i < end; i++ {
		rtv = append(rtv, &il.Items[i])
	}
	return rtv
}

func (il *ValidatorItemList) find(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.Name() == searchItem.Name() {
			return i
		}
	}
	return notFound
}

//------------------------------------------------------------------------------
