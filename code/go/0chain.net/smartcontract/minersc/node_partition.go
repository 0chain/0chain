package minersc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
)

var (
	minerPartitionsName   = encryption.Hash("miner_partitions")
	sharderPartitionsName = encryption.Hash("sharder_partitions")

	minersPartitions  = newNodePartition(minerPartitionsName)
	sharderPartitions = newNodePartition(sharderPartitionsName)
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

type nodePartition struct {
	name string
}

func newNodePartition(name string) *nodePartition {
	return &nodePartition{
		name: name,
	}
}

func (np *nodePartition) add(balances state.StateContextI, n *MinerNode) error {
	part, err := partitions.GetPartitions(balances, np.name)
	if err != nil {
		return err
	}

	return part.AddItem(balances, n)
}

func (np *nodePartition) get(balances state.StateContextI, key string) (*MinerNode, error) {
	part, err := partitions.GetPartitions(balances, np.name)
	if err != nil {
		return nil, err
	}

	var n MinerNode
	if err := part.GetItem(balances, key, &n); err != nil {
		return nil, err
	}

	return &n, nil
}

func (np *nodePartition) update(balances state.StateContextI, n *MinerNode) error {
	part, err := partitions.GetPartitions(balances, np.name)
	if err != nil {
		return err
	}

	return part.UpdateItem(balances, n)
}

func (np *nodePartition) remove(balances state.StateContextI, key string) error {
	part, err := partitions.GetPartitions(balances, np.name)
	if err != nil {
		return err
	}

	return part.RemoveItem(balances, key)
}

//func (np *nodePartition) foreach(balances state.StateContextI, f func(item partitions.PartitionItem) error) error {
//	part, err := partitions.GetPartitions(balances, np.name)
//	if err != nil {
//		return err
//	}
//
//	part.Foreach(balances, func(key string, data []byte) error {
//
//	})
//}

// AddMinerNode adds miner node to miner parititons
func AddMinerNode(balances state.StateContextI, n *MinerNode) error {
	part, err := partitions.GetPartitions(balances, minerPartitionsName)
	if err != nil {
		return err
	}

	return part.AddItem(balances, n)
}

// GetMinerNode gets miner node by id
func GetMinerNode(balances state.StateContextI, id string) (*MinerNode, error) {
	part, err := partitions.GetPartitions(balances, minerPartitionsName)
	if err != nil {
		return nil, err
	}

	var n MinerNode
	if err := part.GetItem(balances, id, &n); err != nil {
		return nil, err
	}

	return &n, nil
}
