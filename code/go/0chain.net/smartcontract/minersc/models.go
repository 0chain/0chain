package minersc

import (
	"encoding/json"
	"errors"
	"net/url"
	"sync"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	AllMinersKey         = datastore.Key(ADDRESS + encryption.Hash("all_miners"))
	AllShardersKey       = datastore.Key(ADDRESS + encryption.Hash("all_sharders"))
	DKGMinersKey         = datastore.Key(ADDRESS + encryption.Hash("dkg_miners"))
	MinersMPKKey         = datastore.Key(ADDRESS + encryption.Hash("miners_mpk"))
	MagicBlockKey        = datastore.Key(ADDRESS + encryption.Hash("magic_block"))
	GlobalNodeKey        = datastore.Key(ADDRESS + encryption.Hash("global_node"))
	GroupShareOrSignsKey = datastore.Key(ADDRESS + encryption.Hash("group_share_or_signs"))
)

var (
	lockAllMiners sync.Mutex
)

// Phases
const (
	Unknown = iota - 1
	Start
	Contribute
	Share
	Publish
	Wait
)

// Pool status
const (
	ACTIVE    = "ACTIVE"
	PENDING   = "PENDING"
	DELETING  = "DELETING"
	CANDELETE = "CAN DELETE"
)

type phaseFunctions func(balances c_state.StateContextI, gn *globalNode) error

type movePhaseFunctions func(balances c_state.StateContextI, pn *PhaseNode, gn *globalNode) bool

type SimpleNodes = map[string]*SimpleNode

func NewSimpleNodes() SimpleNodes {
	return make(map[string]*SimpleNode)
}

type globalNode struct {
	ViewChange   int64   `json:"view_change"`
	MaxN         int     `json:"max_n"`
	MinN         int     `json:"min_n"`
	TPercent     float64 `json:"t_percent"`
	KPercent     float64 `json:"k_percent"`
	LastRound    int64   `json:"last_round"`
	MaxStake     int64   `json:"max_stake"`
	MinStake     int64   `json:"min_stake"`
	InterestRate float64 `json:"interest_rate"`
}

func (gn *globalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *globalNode) Decode(input []byte) error {
	return json.Unmarshal(input, gn)
}

func (gn *globalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *globalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

//MinerNode struct that holds information about the registering miner
type MinerNode struct {
	*SimpleNode `json:"simple_miner"`
	Pending     map[string]*sci.DelegatePool `json:"pending"`
	Active      map[string]*sci.DelegatePool `json:"active"`
	Deleting    map[string]*sci.DelegatePool `json:"deleting"`
}

func NewMinerNode() *MinerNode {
	mn := &MinerNode{SimpleNode: &SimpleNode{}}
	mn.Pending = make(map[string]*sci.DelegatePool)
	mn.Active = make(map[string]*sci.DelegatePool)
	mn.Deleting = make(map[string]*sci.DelegatePool)
	return mn
}

func (mn *MinerNode) getKey() datastore.Key {
	return datastore.Key(ADDRESS + mn.ID)
}

func (mn *MinerNode) Encode() []byte {
	buff, _ := json.Marshal(mn)
	return buff
}

func (mn *MinerNode) decodeFromValues(params url.Values) error {
	mn.N2NHost = params.Get("n2n_host")
	mn.ID = params.Get("id")

	if mn.N2NHost == "" || mn.ID == "" {
		return errors.New("BaseURL or ID is not specified")
	}
	return nil

}

func (mn *MinerNode) Decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	sm, ok := objMap["simple_miner"]
	if ok {
		err = mn.SimpleNode.Decode(*sm)
		if err != nil {
			return err
		}
	}
	pending, ok := objMap["pending"]
	if ok {
		err = DecodeDelegatePools(mn.Pending, pending, &ViewChangeLock{})
		if err != nil {
			return err
		}
	}
	active, ok := objMap["active"]
	if ok {
		err = DecodeDelegatePools(mn.Active, active, &ViewChangeLock{})
		if err != nil {
			return err
		}
	}
	deleting, ok := objMap["deleting"]
	if ok {
		err = DecodeDelegatePools(mn.Deleting, deleting, &ViewChangeLock{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (mn *MinerNode) GetHash() string {
	return util.ToHex(mn.GetHashBytes())
}

func (mn *MinerNode) GetHashBytes() []byte {
	return encryption.RawHash(mn.Encode())
}

type SimpleNode struct {
	ID          string  `json:"id"`
	N2NHost     string  `json:"n2n_host"`
	Host        string  `json:"host"`
	Port        int     `json:"port"`
	PublicKey   string  `json:"public_key"`
	ShortName   string  `json:"short_name"`
	Percentage  float64 `json:"percentage"`
	DelegateID  string  `json:"delegate_id"`
	BuildTag    string  `json:"build_tag"`
	TotalStaked int64   `json:"total_stake"`
}

func (smn *SimpleNode) Encode() []byte {
	buff, _ := json.Marshal(smn)
	return buff
}

func (smn *SimpleNode) Decode(input []byte) error {
	return json.Unmarshal(input, smn)
}

type MinerNodes struct {
	Nodes []*MinerNode
}

func (mn *MinerNodes) Encode() []byte {
	buff, _ := json.Marshal(mn)
	return buff
}

func (mn *MinerNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, mn)
	if err != nil {
		return err
	}
	return nil
}

func (mn *MinerNodes) GetHash() string {
	return util.ToHex(mn.GetHashBytes())
}

func (mn *MinerNodes) GetHashBytes() []byte {
	return encryption.RawHash(mn.Encode())
}

type ViewChangeLock struct {
	DeleteViewChangeSet bool          `json:"delete_view_change_set"`
	DeleteVC            int64         `json:"delete_after_view_change"`
	Owner               datastore.Key `json:"owner"`
}

func (vcl *ViewChangeLock) IsLocked(entity interface{}) bool {
	currentVC, ok := entity.(int64)
	if ok {
		return !vcl.DeleteViewChangeSet || currentVC < vcl.DeleteVC
	}
	return true
}

func (vcl *ViewChangeLock) LockStats(entity interface{}) []byte {
	currentVC, ok := entity.(int64)
	if ok {
		p := &poolStat{ViewChangeLock: vcl, CurrentVC: currentVC, Locked: vcl.IsLocked(currentVC)}
		return p.encode()
	}
	return nil
}

type poolStat struct {
	*ViewChangeLock
	CurrentVC int64 `json:"current_view_change"`
	Locked    bool  `json:"locked"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStat) decode(input []byte) error {
	return json.Unmarshal(input, ps)
}

type UserNode struct {
	ID    string               `json:"id"`
	Pools map[string]*poolInfo `json:"pool_map"`
}

func NewUserNode() *UserNode {
	return &UserNode{Pools: make(map[string]*poolInfo)}
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	return json.Unmarshal(input, un)
}

func (un *UserNode) GetKey() datastore.Key {
	return datastore.Key(ADDRESS + un.ID)
}

func (un *UserNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *UserNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

type poolInfo struct {
	PoolID  string `json:"pool_id"`
	MinerID string `json:"miner_id"`
	Balance int64  `json:"balance"`
}

type deletePool struct {
	MinerID string `json:"id"`
	PoolID  string `json:"pool_id"`
}

func (dp *deletePool) Encode() []byte {
	buff, _ := json.Marshal(dp)
	return buff
}

func (dp *deletePool) Decode(input []byte) error {
	return json.Unmarshal(input, dp)
}

type userPoolsResponse struct {
	*poolInfo
	StakeDiversity float64 `json:"stake_diversity"`
	PoolID         string  `json:"pool_id"`
}

type userResponse struct {
	Pools []*userPoolsResponse `json:"pools"`
}

func (ur *userResponse) Encode() []byte {
	buff, _ := json.Marshal(ur)
	return buff
}

func (ur *userResponse) Decode(input []byte) error {
	return json.Unmarshal(input, ur)
}

type PhaseNode struct {
	Phase        int   `json:"phase"`
	StartRound   int64 `json:"start_round"`
	CurrentRound int64 `json:"current_round"`
	Restarts     int64 `json:"restarts"`
}

func (pn *PhaseNode) GetKey() datastore.Key {
	return datastore.Key(ADDRESS + encryption.Hash("PHASE"))
}

func (pn *PhaseNode) Encode() []byte {
	buff, _ := json.Marshal(pn)
	return buff
}

func (pn *PhaseNode) Decode(input []byte) error {
	return json.Unmarshal(input, pn)
}

func HasPool(pools map[string]*sci.DelegatePool, poolID datastore.Key) bool {
	pool := pools[poolID]
	return pool != nil
}

func AddPool(pools map[string]*sci.DelegatePool, pool *sci.DelegatePool) error {
	if HasPool(pools, pool.ID) {
		return common.NewError("can't add pool", "miner node already has pool")
	}
	pools[pool.ID] = pool
	return nil
}

func DeletePool(pools map[string]*sci.DelegatePool, poolID datastore.Key) error {
	if HasPool(pools, poolID) {
		return common.NewError("can't delete pool", "pool doesn't exist")
	}
	delete(pools, poolID)
	return nil
}

func DecodeDelegatePools(pools map[string]*sci.DelegatePool, poolsBytes *json.RawMessage, tokenlock tokenpool.TokenLockInterface) error {
	var rawMessagesPools map[string]*json.RawMessage
	err := json.Unmarshal(*poolsBytes, &rawMessagesPools)
	if err != nil {
		return err
	}
	for _, raw := range rawMessagesPools {
		tempPool := sci.NewDelegatePool()
		err = tempPool.Decode(*raw, tokenlock)
		if err != nil {
			return err
		}
		err = AddPool(pools, tempPool)
		if err != nil {
			return err
		}
	}
	return nil
}

type DKGMinerNodes struct {
	SimpleNodes
	T              int
	K              int
	N              int
	RevealedShares map[string]int
}

func NewDKGMinerNodes() *DKGMinerNodes {
	return &DKGMinerNodes{SimpleNodes: NewSimpleNodes(), RevealedShares: make(map[string]int)}
}

func (dmn *DKGMinerNodes) Encode() []byte {
	buff, _ := json.Marshal(dmn)
	return buff
}

func (dmn *DKGMinerNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, dmn)
	if err != nil {
		return err
	}
	return nil
}

func (dmn *DKGMinerNodes) GetHash() string {
	return util.ToHex(dmn.GetHashBytes())
}

func (dmn *DKGMinerNodes) GetHashBytes() []byte {
	return encryption.RawHash(dmn.Encode())
}
