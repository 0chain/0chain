package minersc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
)

//go:generate msgp -io=false -tests=false -unexported -v

// MinerNode struct that holds information about the registering miner.
// swagger:model MinerNode
type MinerNode struct {
	*SimpleNode          `json:"simple_miner"`
	*stakepool.StakePool `json:"stake_pool"`
}

func NewMinerNode() *MinerNode {
	mn := &MinerNode{
		SimpleNode: &SimpleNode{},
		StakePool:  stakepool.NewStakePool(),
	}
	return mn
}

type NodePool struct {
	PoolID string `json:"pool_id"`
	*stakepool.DelegatePool
}

func GetSharderKey(sid string) datastore.Key {
	return ADDRESS + sid
}

func (mn *MinerNode) GetKey() datastore.Key {
	return ADDRESS + mn.ID
}

// GetID implements partitions.Item interface.
func (mn *MinerNode) GetID() string {
	return mn.GetKey()
}

func (mn *MinerNode) numDelegates() int {
	var count int
	for _, pool := range mn.Pools {
		if pool.Status == spenum.Pending || pool.Status == spenum.Active {
			count++
		}
	}
	return count
}

func (mn *MinerNode) save(balances cstate.StateContextI) error {
	if _, err := balances.InsertTrieNode(mn.GetKey(), mn); err != nil {
		return fmt.Errorf("saving miner node: %v", err)
	}
	return nil
}

// Encode implements util.Serializable interface.
func (mn *MinerNode) Encode() []byte {
	var b, err = json.Marshal(mn)
	if err != nil {
		panic(err)
	}
	return b
}

// Decode implements util.Serializable interface.
func (mn *MinerNode) Decode(p []byte) error {
	return json.Unmarshal(p, mn)
}

func (mn *MinerNode) GetNodePools(status string) []*NodePool {
	nodePools := make([]*NodePool, 0)
	for id, pool := range mn.Pools {
		nodePool := NodePool{id, pool}
		if len(status) == 0 || pool.Status.String() == status {
			nodePools = append(nodePools, &nodePool)
		}
	}

	return nodePools
}

func (mn *MinerNode) GetNodePool(poolID string) *NodePool {
	dp, ok := mn.Pools[poolID]
	if !ok {
		return nil
	}

	return &NodePool{poolID, dp}
}
