package partitions

import (
	"0chain.net/chaincore/chain/state"
	state2 "0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"errors"
	"fmt"
)

//------------------------------------------------------------------------------

type BlobberRewardNode struct {
	Id                string         `json:"id"`
	SuccessChallenges int            `json:"success_challenges"`
	WritePrice        state2.Balance `json:"write_price"`
}

func (bn *BlobberRewardNode) Encode() []byte {
	var b, err = json.Marshal(bn)
	if err != nil {
		panic(err)
	}
	return b
}

func (bn *BlobberRewardNode) Decode(b []byte) error {
	return json.Unmarshal(b, bn)
}

func (bn *BlobberRewardNode) Data() string {
	return string(bn.Encode())
}

func (bn *BlobberRewardNode) Name() string {
	return bn.Id
}

//------------------------------------------------------------------------------

type blobberRewardItemList struct {
	Key     datastore.Key       `json:"-"`
	Items   []BlobberRewardNode `json:"items"`
	Changed bool                `json:"-"`
}

func (il *blobberRewardItemList) Encode() []byte {
	var b, err = json.Marshal(il)
	if err != nil {
		panic(err)
	}
	return b
}

func (il *blobberRewardItemList) Decode(b []byte) error {
	return json.Unmarshal(b, il)
}

func (il *blobberRewardItemList) save(balances state.StateContextI) error {
	_, err := balances.InsertTrieNode(il.Key, il)
	return err
}

func (il *blobberRewardItemList) get(key datastore.Key, balances state.StateContextI) error {
	val, err := balances.GetTrieNode(key)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il = &blobberRewardItemList{
			Key: key,
		}
	}
	if err := il.Decode(val.Encode()); err != nil {
		return fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	il.Key = key
	return nil
}

func (il *blobberRewardItemList) add(it PartitionItem) {
	var brn BlobberRewardNode
	brn.Decode(it.Encode())
	il.Items = append(il.Items, BlobberRewardNode{
		Id:                it.Name(),
		SuccessChallenges: brn.SuccessChallenges,
		WritePrice:        brn.WritePrice,
	})
	il.Changed = true
}

func (il *blobberRewardItemList) update(it PartitionItem) error {
	var found bool
	for i := range il.itemRange(0, il.length()) {
		if il.Items[i].Name() == it.Name() {
			found = true
			var newItem BlobberRewardNode
			err := newItem.Decode(it.Encode())
			if err != nil {
				return fmt.Errorf("decoding error: %v", err)
			}
			il.Items[i] = BlobberRewardNode{
				Id:                it.Name(),
				SuccessChallenges: newItem.SuccessChallenges,
				WritePrice:        newItem.WritePrice,
			}
		}
	}

	if !found {
		return errors.New("item not found in list")
	}
	il.Changed = true
	return nil
}

func (il *blobberRewardItemList) remove(item PartitionItem) error {
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

func (il *blobberRewardItemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *blobberRewardItemList) length() int {
	return len(il.Items)
}

func (il *blobberRewardItemList) changed() bool {
	return il.Changed
}

func (il *blobberRewardItemList) itemRange(start, end int) []PartitionItem {
	if start > end || end > len(il.Items) {
		return nil
	}

	var rtv []PartitionItem
	for i := start; i < end; i++ {
		rtv = append(rtv, &il.Items[i])
	}
	return rtv
}

func (il *blobberRewardItemList) find(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.Name() == searchItem.Name() {
			return i
		}
	}
	return notFound
}

//------------------------------------------------------------------------------
