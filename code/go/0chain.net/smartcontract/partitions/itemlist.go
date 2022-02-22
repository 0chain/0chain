package partitions

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -v

type ItemList struct {
	Key     string       `json:"-" msg:"-"`
	Items   []StringItem `json:"items"`
	Changed bool         `json:"-" msg:"-"`
}

func (il *ItemList) Encode() []byte {
	var b, err = json.Marshal(il)
	if err != nil {
		panic(err)
	}
	return b
}

func (il *ItemList) Decode(b []byte) error {
	return json.Unmarshal(b, il)
}

func (il *ItemList) save(balances state.StateContextI) error {
	_, err := balances.InsertTrieNode(il.Key, il)
	return err
}

func (il *ItemList) get(key datastore.Key, balances state.StateContextI) error {
	err := balances.GetTrieNode(key, il)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il.Key = key
	}

	return nil
}

func (il *ItemList) add(it PartitionItem) {
	il.Items = append(il.Items, StringItem{it.Name()})
	il.Changed = true
}

func (il *ItemList) remove(item PartitionItem) error {
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

func (il *ItemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *ItemList) length() int {
	return len(il.Items)
}

func (il *ItemList) changed() bool {
	return il.Changed
}

func (il *ItemList) itemRange(start, end int) []PartitionItem {
	if start > end || end > len(il.Items) {
		return nil
	}

	var rtv []PartitionItem
	for i := start; i < end; i++ {
		rtv = append(rtv, &il.Items[i])
	}
	return rtv
}

func (il *ItemList) find(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.Name() == searchItem.Name() {
			return i
		}
	}
	return notFound
}

type StringItem struct {
	Item string `json:"item"`
}

func (ri StringItem) Name() string {
	return ri.Item
}

func (si *StringItem) Encode() []byte {
	var b, err = json.Marshal(si)
	if err != nil {
		panic(err)
	}
	return b
}

func (si *StringItem) Decode(b []byte) error {
	return json.Unmarshal(b, si)
}

func (si *StringItem) Data() string {
	return ""
}

func ItemFromString(name string) PartitionItem {
	return &StringItem{Item: name}
}
