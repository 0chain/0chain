package minersc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
)

//msgp:ignore MinerNode
//go:generate msgp -io=false -tests=false -unexported -v

//
// miner / sharder
//

// MinerNode struct that holds information about the registering miner.
type MinerNode struct {
	*SimpleNode `json:"simple_miner"`
	Pending     map[string]*sci.DelegatePool `json:"pending,omitempty"`
	Active      map[string]*sci.DelegatePool `json:"active,omitempty"`
	Deleting    map[string]*sci.DelegatePool `json:"deleting,omitempty"`
}

func NewMinerNode() *MinerNode {
	mn := &MinerNode{SimpleNode: &SimpleNode{}}
	mn.Pending = make(map[string]*sci.DelegatePool)
	mn.Active = make(map[string]*sci.DelegatePool)
	mn.Deleting = make(map[string]*sci.DelegatePool)
	return mn
}

func getMinerKey(mid string) string {
	return ADDRESS + mid
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

	charge = state.Balance(float64(fees) * mn.ServiceCharge)
	rest = fees - charge
	return
}

func (mn *MinerNode) numDelegates() int {
	return len(mn.Pending) + len(mn.Active)
}

func (mn *MinerNode) numActiveDelegates() int {
	return len(mn.Active)
}

func (mn *MinerNode) save(balances cstate.StateContextI) error {
	if _, err := balances.InsertTrieNode(mn.GetKey(), mn); err != nil {
		return fmt.Errorf("saving miner node: %v", err)
	}
	return nil
}

// minerNodeDecode represents a MinerNode that use ViewChangeLock as tokenLockInterface
// it is for decoding MinerNode bytes
type minerNodeDecode struct {
	*SimpleNode `json:"simple_miner"`
	Pending     map[string]*delegatePool `json:"pending,omitempty"`
	Active      map[string]*delegatePool `json:"active,omitempty"`
	Deleting    map[string]*delegatePool `json:"deleting,omitempty"`
}

func newMinerNodeDecode() *minerNodeDecode {
	mn := &minerNodeDecode{SimpleNode: &SimpleNode{}}
	mn.Pending = make(map[string]*delegatePool)
	mn.Active = make(map[string]*delegatePool)
	mn.Deleting = make(map[string]*delegatePool)
	return mn
}

func newDecodeFromMinerNode(mn *MinerNode) *minerNodeDecode {
	n := newMinerNodeDecode()
	n.SimpleNode = mn.SimpleNode
	for k, pl := range mn.Pending {
		n.Pending[k] = newDelegatePool(pl)
	}

	for k, pl := range mn.Active {
		n.Active[k] = newDelegatePool(pl)
	}

	for k, pl := range mn.Deleting {
		n.Deleting[k] = newDelegatePool(pl)
	}

	return n
}

func (n *minerNodeDecode) toMinerNode() *MinerNode {
	mn := NewMinerNode()
	mn.SimpleNode = n.SimpleNode
	mn.Pending = make(map[string]*sci.DelegatePool, len(n.Pending))
	for k, pl := range n.Pending {
		mn.Pending[k] = pl.toDelegatePool()
	}

	mn.Active = make(map[string]*sci.DelegatePool, len(n.Active))
	for k, pl := range n.Active {
		mn.Active[k] = pl.toDelegatePool()
	}

	mn.Deleting = make(map[string]*sci.DelegatePool, len(n.Deleting))
	for k, pl := range n.Deleting {
		mn.Deleting[k] = pl.toDelegatePool()
	}

	return mn
}

// delegatePool is for decoding delegate pool with ViewChangeLock as the TokenLockInterface
type delegatePool struct {
	*sci.PoolStats `json:"stats"`
	*ZcnTokenPool  `json:"pool"`
}

// toDelegatePool converts the pool struct to *delegatePool
func (dpl *delegatePool) toDelegatePool() *sci.DelegatePool {
	dp := sci.NewDelegatePool()
	dp.PoolStats = dpl.PoolStats
	dp.ZcnPool = dpl.ZcnPool
	dp.TokenLockInterface = dpl.ViewChangeLock
	return dp
}

func newDelegatePool(dp *sci.DelegatePool) *delegatePool {
	pl := &delegatePool{
		PoolStats: dp.PoolStats,
		ZcnTokenPool: &ZcnTokenPool{
			ZcnPool: dp.ZcnPool,
		},
	}

	if dp.TokenLockInterface != nil {
		pl.ViewChangeLock = dp.TokenLockInterface.(*ViewChangeLock)
	}
	return pl
}

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

func (mn *MinerNode) decodeFromValues(params url.Values) error {
	mn.N2NHost = params.Get("n2n_host")
	mn.ID = params.Get("id")

	if mn.N2NHost == "" || mn.ID == "" {
		return errors.New("URL or ID is not specified")
	}
	return nil
}

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
