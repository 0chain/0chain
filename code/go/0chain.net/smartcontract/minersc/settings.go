package minersc

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

const x10 float64 = 10 * 1000 * 1000 * 1000

type Setting int

const (
	MinStake Setting = iota
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
	MaxMint
	OwnerId
	CooldownPeriod
	Cost
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
)

var (
	SettingName = []string{
		"min_stake",
		"max_stake",
		"max_n",
		"min_n",
		"t_percent",
		"k_percent",
		"x_percent",
		"max_s",
		"min_s",
		"max_delegates",
		"reward_round_frequency",
		"reward_rate",
		"share_ratio",
		"block_reward",
		"max_charge",
		"epoch",
		"reward_decline_rate",
		"max_mint",
		"owner_id",
		"cooldown_period",
		"cost",
		"cost.add_miner",
		"cost.add_sharder",
		"cost.delete_miner",
		"cost.miner_health_check",
		"cost.sharder_health_check",
		"cost.contributeMpk",
		"cost.shareSignsOrShares",
		"cost.wait",
		"cost.update_globals",
		"cost.update_settings",
		"cost.update_miner_settings",
		"cost.update_sharder_settings",
		"cost.payFees",
		"cost.feesPaid",
		"cost.mintedTokens",
		"cost.addToDelegatePool",
		"cost.deleteFromDelegatePool",
		"cost.sharder_keep",
	}
	NumberOfSettings = len(SettingName)

	Settings = map[string]struct {
		Setting    Setting
		ConfigType smartcontract.ConfigType
	}{
		"min_stake":                    {MinStake, smartcontract.StateBalance},
		"max_stake":                    {MaxStake, smartcontract.StateBalance},
		"max_n":                        {MaxN, smartcontract.Int},
		"min_n":                        {MinN, smartcontract.Int},
		"t_percent":                    {TPercent, smartcontract.Float64},
		"k_percent":                    {KPercent, smartcontract.Float64},
		"x_percent":                    {XPercent, smartcontract.Float64},
		"max_s":                        {MaxS, smartcontract.Int},
		"min_s":                        {MinS, smartcontract.Int},
		"max_delegates":                {MaxDelegates, smartcontract.Int},
		"reward_round_frequency":       {RewardRoundFrequency, smartcontract.Int64},
		"reward_rate":                  {RewardRate, smartcontract.Float64},
		"share_ratio":                  {ShareRatio, smartcontract.Float64},
		"block_reward":                 {BlockReward, smartcontract.StateBalance},
		"max_charge":                   {MaxCharge, smartcontract.Float64},
		"epoch":                        {Epoch, smartcontract.Int64},
		"reward_decline_rate":          {RewardDeclineRate, smartcontract.Float64},
		"max_mint":                     {MaxMint, smartcontract.StateBalance},
		"owner_id":                     {OwnerId, smartcontract.Key},
		"cooldown_period":              {CooldownPeriod, smartcontract.Int64},
		"cost":                         {Cost, smartcontract.Cost},
		"cost.add_miner":               {CostAddMiner, smartcontract.Cost},
		"cost.add_sharder":             {CostAddSharder, smartcontract.Cost},
		"cost.delete_miner":            {CostDeleteMiner, smartcontract.Cost},
		"cost.miner_health_check":      {CostMinerHealthCheck, smartcontract.Cost},
		"cost.sharder_health_check":    {CostSharderHealthCheck, smartcontract.Cost},
		"cost.contributempk":           {CostContributeMpk, smartcontract.Cost},
		"cost.sharesignsorshares":      {CostShareSignsOrShares, smartcontract.Cost},
		"cost.wait":                    {CostWait, smartcontract.Cost},
		"cost.update_globals":          {CostUpdateGlobals, smartcontract.Cost},
		"cost.update_settings":         {CostUpdateSettings, smartcontract.Cost},
		"cost.update_miner_settings":   {CostUpdateMinerSettings, smartcontract.Cost},
		"cost.update_sharder_settings": {CostUpdateSharderSettings, smartcontract.Cost},
		"cost.payfees":                 {CostPayFees, smartcontract.Cost},
		"cost.feespaid":                {CostFeesPaid, smartcontract.Cost},
		"cost.mintedtokens":            {CostMintedTokens, smartcontract.Cost},
		"cost.addtodelegatepool":       {CostAddToDelegatePool, smartcontract.Cost},
		"cost.deletefromdelegatepool":  {CostDeleteFromDelegatePool, smartcontract.Cost},
		"cost.sharder_keep":            {CostSharderKeep, smartcontract.Cost},
	}
)

func (gn *GlobalNode) setInt(key string, change int) error {
	switch Settings[key].Setting {
	case MaxN:
		gn.MaxN = change
	case MinN:
		gn.MinN = change
	case MaxS:
		gn.MaxS = change
	case MinS:
		gn.MinS = change
	case MaxDelegates:
		gn.MaxDelegates = change
	default:
		return fmt.Errorf("key: %v not implemented as int", key)
	}
	return nil
}

func (gn *GlobalNode) setBalance(key string, change currency.Coin) error {
	switch Settings[key].Setting {
	case MaxMint:
		gn.MaxMint = change
	case MinStake:
		gn.MinStake = change
	case MaxStake:
		gn.MaxStake = change
	case BlockReward:
		gn.BlockReward = change
	default:
		return fmt.Errorf("key: %v not implemented as balance", key)
	}
	return nil
}

func (gn *GlobalNode) setInt64(key string, change int64) error {
	switch Settings[key].Setting {
	case RewardRoundFrequency:
		gn.RewardRoundFrequency = change
	case Epoch:
		gn.Epoch = change
	case CooldownPeriod:
		gn.CooldownPeriod = change
	default:
		return fmt.Errorf("key: %v not implemented as int64", key)
	}
	return nil
}

func (gn *GlobalNode) setFloat64(key string, change float64) error {
	switch Settings[key].Setting {
	case TPercent:
		gn.TPercent = change
	case KPercent:
		gn.KPercent = change
	case XPercent:
		gn.XPercent = change
	case RewardRate:
		gn.RewardRate = change
	case ShareRatio:
		gn.ShareRatio = change
	case MaxCharge:
		gn.MaxCharge = change
	case RewardDeclineRate:
		gn.RewardDeclineRate = change
	default:
		return fmt.Errorf("key: %v not implemented as float64", key)
	}
	return nil
}

func (gn *GlobalNode) setKey(key string, change string) {
	switch Settings[key].Setting {
	case OwnerId:
		gn.OwnerId = change
	default:
		panic("key: " + key + "not implemented as key")
	}
}

func (gn *GlobalNode) setCost(key string, change int) {
	if change < 0 {
		return
	}
	costKey := strings.TrimPrefix(key, fmt.Sprintf("%s.", SettingName[Cost]))
	gn.Cost[costKey] = change
}

func (gn *GlobalNode) set(key string, change string) error {
	key = strings.ToLower(key)
	settings, ok := Settings[key]
	if !ok {
		return fmt.Errorf("unsupported key %v", key)
	}

	switch settings.ConfigType {
	case smartcontract.Int:
		if value, err := strconv.Atoi(change); err == nil {
			if err := gn.setInt(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to int: %v", key, change, err)
		}
	case smartcontract.StateBalance:
		if value, err := strconv.ParseFloat(change, 64); err == nil {
			if err := gn.setBalance(key, currency.Coin(value*x10)); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to state.balance: %v", key, change, err)
		}
	case smartcontract.Int64:
		if value, err := strconv.ParseInt(change, 10, 64); err == nil {
			if err := gn.setInt64(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
	case smartcontract.Float64:
		if value, err := strconv.ParseFloat(change, 64); err == nil {
			if err := gn.setFloat64(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to float64: %v", key, change, err)
		}
	case smartcontract.Key:
		if _, err := hex.DecodeString(change); err != nil {
			return fmt.Errorf("%s must be a hes string: %v", key, err)
		}
		gn.setKey(key, change)
	case smartcontract.Cost:
		if key == SettingName[Cost] {
			return fmt.Errorf("cost update key must follow cost.* format")
		}
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("key %s, unable to convert %v to integer", key, change)
		}
		gn.setCost(key, value)

	default:
		return fmt.Errorf("unsupported type setting %v", smartcontract.ConfigTypeName[Settings[key].ConfigType])
	}

	return nil
}

func (gn *GlobalNode) update(changes smartcontract.StringMap) error {
	for key, value := range changes.Fields {
		if err := gn.set(key, value); err != nil {
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

	var changes smartcontract.StringMap
	if err = changes.Decode(inputData); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if err := gn.update(changes); err != nil {
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
