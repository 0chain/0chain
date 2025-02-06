package minersc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	config2 "0chain.net/core/config"
	"0chain.net/core/encryption"
	"0chain.net/core/util/entitywrapper"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

//msgp:ignore GlobalNode

//go:generate msgp -io=false -tests=false -unexported -v

type globalNodeBase globalNodeV1

func (gnb *globalNodeBase) CommitChangesTo(e entitywrapper.EntityI) {
	switch v := e.(type) {
	case *globalNodeV1:
		*v = globalNodeV1(*gnb)
	case *globalNodeV2:
		v.ApplyBaseChanges(*gnb)
	}
}

type GlobalNode struct {
	entitywrapper.Wrapper
}

func (gn *GlobalNode) TypeName() string {
	return "global_node"
}

func (gn *GlobalNode) UnmarshalMsg(data []byte) ([]byte, error) {
	return gn.UnmarshalMsgType(data, gn.TypeName())
}

func (gn *GlobalNode) UnmarshalJSON(data []byte) error {
	return gn.UnmarshalJSONType(data, gn.TypeName())
}

func (gn *GlobalNode) Msgsize() (s int) {
	return gn.Entity().Msgsize()
}

func (gn *GlobalNode) GetVersion() string {
	return gn.Entity().GetVersion()
}

func (gn *GlobalNode) MustBase() *globalNodeBase {
	b, ok := gn.Base().(*globalNodeBase)
	if !ok {
		logging.Logger.Panic("invalid global node base type")
	}
	return b
}

func (gn *GlobalNode) MustUpdateBase(f func(base *globalNodeBase) error) error {
	return gn.UpdateBase(func(eb entitywrapper.EntityBaseI) error {
		b, ok := eb.(*globalNodeBase)
		if !ok {
			logging.Logger.Panic("invalid global node base type")
		}
		//nolint:errcheck
		f(b)
		return nil
	})
}

func (gn *GlobalNode) GetVCPhaseRounds() map[Phase]int64 {
	switch gn.GetVersion() {
	case entitywrapper.DefaultOriginVersion:
		logging.Logger.Debug("[mvc] get vc phase rounds v1:")
		return PhaseRounds
	case "v2":
		g2 := gn.Entity().(*globalNodeV2)
		pr := make(map[Phase]int64, len(g2.VCPhaseRounds))
		for i, v := range g2.VCPhaseRounds {
			pr[Phase(i)] = int64(v)
		}
		logging.Logger.Debug("[mvc] get vc phase rounds v2:", zap.Any("phase rounds", pr))
		return pr
	default:
		panic("unknown global node version")
	}
}

type globalNodeV1 struct {
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
	MinStake            currency.Coin `json:"min_stake"`
	MinStakePerDelegate currency.Coin `json:"min_stake_per_delegate"`
	HealthCheckPeriod   time.Duration `json:"health_check_period"`

	// Reward rate.
	RewardRate float64 `json:"reward_rate"`
	// ShareRatio is miner/block sharders rewards ratio.
	ShareRatio float64 `json:"share_ratio"`
	// BlockReward
	BlockReward currency.Coin `json:"block_reward"`
	// MaxCharge can be set by a generator.
	MaxCharge float64 `json:"max_charge"` // %
	// Epoch is number of rounds to decline interests and rewards.
	Epoch int64 `json:"epoch"`
	// RewardDeclineRate is ratio of epoch rewards declining.
	RewardDeclineRate float64 `json:"reward_decline_rate"`
	// miner delegates to get paid each round when paying fees and rewards
	NumMinerDelegatesRewarded int `json:"num_miner_delegates_rewarded"`
	// sharders rewarded each round
	NumShardersRewarded int `json:"num_sharders_rewarded"`
	// sharder delegates to get paid each round when paying fees and rewards
	NumSharderDelegatesRewarded int `json:"num_sharder_delegates_rewarded"`
	// PrevMagicBlock keeps previous magic block to make Miner SC more stable.
	// In case latestFinalizedMagicBlock of a miner works incorrect. We are
	// using this previous MB or latestFinalizedMagicBlock for genesis block.
	PrevMagicBlock *block.MagicBlock `json:"prev_magic_block"`

	// If viewchange is false then this will be used to pay interests and rewards to miner/sharders.
	RewardRoundFrequency int64          `json:"reward_round_frequency"`
	OwnerId              string         `json:"owner_id"`
	CooldownPeriod       int64          `json:"cooldown_period"`
	Cost                 map[string]int `json:"cost"`
}

func (gn *globalNodeV1) GetVersion() string {
	return entitywrapper.DefaultOriginVersion
}

func (gn *globalNodeV1) GetBase() entitywrapper.EntityBaseI {
	b := globalNodeBase(*gn)
	return &b
}

func (gn *globalNodeV1) MigrateFrom(e entitywrapper.EntityI) error {
	// nothing to migrate as this is original version of the global node
	return nil
}

func (gn *globalNodeV1) InitVersion() {
	// do nothing cause it's original version of global node
}

func (gn *GlobalNode) readConfig(balances cstate.StateContextI) (err error) {
	const pfx = "smart_contracts.minersc."
	gnb := globalNodeV1{}
	gnb.MinStake, err = currency.ParseZCN(config2.SmartContractConfig.GetFloat64(pfx + SettingName[MinStake]))
	if err != nil {
		return err
	}

	gnb.MinStakePerDelegate, err = currency.ParseZCN(config2.SmartContractConfig.GetFloat64(pfx + SettingName[MinStakePerDelegate]))
	if err != nil {
		return err
	}

	gnb.MaxStake, err = currency.ParseZCN(config2.SmartContractConfig.GetFloat64(pfx + SettingName[MaxStake]))
	if err != nil {
		return err
	}
	gnb.HealthCheckPeriod = config2.SmartContractConfig.GetDuration(pfx + SettingName[HealthCheckPeriod])

	gnb.MaxN = config2.SmartContractConfig.GetInt(pfx + SettingName[MaxN])
	gnb.MinN = config2.SmartContractConfig.GetInt(pfx + SettingName[MinN])
	gnb.TPercent = config2.SmartContractConfig.GetFloat64(pfx + SettingName[TPercent])
	gnb.KPercent = config2.SmartContractConfig.GetFloat64(pfx + SettingName[KPercent])
	gnb.XPercent = config2.SmartContractConfig.GetFloat64(pfx + SettingName[XPercent])
	gnb.MaxS = config2.SmartContractConfig.GetInt(pfx + SettingName[MaxS])
	gnb.MinS = config2.SmartContractConfig.GetInt(pfx + SettingName[MinS])
	gnb.MaxDelegates = config2.SmartContractConfig.GetInt(pfx + SettingName[MaxDelegates])
	gnb.RewardRoundFrequency = config2.SmartContractConfig.GetInt64(pfx + SettingName[RewardRoundFrequency])
	gnb.RewardRate = config2.SmartContractConfig.GetFloat64(pfx + SettingName[RewardRate])
	gnb.ShareRatio = config2.SmartContractConfig.GetFloat64(pfx + SettingName[ShareRatio])
	gnb.NumMinerDelegatesRewarded = config2.SmartContractConfig.GetInt(pfx + SettingName[NumMinerDelegatesRewarded])
	gnb.NumShardersRewarded = config2.SmartContractConfig.GetInt(pfx + SettingName[NumShardersRewarded])
	gnb.NumSharderDelegatesRewarded = config2.SmartContractConfig.GetInt(pfx + SettingName[NumSharderDelegatesRewarded])
	gnb.BlockReward, err = currency.ParseZCN(config2.SmartContractConfig.GetFloat64(pfx + SettingName[BlockReward]))
	if err != nil {
		return err
	}
	gnb.MaxCharge = config2.SmartContractConfig.GetFloat64(pfx + SettingName[MaxCharge])
	gnb.Epoch = config2.SmartContractConfig.GetInt64(pfx + SettingName[Epoch])
	gnb.RewardDeclineRate = config2.SmartContractConfig.GetFloat64(pfx + SettingName[RewardDeclineRate])
	gnb.OwnerId = config2.SmartContractConfig.GetString(pfx + SettingName[OwnerId])
	gnb.CooldownPeriod = config2.SmartContractConfig.GetInt64(pfx + SettingName[CooldownPeriod])
	gnb.Cost = config2.SmartContractConfig.GetStringMapInt(pfx + "cost")

	gn.SetEntity(&gnb)
	return nil
}

func (gn *GlobalNode) validate() error {
	gnb := gn.MustBase()
	if gnb.MinN < 1 {
		return fmt.Errorf("min_n is too small: %d", gnb.MinN)
	}
	if gnb.MaxN < gnb.MinN {
		return fmt.Errorf("max_n is less than min_n: %d < %d",
			gnb.MaxN, gnb.MinN)
	}

	if gnb.MinS < 1 {
		return fmt.Errorf("min_s is too small: %d", gnb.MinS)
	}
	if gnb.MaxS < gnb.MinS {
		return fmt.Errorf("max_s is less than min_s: %d < %d",
			gnb.MaxS, gnb.MinS)
	}

	if gnb.MaxDelegates <= 0 {
		return fmt.Errorf("max_delegates is too small: %d", gnb.MaxDelegates)
	}
	if gnb.NumSharderDelegatesRewarded < 0 {
		return fmt.Errorf("%s cannot be negative: %d",
			NumSharderDelegatesRewarded.String(), gnb.NumSharderDelegatesRewarded)
	}
	if gnb.NumMinerDelegatesRewarded < 0 {
		return fmt.Errorf("%s cannot be negative: %d",
			NumMinerDelegatesRewarded.String(), gnb.NumMinerDelegatesRewarded)
	}
	if gnb.NumShardersRewarded < 0 {
		return fmt.Errorf("%s cannot be negative: %d",
			NumShardersRewarded.String(), gnb.NumShardersRewarded)
	}
	return nil
}

func (gn *GlobalNode) getConfigMap() (config2.StringMap, error) {
	var out config2.StringMap
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
		if info.ConfigType == config2.CurrencyCoin {
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

	if isVCPRounds(key.String()) {
		value, err := gn.getVCRounds(key.String())
		return value, err
	}

	gnb := gn.MustBase()

	switch key {
	case MinStake:
		return gnb.MinStake, nil
	case MinStakePerDelegate:
		return gnb.MinStakePerDelegate, nil
	case HealthCheckPeriod:
		return gnb.HealthCheckPeriod, nil
	case MaxStake:
		return gnb.MaxStake, nil
	case MaxN:
		return gnb.MaxN, nil
	case MinN:
		return gnb.MinN, nil
	case TPercent:
		return gnb.TPercent, nil
	case KPercent:
		return gnb.KPercent, nil
	case XPercent:
		return gnb.XPercent, nil
	case MaxS:
		return gnb.MaxS, nil
	case MinS:
		return gnb.MinS, nil
	case MaxDelegates:
		return gnb.MaxDelegates, nil
	case RewardRoundFrequency:
		return gnb.RewardRoundFrequency, nil
	case RewardRate:
		return gnb.RewardRate, nil
	case NumMinerDelegatesRewarded:
		return gnb.NumMinerDelegatesRewarded, nil
	case NumShardersRewarded:
		return gnb.NumShardersRewarded, nil
	case NumSharderDelegatesRewarded:
		return gnb.NumSharderDelegatesRewarded, nil
	case ShareRatio:
		return gnb.ShareRatio, nil
	case BlockReward:
		return gnb.BlockReward, nil
	case MaxCharge:
		return gnb.MaxCharge, nil
	case Epoch:
		return gnb.Epoch, nil
	case RewardDeclineRate:
		return gnb.RewardDeclineRate, nil
	case OwnerId:
		return gnb.OwnerId, nil
	case CooldownPeriod:
		return gnb.CooldownPeriod, nil
	default:
		logging.Logger.Debug("Setting not implemented", zap.String("key", key.String()))
		return nil, errors.New("Setting not implemented")
	}
}

// The prevMagicBlock from the global node (saved on previous VC) or LFMB of
// the balances if missing (genesis case);
func (gn *GlobalNode) prevMagicBlock(balances cstate.StateContextI) (
	pmb *block.MagicBlock) {
	gnb := gn.MustBase()

	if gnb.PrevMagicBlock != nil {
		return gnb.PrevMagicBlock
	}
	return balances.GetChainCurrentMagicBlock()
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

func (gn *GlobalNode) epochDecline() {
	// keep existing value for logs
	gn.MustUpdateBase(func(gnb *globalNodeBase) error {
		var rr = gnb.RewardRate
		// decline the value
		gnb.RewardRate = gnb.RewardRate * (1.0 - gnb.RewardDeclineRate)

		// log about the epoch declining
		logging.Logger.Info("miner sc: epoch decline",
			zap.Int64("round", gnb.LastRound),
			zap.Float64("reward_decline_rate", gnb.RewardDeclineRate),
			zap.Float64("prev_reward_rate", rr),
			zap.Float64("new_reward_rate", gnb.RewardRate),
		)
		return nil
	})
}

// calculate miner/block sharders fees
func (gn *GlobalNode) splitByShareRatio(fees currency.Coin) (
	miner, sharders currency.Coin, err error) {
	gnb := gn.MustBase()
	fFees, err := fees.Float64()
	if err != nil {
		return 0, 0, err
	}
	miner, err = currency.Float64ToCoin(fFees * gnb.ShareRatio)
	if err != nil {
		return 0, 0, err
	}
	sharders, err = currency.MinusCoin(fees, miner)
	return
}

func (gn *GlobalNode) setLastRound(round int64) {
	gn.MustUpdateBase(func(gnb *globalNodeBase) error {
		gnb.LastRound = round
		if round%gnb.Epoch == 0 {
			gn.epochDecline()
		}
		return nil
	})
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

func (gn *GlobalNode) Clone() statecache.Value {
	// cg := &GlobalNode{}
	// *cg = *gn

	// if gn.PrevMagicBlock != nil {
	// 	cg.PrevMagicBlock = gn.PrevMagicBlock.Clone()
	// }
	v, err := gn.MarshalMsg(nil)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal GlobalNode: %v", err))
	}

	ng := &GlobalNode{}
	_, err = ng.UnmarshalMsg(v)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal GlobalNode: %v", err))
	}

	return ng
}

func (gn *GlobalNode) CopyFrom(v interface{}) bool {
	cg, ok := v.(*GlobalNode)
	if !ok {
		return false
	}

	ccg := cg.Clone().(*GlobalNode)
	*gn = *ccg
	return true
}

func NewGlobalNode(owner string, cost map[string]int) *GlobalNode {
	gnV2 := &globalNodeV2{
		globalNodeV1: globalNodeV1{
			OwnerId: owner,
			Cost:    cost,
		},
	}

	var gn = &GlobalNode{}
	gn.SetEntity(gnV2)

	return gn
}
