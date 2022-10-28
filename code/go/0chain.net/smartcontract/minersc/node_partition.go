package minersc

import (
	"errors"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
)

var (
	minerPartitionsName   = encryption.Hash("miner_partitions")
	sharderPartitionsName = encryption.Hash("sharder_partitions")

	minersPartitions   = newNodePartition(minerPartitionsName)
	shardersPartitions = newNodePartition(sharderPartitionsName)
)

// InitPartitions initialize nodes partitions
func InitPartitions(balances state.StateContextI) error {
	_, err := partitions.CreateIfNotExists(balances, minerPartitionsName, 20)
	if err != nil {
		return err
	}

	_, err = partitions.CreateIfNotExists(balances, sharderPartitionsName, 20)
	if err != nil {
		return err
	}

	return nil
}

// GetPartitions returns partitions of given node type
func GetPartitions(balances state.StateContextI, nodeType NodeType) (*partitions.Partitions, error) {
	switch nodeType {
	case NodeTypeMiner:
		return minersPartitions.getPart(balances)
	case NodeTypeSharder:
		return shardersPartitions.getPart(balances)
	default:
		return nil, errors.New("unknown node type of partitions")
	}
}

type nodePartition struct {
	name string
}

func newNodePartition(name string) *nodePartition {
	return &nodePartition{
		name: name,
	}
}

func (np *nodePartition) add(balances state.StateContextI, n *MinerNode) error {
	return partitions.Update(balances, np.name, func(part *partitions.Partitions) error {
		_, err := part.AddItem(balances, n)
		return err
	})

}

func (np *nodePartition) get(balances state.StateContextI, key string) (*MinerNode, error) {
	var n MinerNode
	if err := partitions.View(balances, np.name, func(part *partitions.Partitions) error {
		return part.GetItem(balances, key, &n)
	}); err != nil {
		return nil, err
	}

	return &n, nil
}

func (np *nodePartition) remove(balances state.StateContextI, key string) error {
	return partitions.Update(balances, np.name, func(part *partitions.Partitions) error {
		return part.RemoveItem(balances, key)
	})
}

func (np *nodePartition) exist(balances state.StateContextI, key string) (bool, error) {
	var exist bool
	if err := partitions.View(balances, np.name, func(part *partitions.Partitions) error {
		var err error
		exist, err = part.Exist(balances, key)
		return err
	}); err != nil {
		return false, err
	}

	return exist, nil
}

func (np *nodePartition) getPart(balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.GetPartitions(balances, np.name)
}

type changesCount struct {
	count int
}

func (c *changesCount) increase() {
	c.count++
}

func forEachNodesWithPart(balances state.StateContextI, part *partitions.Partitions, f func(partIndex int, n *MinerNode, cc *changesCount) (bool, error)) error {
	return part.Foreach(balances, func(_ string, data []byte, partIndex int) ([]byte, bool, error) {
		n := NewMinerNode()
		_, err := n.UnmarshalMsg(data)
		if err != nil {
			return nil, false, err
		}

		var cc changesCount
		bk, err := f(partIndex, n, &cc)
		if err != nil {
			return nil, false, err
		}

		if cc.count > 0 {
			newData, err := n.MarshalMsg(nil)
			if err != nil {
				return nil, false, err
			}

			return newData, bk, nil
		}

		return data, bk, nil
	})
}

func (np *nodePartition) update(balances state.StateContextI, id string, f func(n *MinerNode) error) error {
	return partitions.Update(balances, np.name, func(part *partitions.Partitions) error {
		return part.Update(balances, GetNodeKey(id), func(data []byte) ([]byte, error) {
			n := NewMinerNode()
			_, err := n.UnmarshalMsg(data)
			if err != nil {
				return nil, err
			}

			if err := f(n); err != nil {
				return nil, err
			}

			return n.MarshalMsg(nil)
		})
	})
}
