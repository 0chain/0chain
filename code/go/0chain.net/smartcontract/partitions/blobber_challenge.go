package partitions

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

//------------------------------------------------------------------------------

type BlobberChallengeNode struct {
	BlobberID    string  `json:"blobber_id"`
	UsedCapacity float64 `json:"used_capacity"`
}

func (bcn *BlobberChallengeNode) Encode() []byte {
	var b, err = json.Marshal(bcn)
	if err != nil {
		panic(err)
	}
	return b
}

func (bcn *BlobberChallengeNode) Decode(b []byte) error {
	return json.Unmarshal(b, bcn)
}

func (bcn *BlobberChallengeNode) Data() string {
	return string(bcn.Encode())
}

func (bcn *BlobberChallengeNode) Name() string {
	return bcn.BlobberID
}

//------------------------------------------------------------------------------

type blobberChallengeItemList struct {
	Key     string                 `json:"-"`
	Items   []BlobberChallengeNode `json:"items"`
	Changed bool                   `json:"-"`
}

func (il *blobberChallengeItemList) Encode() []byte {
	var b, err = json.Marshal(il)
	if err != nil {
		panic(err)
	}
	return b
}

func (il *blobberChallengeItemList) Decode(b []byte) error {
	return json.Unmarshal(b, il)
}

func (il *blobberChallengeItemList) save(balances state.StateContextI) error {
	_, err := balances.InsertTrieNode(il.Key, il)
	return err
}

func (il *blobberChallengeItemList) get(key datastore.Key, balances state.StateContextI) error {
	err := balances.GetTrieNode(key, il)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il.Key = key
	}
	return nil
}

func (il *blobberChallengeItemList) add(it PartitionItem) error {

	for _, bc := range il.Items {
		if bc.Name() == it.Name() {
			return errors.New("blobber challenge item already exists")
		}
	}

	il.Items = append(il.Items, BlobberChallengeNode{
		BlobberID: it.Name(),
	})
	il.Changed = true
	return nil
}

func (il *blobberChallengeItemList) remove(item PartitionItem) error {
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

func (il *blobberChallengeItemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *blobberChallengeItemList) length() int {
	return len(il.Items)
}

func (il *blobberChallengeItemList) changed() bool {
	return il.Changed
}

func (il *blobberChallengeItemList) itemRange(start, end int) []PartitionItem {
	if start > end || end > len(il.Items) {
		return nil
	}

	var rtv []PartitionItem
	for i := start; i < end; i++ {
		rtv = append(rtv, &il.Items[i])
	}
	return rtv
}

func (il *blobberChallengeItemList) find(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.Name() == searchItem.Name() {
			return i
		}
	}
	return notFound
}

func (il *blobberChallengeItemList) update(it PartitionItem) error {

	val, ok := it.(*BlobberChallengeNode)
	if !ok {
		return errors.New("invalid item")
	}

	for i := 0; i < il.length(); i++ {
		if il.Items[i].Name() == it.Name() {
			newItem := *val
			err := newItem.Decode(it.Encode())
			if err != nil {
				return fmt.Errorf("decoding error: %v", err)
			}
			il.Items[i] = newItem
			il.Changed = true
			return nil
		}
	}
	return errors.New("item not found")
}

//------------------------------------------------------------------------------
