package minersc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/provider"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/statecache"
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
		SimpleNode: &SimpleNode{
			Provider: provider.Provider{},
		},
		StakePool: stakepool.NewStakePool(),
	}

	mn.Minter = cstate.MinterMiner
	return mn
}

//nolint
func (m *MinerNode) clone() *MinerNode {
	clone := &MinerNode{
		SimpleNode: &SimpleNode{},
		StakePool:  &stakepool.StakePool{},
	}
	*clone.SimpleNode = *m.SimpleNode
	*clone.StakePool = *m.StakePool
	clone.StakePool.Pools = make(map[string]*stakepool.DelegatePool)
	for k, v := range m.StakePool.Pools {
		dp := *v
		clone.StakePool.Pools[k] = &dp
	}
	return clone
}

func (m *MinerNode) Clone() statecache.Value {
	v, err := m.MarshalMsg(nil)
	if err != nil {
		panic(fmt.Sprintf("could not marshal miner node: %v", err))
	}

	newMn := NewMinerNode()
	_, err = newMn.UnmarshalMsg(v)
	if err != nil {
		panic(fmt.Sprintf("could not unmarshal miner node: %v", err))
	}

	return newMn
}

func (m *MinerNode) CopyFrom(v interface{}) bool {
	if mn, ok := v.(*MinerNode); ok {
		cmn := mn.Clone().(*MinerNode)
		*m = *cmn
		return true
	}
	return false
}

// swagger:model NodePool
type NodePool struct {
	PoolID string `json:"pool_id"`
	*stakepool.DelegatePool
}

func GetSharderKey(sid string) datastore.Key {
	return provider.GetKey(sid)
}

func (mn *MinerNode) GetKey() datastore.Key {
	return provider.GetKey(mn.ID)
}

//nolint
func (mn *MinerNode) numDelegates() int {
	var count int
	for _, pool := range mn.Pools {
		if pool.Status == spenum.Pending || pool.Status == spenum.Active {
			count++
		}
	}
	return count
}

func (mn *MinerNode) Save(p spenum.Provider, id string, balances cstate.StateContextI) error {
	return mn.save(balances)
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
	orderedPoolIds := mn.OrderedPoolIds()
	for _, id := range orderedPoolIds {
		pool := mn.Pools[id]
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
