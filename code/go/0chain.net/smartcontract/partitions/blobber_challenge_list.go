package partitions

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"fmt"
)

//------------------------------------------------------------------------------

type BlobberChallengeNode struct {
	Id  string `json:"id"`
	Url string `json:"url"`
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
	return bcn.Url
}

func (bcn *BlobberChallengeNode) Name() string {
	return bcn.Id
}

//------------------------------------------------------------------------------

type blobberChallengeItemList struct {
	Key     datastore.Key          `json:"-"`
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
	val, err := balances.GetTrieNode(key)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il = &blobberChallengeItemList{
			Key: key,
		}
	}
	if err := il.Decode(val.Encode()); err != nil {
		return fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	il.Key = key
	return nil
}

func (il *blobberChallengeItemList) add(it PartitionItem) {
	il.Items = append(il.Items, BlobberChallengeNode{
		Id:  it.Name(),
		Url: it.Data(),
	})
	il.Changed = true
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

//------------------------------------------------------------------------------
