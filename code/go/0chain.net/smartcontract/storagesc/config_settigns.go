package storagesc

import (
	"fmt"
	"strconv"
	"time"

	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"0chain.net/smartcontract"

	chainState "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

type Setting int

var settingChangesKey = ADDRESS + encryption.Hash("setting_changes")

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

func (sc *scConfig) getConfigMap() smartcontract.StringMap {
	var im smartcontract.StringMap
	im.Fields = make(map[string]string)
	for key, info := range Settings {
		iSetting := sc.get(info.setting)
		if info.configType == smartcontract.StateBalance {
			sbSetting, ok := iSetting.(state.Balance)
			if !ok {
				panic(fmt.Sprintf("%s key not implemented as state.balance", key))
			}
			iSetting = float64(sbSetting) / x10
		}
		im.Fields[key] = fmt.Sprintf("%v", iSetting)
	}
	return im
}

func (sc *scConfig) setInt(key string, change int) {
	switch Settings[key].setting {
	case FreeAllocationDataShards:
		sc.FreeAllocationSettings.DataShards = change
	case FreeAllocationParityShards:
		sc.FreeAllocationSettings.ParityShards = change
	case FailedChallengesToCancel:
		sc.FailedChallengesToCancel = change
	case FailedChallengesToRevokeMinLock:
		sc.FailedChallengesToRevokeMinLock = change
	case MaxChallengesPerGeneration:
		sc.MaxChallengesPerGeneration = change
	case MaxDelegates:
		sc.MaxDelegates = change
	default:
		panic("key: " + key + "not implemented as int")
	}
}

func (sc *scConfig) setBalance(key string, change state.Balance) {
	switch Settings[key].setting {
	case MaxMint:
		sc.MaxMint = change
	case MaxTotalFreeAllocation:
		sc.MaxTotalFreeAllocation = change
	case MaxIndividualFreeAllocation:
		sc.MaxIndividualFreeAllocation = change
	case FreeAllocationReadPriceRangeMin:
		sc.FreeAllocationSettings.ReadPriceRange.Min = change
	case FreeAllocationReadPriceRangeMax:
		sc.FreeAllocationSettings.ReadPriceRange.Max = change
	case FreeAllocationWritePriceRangeMin:
		sc.FreeAllocationSettings.WritePriceRange.Min = change
	case FreeAllocationWritePriceRangeMax:
		sc.FreeAllocationSettings.WritePriceRange.Max = change
	case MaxReadPrice:
		sc.MaxReadPrice = change
	case MaxWritePrice:
		sc.MaxWritePrice = change
	case BlockRewardBlockReward:
		if sc.BlockReward == nil {
			sc.BlockReward = &blockReward{}
		}
		sc.BlockReward.BlockReward = change
	case BlockRewardQualifyingStake:
		if sc.BlockReward == nil {
			sc.BlockReward = &blockReward{}
		}
		sc.BlockReward.QualifyingStake = change
	default:
		panic("key: " + key + "not implemented as balance")
	}
}

func (sc *scConfig) setInt64(key string, change int64) {
	switch Settings[key].setting {
	case MinAllocSize:
		sc.MinAllocSize = change
	case MinBlobberCapacity:
		sc.MinBlobberCapacity = change
	case ReadPoolMinLock:
		if sc.ReadPool == nil {
			sc.ReadPool = &readPoolConfig{}
		}
		sc.ReadPool.MinLock = change
	case WritePoolMinLock:
		if sc.WritePool == nil {
			sc.WritePool = &writePoolConfig{}
		}
		sc.WritePool.MinLock = change
	case StakePoolMinLock:
		if sc.StakePool == nil {
			sc.StakePool = &stakePoolConfig{}
		}
		sc.StakePool.MinLock = change
	case FreeAllocationSize:
		sc.FreeAllocationSettings.Size = change
	default:
		panic("key: " + key + "not implemented as int64")
	}
}

func (sc *scConfig) setFloat64(key string, change float64) {
	switch Settings[key].setting {
	case StakePoolInterestRate:
		if sc.StakePool == nil {
			sc.StakePool = &stakePoolConfig{}
		}
		sc.StakePool.InterestRate = change
	case FreeAllocationReadPoolFraction:
		sc.FreeAllocationSettings.ReadPoolFraction = change
	case ValidatorReward:
		sc.ValidatorReward = change
	case BlobberSlash:
		sc.BlobberSlash = change
	case ChallengeGenerationRate:
		sc.ChallengeGenerationRate = change
	case BlockRewardSharderWeight:
		if sc.BlockReward == nil {
			sc.BlockReward = &blockReward{}
		}
		sc.BlockReward.SharderWeight = change
	case BlockRewardMinerWeight:
		if sc.BlockReward == nil {
			sc.BlockReward = &blockReward{}
		}
		sc.BlockReward.MinerWeight = change
	case BlockRewardBlobberCapacityWeight:
		if sc.BlockReward == nil {
			sc.BlockReward = &blockReward{}
		}
		sc.BlockReward.BlobberCapacityWeight = change
	case BlockRewardBlobberUsageWeight:
		if sc.BlockReward == nil {
			sc.BlockReward = &blockReward{}
		}
		sc.BlockReward.BlobberUsageWeight = change
	default:
		panic("key: " + key + "not implemented as float64")
	}
}

func (sc *scConfig) setDuration(key string, change time.Duration) {
	switch Settings[key].setting {
	case TimeUnit:
		sc.TimeUnit = change
	case MinAllocDuration:
		sc.MinAllocDuration = change
	case MaxChallengeCompletionTime:
		sc.MaxChallengeCompletionTime = change
	case MinOfferDuration:
		sc.MinOfferDuration = change
	case ReadPoolMinLockPeriod:
		if sc.ReadPool == nil {
			sc.ReadPool = &readPoolConfig{}
		}
		sc.ReadPool.MinLockPeriod = change
	case ReadPoolMaxLockPeriod:
		if sc.ReadPool == nil {
			sc.ReadPool = &readPoolConfig{}
		}
		sc.ReadPool.MaxLockPeriod = change
	case WritePoolMinLockPeriod:
		if sc.WritePool == nil {
			sc.WritePool = &writePoolConfig{}
		}
		sc.WritePool.MinLockPeriod = change
	case WritePoolMaxLockPeriod:
		if sc.WritePool == nil {
			sc.WritePool = &writePoolConfig{}
		}
		sc.WritePool.MaxLockPeriod = change
	case StakePoolInterestInterval:
		if sc.StakePool == nil {
			sc.StakePool = &stakePoolConfig{}
		}
		sc.StakePool.InterestInterval = change
	case FreeAllocationDuration:
		sc.FreeAllocationSettings.Duration = change
	case FreeAllocationMaxChallengeCompletionTime:
		sc.FreeAllocationSettings.MaxChallengeCompletionTime = change
	default:
		panic("key: " + key + "not implemented as duration")
	}
}

func (sc *scConfig) setBoolean(key string, change bool) {
	switch Settings[key].setting {
	case ChallengeEnabled:
		sc.ChallengeEnabled = change
	case ExposeMpt:
		sc.ExposeMpt = change
	default:
		panic("key: " + key + "not implemented as boolean")
	}
}

func (sc *scConfig) set(key string, change string) error {
	switch Settings[key].configType {
	case smartcontract.Int:
		if value, err := strconv.Atoi(change); err == nil {
			sc.setInt(key, value)
		} else {
			return fmt.Errorf("cannot convert key %s value %v to int: %v", key, change, err)
		}
	case smartcontract.StateBalance:
		if value, err := strconv.ParseFloat(change, 64); err == nil {
			sc.setBalance(key, state.Balance(value*x10))
		} else {
			return fmt.Errorf("cannot convert key %s value %v to state.balance: %v", key, change, err)
		}
	case smartcontract.Int64:
		if value, err := strconv.ParseInt(change, 10, 64); err == nil {
			sc.setInt64(key, value)
		} else {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
	case smartcontract.Float64:
		if value, err := strconv.ParseFloat(change, 64); err == nil {
			sc.setFloat64(key, value)
		} else {
			return fmt.Errorf("cannot convert key %s value %v to float64: %v", key, change, err)
		}
	case smartcontract.Duration:
		if value, err := time.ParseDuration(change); err == nil {
			sc.setDuration(key, value)
		} else {
			return fmt.Errorf("cannot convert key %s value %v to duration: %v", key, change, err)
		}
	case smartcontract.Boolean:
		if value, err := strconv.ParseBool(change); err == nil {
			sc.setBoolean(key, value)
		} else {
			return fmt.Errorf("cannot convert key %s value %v to boolean: %v", key, change, err)
		}
	default:
		panic("unsupported type setting " + smartcontract.ConfigTypeName[Settings[key].configType])
	}
	return nil
}

func (sc *scConfig) get(key Setting) interface{} {
	switch key {
	case MaxMint:
		return sc.MaxMint
	case TimeUnit:
		return sc.TimeUnit
	case MinAllocSize:
		return sc.MinAllocSize
	case MinAllocDuration:
		return sc.MinAllocDuration
	case MaxChallengeCompletionTime:
		return sc.MaxChallengeCompletionTime
	case MinOfferDuration:
		return sc.MinOfferDuration
	case MinBlobberCapacity:
		return sc.MinBlobberCapacity
	case ReadPoolMinLock:
		return sc.ReadPool.MinLock
	case ReadPoolMinLockPeriod:
		return sc.ReadPool.MinLockPeriod
	case ReadPoolMaxLockPeriod:
		return sc.ReadPool.MaxLockPeriod
	case WritePoolMinLock:
		return sc.WritePool.MinLock
	case WritePoolMinLockPeriod:
		return sc.WritePool.MinLockPeriod
	case WritePoolMaxLockPeriod:
		return sc.WritePool.MaxLockPeriod
	case StakePoolMinLock:
		return sc.StakePool.MinLock
	case StakePoolInterestRate:
		return sc.StakePool.InterestRate
	case StakePoolInterestInterval:
		return sc.StakePool.InterestInterval
	case MaxTotalFreeAllocation:
		return sc.MaxTotalFreeAllocation
	case MaxIndividualFreeAllocation:
		return sc.MaxIndividualFreeAllocation
	case FreeAllocationDataShards:
		return sc.FreeAllocationSettings.DataShards
	case FreeAllocationParityShards:
		return sc.FreeAllocationSettings.ParityShards
	case FreeAllocationSize:
		return sc.FreeAllocationSettings.Size
	case FreeAllocationDuration:
		return sc.FreeAllocationSettings.Duration
	case FreeAllocationReadPriceRangeMin:
		return sc.FreeAllocationSettings.ReadPriceRange.Min
	case FreeAllocationReadPriceRangeMax:
		return sc.FreeAllocationSettings.ReadPriceRange.Max
	case FreeAllocationWritePriceRangeMin:
		return sc.FreeAllocationSettings.WritePriceRange.Min
	case FreeAllocationWritePriceRangeMax:
		return sc.FreeAllocationSettings.WritePriceRange.Max
	case FreeAllocationMaxChallengeCompletionTime:
		return sc.FreeAllocationSettings.MaxChallengeCompletionTime
	case FreeAllocationReadPoolFraction:
		return sc.FreeAllocationSettings.ReadPoolFraction
	case ValidatorReward:
		return sc.ValidatorReward
	case BlobberSlash:
		return sc.BlobberSlash
	case MaxReadPrice:
		return sc.MaxReadPrice
	case MaxWritePrice:
		return sc.MaxWritePrice
	case FailedChallengesToCancel:
		return sc.FailedChallengesToCancel
	case FailedChallengesToRevokeMinLock:
		return sc.FailedChallengesToRevokeMinLock
	case ChallengeEnabled:
		return sc.ChallengeEnabled
	case ChallengeGenerationRate:
		return sc.ChallengeGenerationRate
	case MaxChallengesPerGeneration:
		return sc.MaxChallengesPerGeneration
	case MaxDelegates:
		return sc.MaxDelegates
	case BlockRewardBlockReward:
		return sc.BlockReward.BlockReward
	case BlockRewardQualifyingStake:
		return sc.BlockReward.QualifyingStake
	case BlockRewardSharderWeight:
		return sc.BlockReward.SharderWeight
	case BlockRewardMinerWeight:
		return sc.BlockReward.MinerWeight
	case BlockRewardBlobberCapacityWeight:
		return sc.BlockReward.BlobberCapacityWeight
	case BlockRewardBlobberUsageWeight:
		return sc.BlockReward.BlobberUsageWeight
	case ExposeMpt:
		return sc.ExposeMpt
	default:
		panic("Setting not implemented")
	}
}

func (sc *scConfig) update(changes smartcontract.StringMap) error {
	for key, value := range changes.Fields {
		if err := sc.set(key, value); err != nil {
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
	if t.ClientID != owner {
		return "", common.NewError("update_settings",
			"unauthorized access - only the owner can update the variables")
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
	_ *transaction.Transaction,
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
