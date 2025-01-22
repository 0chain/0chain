package minersc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"0chain.net/core/config"
	"0chain.net/core/util/entitywrapper"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

const x10 float64 = 10 * 1000 * 1000 * 1000

type Setting int

const (
	MinStake            Setting = iota
	MinStakePerDelegate Setting = iota
	MaxStake
	MaxN
	MinN
	TPercent
	KPercent
	XPercent
	MaxS
	MinS
	MaxDelegates
	RewardRoundFrequency
	RewardRate
	ShareRatio
	BlockReward
	MaxCharge
	Epoch
	RewardDeclineRate
	NumMinerDelegatesRewarded
	NumShardersRewarded
	NumSharderDelegatesRewarded
	OwnerId
	CooldownPeriod
	CostAddMiner
	CostAddSharder
	CostDeleteMiner
	CostMinerHealthCheck
	CostSharderHealthCheck
	CostContributeMpk
	CostShareSignsOrShares
	CostWait
	CostUpdateGlobals
	CostUpdateSettings
	CostUpdateMinerSettings
	CostUpdateSharderSettings
	CostPayFees
	CostFeesPaid
	CostMintedTokens
	CostAddToDelegatePool
	CostDeleteFromDelegatePool
	CostSharderKeep
	CostKillMiner
	CostKillSharder
	HealthCheckPeriod
	VCStartRounds
	VCContributeRounds
	VCShareRounds
	VCPublishRounds
	VCWaitRounds
	NumberOfSettings
)

func (s Setting) String() string {
	if s >= NumberOfSettings { // should never happen
		return ""
	}
	return SettingName[s]
}

var (
	SettingName = make([]string, NumberOfSettings)
	Settings    map[string]struct {
		Setting    Setting
		ConfigType config.ConfigType
	}
)

func init() {
	initSettingName()
	initSettings()
}

func initSettingName() {
	SettingName[MinStake] = "min_stake"
	SettingName[MinStakePerDelegate] = "min_stake_per_delegate"
	SettingName[MaxStake] = "max_stake"
	SettingName[MaxN] = "max_n"
	SettingName[MinN] = "min_n"
	SettingName[TPercent] = "t_percent"
	SettingName[KPercent] = "k_percent"
	SettingName[XPercent] = "x_percent"
	SettingName[MaxS] = "max_s"
	SettingName[MinS] = "min_s"
	SettingName[MaxDelegates] = "max_delegates"
	SettingName[RewardRoundFrequency] = "reward_round_frequency"
	SettingName[RewardRate] = "reward_rate"
	SettingName[ShareRatio] = "share_ratio"
	SettingName[BlockReward] = "block_reward"
	SettingName[MaxCharge] = "max_charge"
	SettingName[Epoch] = "epoch"
	SettingName[RewardDeclineRate] = "reward_decline_rate"
	SettingName[NumMinerDelegatesRewarded] = "num_miner_delegates_rewarded"
	SettingName[NumShardersRewarded] = "num_sharders_rewarded"
	SettingName[NumSharderDelegatesRewarded] = "num_sharder_delegates_rewarded"
	SettingName[OwnerId] = "owner_id"
	SettingName[CooldownPeriod] = "cooldown_period"
	SettingName[HealthCheckPeriod] = "health_check_period"
	SettingName[CostAddMiner] = "cost.add_miner"
	SettingName[CostAddSharder] = "cost.add_sharder"
	SettingName[CostDeleteMiner] = "cost.delete_miner"
	SettingName[CostMinerHealthCheck] = "cost.miner_health_check"
	SettingName[CostSharderHealthCheck] = "cost.sharder_health_check"
	SettingName[CostContributeMpk] = strings.ToLower("cost.contributeMpk")
	SettingName[CostShareSignsOrShares] = strings.ToLower("cost.shareSignsOrShares")
	SettingName[CostWait] = "cost.wait"
	SettingName[CostUpdateGlobals] = "cost.update_globals"
	SettingName[CostUpdateSettings] = "cost.update_settings"
	SettingName[CostUpdateMinerSettings] = "cost.update_miner_settings"
	SettingName[CostUpdateSharderSettings] = "cost.update_sharder_settings"
	SettingName[CostPayFees] = strings.ToLower("cost.payFees")
	SettingName[CostFeesPaid] = strings.ToLower("cost.feesPaid")
	SettingName[CostMintedTokens] = strings.ToLower("cost.mintedTokens")
	SettingName[CostAddToDelegatePool] = strings.ToLower("cost.addToDelegatePool")
	SettingName[CostDeleteFromDelegatePool] = strings.ToLower("cost.deleteFromDelegatePool")
	SettingName[CostSharderKeep] = "cost.sharder_keep"
	SettingName[CostKillMiner] = "cost.kill_miner"
	SettingName[CostKillSharder] = "cost.kill_sharder"
	SettingName[VCStartRounds] = "vc_rounds.start"
	SettingName[VCContributeRounds] = "vc_rounds.contribute"
	SettingName[VCShareRounds] = "vc_rounds.share"
	SettingName[VCPublishRounds] = "vc_rounds.publish"
	SettingName[VCWaitRounds] = "vc_rounds.wait"
}

func initSettings() {
	Settings = map[string]struct {
		Setting    Setting
		ConfigType config.ConfigType
	}{
		MinStake.String():                    {MinStake, config.CurrencyCoin},
		MinStakePerDelegate.String():         {MinStakePerDelegate, config.CurrencyCoin},
		MaxStake.String():                    {MaxStake, config.CurrencyCoin},
		MaxN.String():                        {MaxN, config.Int},
		MinN.String():                        {MinN, config.Int},
		TPercent.String():                    {TPercent, config.Float64},
		KPercent.String():                    {KPercent, config.Float64},
		XPercent.String():                    {XPercent, config.Float64},
		MaxS.String():                        {MaxS, config.Int},
		MinS.String():                        {MinS, config.Int},
		MaxDelegates.String():                {MaxDelegates, config.Int},
		RewardRoundFrequency.String():        {RewardRoundFrequency, config.Int64},
		RewardRate.String():                  {RewardRate, config.Float64},
		ShareRatio.String():                  {ShareRatio, config.Float64},
		BlockReward.String():                 {BlockReward, config.CurrencyCoin},
		MaxCharge.String():                   {MaxCharge, config.Float64},
		Epoch.String():                       {Epoch, config.Int64},
		RewardDeclineRate.String():           {RewardDeclineRate, config.Float64},
		NumMinerDelegatesRewarded.String():   {NumMinerDelegatesRewarded, config.Int},
		NumShardersRewarded.String():         {NumShardersRewarded, config.Int},
		NumSharderDelegatesRewarded.String(): {NumSharderDelegatesRewarded, config.Int},
		OwnerId.String():                     {OwnerId, config.Key},
		CooldownPeriod.String():              {CooldownPeriod, config.Int64},
		HealthCheckPeriod.String():           {HealthCheckPeriod, config.Duration},
		CostAddMiner.String():                {CostAddMiner, config.Cost},
		CostAddSharder.String():              {CostAddSharder, config.Cost},
		CostDeleteMiner.String():             {CostDeleteMiner, config.Cost},
		CostMinerHealthCheck.String():        {CostMinerHealthCheck, config.Cost},
		CostSharderHealthCheck.String():      {CostSharderHealthCheck, config.Cost},
		CostContributeMpk.String():           {CostContributeMpk, config.Cost},
		CostShareSignsOrShares.String():      {CostShareSignsOrShares, config.Cost},
		CostWait.String():                    {CostWait, config.Cost},
		CostUpdateGlobals.String():           {CostUpdateGlobals, config.Cost},
		CostUpdateSettings.String():          {CostUpdateSettings, config.Cost},
		CostUpdateMinerSettings.String():     {CostUpdateMinerSettings, config.Cost},
		CostUpdateSharderSettings.String():   {CostUpdateSharderSettings, config.Cost},
		CostPayFees.String():                 {CostPayFees, config.Cost},
		CostFeesPaid.String():                {CostFeesPaid, config.Cost},
		CostMintedTokens.String():            {CostMintedTokens, config.Cost},
		CostAddToDelegatePool.String():       {CostAddToDelegatePool, config.Cost},
		CostDeleteFromDelegatePool.String():  {CostDeleteFromDelegatePool, config.Cost},
		CostSharderKeep.String():             {CostSharderKeep, config.Cost},
		CostKillMiner.String():               {CostKillMiner, config.Cost},
		CostKillSharder.String():             {CostKillSharder, config.Cost},
		VCStartRounds.String():               {VCStartRounds, config.Int},
		VCContributeRounds.String():          {VCContributeRounds, config.Int},
		VCShareRounds.String():               {VCShareRounds, config.Int},
		VCPublishRounds.String():             {VCPublishRounds, config.Int},
		VCWaitRounds.String():                {VCWaitRounds, config.Int},
	}
}

func initGlobalNodeVCRounds(g2 *globalNodeV2) {
	g2.VCPhaseRounds = make([]int, len(PhaseRounds))

	g2.VCPhaseRounds[int(Start)] = int(PhaseRounds[Start])
	g2.VCPhaseRounds[int(Contribute)] = int(PhaseRounds[Contribute])
	g2.VCPhaseRounds[int(Share)] = int(PhaseRounds[Share])
	g2.VCPhaseRounds[int(Publish)] = int(PhaseRounds[Publish])
	g2.VCPhaseRounds[int(Wait)] = int(PhaseRounds[Wait])
}

func (gn *GlobalNode) setInt(balances cstate.StateContextI, key string, change int) error {
	return cstate.WithActivation(balances, "hermes", func() error {
		return gn.setIntBase(key, change)
	}, func() error {
		switch gn.GetVersion() {
		case entitywrapper.DefaultOriginVersion:
			return gn.Update(&globalNodeV2{}, func(e entitywrapper.EntityI) error {
				g2 := e.(*globalNodeV2)
				initGlobalNodeVCRounds(g2)

				// apply changes if any
				if isVCPRounds(key) {
					vcPrefixLen := len(vcRoundsPrefix)
					phase := StringToPhase(key[vcPrefixLen:])
					if phase == Unknown {
						return fmt.Errorf("unknown phase: %s", key)
					}

					g2.VCPhaseRounds[int(phase)] = int(change)
					logging.Logger.Debug("[mvc] migrate VC phase rounds", zap.Any("vc phase rounds", g2.VCPhaseRounds))
				}
				return nil
			})
		case "v2":
			logging.Logger.Debug("[mvc] update VC phase round", zap.String("phase", key), zap.Int("rounds", change))
			return gn.setIntV2(key, change)
		}
		return nil
	})
}

func (gn *GlobalNode) setIntBase(key string, change int) error {
	return gn.MustUpdateBase(func(base *globalNodeBase) error {
		switch Settings[key].Setting {
		case MaxN:
			base.MaxN = change
		case MinN:
			base.MinN = change
		case MaxS:
			base.MaxS = change
		case MinS:
			base.MinS = change
		case MaxDelegates:
			base.MaxDelegates = change
		case NumMinerDelegatesRewarded:
			base.NumMinerDelegatesRewarded = change
		case NumShardersRewarded:
			base.NumShardersRewarded = change
		case NumSharderDelegatesRewarded:
			base.NumSharderDelegatesRewarded = change
		default:
			return fmt.Errorf("key: %v not implemented as int", key)
		}
		return nil
	})
}

func (gn *GlobalNode) setIntV2(key string, change int) error {
	g2 := gn.Entity().(*globalNodeV2)
	switch Settings[key].Setting {
	case MaxN:
		g2.MaxN = change
	case MinN:
		g2.MinN = change
	case MaxS:
		g2.MaxS = change
	case MinS:
		g2.MinS = change
	case MaxDelegates:
		g2.MaxDelegates = change
	case NumMinerDelegatesRewarded:
		g2.NumMinerDelegatesRewarded = change
	case NumShardersRewarded:
		g2.NumShardersRewarded = change
	case NumSharderDelegatesRewarded:
		g2.NumSharderDelegatesRewarded = change
	case VCStartRounds, VCContributeRounds, VCShareRounds, VCPublishRounds, VCWaitRounds:
		vcPrefixLen := len(vcRoundsPrefix)
		phase := StringToPhase(key[vcPrefixLen:])
		if phase == Unknown {
			return fmt.Errorf("unknown phase: %s", key)
		}
		g2.VCPhaseRounds[int(phase)] = int(change)
	default:
		return fmt.Errorf("key: %v not implemented as int", key)
	}
	gn.SetEntity(g2)
	return nil
}

func (gn *GlobalNode) setDuration(key string, change time.Duration) error {
	return gn.MustUpdateBase(func(base *globalNodeBase) error {
		switch Settings[key].Setting {
		case HealthCheckPeriod:
			base.HealthCheckPeriod = change
		default:
			return fmt.Errorf("key: %v not implemented as int", key)
		}
		return nil
	})
}

func (gn *GlobalNode) setBalance(key string, change currency.Coin) error {
	return gn.MustUpdateBase(func(base *globalNodeBase) error {
		switch Settings[key].Setting {
		case MinStake:
			base.MinStake = change
		case MinStakePerDelegate:
			base.MinStakePerDelegate = change
		case MaxStake:
			base.MaxStake = change
		case BlockReward:
			base.BlockReward = change
		default:
			return fmt.Errorf("key: %v not implemented as balance", key)
		}
		return nil
	})
}

func (gn *GlobalNode) setInt64(key string, change int64) error {
	return gn.MustUpdateBase(func(base *globalNodeBase) error {
		switch Settings[key].Setting {
		case RewardRoundFrequency:
			base.RewardRoundFrequency = change
		case Epoch:
			base.Epoch = change
		case CooldownPeriod:
			base.CooldownPeriod = change
		default:
			return fmt.Errorf("key: %v not implemented as int64", key)
		}
		return nil
	})
}

func (gn *GlobalNode) setFloat64(key string, change float64) error {
	return gn.MustUpdateBase(func(base *globalNodeBase) error {
		switch Settings[key].Setting {
		case TPercent:
			base.TPercent = change
		case KPercent:
			base.KPercent = change
		case XPercent:
			base.XPercent = change
		case RewardRate:
			base.RewardRate = change
		case ShareRatio:
			base.ShareRatio = change
		case MaxCharge:
			base.MaxCharge = change
		case RewardDeclineRate:
			base.RewardDeclineRate = change
		default:
			return fmt.Errorf("key: %v not implemented as float64", key)
		}
		return nil
	})
}

func (gn *GlobalNode) setKey(key string, change string) {
	switch Settings[key].Setting {
	case OwnerId:
		gn.MustUpdateBase(func(base *globalNodeBase) error {
			base.OwnerId = change
			return nil
		})
	default:
		panic("key: " + key + "not implemented as key")
	}
}

const costPrefix = "cost."
const vcRoundsPrefix = "vc_rounds."

func (gn *GlobalNode) setCost(key string, change int) error {
	if !isCost(key) {
		return fmt.Errorf("key: %v is not a cost", key)
	}
	return gn.MustUpdateBase(func(base *globalNodeBase) error {
		if base.Cost == nil {
			base.Cost = make(map[string]int)
		}
		base.Cost[strings.TrimPrefix(key, costPrefix)] = change
		return nil
	})
}

func (gn *GlobalNode) getCost(key string) (int, error) {
	if !isCost(key) {
		return 0, fmt.Errorf("key: %v is not a cost", key)
	}
	gnb := gn.MustBase()
	if gnb.Cost == nil {
		return 0, errors.New("cost object is nil")
	}
	value, ok := gnb.Cost[strings.TrimPrefix(key, costPrefix)]
	if !ok {
		return 0, fmt.Errorf("cost %s not set", key)
	}
	return value, nil
}

func isCost(key string) bool {
	if len(key) <= len(costPrefix) {
		return false
	}
	return key[:len(costPrefix)] == costPrefix
}

func isVCPRounds(key string) bool {
	if len(key) <= len(vcRoundsPrefix) {
		return false
	}
	return key[:len(vcRoundsPrefix)] == vcRoundsPrefix
}

func (gn *GlobalNode) getVCRounds(key string) (int64, error) {
	if !isVCPRounds(key) {
		return 0, fmt.Errorf("key: %v is not a vc_rounds", key)
	}

	k := strings.TrimPrefix(key, vcRoundsPrefix)
	p := StringToPhase(k)
	phaseRounds := gn.GetVCPhaseRounds()
	r, ok := phaseRounds[p]
	if !ok {
		return 0, fmt.Errorf("phase rounds not found, key: %s", key)
	}

	return r, nil
}

func (gn *GlobalNode) set(balances cstate.StateContextI, key string, change string) error {
	if isCost(key) {
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
		if err := gn.setCost(key, value); err != nil {
			return err
		}

		return nil
	}

	if isVCPRounds(key) {
		if err := cstate.WithActivation(balances, "hermes", func() error {
			return fmt.Errorf("unsupported key %v", key)
		}, func() error {
			return nil
		}); err != nil {
			return err
		}
	}

	settings, ok := Settings[key]
	if !ok {
		return fmt.Errorf("unsupported key %v", key)
	}

	switch settings.ConfigType {
	case config.Int:
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int: %v", key, change, err)
		}
		if err := gn.setInt(balances, key, value); err != nil {
			return err
		}
	case config.CurrencyCoin:
		value, err := strconv.ParseFloat(change, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to state.balance: %v", key, change, err)
		}
		coinV, err := currency.ParseZCN(value)
		if err != nil {
			return err
		}
		if err := gn.setBalance(key, coinV); err != nil {
			return err
		}
	case config.Int64:
		value, err := strconv.ParseInt(change, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
		if err := gn.setInt64(key, value); err != nil {
			return err
		}
	case config.Duration:
		value, err := time.ParseDuration(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to duration: %v", key, change, err)
		}
		if err := gn.setDuration(key, value); err != nil {
			return err
		}
	case config.Float64:
		value, err := strconv.ParseFloat(change, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to float64: %v", key, change, err)
		}
		if err := gn.setFloat64(key, value); err != nil {
			return err
		}
	case config.Key:
		if _, err := hex.DecodeString(change); err != nil {
			return fmt.Errorf("%s must be a hex string: %v", key, err)
		}
		gn.setKey(key, change)
	default:
		return fmt.Errorf("unsupported type setting %v", config.ConfigTypeName[Settings[key].ConfigType])
	}

	return nil
}

func (gn *GlobalNode) update(balances cstate.StateContextI, changes config.StringMap) error {
	for key, value := range changes.Fields {
		if err := gn.set(balances, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (msc *MinerSmartContract) updateSettings(
	t *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	if err := smartcontractinterface.AuthorizeWithOwner("update_settings", func() bool {
		get, _ := gn.Get(OwnerId)
		return get == t.ClientID
	}); err != nil {
		return "", err
	}

	var changes config.StringMap
	if err = changes.Decode(inputData); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if err := gn.update(balances, changes); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if err = gn.validate(); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if err := gn.save(balances); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	return "", nil
}
