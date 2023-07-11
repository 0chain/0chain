package stakepool

import (
	cstate "0chain.net/chaincore/chain/state"
	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp -v -io=false -tests=false

// NodeStakePool structs that use msgp.Raw like json.RawMessage so we
// can encode/decode the stake pool parts without losing data.
type NodeStakePool struct {
	ID         string `json:"-" msg:"-"`
	*msgp.Raw  `msg:"SimpleNode"`
	*StakePool `json:"stake_pool"`
}

func NewNodeStakePool() *NodeStakePool {
	return &NodeStakePool{
		StakePool: NewStakePool(),
	}
}

func providerKey(id string) string {
	return "provider:" + id
}

func (nsp *NodeStakePool) Get(balances cstate.CommonStateContextI) error {
	k := providerKey(nsp.ID)
	return balances.GetTrieNode(k, nsp)
}

func (nsp *NodeStakePool) Save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(providerKey(nsp.ID), nsp)
	return err
}
