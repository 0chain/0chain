package partitions

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"errors"
	"fmt"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

//------------------------------------------------------------------------------

type BlobberChallengeAllocationNode struct {
	ID string `json:"id"`
}

func (bcn *BlobberChallengeAllocationNode) Encode() []byte {
	var b, err = json.Marshal(bcn)
	if err != nil {
		panic(err)
	}
	return b
}

func (bcn *BlobberChallengeAllocationNode) Decode(b []byte) error {
	return json.Unmarshal(b, bcn)
}

func (bcn *BlobberChallengeAllocationNode) Data() string {
	return ""
}

func (bcn *BlobberChallengeAllocationNode) Name() string {
	return bcn.ID
}

//------------------------------------------------------------------------------

type blobberChallengeAllocationItemList struct {
	Key     string                           `json:"-"`
	Items   []BlobberChallengeAllocationNode `json:"items"`
	Changed bool                             `json:"-"`
}

func (il *blobberChallengeAllocationItemList) Encode() []byte {
	var b, err = json.Marshal(il)
	if err != nil {
		panic(err)
	}
	return b
}

func (il *blobberChallengeAllocationItemList) Decode(b []byte) error {
	return json.Unmarshal(b, il)
}

func (il *blobberChallengeAllocationItemList) save(balances state.StateContextI) error {
	_, err := balances.InsertTrieNode(il.Key, il)
	return err
}

func (il *blobberChallengeAllocationItemList) get(key datastore.Key, balances state.StateContextI) error {
	err := balances.GetTrieNode(key, il)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		il.Key = key
	}
	return nil
}

func (il *blobberChallengeAllocationItemList) add(it PartitionItem) error {
	for _, bi := range il.Items {
		if bi.ID == it.Name() {
			return errors.New("blobber_challenge_allocation item already exists")
		}
	}
	il.Items = append(il.Items, BlobberChallengeAllocationNode{
		ID: it.Name(),
	})
	il.Changed = true
	return nil
}

func (il *blobberChallengeAllocationItemList) remove(item PartitionItem) error {
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

func (il *blobberChallengeAllocationItemList) cutTail() PartitionItem {
	if len(il.Items) == 0 {
		return nil
	}

	tail := il.Items[len(il.Items)-1]
	il.Items = il.Items[:len(il.Items)-1]
	il.Changed = true
	return &tail
}

func (il *blobberChallengeAllocationItemList) length() int {
	return len(il.Items)
}

func (il *blobberChallengeAllocationItemList) changed() bool {
	return il.Changed
}

func (il *blobberChallengeAllocationItemList) itemRange(start, end int) []PartitionItem {
	if start > end || end > len(il.Items) {
		return nil
	}

	var rtv []PartitionItem
	for i := start; i < end; i++ {
		rtv = append(rtv, &il.Items[i])
	}
	return rtv
}

func (il *blobberChallengeAllocationItemList) find(searchItem PartitionItem) int {
	for i, item := range il.Items {
		if item.Name() == searchItem.Name() {
			return i
		}
	}
	return notFound
}

func (il *blobberChallengeAllocationItemList) update(it PartitionItem) error {

	val, ok := it.(*BlobberChallengeAllocationNode)
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
