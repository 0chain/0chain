package minersc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
)

//go:generate msgp -io=false -tests=false -unexported -v

// MinerNode struct that holds information about the registering miner.
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

func GetSharderKey(sid string) datastore.Key {
	return ADDRESS + sid
}

func (mn *MinerNode) GetKey() datastore.Key {
	return ADDRESS + mn.ID
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

func (mn *MinerNode) decodeFromValues(params url.Values) error {
	mn.N2NHost = params.Get("n2n_host")
	mn.ID = params.Get("id")

	if mn.N2NHost == "" || mn.ID == "" {
		return errors.New("URL or ID is not specified")
	}
	return nil
}
