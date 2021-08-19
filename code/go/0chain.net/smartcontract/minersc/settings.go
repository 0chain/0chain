package minersc

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/state"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

type Setting int
type ConfigType int

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
	InterestRate
	RewardRate
	ShareRatio
	BlockReward
	MaxCharge
	Epoch
	RewardDeclineRate
	InterestDeclineRate
	MaxMint
	NumberOfSettings
)

const (
	Int ConfigType = iota
	StateBalance
	Int64
	Float64
	Duration
	Boolean

	NumberOfTypes
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
		"interest_rate",
		"reward_rate",
		"share_ratio",
		"block_reward",
		"max_charge",
		"epoch",
		"reward_decline_rate",
		"interest_decline_rate",
		"max_mint",
	}

	ConfitTypeName = []string{
		"int", "state.Balance", "int64", "float64", "time.duration", "bool",
	}

	Settings = map[string]struct {
		Setting    Setting
		ConfigType ConfigType
	}{
		"min_stake":              {MinStake, StateBalance},
		"max_stake":              {MaxStake, StateBalance},
		"max_n":                  {MaxN, Int},
		"min_n":                  {MinN, Int},
		"t_percent":              {TPercent, Float64},
		"k_percent":              {KPercent, Float64},
		"x_percent":              {XPercent, Float64},
		"max_s":                  {MaxS, Int},
		"min_s":                  {MinS, Int},
		"max_delegates":          {MaxDelegates, Int},
		"reward_round_frequency": {RewardRoundFrequency, Int64},
		"interest_rate":          {InterestRate, Float64},
		"reward_rate":            {RewardRate, Float64},
		"share_ratio":            {ShareRatio, Float64},
		"block_reward":           {BlockReward, StateBalance},
		"max_charge":             {MaxCharge, Float64},
		"epoch":                  {Epoch, Int64},
		"reward_decline_rate":    {RewardDeclineRate, Float64},
		"interest_decline_rate":  {InterestDeclineRate, Float64},
		"max_mint":               {MaxMint, StateBalance},
	}
)

type InputMap struct {
	Fields map[string]interface{} `json:"fields"`
}

func (im *InputMap) Decode(input []byte) error {
	err := json.Unmarshal(input, im)
	if err != nil {
		return err
	}
	return nil
}

func (im *InputMap) Encode() []byte {
	buff, _ := json.Marshal(im)
	return buff
}

func (gn *GlobalNode) setInt(key string, change int) {
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
		panic("key: " + key + "not implemented as int")
	}
}

func (gn *GlobalNode) setBalance(key string, change state.Balance) {
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
		panic("key: " + key + "not implemented as balance")
	}
}

func (gn *GlobalNode) setInt64(key string, change int64) {
	switch Settings[key].Setting {
	case RewardRoundFrequency:
		gn.RewardRoundFrequency = change
	case Epoch:
		gn.Epoch = change
	default:
		panic("key: " + key + "not implemented as balance")
	}
}

func (gn *GlobalNode) setFloat64(key string, change float64) {
	switch Settings[key].Setting {
	case TPercent:
		gn.TPercent = change
	case KPercent:
		gn.KPercent = change
	case XPercent:
		gn.XPercent = change
	case InterestRate:
		gn.InterestRate = change
	case RewardRate:
		gn.RewardRate = change
	case ShareRatio:
		gn.ShareRatio = change
	case MaxCharge:
		gn.MaxCharge = change
	case RewardDeclineRate:
		gn.RewardDeclineRate = change
	case InterestDeclineRate:
		gn.InterestDeclineRate = change
	default:
		panic("key: " + key + "not implemented as balance")
	}
}

func (gn *GlobalNode) set(key string, change interface{}) error {
	switch Settings[key].ConfigType {
	case Int:
		if fChange, ok := change.(float64); ok {
			gn.setInt(key, int(fChange))
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case StateBalance:
		if fChange, ok := change.(float64); ok {
			gn.setBalance(key, state.Balance(fChange))
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case Int64:
		if fChange, ok := change.(float64); ok {
			gn.setInt64(key, int64(fChange))
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case Float64:
		if fChange, ok := change.(float64); ok {
			gn.setFloat64(key, fChange)
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	default:
		panic("unsupported type setting " + ConfitTypeName[Settings[key].ConfigType])
	}

	return nil
}

func (gn *GlobalNode) update(changes InputMap) error {
	for key, value := range changes.Fields {
		if err := gn.set(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (msc *MinerSmartContract) UpdateSettings(
	t *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {

	if t.ClientID != owner {
		return "", common.NewError("update_settings",
			"unauthorized access - only the owner can update the variables")
	}

	var changes InputMap
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
