package partitions

import (
	"0chain.net/chaincore/chain/state"
	state2 "0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"errors"
	"fmt"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

//------------------------------------------------------------------------------

type BlobberRewardNode struct {
	ID                string         `json:"id"`
	SuccessChallenges int            `json:"success_challenges"`
	WritePrice        state2.Balance `json:"write_price"`
	ReadPrice         state2.Balance `json:"read_price"`
	TotalData         float64        `json:"total_data"`
	DataRead          float64        `json:"data_read"`
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
	return bn.ID
}

//------------------------------------------------------------------------------

type blobberRewardItemList struct {
	Key     string              `json:"-"`
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
	err := balances.GetTrieNode(key, il)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il.Key = key
	}
	return nil
}

func (il *blobberRewardItemList) add(it PartitionItem) error {
	for _, bi := range il.Items {
		if bi.ID == it.Name() {
			return errors.New("blobber reward item already exists")
		}
	}
	brn, ok := it.(*BlobberRewardNode)
	if !ok {
		return errors.New("not a blobber reward item")
	}
	il.Items = append(il.Items, BlobberRewardNode{
		ID:                it.Name(),
		SuccessChallenges: brn.SuccessChallenges,
		WritePrice:        brn.WritePrice,
		ReadPrice:         brn.ReadPrice,
		TotalData:         brn.TotalData,
		DataRead:          brn.DataRead,
	})
	il.Changed = true
	return nil
}

func (il *blobberRewardItemList) update(it PartitionItem) error {
	val, ok := it.(*BlobberRewardNode)
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
