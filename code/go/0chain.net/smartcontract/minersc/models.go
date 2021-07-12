package minersc

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"sync"

	"github.com/0chain/gosdk/core/common/errors"

	"0chain.net/chaincore/block"
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

// Phase number.
type Phase int

// known phases
const (
	Unknown Phase = iota - 1
	Start
	Contribute
	Share
	Publish
	Wait
)

func (p Phase) String() string {
	switch p {
	case Unknown:
		return "unknown"
	case Start:
		return "start"
	case Contribute:
		return "contribute"
	case Share:
		return "share"
	case Publish:
		return "publish"
	case Wait:
		return "wait"
	default:
	}
	return fmt.Sprintf("Phase<%d>", int(p))
}

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
	phaseFunctions func(balances cstate.StateContextI, gn *GlobalNode) (
		err error)
	movePhaseFunctions func(balances cstate.StateContextI, pn *PhaseNode,
		gn *GlobalNode) error
	smartContractFunction func(t *transaction.Transaction, inputData []byte,
		gn *GlobalNode, balances cstate.StateContextI) (string, error)

	SimpleNodes = map[string]*SimpleNode
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

// The Config represents GlobalNode with phases rounds.
// It used in SC /config handler as response.
type Config struct {
	GlobalNode

	StartRounds      int64 `json:"start_rounds"`
	ContributeRounds int64 `json:"contribute_rounds"`
	ShareRounds      int64 `json:"share_rounds"`
	PublishRounds    int64 `json:"publish_rounds"`
	WaitRounds       int64 `json:"wait_rounds"`
}

type GlobalNode struct {
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

	// PrevMagicBlock keeps previous magic block to make Miner SC more stable.
	// In case latestFinalizedMagicBlock of a miner works incorrect. We are
	// using this previous MB or latestFinalizedMagicBlock for genesis block.
	PrevMagicBlock *block.MagicBlock `json:"prev_magic_block"`

	// Minted tokens by SC.
	Minted state.Balance `json:"minted"`

	// If viewchange is false then this will be used to pay interests and rewards to miner/sharders.
	RewardRoundFrequency int64 `json:"reward_round_frequency"`
}

// The prevMagicBlock from the global node (saved on previous VC) or LFMB of
// the balances if missing (genesis case);
func (gn *GlobalNode) prevMagicBlock(balances cstate.StateContextI) (
	pmb *block.MagicBlock) {

	if gn.PrevMagicBlock != nil {
		return gn.PrevMagicBlock
	}
	return balances.GetLastestFinalizedMagicBlock().MagicBlock
}

// has previous miner in all miners list
func (gn *GlobalNode) hasPrevMiner(miners *MinerNodes,
	balances cstate.StateContextI) (has bool) {

	var pmb = gn.prevMagicBlock(balances)

	for _, mn := range miners.Nodes {
		if pmb.Miners.HasNode(mn.ID) {
			return true
		}
	}

	return // false, hasn't
}

// has previous miner in given MPKs
func (gn *GlobalNode) hasPrevMinerInMPKs(mpks *block.Mpks,
	balances cstate.StateContextI) (has bool) {

	if len(mpks.Mpks) == 0 {
		Logger.Error("empty miners mpks keys")
		return
	}

	var pmb = gn.prevMagicBlock(balances)

	for id := range mpks.Mpks {
		if pmb.Miners.HasNode(id) {
			return true
		}
	}

	Logger.Debug("has no prev miner in MPKs", zap.Int64("prev_mb_round", pmb.StartingRound))
	return // false, hasn't
}

// has previous miner in given GSoS
func (gn *GlobalNode) hasPrevMinerInGSoS(gsos *block.GroupSharesOrSigns,
	balances cstate.StateContextI) (has bool) {

	if len(gsos.Shares) == 0 {
		Logger.Error("empty sharder or sign keys")
		return
	}

	var pmb = gn.prevMagicBlock(balances)

	for id := range gsos.Shares {
		if pmb.Miners.HasNode(id) {
			return true
		}
	}

	Logger.Debug("has no prev miner in GSoS",
		zap.Int64("prev_mb_round", pmb.StartingRound),
		zap.Int("mb miner len", len(pmb.Miners.Nodes)),
	)
	return // false, hasn't
}

// of DKG miners
func (gn *GlobalNode) hasPrevDKGMiner(dkgmns SimpleNodes,
	balances cstate.StateContextI) (has bool) {

	var pmb = gn.prevMagicBlock(balances)

	for id := range dkgmns {
		if pmb.Miners.HasNode(id) {
			return true
		}
	}

	return // false, hasn't
}

// of DKG miners sorted list
func (gn *GlobalNode) hasPrevDKGMinerInList(list []*SimpleNode,
	balances cstate.StateContextI) (has bool) {

	var pmb = gn.prevMagicBlock(balances)

	for _, node := range list {
		if pmb.Miners.HasNode(node.ID) {
			return true
		}
	}

	return // false, hasn't
}

// Receive list of ranked miners and extract miners of previous MB preserving
// order. The given list not modified.
func (gn *GlobalNode) rankedPrevDKGMiners(list []*SimpleNode,
	balances cstate.StateContextI) (prev []*SimpleNode) {

	var pmb = gn.prevMagicBlock(balances)
	prev = make([]*SimpleNode, 0, len(list))

	for _, node := range list {
		if pmb.Miners.HasNode(node.ID) {
			prev = append(prev, node)
		}
	}

	return // false, hasn't
}

//
func (gn *GlobalNode) hasPrevSharderInList(list []*MinerNode,
	balances cstate.StateContextI) (has bool) {

	var pmb = gn.prevMagicBlock(balances)

	for _, node := range list {
		if pmb.Sharders.HasNode(node.ID) {
			return true
		}
	}

	return // false, hasn't
}

// Receive list of ranked sharders and extract sharder of previous MB preserving
// order. The given list not modified.
func (gn *GlobalNode) rankedPrevSharders(list []*MinerNode,
	balances cstate.StateContextI) (prev []*MinerNode) {

	var pmb = gn.prevMagicBlock(balances)
	prev = make([]*MinerNode, 0, len(list))

	for _, node := range list {
		if pmb.Sharders.HasNode(node.ID) {
			prev = append(prev, node)
		}
	}

	return // false, hasn't
}

// has previous sharder in sharders keep list
func (gn *GlobalNode) hasPrevShader(sharders *MinerNodes,
	balances cstate.StateContextI) (has bool) {

	var pmb = gn.prevMagicBlock(balances)

	for _, sn := range sharders.Nodes {
		if pmb.Sharders.HasNode(sn.ID) {
			return true
		}
	}

	return // false, hasn't
}

func (gn *GlobalNode) canMint() bool {
	return gn.Minted < gn.MaxMint
}

func (gn *GlobalNode) epochDecline() {
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
func (gn *GlobalNode) splitByShareRatio(fees state.Balance) (
	miner, sharders state.Balance) {

	miner = state.Balance(float64(fees) * gn.ShareRatio)
	sharders = fees - miner
	return
}

func (gn *GlobalNode) setLastRound(round int64) {
	gn.LastRound = round
	if round%gn.Epoch == 0 {
		gn.epochDecline()
	}
}

func (gn *GlobalNode) save(balances cstate.StateContextI) (err error) {
	if _, err = balances.InsertTrieNode(GlobalNodeKey, gn); err != nil {
		return errors.Newf("", "saving global node: %v", err)
	}
	return
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *GlobalNode) Decode(input []byte) error {
	return json.Unmarshal(input, gn)
}

func (gn *GlobalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *GlobalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

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

func (mn *MinerNode) numActiveDelegates() int {
	return len(mn.Active)
}

func (mn *MinerNode) save(balances cstate.StateContextI) error {
	//var key datastore.Key
	//if key, err = balances.InsertTrieNode(mn.getKey(), mn); err != nil {
	if _, err := balances.InsertTrieNode(mn.getKey(), mn); err != nil {
		return errors.Newf("", "saving miner node: %v", err)
	}

	//Logger.Debug("MinerNode save successfully",
	//	zap.String("path", encryption.Hash(mn.getKey())),
	//	zap.String("new root key", hex.EncodeToString([]byte(key))))
	return nil
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

// NodeType used in pools statistic.
type NodeType int

// known node types of the Miner SC
const (
	NodeTypeUnknown NodeType = iota // unknown (zero)
	NodeTypeMiner                   // miner node
	NodeTypeSharder                 // sharder node
)

// String converted NodeType to string.
func (nt NodeType) String() string {
	switch nt {
	case NodeTypeUnknown:
		return "unknown"
	case NodeTypeMiner:
		return "miner"
	case NodeTypeSharder:
		return "sharder"
	default:
		return fmt.Sprintf("unknown node type: %d", int(nt))
	}
}

// MarshalJSON converts NodeType to appropriate JSON
// value represented as string.
func (nt NodeType) MarshalJSON() (p []byte, err error) {
	return json.Marshal(nt.String())
}

// UnmarsalJSON converts JSON value back to NodeType.
func (nt *NodeType) UnmarshalJSON(p []byte) (err error) {
	var nts string
	if err = json.Unmarshal(p, &nts); err != nil {
		return
	}
	switch nts {
	case "unknown":
		(*nt) = NodeTypeUnknown
	case "miner":
		(*nt) = NodeTypeMiner
	case "sharder":
		(*nt) = NodeTypeSharder
	default:
		err = errors.Newf("", "unknown node type: %q", nts)
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
	Path        string `json:"path"`
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

	// NodeType used for delegate pools statistic.
	NodeType NodeType `json:"node_type,omitempty"`

	// LastHealthCheck used to check for active node
	LastHealthCheck common.Timestamp `json:"last_health_check"`
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

type delegatePoolStat struct {
	ID           datastore.Key `json:"id"`            // pool ID
	Balance      state.Balance `json:"balance"`       //
	InterestPaid state.Balance `json:"interest_paid"` //
	RewardPaid   state.Balance `json:"reward_paid"`   //
	Status       string        `json:"status"`        //
	High         state.Balance `json:"high"`          // }
	Low          state.Balance `json:"low"`           // }
}

func newDelegatePoolStat(dp *sci.DelegatePool) (dps *delegatePoolStat) {
	dps = new(delegatePoolStat)
	dps.ID = dp.ID
	dps.Balance = dp.Balance
	dps.InterestPaid = dp.InterestPaid
	dps.RewardPaid = dp.RewardPaid
	dps.Status = dp.Status
	dps.High = dp.High
	dps.Low = dp.Low
	return
}

// A userPools represents response for user pools requests.
type userPools struct {
	Pools map[string]map[string][]*delegatePoolStat `json:"pools"`
}

func newUserPools() (ups *userPools) {
	ups = new(userPools)
	ups.Pools = make(map[string]map[string][]*delegatePoolStat)
	return
}

// UserNode keeps references to all user's pools.
type UserNode struct {
	ID    string                            `json:"id"`       // client ID
	Pools map[datastore.Key][]datastore.Key `json:"pool_map"` // node_id -> [pool_id]
}

func NewUserNode() *UserNode {
	return &UserNode{Pools: make(map[datastore.Key][]datastore.Key)}
}

func (un *UserNode) save(balances cstate.StateContextI) (err error) {

	if len(un.Pools) > 0 {
		if _, err = balances.InsertTrieNode(un.GetKey(), un); err != nil {
			return errors.Newf("", "saving user node: %v", err)
		}
	} else {
		if _, err = balances.DeleteTrieNode(un.GetKey()); err != nil {
			return errors.Newf("", "deleting user node: %v", err)
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

type PhaseNode struct {
	Phase        Phase `json:"phase"`
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
		return errors.New("can't add pool", "miner node already has pool")
	}
	pools[pool.ID] = pool
	return nil
}

func DeletePool(pools map[string]*sci.DelegatePool, poolID datastore.Key) error {
	if HasPool(pools, poolID) {
		return errors.New("can't delete pool", "pool doesn't exist")
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
	T              int             `json:"t"`
	K              int             `json:"k"`
	N              int             `json:"n"`
	RevealedShares map[string]int  `json:"revealed_shares"`
	Waited         map[string]bool `json:"waited"`

	// StartRound used to filter responses from old MB where sharders comes up.
	StartRound int64 `json:"start_round"`
}

func (dkgmn *DKGMinerNodes) setConfigs(gn *GlobalNode) {
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

// The min_n is checked before the calculateTKN call, so, the n >= min_n.
// The calculateTKN used to set initial T, K, and N.
func (dkgmn *DKGMinerNodes) calculateTKN(gn *GlobalNode, n int) {
	dkgmn.setConfigs(gn)
	var m = min(dkgmn.MaxN, n)
	dkgmn.N = m
	dkgmn.K = int(math.Ceil(dkgmn.KPercent * float64(m)))
	dkgmn.T = int(math.Ceil(dkgmn.TPercent * float64(m)))
}

func (dkgmn *DKGMinerNodes) reduce(n int, gn *GlobalNode,
	balances cstate.StateContextI) int {

	var list []*SimpleNode
	for _, node := range dkgmn.SimpleNodes {
		list = append(list, node)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalStaked > list[j].TotalStaked ||
			list[i].ID < list[j].ID
	})

	if !gn.hasPrevDKGMinerInList(list[:n], balances) {
		var prev = gn.rankedPrevDKGMiners(list, balances)
		if len(prev) == 0 {
			panic("must not happen")
		}
		list[n-1] = prev[0]
	}

	list = list[:n]
	dkgmn.SimpleNodes = make(SimpleNodes)
	for _, node := range list {
		dkgmn.SimpleNodes[node.ID] = node
	}
	return dkgmn.MaxN
}

func simpleNodesKeys(sns SimpleNodes) (ks []string) {
	ks = make([]string, 0, len(sns))
	for k := range sns {
		ks = append(ks, k)
	}
	return
}

// The recalculateTKN reconstructs and checks current DKG list. It never affects
// T, K, and N.
func (dkgmn *DKGMinerNodes) recalculateTKN(final bool, gn *GlobalNode,
	balances cstate.StateContextI) (err error) {

	var n = len(dkgmn.SimpleNodes)

	// check the lower boundary
	if n < dkgmn.MinN {
		return errors.Newf("", "to few miners: %d, want at least: %d", n, dkgmn.MinN)
	}

	if !gn.hasPrevDKGMiner(dkgmn.SimpleNodes, balances) {
		return errors.Newf("", "missing miner from previous set, n: %d, list: %s",
			n, simpleNodesKeys(dkgmn.SimpleNodes))
	}

	// check upper boundary for a final recalculation
	if final && n > dkgmn.MaxN {
		dkgmn.reduce(dkgmn.MaxN, gn, balances)
	}

	// Note: don't recalculate anything here.

	// var m = min(dkgmn.MaxN, n)
	// dkgmn.N = m
	// dkgmn.K = int(math.Ceil(dkgmn.KPercent * float64(m)))
	// dkgmn.T = int(math.Ceil(dkgmn.TPercent * float64(m)))
	return
}

func NewDKGMinerNodes() *DKGMinerNodes {
	return &DKGMinerNodes{
		SimpleNodes:    NewSimpleNodes(),
		RevealedShares: make(map[string]int),
		Waited:         make(map[string]bool),
	}
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

// getMinersList returns miners list
func getMinersList(state cstate.StateContextI) (*MinerNodes, error) {
	minerNodes, err := getNodesList(state, AllMinersKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		return &MinerNodes{}, nil
	}

	return minerNodes, nil
}

func updateMinersList(state cstate.StateContextI, miners *MinerNodes) error {
	if _, err := state.InsertTrieNode(AllMinersKey, miners); err != nil {
		return errors.Wrap(err, "update_all_miners_list_failed")
	}
	return nil
}

// getDKGMinersList gets dkg miners list
func getDKGMinersList(state cstate.StateContextI) (*DKGMinerNodes, error) {
	dkgMiners := NewDKGMinerNodes()
	allMinersDKGBytes, err := state.GetTrieNode(DKGMinersKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		return dkgMiners, nil
	}

	if err := dkgMiners.Decode(allMinersDKGBytes.Encode()); err != nil {
		return nil, errors.Newf("", "decode DKGMinersKey failed, err: %v", err)
	}

	return dkgMiners, nil
}

// updateDKGMinersList update the dkg miners list
func updateDKGMinersList(state cstate.StateContextI, dkgMiners *DKGMinerNodes) error {
	_, err := state.InsertTrieNode(DKGMinersKey, dkgMiners)
	return err
}

func getMinersMPKs(state cstate.StateContextI) (*block.Mpks, error) {
	var mpksBytes util.Serializable
	mpksBytes, err := state.GetTrieNode(MinersMPKKey)
	if err != nil {
		return nil, err
	}

	mpks := block.NewMpks()
	if err := mpks.Decode(mpksBytes.Encode()); err != nil {
		return nil, errors.Newf("", "failed to decode node MinersMPKKey, err: %v", err)
	}

	return mpks, nil
}

func updateMinersMPKs(state cstate.StateContextI, mpks *block.Mpks) error {
	_, err := state.InsertTrieNode(MinersMPKKey, mpks)
	return err
}

func getMagicBlock(state cstate.StateContextI) (*block.MagicBlock, error) {
	magicBlockBytes, err := state.GetTrieNode(MagicBlockKey)
	if err != nil {
		return nil, err
	}

	magicBlock := block.NewMagicBlock()
	if err = magicBlock.Decode(magicBlockBytes.Encode()); err != nil {
		return nil, errors.Newf("", "failed to decode MagicBlockKey, err: %v", err)
	}

	return magicBlock, nil
}

func updateMagicBlock(state cstate.StateContextI, magicBlock *block.MagicBlock) error {
	_, err := state.InsertTrieNode(MagicBlockKey, magicBlock)
	return err
}

func getGroupShareOrSigns(state cstate.StateContextI) (*block.GroupSharesOrSigns, error) {
	groupBytes, err := state.GetTrieNode(GroupShareOrSignsKey)
	if err != nil {
		return nil, err
	}

	var gsos = block.NewGroupSharesOrSigns()
	if err = gsos.Decode(groupBytes.Encode()); err != nil {
		return nil, errors.Newf("", "failed to decode GroupShareOrSignKey, err: %v", err)
	}

	return gsos, nil
}

func updateGroupShareOrSigns(state cstate.StateContextI, gsos *block.GroupSharesOrSigns) error {
	_, err := state.InsertTrieNode(GroupShareOrSignsKey, gsos)
	return err
}

// getShardersKeepList returns the sharder list
func getShardersKeepList(balances cstate.StateContextI) (*MinerNodes, error) {
	sharders, err := getNodesList(balances, ShardersKeepKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return &MinerNodes{}, nil
	}

	return sharders, nil
}

func updateShardersKeepList(state cstate.StateContextI, sharders *MinerNodes) error {
	_, err := state.InsertTrieNode(ShardersKeepKey, sharders)
	return err
}

// getAllShardersKeepList returns the sharder list
func getAllShardersList(balances cstate.StateContextI) (*MinerNodes, error) {
	sharders, err := getNodesList(balances, AllShardersKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return &MinerNodes{}, nil
	}
	return sharders, nil
}

func updateAllShardersList(state cstate.StateContextI, sharders *MinerNodes) error {
	_, err := state.InsertTrieNode(AllShardersKey, sharders)
	return err
}

func getNodesList(balances cstate.StateContextI, key datastore.Key) (*MinerNodes, error) {
	nodesBytes, err := balances.GetTrieNode(key)
	if err != nil {
		return nil, err
	}

	nodesList := &MinerNodes{}
	if err = nodesList.Decode(nodesBytes.Encode()); err != nil {
		return nil, err
	}

	return nodesList, nil
}
