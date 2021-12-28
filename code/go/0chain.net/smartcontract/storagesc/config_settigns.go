package storagesc

import (
	"0chain.net/chaincore/smartcontractinterface"
	"fmt"
	"strconv"
	"time"

	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"0chain.net/core/datastore"
	"0chain.net/smartcontract"

	chainState "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

type Setting int

var settingChangesKey = datastore.Key(ADDRESS + encryption.Hash("setting_changes"))

const x10 = 10 * 1000 * 1000 * 1000

const (
	MaxMint Setting = iota
	TimeUnit
	MinAllocSize
	MinAllocDuration
	MaxChallengeCompletionTime
	MinOfferDuration
	MinBlobberCapacity

	ReadPoolMinLock
	ReadPoolMinLockPeriod
	ReadPoolMaxLockPeriod

	WritePoolMinLock
	WritePoolMinLockPeriod
	WritePoolMaxLockPeriod

	StakePoolMinLock
	StakePoolInterestRate
	StakePoolInterestInterval

	MaxTotalFreeAllocation
	MaxIndividualFreeAllocation

	FreeAllocationDataShards
	FreeAllocationParityShards
	FreeAllocationSize
	FreeAllocationDuration
	FreeAllocationReadPriceRangeMin
	FreeAllocationReadPriceRangeMax
	FreeAllocationWritePriceRangeMin
	FreeAllocationWritePriceRangeMax
	FreeAllocationMaxChallengeCompletionTime
	FreeAllocationReadPoolFraction

	ValidatorReward
	BlobberSlash
	MaxReadPrice
	MaxWritePrice
	MinWritePrice
	FailedChallengesToCancel
	FailedChallengesToRevokeMinLock
	ChallengeEnabled
	ChallengeGenerationRate
	MaxChallengesPerGeneration
	MaxDelegates

	BlockRewardBlockReward
	BlockRewardQualifyingStake
	BlockRewardSharderWeight
	BlockRewardMinerWeight
	BlockRewardBlobberCapacityWeight
	BlockRewardBlobberUsageWeight

	ExposeMpt

	NumberOfSettings
)

var (
	SettingName = []string{
		"max_mint",
		"time_unit",
		"min_alloc_size",
		"min_alloc_duration",
		"max_challenge_completion_time",
		"min_offer_duration",
		"min_blobber_capacity",

		"readpool.min_lock",
		"readpool.min_lock_period",
		"readpool.max_lock_period",

		"writepool.min_lock",
		"writepool.min_lock_period",
		"writepool.max_lock_period",

		"stakepool.min_lock",
		"stakepool.interest_rate",
		"stakepool.interest_interval",

		"max_total_free_allocation",
		"max_individual_free_allocation",

		"free_allocation_settings.data_shards",
		"free_allocation_settings.parity_shards",
		"free_allocation_settings.size",
		"free_allocation_settings.duration",
		"free_allocation_settings.read_price_range.min",
		"free_allocation_settings.read_price_range.max",
		"free_allocation_settings.write_price_range.min",
		"free_allocation_settings.write_price_range.max",
		"free_allocation_settings.max_challenge_completion_time",
		"free_allocation_settings.read_pool_fraction",

		"validator_reward",
		"blobber_slash",
		"max_read_price",
		"max_write_price",
		"max_write_price",
		"failed_challenges_to_cancel",
		"failed_challenges_to_revoke_min_lock",
		"challenge_enabled",
		"challenge_rate_per_mb_min",
		"max_challenges_per_generation",
		"max_delegates",

		"block_reward.block_reward",
		"block_reward.qualifying_stake",
		"block_reward.sharder_ratio",
		"block_reward.miner_ratio",
		"block_reward.blobber_capacity_ratio",
		"block_reward.blobber_usage_ratio",

		"expose_mpt",
	}

	Settings = map[string]struct {
		setting    Setting
		configType smartcontract.ConfigType
	}{
		"max_mint":                      {MaxMint, smartcontract.StateBalance},
		"time_unit":                     {TimeUnit, smartcontract.Duration},
		"min_alloc_size":                {MinAllocSize, smartcontract.Int64},
		"min_alloc_duration":            {MinAllocDuration, smartcontract.Duration},
		"max_challenge_completion_time": {MaxChallengeCompletionTime, smartcontract.Duration},
		"min_offer_duration":            {MinOfferDuration, smartcontract.Duration},
		"min_blobber_capacity":          {MinBlobberCapacity, smartcontract.Int64},

		"readpool.min_lock":        {ReadPoolMinLock, smartcontract.Int64},
		"readpool.min_lock_period": {ReadPoolMinLockPeriod, smartcontract.Duration},
		"readpool.max_lock_period": {ReadPoolMaxLockPeriod, smartcontract.Duration},

		"writepool.min_lock":        {WritePoolMinLock, smartcontract.Int64},
		"writepool.min_lock_period": {WritePoolMinLockPeriod, smartcontract.Duration},
		"writepool.max_lock_period": {WritePoolMaxLockPeriod, smartcontract.Duration},

		"stakepool.min_lock":          {StakePoolMinLock, smartcontract.Int64},
		"stakepool.interest_rate":     {StakePoolInterestRate, smartcontract.Float64},
		"stakepool.interest_interval": {StakePoolInterestInterval, smartcontract.Duration},

		"max_total_free_allocation":      {MaxTotalFreeAllocation, smartcontract.StateBalance},
		"max_individual_free_allocation": {MaxIndividualFreeAllocation, smartcontract.StateBalance},

		"free_allocation_settings.data_shards":                   {FreeAllocationDataShards, smartcontract.Int},
		"free_allocation_settings.parity_shards":                 {FreeAllocationParityShards, smartcontract.Int},
		"free_allocation_settings.size":                          {FreeAllocationSize, smartcontract.Int64},
		"free_allocation_settings.duration":                      {FreeAllocationDuration, smartcontract.Duration},
		"free_allocation_settings.read_price_range.min":          {FreeAllocationReadPriceRangeMin, smartcontract.StateBalance},
		"free_allocation_settings.read_price_range.max":          {FreeAllocationReadPriceRangeMax, smartcontract.StateBalance},
		"free_allocation_settings.write_price_range.min":         {FreeAllocationWritePriceRangeMin, smartcontract.StateBalance},
		"free_allocation_settings.write_price_range.max":         {FreeAllocationWritePriceRangeMax, smartcontract.StateBalance},
		"free_allocation_settings.max_challenge_completion_time": {FreeAllocationMaxChallengeCompletionTime, smartcontract.Duration},
		"free_allocation_settings.read_pool_fraction":            {FreeAllocationReadPoolFraction, smartcontract.Float64},

		"validator_reward":                     {ValidatorReward, smartcontract.Float64},
		"blobber_slash":                        {BlobberSlash, smartcontract.Float64},
		"max_read_price":                       {MaxReadPrice, smartcontract.StateBalance},
		"max_write_price":                      {MaxWritePrice, smartcontract.StateBalance},
		"min_write_price":                      {MinWritePrice, smartcontract.StateBalance},
		"failed_challenges_to_cancel":          {FailedChallengesToCancel, smartcontract.Int},
		"failed_challenges_to_revoke_min_lock": {FailedChallengesToRevokeMinLock, smartcontract.Int},
		"challenge_enabled":                    {ChallengeEnabled, smartcontract.Boolean},
		"challenge_rate_per_mb_min":            {ChallengeGenerationRate, smartcontract.Float64},
		"max_challenges_per_generation":        {MaxChallengesPerGeneration, smartcontract.Int},
		"max_delegates":                        {MaxDelegates, smartcontract.Int},

		"block_reward.block_reward":           {BlockRewardBlockReward, smartcontract.StateBalance},
		"block_reward.qualifying_stake":       {BlockRewardQualifyingStake, smartcontract.StateBalance},
		"block_reward.sharder_ratio":          {BlockRewardSharderWeight, smartcontract.Float64},
		"block_reward.miner_ratio":            {BlockRewardMinerWeight, smartcontract.Float64},
		"block_reward.blobber_capacity_ratio": {BlockRewardBlobberCapacityWeight, smartcontract.Float64},
		"block_reward.blobber_usage_ratio":    {BlockRewardBlobberUsageWeight, smartcontract.Float64},

		"expose_mpt": {ExposeMpt, smartcontract.Boolean},
	}
)

func (conf *scConfig) getConfigMap() (smartcontract.StringMap, error) {
	var im smartcontract.StringMap
	im.Fields = make(map[string]string)
	for key, info := range Settings {
		iSetting := conf.get(info.setting)
		if info.configType == smartcontract.StateBalance {
			sbSetting, ok := iSetting.(state.Balance)
			if !ok {
				return im, fmt.Errorf("%s key not implemented as state.balance", key)
			}
			iSetting = float64(sbSetting) / x10
		}
		im.Fields[key] = fmt.Sprintf("%v", iSetting)
	}
	return im, nil
}

func (conf *scConfig) setInt(key string, change int) error {
	switch Settings[key].setting {
	case FreeAllocationDataShards:
		conf.FreeAllocationSettings.DataShards = change
	case FreeAllocationParityShards:
		conf.FreeAllocationSettings.ParityShards = change
	case FailedChallengesToCancel:
		conf.FailedChallengesToCancel = change
	case FailedChallengesToRevokeMinLock:
		conf.FailedChallengesToRevokeMinLock = change
	case MaxChallengesPerGeneration:
		conf.MaxChallengesPerGeneration = change
	case MaxDelegates:
		conf.MaxDelegates = change
	default:
		return fmt.Errorf("key: %v not implemented as int", key)
	}

	return nil
}

func (conf *scConfig) setBalance(key string, change state.Balance) error {
	switch Settings[key].setting {
	case MaxMint:
		conf.MaxMint = change
	case MaxTotalFreeAllocation:
		conf.MaxTotalFreeAllocation = change
	case MaxIndividualFreeAllocation:
		conf.MaxIndividualFreeAllocation = change
	case FreeAllocationReadPriceRangeMin:
		conf.FreeAllocationSettings.ReadPriceRange.Min = change
	case FreeAllocationReadPriceRangeMax:
		conf.FreeAllocationSettings.ReadPriceRange.Max = change
	case FreeAllocationWritePriceRangeMin:
		conf.FreeAllocationSettings.WritePriceRange.Min = change
	case FreeAllocationWritePriceRangeMax:
		conf.FreeAllocationSettings.WritePriceRange.Max = change
	case MaxReadPrice:
		conf.MaxReadPrice = change
	case MaxWritePrice:
		conf.MaxWritePrice = change
	case MinWritePrice:
		conf.MinWritePrice = change
	case BlockRewardBlockReward:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.BlockReward = change
	case BlockRewardQualifyingStake:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.QualifyingStake = change
	default:
		return fmt.Errorf("key: %v not implemented as balance", key)
	}

	return nil
}

func (conf *scConfig) setInt64(key string, change int64) error {
	switch Settings[key].setting {
	case MinAllocSize:
		conf.MinAllocSize = change
	case MinBlobberCapacity:
		conf.MinBlobberCapacity = change
	case ReadPoolMinLock:
		if conf.ReadPool == nil {
			conf.ReadPool = &readPoolConfig{}
		}
		conf.ReadPool.MinLock = change
	case WritePoolMinLock:
		if conf.WritePool == nil {
			conf.WritePool = &writePoolConfig{}
		}
		conf.WritePool.MinLock = change
	case StakePoolMinLock:
		if conf.StakePool == nil {
			conf.StakePool = &stakePoolConfig{}
		}
		conf.StakePool.MinLock = change
	case FreeAllocationSize:
		conf.FreeAllocationSettings.Size = change
	default:
		return fmt.Errorf("key: %v not implemented as int64", key)
	}

	return nil
}

func (conf *scConfig) setFloat64(key string, change float64) error {
	switch Settings[key].setting {
	case StakePoolInterestRate:
		if conf.StakePool == nil {
			conf.StakePool = &stakePoolConfig{}
		}
		conf.StakePool.InterestRate = change
	case FreeAllocationReadPoolFraction:
		conf.FreeAllocationSettings.ReadPoolFraction = change
	case ValidatorReward:
		conf.ValidatorReward = change
	case BlobberSlash:
		conf.BlobberSlash = change
	case ChallengeGenerationRate:
		conf.ChallengeGenerationRate = change
	case BlockRewardSharderWeight:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.SharderWeight = change
	case BlockRewardMinerWeight:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.MinerWeight = change
	case BlockRewardBlobberCapacityWeight:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.BlobberCapacityWeight = change
	case BlockRewardBlobberUsageWeight:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.BlobberUsageWeight = change
	default:
		return fmt.Errorf("key: %v not implemented as float64", key)
	}
	return nil
}

func (conf *scConfig) setDuration(key string, change time.Duration) error {
	switch Settings[key].setting {
	case TimeUnit:
		conf.TimeUnit = change
	case MinAllocDuration:
		conf.MinAllocDuration = change
	case MaxChallengeCompletionTime:
		conf.MaxChallengeCompletionTime = change
	case MinOfferDuration:
		conf.MinOfferDuration = change
	case ReadPoolMinLockPeriod:
		if conf.ReadPool == nil {
			conf.ReadPool = &readPoolConfig{}
		}
		conf.ReadPool.MinLockPeriod = change
	case ReadPoolMaxLockPeriod:
		if conf.ReadPool == nil {
			conf.ReadPool = &readPoolConfig{}
		}
		conf.ReadPool.MaxLockPeriod = change
	case WritePoolMinLockPeriod:
		if conf.WritePool == nil {
			conf.WritePool = &writePoolConfig{}
		}
		conf.WritePool.MinLockPeriod = change
	case WritePoolMaxLockPeriod:
		if conf.WritePool == nil {
			conf.WritePool = &writePoolConfig{}
		}
		conf.WritePool.MaxLockPeriod = change
	case StakePoolInterestInterval:
		if conf.StakePool == nil {
			conf.StakePool = &stakePoolConfig{}
		}
		conf.StakePool.InterestInterval = change
	case FreeAllocationDuration:
		conf.FreeAllocationSettings.Duration = change
	case FreeAllocationMaxChallengeCompletionTime:
		conf.FreeAllocationSettings.MaxChallengeCompletionTime = change
	default:
		return fmt.Errorf("key: %v not implemented as duration", key)
	}
	return nil
}

func (conf *scConfig) setBoolean(key string, change bool) error {
	switch Settings[key].setting {
	case ChallengeEnabled:
		conf.ChallengeEnabled = change
	case ExposeMpt:
		conf.ExposeMpt = change
	default:
		return fmt.Errorf("key: %v not implemented as boolean", key)
	}
	return nil
}

func (conf *scConfig) set(key string, change string) error {
	s, ok := Settings[key]
	if !ok {
		return fmt.Errorf("unknown key %s, can't set value %v", key, change)
	}

	switch s.configType {
	case smartcontract.Int:
		if value, err := strconv.Atoi(change); err == nil {
			if err := conf.setInt(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to int: %v", key, change, err)
		}
	case smartcontract.StateBalance:
		if value, err := strconv.ParseFloat(change, 64); err == nil {
			if err := conf.setBalance(key, state.Balance(value*x10)); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to state.balance: %v", key, change, err)
		}
	case smartcontract.Int64:
		if value, err := strconv.ParseInt(change, 10, 64); err == nil {
			if err := conf.setInt64(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
	case smartcontract.Float64:
		if value, err := strconv.ParseFloat(change, 64); err == nil {
			if err := conf.setFloat64(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to float64: %v", key, change, err)
		}
	case smartcontract.Duration:
		if value, err := time.ParseDuration(change); err == nil {
			if err := conf.setDuration(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to duration: %v", key, change, err)
		}
	case smartcontract.Boolean:
		if value, err := strconv.ParseBool(change); err == nil {
			if err := conf.setBoolean(key, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot convert key %s value %v to boolean: %v", key, change, err)
		}
	default:
		panic("unsupported type setting " + smartcontract.ConfigTypeName[Settings[key].configType])
	}
	return nil
}

func (conf *scConfig) get(key Setting) interface{} {
	switch key {
	case MaxMint:
		return conf.MaxMint
	case TimeUnit:
		return conf.TimeUnit
	case MinAllocSize:
		return conf.MinAllocSize
	case MinAllocDuration:
		return conf.MinAllocDuration
	case MaxChallengeCompletionTime:
		return conf.MaxChallengeCompletionTime
	case MinOfferDuration:
		return conf.MinOfferDuration
	case MinBlobberCapacity:
		return conf.MinBlobberCapacity
	case ReadPoolMinLock:
		return conf.ReadPool.MinLock
	case ReadPoolMinLockPeriod:
		return conf.ReadPool.MinLockPeriod
	case ReadPoolMaxLockPeriod:
		return conf.ReadPool.MaxLockPeriod
	case WritePoolMinLock:
		return conf.WritePool.MinLock
	case WritePoolMinLockPeriod:
		return conf.WritePool.MinLockPeriod
	case WritePoolMaxLockPeriod:
		return conf.WritePool.MaxLockPeriod
	case StakePoolMinLock:
		return conf.StakePool.MinLock
	case StakePoolInterestRate:
		return conf.StakePool.InterestRate
	case StakePoolInterestInterval:
		return conf.StakePool.InterestInterval
	case MaxTotalFreeAllocation:
		return conf.MaxTotalFreeAllocation
	case MaxIndividualFreeAllocation:
		return conf.MaxIndividualFreeAllocation
	case FreeAllocationDataShards:
		return conf.FreeAllocationSettings.DataShards
	case FreeAllocationParityShards:
		return conf.FreeAllocationSettings.ParityShards
	case FreeAllocationSize:
		return conf.FreeAllocationSettings.Size
	case FreeAllocationDuration:
		return conf.FreeAllocationSettings.Duration
	case FreeAllocationReadPriceRangeMin:
		return conf.FreeAllocationSettings.ReadPriceRange.Min
	case FreeAllocationReadPriceRangeMax:
		return conf.FreeAllocationSettings.ReadPriceRange.Max
	case FreeAllocationWritePriceRangeMin:
		return conf.FreeAllocationSettings.WritePriceRange.Min
	case FreeAllocationWritePriceRangeMax:
		return conf.FreeAllocationSettings.WritePriceRange.Max
	case FreeAllocationMaxChallengeCompletionTime:
		return conf.FreeAllocationSettings.MaxChallengeCompletionTime
	case FreeAllocationReadPoolFraction:
		return conf.FreeAllocationSettings.ReadPoolFraction
	case ValidatorReward:
		return conf.ValidatorReward
	case BlobberSlash:
		return conf.BlobberSlash
	case MaxReadPrice:
		return conf.MaxReadPrice
	case MaxWritePrice:
		return conf.MaxWritePrice
	case MinWritePrice:
		return conf.MinWritePrice
	case FailedChallengesToCancel:
		return conf.FailedChallengesToCancel
	case FailedChallengesToRevokeMinLock:
		return conf.FailedChallengesToRevokeMinLock
	case ChallengeEnabled:
		return conf.ChallengeEnabled
	case ChallengeGenerationRate:
		return conf.ChallengeGenerationRate
	case MaxChallengesPerGeneration:
		return conf.MaxChallengesPerGeneration
	case MaxDelegates:
		return conf.MaxDelegates
	case BlockRewardBlockReward:
		return conf.BlockReward.BlockReward
	case BlockRewardQualifyingStake:
		return conf.BlockReward.QualifyingStake
	case BlockRewardSharderWeight:
		return conf.BlockReward.SharderWeight
	case BlockRewardMinerWeight:
		return conf.BlockReward.MinerWeight
	case BlockRewardBlobberCapacityWeight:
		return conf.BlockReward.BlobberCapacityWeight
	case BlockRewardBlobberUsageWeight:
		return conf.BlockReward.BlobberUsageWeight
	case ExposeMpt:
		return conf.ExposeMpt
	default:
		panic("Setting not implemented")
	}
}

func (conf *scConfig) update(changes smartcontract.StringMap) error {
	for key, value := range changes.Fields {
		if err := conf.set(key, value); err != nil {
			return err
		}
	}
	return nil
}

// updateSettings is SC function used by SC owner
// to update storage SC configurations
func (ssc *StorageSmartContract) updateSettings(
	t *transaction.Transaction,
	input []byte,
	balances chainState.StateContextI,
) (resp string, err error) {
	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("update_settings",
			"can't get config: "+err.Error())
	}

	if err := smartcontractinterface.AuthorizeWithOwner("update_settings", func() bool {
		return conf.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	var newChanges smartcontract.StringMap
	if err = newChanges.Decode(input); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if len(newChanges.Fields) == 0 {
		return "", nil
	}

	updateChanges, err := getSettingChanges(balances)
	if err != nil {
		return "", common.NewError("update_settings, getting setting changes", err.Error())
	}

	for key, value := range newChanges.Fields {
		updateChanges.Fields[key] = value
	}

	_, err = balances.InsertTrieNode(settingChangesKey, updateChanges)
	if err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	return "", nil
}

func (ssc *StorageSmartContract) commitSettingChanges(
	t *transaction.Transaction,
	_ []byte,
	balances chainState.StateContextI,
) (resp string, err error) {
	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("update_settings",
			"can't get config: "+err.Error())
	}

	changes, err := getSettingChanges(balances)
	if err != nil {
		return "", common.NewError("commitSettingChanges, getting setting changes", err.Error())
	}

	if len(changes.Fields) == 0 {
		return "", nil
	}

	if err := conf.update(*changes); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if err = conf.validate(); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	return "", nil
}

func getSettingChanges(balances cstate.StateContextI) (*smartcontract.StringMap, error) {
	val, err := balances.GetTrieNode(settingChangesKey)
	if err != nil || val == nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return smartcontract.NewStringMap(), nil
	}

	var changes = new(smartcontract.StringMap)
	err = changes.Decode(val.Encode())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	if changes.Fields == nil {
		return smartcontract.NewStringMap(), nil
	}
	return changes, nil
}
