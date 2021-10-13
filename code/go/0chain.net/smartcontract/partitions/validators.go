package partitions

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//------------------------------------------------------------------------------

type ValidatorItem struct {
	StringItem
	Url string `json:"url"`
}

func (si *ValidatorItem) Encode() []byte {
	var b, err = json.Marshal(si)
	if err != nil {
		panic(err)
	}
	return b
}

func (si *ValidatorItem) Decode(b []byte) error {
	return json.Unmarshal(b, si)
}

func (si *ValidatorItem) Data() []byte {
	return []byte(si.Url)
}

func NewValidatorItem(name, url string) PartitionItem {
	return &ValidatorItem{
		StringItem: StringItem{name},
		Url:        url,
	}
}

//------------------------------------------------------------------------------

type validatorItemList struct {
	Key     datastore.Key   `json:"-"`
	Items   []ValidatorItem `json:"items"`
	Changed bool            `json:"-"`
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
	val, err := balances.GetTrieNode(key)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il = &validatorItemList{
			Key: key,
		}
	}
	if err := il.Decode(val.Encode()); err != nil {
		return fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	il.Key = key
	return nil
}

func (il *validatorItemList) add(it PartitionItem) {
	vit, ok := it.(*ValidatorItem)
	ok = ok
	il.Items = append(il.Items, ValidatorItem{
		StringItem: StringItem{it.Name()},
		Url:        vit.Url,
	})
	il.Changed = true
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
