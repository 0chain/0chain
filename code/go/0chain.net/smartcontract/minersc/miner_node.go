package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

////////////msgp:ignore MinerNode dp

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

// calculate service charge from fees
func (mn *MinerNode) splitByServiceCharge(fees state.Balance) (
	charge, rest state.Balance) {

	charge = state.Balance(float64(fees) * mn.Settings.ServiceCharge)
	rest = fees - charge
	return
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

// minerNodeDecode represents a MinerNode that use ViewChangeLock as tokenLockInterface
// it is for decoding MinerNode bytes
type minerNodeDecode struct {
	*SimpleNode          `json:"simple_miner"`
	*stakepool.StakePool `json:"stake_pool"`
}

func newMinerNodeDecode() *minerNodeDecode {
	mn := &minerNodeDecode{SimpleNode: &SimpleNode{}}
	mn.StakePool = stakepool.NewStakePool()
	return mn
}

func newDecodeFromMinerNode(mn *MinerNode) *minerNodeDecode {
	n := newMinerNodeDecode()
	n.SimpleNode = mn.SimpleNode
	n.StakePool = mn.StakePool.Copy()
	return n
}

func (n *minerNodeDecode) toMinerNode() *MinerNode {
	mn := NewMinerNode()
	mn.SimpleNode = n.SimpleNode
	mn.StakePool = n.StakePool.Copy()
	return mn
}

/*
// ZcnTokenPool represents the struct for decoding pool in delegatePool
type ZcnTokenPool struct {
	tokenpool.ZcnPool `json:"pool"`
	*ViewChangeLock   `json:"lock"`
}

func (mn *MinerNode) Encode() []byte {
	buff, _ := json.Marshal(mn)
	return buff
}

// Decode decodes the miner node from bytes
func (mn *MinerNode) Decode(input []byte) error {
	n := newMinerNodeDecode()
	if err := json.Unmarshal(input, n); err != nil {
		return err
	}

	nn := n.toMinerNode()
	*mn = *nn
	return nil
}

func (mn *MinerNode) MarshalMsg(o []byte) ([]byte, error) {
	d := newDecodeFromMinerNode(mn)
	return d.MarshalMsg(o)
}

func (mn *MinerNode) UnmarshalMsg(data []byte) ([]byte, error) {
	d := newMinerNodeDecode()
	o, err := d.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	dmn := d.toMinerNode()
	*mn = *dmn
	return o, nil
}

func (mn *MinerNode) Msgsize() int {
	d := newDecodeFromMinerNode(mn)
	return d.Msgsize()
}
*/
func (mn *MinerNode) decodeFromValues(params url.Values) error {
	mn.N2NHost = params.Get("n2n_host")
	mn.ID = params.Get("id")

	if mn.N2NHost == "" || mn.ID == "" {
		return errors.New("URL or ID is not specified")
	}
	return nil
}

/*
func (mn *MinerNode) orderedActivePools() (ops []*sci.DelegatePool) {
	var keys []string
	for k := range mn.Active {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ops = make([]*sci.DelegatePool, 0, len(keys))
	for _, key := range keys {
		ops = append(ops, mn.Active[key])
	}
	return
}
*/
