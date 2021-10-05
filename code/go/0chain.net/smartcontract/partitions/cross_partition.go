package partitions

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type crossPartition struct {
	Name            string
	PartitionMap    map[string]*revolvingPartition `json:"partition_map"`
	ItemToPartition ItemToPartition                `json:"item_to_partition"`
	RangeIterator   PartitionIterator              `json:"partition_iterator"`
	Validate        Validator                      `json:"validator"`
}

func (cp *crossPartition) Add(
	item PartitionItem,
	balances state.StateContextI,
) error {
	name, err := cp.ItemToPartition(item)
	if err != nil {
		return err
	}
	partition, err := cp.getOrCreatePartition(name, balances)
	if err != nil {
		return err
	}
	partition.add(item.Name())
	return nil
}

func (cp *crossPartition) Remove(
	item PartitionItem,
	balances state.StateContextI,
) error {
	partitionName, err := cp.ItemToPartition(item)
	if err != nil {
		return err
	}
	partition, err := cp.getPartition(partitionName, balances)
	if err != nil {
		return nil
	}
	return partition.remove(item)
}

func (cp *crossPartition) Change(
	old, new PartitionItem,
	balances state.StateContextI,
) error {
	err := cp.Remove(old, balances)
	if err != nil {
		return err
	}
	return cp.Add(new, balances)
}

func (cp *crossPartition) GetItems(
	pRange PartitionRange,
	want int,
	balances state.StateContextI,
) ([]string, error) {
	if err := cp.RangeIterator.Start(pRange); err != nil {
		return nil, err
	}
	if want < 1 {
		return nil, fmt.Errorf("must get at least one item, not %v", want)
	}

	got := 0
	var list []string
	for next := cp.RangeIterator.Next(); next != ""; next = cp.RangeIterator.Next() {
		partition, err := cp.getPartition(next, balances)
		if err != nil {
			if err != util.ErrValueNotPresent {
				return nil, err
			}
			continue
		}
		for i := 0; i < len(partition.Items); i++ {
			next := partition.get()
			if cp.Validate(pRange, next) {
				list = append(list, next)
				got++
				if got == want {
					return list, nil
				}
			}

		}
	}
	return nil, fmt.Errorf("insuffient items found, wanted %d", want)

}

func (cp *crossPartition) SetPartitionHandler(f ItemToPartition) {
	cp.ItemToPartition = f
}
func (cp *crossPartition) SetPartitionIterator(iterator PartitionIterator) {
	cp.RangeIterator = iterator
}

func (cp *crossPartition) partitionKey(key string) datastore.Key {
	return datastore.Key(cp.Name + encryption.Hash(":key"))
}

func (cp crossPartition) SetItemValidator(f Validator) {
	cp.Validate = f
}

func (cp *crossPartition) getPartition(
	key string,
	balances state.StateContextI,
) (*revolvingPartition, error) {
	partition, ok := cp.PartitionMap[key]
	if ok {
		return partition, nil
	}

	var rp revolvingPartition
	val, err := balances.GetTrieNode(cp.partitionKey(key))
	if err != nil {
		return nil, err
	}
	if err := rp.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	cp.PartitionMap[key] = &rp
	return &rp, nil
}

func (cp *crossPartition) getOrCreatePartition(
	key string,
	balances state.StateContextI,
) (*revolvingPartition, error) {
	rp, err := cp.getPartition(key, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		rp = new(revolvingPartition)
		cp.PartitionMap[key] = rp
	}
	return rp, nil
}

func (cp *crossPartition) Encode() []byte {
	var b, err = json.Marshal(cp)
	if err != nil {
		panic(err)
	}
	return b
}

func (cp *crossPartition) Decode(b []byte) error {
	return json.Unmarshal(b, cp)
}

func (cp *crossPartition) save(balances state.StateContextI) error {
	for key, partition := range cp.PartitionMap {
		if err := partition.save(cp.partitionKey(key), balances); err != nil {
			return nil
		}
	}

	_, err := balances.InsertTrieNode(cp.Name, cp)
	if err != nil {
		return err
	}
	return nil
}

func getCrossPartitionTable(
	name datastore.Key,
	balances state.StateContextI,
) (*crossPartition, error) {
	var cp crossPartition
	val, err := balances.GetTrieNode(name)
	if err != nil {
		return nil, err
	}
	if err := cp.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return &cp, nil
}

//------------------------------------------------------------------------------

type revolvingPartition struct {
	Items   []string `json:"items"`
	Head    int      `json:"head"`
	changed bool
}

func (rp *revolvingPartition) Encode() []byte {
	var b, err = json.Marshal(rp)
	if err != nil {
		panic(err)
	}
	return b
}

func (rp *revolvingPartition) Decode(b []byte) error {
	return json.Unmarshal(b, rp)
}

func (rp *revolvingPartition) save(
	key datastore.Key,
	balances state.StateContextI,
) error {
	if rp.changed {
		_, err := balances.InsertTrieNode(key, rp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rl *revolvingPartition) get() string {
	rtv := rl.Items[rl.Head]
	rl.Head++
	if rl.Head >= len(rl.Items) {
		rl.Head = 0
	}
	return rtv
}

func (rl *revolvingPartition) add(item string) int {
	rl.Items = append(rl.Items, item)
	rl.changed = true
	return len(rl.Items) - 1
}

func (rl *revolvingPartition) remove(item PartitionItem) error {
	index := rl.find(item)
	if index == notFound {
		return fmt.Errorf("cannot find %v in partition", item)
	}
	rl.Items[index] = rl.Items[len(rl.Items)-1]
	rl.Items = rl.Items[:len(rl.Items)-1]
	rl.changed = true
	return nil
}

func (rl *revolvingPartition) find(item PartitionItem) int {
	for i, val := range rl.Items {
		if item.Name() == val {
			return i
		}
	}
	return notFound
}

//------------------------------------------------------------------------------
