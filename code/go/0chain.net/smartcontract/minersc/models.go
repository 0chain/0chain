package minersc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"

	"0chain.net/smartcontract/zbig"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

//go:generate msgp -io=false -tests=false -v

var validate *validator.Validate

func init() {
	validate = validator.New()
}

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
)

func globalKeyHash(name string) datastore.Key {
	return datastore.Key(ADDRESS + encryption.Hash(name))
}

func NewSimpleNodes() SimpleNodes {
	return make(map[string]*SimpleNode)
}

// not thread safe
// swagger:model SimpleNodes
type SimpleNodes map[string]*SimpleNode

// Pooler represents a pool interface
type Pooler interface {
	HasNode(id string) bool
}

func (sns SimpleNodes) reduce(limit int, xPercent float64, pmbrss int64, pmbnp Pooler) (maxNodes int) {
	var pmbNodes, newNodes, selectedNodes []*SimpleNode

	// separate previous mb miners and new miners from dkg miners list
	for _, sn := range sns {
		if pmbnp != nil && pmbnp.HasNode(sn.ID) {
			pmbNodes = append(pmbNodes, sn)
			continue
		}
		newNodes = append(newNodes, sn)
	}

	// sort pmb nodes by total stake: desc
	sort.SliceStable(pmbNodes, func(i, j int) bool {
		if pmbNodes[i].TotalStaked == pmbNodes[j].TotalStaked {
			return pmbNodes[i].ID < pmbNodes[j].ID
		}

		return pmbNodes[i].TotalStaked > pmbNodes[j].TotalStaked
	})

	// calculate max nodes count for next mb
	maxNodes = min(limit, len(sns))

	// get number of nodes from previous mb that are required to be part of next mb
	x := min(len(pmbNodes), int(math.Ceil(xPercent*float64(maxNodes))))
	y := maxNodes - x

	// select first x nodes from pmb miners
	selectedNodes = pmbNodes[:x]

	// add rest of the pmb miners into new miners list
	newNodes = append(newNodes, pmbNodes[x:]...)
	sort.SliceStable(newNodes, func(i, j int) bool {
		if newNodes[i].TotalStaked == newNodes[j].TotalStaked {
			return newNodes[i].ID < newNodes[j].ID
		}

		return newNodes[i].TotalStaked > newNodes[j].TotalStaked
	})

	if len(newNodes) <= y {
		// less than allowed nodes remaining
		selectedNodes = append(selectedNodes, newNodes...)

	} else if y > 0 {
		// more than allowed nodes remaining

		// find the range of nodes with equal stakes, start (s), end (e)
		s, e := 0, len(newNodes)
		stake := newNodes[y-1].TotalStaked
		for i, sn := range newNodes {
			if s == 0 && sn.TotalStaked == stake {
				s = i
			} else if sn.TotalStaked < stake {
				e = i
				break
			}
		}

		// select nodes that don't have equal stake
		selectedNodes = append(selectedNodes, newNodes[:s]...)

		// resolve equal stake condition by randomly selecting nodes with equal stake
		newNodes = newNodes[s:e]
		for _, j := range rand.New(rand.NewSource(pmbrss)).Perm(len(newNodes)) {
			if len(selectedNodes) < maxNodes {
				selectedNodes = append(selectedNodes, newNodes[j])
			}
		}

	}

	// update map with selected nodes
	for k := range sns {
		delete(sns, k)
	}
	for _, sn := range selectedNodes {
		sns[sn.ID] = sn
	}

	return maxNodes
}

//
// global
//

type GlobalNode struct {
	ViewChange   int64   `json:"view_change"`
	MaxN         int     `json:"max_n"`         // } miners limits
	MinN         int     `json:"min_n"`         // }
	MaxS         int     `json:"max_s"`         // } sharders limits
	MinS         int     `json:"min_s"`         // }
	MaxDelegates int     `json:"max_delegates"` // } limited by the SC
	TPercent     float64 `json:"t_percent"`
	KPercent     float64 `json:"k_percent"`
	XPercent     float64 `json:"x_percent"`
	LastRound    int64   `json:"last_round"`
	// MaxStake boundary of SC.
	MaxStake currency.Coin `json:"max_stake"`
	// MinStake boundary of SC.
	MinStake currency.Coin `json:"min_stake"`

	// Reward rate.
	RewardRate float64 `json:"reward_rate"`
	// ShareRatio is miner/block sharders rewards ratio.
	ShareRatio float64 `json:"share_ratio"`
	// BlockReward
	BlockReward currency.Coin `json:"block_reward"`
	// MaxCharge can be set by a generator.
	MaxCharge zbig.BigRat `json:"max_charge"` // %
	// Epoch is number of rounds to decline interests and rewards.
	Epoch int64 `json:"epoch"`
	// RewardDeclineRate is ratio of epoch rewards declining.
	RewardDeclineRate float64 `json:"reward_decline_rate"`
	// MaxMint is minting boundary for SC.
	MaxMint currency.Coin `json:"max_mint"`

	// PrevMagicBlock keeps previous magic block to make Miner SC more stable.
	// In case latestFinalizedMagicBlock of a miner works incorrect. We are
	// using this previous MB or latestFinalizedMagicBlock for genesis block.
	PrevMagicBlock *block.MagicBlock `json:"prev_magic_block"`

	// Minted tokens by SC.
	Minted currency.Coin `json:"minted"`

	// If viewchange is false then this will be used to pay interests and rewards to miner/sharders.
	RewardRoundFrequency int64          `json:"reward_round_frequency"`
	OwnerId              string         `json:"owner_id"`
	CooldownPeriod       int64          `json:"cooldown_period"`
	Cost                 map[string]int `json:"cost"`
}

func (gn *GlobalNode) readConfig() (err error) {
	const pfx = "smart_contracts.minersc."
	gn.MinStake, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(pfx + SettingName[MinStake]))
	if err != nil {
		return
	}
	gn.MaxStake, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(pfx + SettingName[MaxStake]))
	if err != nil {
		return
	}
	gn.MaxN = config.SmartContractConfig.GetInt(pfx + SettingName[MaxN])
	gn.MinN = config.SmartContractConfig.GetInt(pfx + SettingName[MinN])
	gn.TPercent = config.SmartContractConfig.GetFloat64(pfx + SettingName[TPercent])
	gn.KPercent = config.SmartContractConfig.GetFloat64(pfx + SettingName[KPercent])
	gn.XPercent = config.SmartContractConfig.GetFloat64(pfx + SettingName[XPercent])
	gn.MaxS = config.SmartContractConfig.GetInt(pfx + SettingName[MaxS])
	gn.MinS = config.SmartContractConfig.GetInt(pfx + SettingName[MinS])
	gn.MaxDelegates = config.SmartContractConfig.GetInt(pfx + SettingName[MaxDelegates])
	gn.RewardRoundFrequency = config.SmartContractConfig.GetInt64(pfx + SettingName[RewardRoundFrequency])
	gn.RewardRate = config.SmartContractConfig.GetFloat64(pfx + SettingName[RewardRate])
	gn.ShareRatio = config.SmartContractConfig.GetFloat64(pfx + SettingName[ShareRatio])
	gn.BlockReward, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(pfx + SettingName[BlockReward]))
	if err != nil {
		return
	}
	gn.MaxCharge = *zbig.BigRatFromFloat64(config.SmartContractConfig.GetFloat64(pfx + SettingName[MaxCharge]))
	gn.Epoch = config.SmartContractConfig.GetInt64(pfx + SettingName[Epoch])
	gn.RewardDeclineRate = config.SmartContractConfig.GetFloat64(pfx + SettingName[RewardDeclineRate])
	gn.MaxMint, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(pfx + SettingName[MaxMint]))
	if err != nil {
		return
	}
	gn.OwnerId = config.SmartContractConfig.GetString(pfx + SettingName[OwnerId])
	gn.CooldownPeriod = config.SmartContractConfig.GetInt64(pfx + SettingName[CooldownPeriod])
	gn.Cost = config.SmartContractConfig.GetStringMapInt(pfx + "cost")
	return nil
}

func (gn *GlobalNode) validate() error {
	if gn.MinN < 1 {
		return fmt.Errorf("min_n is too small: %d", gn.MinN)
	}
	if gn.MaxN < gn.MinN {
		return fmt.Errorf("max_n is less than min_n: %d < %d",
			gn.MaxN, gn.MinN)
	}

	if gn.MinS < 1 {
		return fmt.Errorf("min_s is too small: %d", gn.MinS)
	}
	if gn.MaxS < gn.MinS {
		return fmt.Errorf("max_s is less than min_s: %d < %d",
			gn.MaxS, gn.MinS)
	}

	if gn.MaxDelegates <= 0 {
		return fmt.Errorf("max_delegates is too small: %d", gn.MaxDelegates)
	}
	return nil
}

func (gn *GlobalNode) getConfigMap() (smartcontract.StringMap, error) {
	var out smartcontract.StringMap
	out.Fields = make(map[string]string)
	for _, key := range SettingName {
		info, ok := Settings[key]
		if !ok {
			return out, fmt.Errorf("SettingName %s not found in Settings", key)
		}
		iSetting, err := gn.Get(info.Setting)
		if err != nil {
			return out, err
		}
		if info.ConfigType == smartcontract.CurrencyCoin {
			sbSetting, ok := iSetting.(currency.Coin)
			if !ok {
				return out, fmt.Errorf("%s key not implemented as state.balance", key)
			}
			iSetting = float64(sbSetting) / x10
		}
		out.Fields[key] = fmt.Sprintf("%v", iSetting)
	}
	return out, nil
}

func (gn *GlobalNode) Get(key Setting) (interface{}, error) {
	if isCost(key.String()) {
		value, _ := gn.getCost(key.String())
		return value, nil
	}

	switch key {
	case MinStake:
		return gn.MinStake, nil
	case MaxStake:
		return gn.MaxStake, nil
	case MaxN:
		return gn.MaxN, nil
	case MinN:
		return gn.MinN, nil
	case TPercent:
		return gn.TPercent, nil
	case KPercent:
		return gn.KPercent, nil
	case XPercent:
		return gn.XPercent, nil
	case MaxS:
		return gn.MaxS, nil
	case MinS:
		return gn.MinS, nil
	case MaxDelegates:
		return gn.MaxDelegates, nil
	case RewardRoundFrequency:
		return gn.RewardRoundFrequency, nil
	case RewardRate:
		return gn.RewardRate, nil
	case ShareRatio:
		return gn.ShareRatio, nil
	case BlockReward:
		return gn.BlockReward, nil
	case MaxCharge:
		return gn.MaxCharge, nil
	case Epoch:
		return gn.Epoch, nil
	case RewardDeclineRate:
		return gn.RewardDeclineRate, nil
	case MaxMint:
		return gn.MaxMint, nil
	case OwnerId:
		return gn.OwnerId, nil
	case CooldownPeriod:
		return gn.CooldownPeriod, nil
	default:
		return nil, errors.New("Setting not implemented")
	}
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
		logging.Logger.Error("empty miners mpks keys")
		return
	}

	var pmb = gn.prevMagicBlock(balances)

	for id := range mpks.Mpks {
		if pmb.Miners.HasNode(id) {
			return true
		}
	}

	logging.Logger.Debug("has no prev miner in MPKs", zap.Int64("prev_mb_round", pmb.StartingRound))
	return // false, hasn't
}

// has previous miner in given GSoS
func (gn *GlobalNode) hasPrevMinerInGSoS(gsos *block.GroupSharesOrSigns,
	balances cstate.StateContextI) (has bool) {

	if len(gsos.Shares) == 0 {
		logging.Logger.Error("empty sharder or sign keys")
		return
	}

	var pmb = gn.prevMagicBlock(balances)

	for id := range gsos.Shares {
		if pmb.Miners.HasNode(id) {
			return true
		}
	}

	logging.Logger.Debug("has no prev miner in GSoS",
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

// hasPrevSharderInList checks if there are nodes in previous magic block sharder list
func hasPrevSharderInList(prevMB *block.MagicBlock, nodes []*MinerNode) bool {
	for _, n := range nodes {
		if prevMB.Sharders.HasNode(n.ID) {
			return true
		}
	}

	return false
}

// rankedPrevSharders receives a list of ranked sharders and extract sharder of
// previous MB preserving order. The given list not modified.
func rankedPrevSharders(prevMB *block.MagicBlock, list []*MinerNode) []*MinerNode {
	prev := make([]*MinerNode, 0, len(list))

	for _, node := range list {
		if prevMB.Sharders.HasNode(node.ID) {
			prev = append(prev, node)
		}
	}

	return prev
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
	var rr = gn.RewardRate
	// decline the value
	gn.RewardRate = gn.RewardRate * (1.0 - gn.RewardDeclineRate)

	// log about the epoch declining
	logging.Logger.Info("miner sc: epoch decline",
		zap.Int64("round", gn.LastRound),
		zap.Float64("reward_decline_rate", gn.RewardDeclineRate),
		zap.Float64("prev_reward_rate", rr),
		zap.Float64("new_reward_rate", gn.RewardRate),
	)
}

// calculate miner/block sharders fees
func (gn *GlobalNode) splitByShareRatio(fees currency.Coin) (
	miner, sharders currency.Coin, err error) {

	fFees, err := fees.Float64()
	if err != nil {
		return 0, 0, err
	}
	miner, err = currency.Float64ToCoin(fFees * gn.ShareRatio)
	if err != nil {
		return 0, 0, err
	}
	sharders, err = currency.MinusCoin(fees, miner)
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
		return fmt.Errorf("saving global node: %v", err)
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
		err = fmt.Errorf("unknown node type: %q", nts)
	}
	return
}

// swagger:model SimpleNodeGeolocation
type SimpleNodeGeolocation struct {
	Latitude  zbig.BigRat `json:"latitude" msg:"latitude,extension"`
	Longitude zbig.BigRat `json:"longitude" msg:"longitude,extension"`
}

// swagger:model SimpleNode
type SimpleNode struct {
	ID          string                `json:"id" validate:"hexadecimal,len=64"`
	N2NHost     string                `json:"n2n_host"`
	Host        string                `json:"host"`
	Port        int                   `json:"port"`
	Geolocation SimpleNodeGeolocation `json:"geolocation"`
	Path        string                `json:"path"`
	PublicKey   string                `json:"public_key"`
	ShortName   string                `json:"short_name"`
	BuildTag    string                `json:"build_tag"`
	TotalStaked currency.Coin         `json:"total_stake"`
	Delete      bool                  `json:"delete"`

	// settings and statistic

	// NodeType used for delegate pools statistic.
	NodeType NodeType `json:"node_type,omitempty"`

	// LastHealthCheck used to check for active node
	LastHealthCheck common.Timestamp `json:"last_health_check"`

	// Status will be set either node.NodeStatusActive or node.NodeStatusInactive
	Status int `json:"-" msg:"-"`

	//LastSettingUpdateRound will be set to round number when settings were updated
	LastSettingUpdateRound int64 `json:"last_setting_update_round"`
}

func (smn *SimpleNode) Encode() []byte {
	buff, _ := json.Marshal(smn)
	return buff
}

func (smn *SimpleNode) Decode(input []byte) error {
	return json.Unmarshal(input, smn)
}

func (smn *SimpleNode) Validate() error {
	return validate.Struct(smn)
}

type ViewChangeLock struct {
	DeleteViewChangeSet bool   `json:"delete_view_change_set"`
	DeleteVC            int64  `json:"delete_after_view_change"`
	Owner               string `json:"owner"`
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

// swagger:model
type delegatePoolStat struct {
	ID         datastore.Key `json:"id"`
	Balance    currency.Coin `json:"balance"`
	Reward     currency.Coin `json:"reward"`      // uncollected reread
	RewardPaid currency.Coin `json:"reward_paid"` // total reward all time
	Status     string        `json:"status"`
}

type deletePool struct {
	MinerID string `json:"id"`
}

func (dp *deletePool) Encode() []byte {
	buff, _ := json.Marshal(dp)
	return buff
}

func (dp *deletePool) Decode(input []byte) error {
	return json.Unmarshal(input, dp)
}

// swagger:model PhaseNode
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

// swagger:model DKGMinerNodes
type DKGMinerNodes struct {
	MinN     int     `json:"min_n"`
	MaxN     int     `json:"max_n"`
	TPercent float64 `json:"t_percent"`
	KPercent float64 `json:"k_percent"`

	SimpleNodes    `json:"simple_nodes"`
	T              int             `json:"t"`
	K              int             `json:"k"`
	N              int             `json:"n"`
	XPercent       float64         `json:"x_percent"`
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
	dkgmn.XPercent = gn.XPercent
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

func simpleNodesKeys(sns SimpleNodes) (ks []string) {
	ks = make([]string, 0, len(sns))
	for k := range sns {
		ks = append(ks, k)
	}
	return
}

// reduce method checks boundaries and if final, reduces the
// list to adhere to the limits (min_n, max_n) and conditions
func (dkgmn *DKGMinerNodes) reduceNodes(
	final bool,
	gn *GlobalNode,
	balances cstate.StateContextI) (err error) {

	var n = len(dkgmn.SimpleNodes)

	if n < dkgmn.MinN {
		return fmt.Errorf("too few miners: %d, want at least: %d", n, dkgmn.MinN)
	}

	if !gn.hasPrevDKGMiner(dkgmn.SimpleNodes, balances) {
		return fmt.Errorf("missing miner from previous set, n: %d, list: %s",
			n, simpleNodesKeys(dkgmn.SimpleNodes))
	}

	if final {
		simpleNodes := make(SimpleNodes)
		for k, v := range dkgmn.SimpleNodes {
			simpleNodes[k] = v
		}
		var pmbrss int64
		var pmbnp *node.Pool
		pmb := balances.GetLastestFinalizedMagicBlock()
		if pmb != nil {
			pmbrss = pmb.RoundRandomSeed
			if pmb.MagicBlock != nil {
				pmbnp = pmb.MagicBlock.Miners
			}
		}
		simpleNodes.reduce(gn.MaxN, gn.XPercent, pmbrss, pmbnp)
		dkgmn.SimpleNodes = simpleNodes
	}

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
func getMinersList(state cstate.QueryStateContextI) (*MinerNodes, error) {
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
		return common.NewError("update_all_miners_list_failed", err.Error())
	}
	return nil
}

// getDKGMinersList gets dkg miners list
func getDKGMinersList(state cstate.CommonStateContextI) (*DKGMinerNodes, error) {
	dkgMiners := NewDKGMinerNodes()
	err := state.GetTrieNode(DKGMinersKey, dkgMiners)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		return NewDKGMinerNodes(), nil
	}

	return dkgMiners, nil
}

// updateDKGMinersList update the dkg miners list
func updateDKGMinersList(state cstate.StateContextI, dkgMiners *DKGMinerNodes) error {
	logging.Logger.Info("update dkg miners list", zap.Int("len", len(dkgMiners.SimpleNodes)))
	_, err := state.InsertTrieNode(DKGMinersKey, dkgMiners)
	return err
}

func getMinersMPKs(state cstate.CommonStateContextI) (*block.Mpks, error) {
	mpks := block.NewMpks()
	err := state.GetTrieNode(MinersMPKKey, mpks)
	if err != nil {
		return nil, err
	}

	return mpks, nil
}

func updateMinersMPKs(state cstate.StateContextI, mpks *block.Mpks) error {
	_, err := state.InsertTrieNode(MinersMPKKey, mpks)
	return err
}

func getMagicBlock(state cstate.CommonStateContextI) (*block.MagicBlock, error) {
	magicBlock := block.NewMagicBlock()
	err := state.GetTrieNode(MagicBlockKey, magicBlock)
	if err != nil {
		return nil, err
	}

	return magicBlock, nil
}

func updateMagicBlock(state cstate.StateContextI, magicBlock *block.MagicBlock) error {
	_, err := state.InsertTrieNode(MagicBlockKey, magicBlock)
	return err
}

func getGroupShareOrSigns(state cstate.CommonStateContextI) (*block.GroupSharesOrSigns, error) {
	var gsos = block.NewGroupSharesOrSigns()
	err := state.GetTrieNode(GroupShareOrSignsKey, gsos)
	if err != nil {
		return nil, err
	}

	return gsos, nil
}

func updateGroupShareOrSigns(state cstate.StateContextI, gsos *block.GroupSharesOrSigns) error {
	_, err := state.InsertTrieNode(GroupShareOrSignsKey, gsos)
	return err
}

// getShardersKeepList returns the sharder list
func getShardersKeepList(balances cstate.CommonStateContextI) (*MinerNodes, error) {
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

func getNodesList(balances cstate.CommonStateContextI, key datastore.Key) (*MinerNodes, error) {
	nodesList := &MinerNodes{}
	err := balances.GetTrieNode(key, nodesList)
	if err != nil {
		return nil, err
	}

	return nodesList, nil
}

// quick fix: localhost check + duplicate check
// TODO: remove this after more robust challenge based node addtion/health_check is added
func quickFixDuplicateHosts(nn *MinerNode, allNodes []*MinerNode) error {
	localhost := regexp.MustCompile(`^(?:(?:https|http)\:\/\/)?(?:localhost|127\.0\.0\.1)(?:\:\d+)?(?:\/.*)?$`)
	host := strings.TrimSpace(nn.Host)
	n2nhost := strings.TrimSpace(nn.N2NHost)
	port := nn.Port
	if n2nhost == "" || localhost.MatchString(n2nhost) {
		return fmt.Errorf("invalid n2nhost: '%v'", n2nhost)
	}
	if host == "" || localhost.MatchString(host) {
		host = n2nhost
	}
	for _, n := range allNodes {
		if n.ID != nn.ID && n2nhost == n.N2NHost && n.Port == port {
			return fmt.Errorf("n2nhost:port already exists: '%v:%v'", n2nhost, port)
		}
		if n.ID != nn.ID && host == n.Host && n.Port == port {
			return fmt.Errorf("host:port already exists: '%v:%v'", host, port)
		}
	}
	nn.Host, nn.N2NHost, nn.Port = host, n2nhost, port
	return nil
}
