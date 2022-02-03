package partitions

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type itemList struct {
	Key     datastore.Key `json:"-"`
	Items   []StringItem  `json:"items"`
	Changed bool          `json:"-"`
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

func (il *itemList) save(balances state.StateContextI) error {
	err := balances.InsertTrieNode(il.Key, il)
	return err
}

func getItemList(key datastore.Key, balances state.StateContextI) (*itemList, error) {
	var il *itemList
	raw, err := balances.GetTrieNode(key, il)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		il = &itemList{
			Key: key,
		}
		return il, nil
	}
	var ok bool
	if il, ok = raw.(*itemList); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}
	il.Key = key
	return il, nil
}

func (il *itemList) add(it PartitionItem) {
	il.Items = append(il.Items, StringItem{it.Name()})
	il.Changed = true
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
	il.Changed = true
	return nil
}

func (il *itemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *itemList) length() int {
	return len(il.Items)
}

func (il *itemList) changed() bool {
	return il.Changed
}

func (il *itemList) itemRange(start, end int) []PartitionItem {
	if start > end || end > len(il.Items) {
		return nil
	}

	var rtv []PartitionItem
	for i := start; i < end; i++ {
		rtv = append(rtv, &il.Items[i])
	}
	return rtv
}

func (il *itemList) find(searchItem PartitionItem) int {
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
