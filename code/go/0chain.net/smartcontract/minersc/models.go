package minersc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"sync"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
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

var (
	AllMinersKey         = globalKeyHash("all_miners")
	AllShardersKey       = globalKeyHash("all_sharders")
	DKGMinersKey         = globalKeyHash("dkg_miners")
	MinersMPKKey         = globalKeyHash("miners_mpk")
	MagicBlockKey        = globalKeyHash("magic_block")
	GlobalNodeKey        = globalKeyHash("global_node")
	GroupShareOrSignsKey = globalKeyHash("group_share_or_signs")
	ShardersKeepKey      = globalKeyHash("sharders_keep")
	PhaseKey             = globalKeyHash("phase")

	lockAllMiners sync.Mutex
)

type (
	phaseFunctions        func(balances cstate.StateContextI, gn *globalNode) error
	movePhaseFunctions    func(balances cstate.StateContextI, pn *PhaseNode, gn *globalNode) bool
	smartContractFunction func(t *transaction.Transaction, inputData []byte, gn *globalNode, balances cstate.StateContextI) (string, error)
	SimpleNodes           = map[string]*SimpleNode
)

func globalKeyHash(name string) datastore.Key {
	return datastore.Key(ADDRESS + encryption.Hash(name))
}

func NewSimpleNodes() SimpleNodes {
	return make(map[string]*SimpleNode)
}

//
// global
//

type globalNode struct {
	ViewChange   int64   `json:"view_change"`
	MaxN         int     `json:"max_n"`         // } miners limits
	MinN         int     `json:"min_n"`         // }
	MaxS         int     `json:"max_s"`         // } sharders limits
	MinS         int     `json:"min_s"`         // }
	MaxDelegates int     `json:"max_delegates"` // } limited by the SC
	TPercent     float64 `json:"t_percent"`
	KPercent     float64 `json:"k_percent"`
	LastRound    int64   `json:"last_round"`
	// MaxStake boundary of SC.
	MaxStake state.Balance `json:"max_stake"`
	// MinStake boundary of SC.
	MinStake state.Balance `json:"min_stake"`

	// Stake interests.
	InterestRate float64 `json:"interest_rate"`
	// Reward rate.
	RewardRate float64 `json:"reward_rate"`
	// ShareRatio is miner/block sharders rewards ratio.
	ShareRatio float64 `json:"share_ratio"`
	// BlockReward
	BlockReward state.Balance `json:"block_reward"`
	// MaxCharge can be set by a generator.
	MaxCharge float64 `json:"max_charge"` // %
	// Epoch is number of rounds to decline interests and rewards.
	Epoch int64 `json:"epoch"`
	// RewardDeclineRate is ratio of epoch rewards declining.
	RewardDeclineRate float64 `json:"reward_decline_rate"`
	// InterestDeclineRate is ratio of epoch interests declining.
	InterestDeclineRate float64 `json:"interest_decline_rate"`
	// MaxMint is minting boundary for SC.
	MaxMint state.Balance `json:"max_mint"`

	// Minted tokens by SC.
	Minted state.Balance `json:"minted"`
}

func (gn *globalNode) canMint() bool {
	return gn.Minted < gn.MaxMint
}

func (gn *globalNode) epochDecline() {
	// keep existing value for logs
	var ir, rr = gn.InterestRate, gn.RewardRate
	// decline the value
	gn.RewardRate = gn.RewardRate * (1.0 - gn.RewardDeclineRate)
	gn.InterestRate = gn.InterestRate * (1.0 - gn.InterestDeclineRate)

	// log about the epoch declining
	Logger.Info("miner sc: epoch decline",
		zap.Int64("round", gn.LastRound),
		zap.Float64("reward_decline_rate", gn.RewardDeclineRate),
		zap.Float64("interest_decline_rate", gn.InterestDeclineRate),
		zap.Float64("prev_reward_rate", rr),
		zap.Float64("prev_interest_rate", ir),
		zap.Float64("new_reward_rate", gn.RewardRate),
		zap.Float64("new_interest_rate", gn.InterestRate),
	)
}

// calculate miner/block sharders fees
func (gn *globalNode) splitByShareRatio(fees state.Balance) (
	miner, sharders state.Balance) {

	miner = state.Balance(float64(fees) * gn.ShareRatio)
	sharders = fees - miner
	return
}

func (gn *globalNode) setLastRound(round int64) {
	gn.LastRound = round
	if round%gn.Epoch == 0 {
		gn.epochDecline()
	}
}

func (gn *globalNode) save(balances cstate.StateContextI) (err error) {
	if _, err = balances.InsertTrieNode(GlobalNodeKey, gn); err != nil {
		return fmt.Errorf("saving global node: %v", err)
	}
	return
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

//
// miner / sharder
//

//MinerNode struct that holds information about the registering miner
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

func getMinerKey(mid string) datastore.Key {
	return datastore.Key(ADDRESS + mid)
}

func getSharderKey(sid string) datastore.Key {
	return datastore.Key(ADDRESS + sid)
}

func (mn *MinerNode) getKey() datastore.Key {
	return datastore.Key(ADDRESS + mn.ID)
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

func (mn *MinerNode) save(balances cstate.StateContextI) (err error) {
	if _, err = balances.InsertTrieNode(mn.getKey(), mn); err != nil {
		return fmt.Errorf("saving miner node: %v", err)
	}
	return
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
	var objMap map[string]json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	sm, ok := objMap["simple_miner"]
	if ok {
		err = mn.SimpleNode.Decode(sm)
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

type Stat struct {
	// for miner (totals)
	GeneratorRewards state.Balance `json:"generator_rewards,omitempty"`
	GeneratorFees    state.Balance `json:"generator_fees,omitempty"`
	// for sharder (totals)
	SharderRewards state.Balance `json:"sharder_rewards,omitempty"`
	SharderFees    state.Balance `json:"sharder_fees,omitempty"`
}

type SimpleNode struct {
	ID          string `json:"id"`
	N2NHost     string `json:"n2n_host"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	PublicKey   string `json:"public_key"`
	ShortName   string `json:"short_name"`
	BuildTag    string `json:"build_tag"`
	TotalStaked int64  `json:"total_stake"`

	// settings and statistic

	// DelegateWallet grabs node rewards (excluding stake rewards) and
	// controls the node setting. If the DelegateWallet hasn't been provided,
	// then node ID used (for genesis nodes, for example).
	DelegateWallet string `json:"delegate_wallet"` // ID
	// ServiceChange is % that miner node grabs where it's generator.
	ServiceCharge float64 `json:"service_charge"` // %
	// NumberOfDelegates is max allowed number of delegate pools.
	NumberOfDelegates int `json:"number_of_delegates"`
	// MinStake allowed by node.
	MinStake state.Balance `json:"min_stake"`
	// MaxStake allowed by node.
	MaxStake state.Balance `json:"max_stake"`

	// Stat contains node statistic.
	Stat Stat `json:"stat"`
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

func (mn *MinerNodes) FindNodeById(id string) *MinerNode {
	for _, minerNode := range mn.Nodes {
		if minerNode.ID == id {
			return minerNode
		}
	}
	return nil
}

type ViewChangeLock struct {
	DeleteViewChangeSet bool          `json:"delete_view_change_set"`
	DeleteVC            int64         `json:"delete_after_view_change"`
	Owner               datastore.Key `json:"owner"`
}

func (vcl *ViewChangeLock) IsLocked(entity interface{}) bool {
	if entity == nil {
		return false
	}
	currentVC, ok := entity.(int64)
	if ok {
		return !vcl.DeleteViewChangeSet || currentVC < vcl.DeleteVC
	}
	if currentVC == 0 {
		return false // forced unlock
	}
	return true
}

func (vcl *ViewChangeLock) LockStats(entity interface{}) []byte {
	currentVC, ok := entity.(int64)
	if ok {
		p := &poolStat{
			ViewChangeLock: vcl,
			CurrentVC:      currentVC,
			Locked:         vcl.IsLocked(currentVC),
		}
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

func (un *UserNode) save(balances cstate.StateContextI) (err error) {

	if len(un.Pools) > 0 {
		if _, err = balances.InsertTrieNode(un.GetKey(), un); err != nil {
			return fmt.Errorf("saving user node: %v", err)
		}
	} else {
		if _, err = balances.DeleteTrieNode(un.GetKey()); err != nil {
			return fmt.Errorf("deleting user node: %v", err)
		}
	}

	return
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
	PoolID  string        `json:"pool_id"`
	MinerID string        `json:"miner_id"`
	Balance state.Balance `json:"balance"`
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
	return PhaseKey
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

func DecodeDelegatePools(pools map[string]*sci.DelegatePool,
	poolsBytes json.RawMessage, tokenlock tokenpool.TokenLockInterface) error {

	var rawMessagesPools map[string]json.RawMessage
	err := json.Unmarshal(poolsBytes, &rawMessagesPools)
	if err != nil {
		return err
	}
	for _, raw := range rawMessagesPools {
		tempPool := sci.NewDelegatePool()
		err = tempPool.Decode(raw, tokenlock)
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
	MinN     int     `json:"min_n"`
	MaxN     int     `json:"max_n"`
	TPercent float64 `json:"t_percent"`
	KPercent float64 `json:"k_percent"`

	SimpleNodes    `json:"simple_nodes"`
	T              int            `json:"t"`
	K              int            `json:"k"`
	N              int            `json:"n"`
	RevealedShares map[string]int `json:"revealed_shares"`
	// S              int         // number of sharders
	// Sharders       SimpleNodes // sharders
}

func (dkgmn *DKGMinerNodes) setConfigs(gn *globalNode) {
	dkgmn.MinN = gn.MinN
	dkgmn.MaxN = gn.MaxN
	dkgmn.TPercent = gn.TPercent
	dkgmn.KPercent = gn.KPercent
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func (dkgmn *DKGMinerNodes) calculateTKN(gn *globalNode, n int) {
	dkgmn.setConfigs(gn)
	var m = min(dkgmn.MaxN, n)
	dkgmn.N = n
	dkgmn.K = int(math.Ceil(dkgmn.KPercent * float64(m)))
	dkgmn.T = int(math.Ceil(dkgmn.TPercent * float64(m)))
}

func (dkgmn *DKGMinerNodes) reduce(n int) int {
	var list []*SimpleNode
	for _, node := range dkgmn.SimpleNodes {
		list = append(list, node)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalStaked > list[j].TotalStaked
	})
	list = list[:n]
	dkgmn.SimpleNodes = make(SimpleNodes)
	for _, node := range list {
		dkgmn.SimpleNodes[node.ID] = node
	}
	return dkgmn.MaxN
}

func (dkgmn *DKGMinerNodes) recalculateTKN(final bool) (err error) {
	var n = len(dkgmn.SimpleNodes)
	// check the lower boundary
	if n < dkgmn.MinN {
		return fmt.Errorf("to few miners: %d, want at least: %d", n, dkgmn.MinN)
	}
	// check upper boundary for a final recalculation
	if final && n > dkgmn.MaxN {
		n = dkgmn.reduce(dkgmn.MaxN)
	}
	var m = min(dkgmn.MaxN, n)
	dkgmn.N = n
	dkgmn.K = int(math.Ceil(dkgmn.KPercent * float64(m)))
	dkgmn.T = int(math.Ceil(dkgmn.TPercent * float64(m)))
	return
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
