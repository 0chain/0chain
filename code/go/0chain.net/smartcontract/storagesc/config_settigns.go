package storagesc

import (
	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"encoding/json"
	"fmt"
	"time"
)

type Setting int
type ConfigType int

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
	ConfitTypeName = []string{
		"int", "state.Balance", "int64", "float64", "time.duration", "bool",
	}
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
		configType ConfigType
	}{
		"max_mint":                      {MaxMint, StateBalance},
		"time_unit":                     {TimeUnit, Duration},
		"min_alloc_size":                {MinAllocSize, Int64},
		"min_alloc_duration":            {MinAllocDuration, Duration},
		"max_challenge_completion_time": {MaxChallengeCompletionTime, Duration},
		"min_offer_duration":            {MinOfferDuration, Duration},
		"min_blobber_capacity":          {MinBlobberCapacity, Int64},

		"readpool.min_lock":        {ReadPoolMinLock, Int64},
		"readpool.min_lock_period": {ReadPoolMinLockPeriod, Duration},
		"readpool.max_lock_period": {ReadPoolMaxLockPeriod, Duration},

		"writepool.min_lock":        {WritePoolMinLock, Int64},
		"writepool.min_lock_period": {WritePoolMinLockPeriod, Duration},
		"writepool.max_lock_period": {WritePoolMaxLockPeriod, Duration},

		"stakepool.min_lock":          {StakePoolMinLock, Int64},
		"stakepool.interest_rate":     {StakePoolInterestRate, Float64},
		"stakepool.interest_interval": {StakePoolInterestInterval, Duration},

		"max_total_free_allocation":      {MaxTotalFreeAllocation, StateBalance},
		"max_individual_free_allocation": {MaxIndividualFreeAllocation, StateBalance},

		"free_allocation_settings.data_shards":                   {FreeAllocationDataShards, Int},
		"free_allocation_settings.parity_shards":                 {FreeAllocationParityShards, Int},
		"free_allocation_settings.size":                          {FreeAllocationSize, Int64},
		"free_allocation_settings.duration":                      {FreeAllocationDuration, Duration},
		"free_allocation_settings.read_price_range.min":          {FreeAllocationReadPriceRangeMin, StateBalance},
		"free_allocation_settings.read_price_range.max":          {FreeAllocationReadPriceRangeMax, StateBalance},
		"free_allocation_settings.write_price_range.min":         {FreeAllocationWritePriceRangeMin, StateBalance},
		"free_allocation_settings.write_price_range.max":         {FreeAllocationWritePriceRangeMax, StateBalance},
		"free_allocation_settings.max_challenge_completion_time": {FreeAllocationMaxChallengeCompletionTime, Duration},
		"free_allocation_settings.read_pool_fraction":            {FreeAllocationReadPoolFraction, Float64},

		"validator_reward":                     {ValidatorReward, Float64},
		"blobber_slash":                        {BlobberSlash, Float64},
		"max_read_price":                       {MaxReadPrice, StateBalance},
		"max_write_price":                      {MaxWritePrice, StateBalance},
		"failed_challenges_to_cancel":          {FailedChallengesToCancel, Int},
		"failed_challenges_to_revoke_min_lock": {FailedChallengesToRevokeMinLock, Int},
		"challenge_enabled":                    {ChallengeEnabled, Boolean},
		"challenge_rate_per_mb_min":            {ChallengeGenerationRate, Float64},
		"max_challenges_per_generation":        {MaxChallengesPerGeneration, Int},
		"max_delegates":                        {MaxDelegates, Int},

		"block_reward.block_reward":           {BlockRewardBlockReward, StateBalance},
		"block_reward.qualifying_stake":       {BlockRewardQualifyingStake, StateBalance},
		"block_reward.sharder_ratio":          {BlockRewardSharderWeight, Float64},
		"block_reward.miner_ratio":            {BlockRewardMinerWeight, Float64},
		"block_reward.blobber_capacity_ratio": {BlockRewardBlobberCapacityWeight, Float64},
		"block_reward.blobber_usage_ratio":    {BlockRewardBlobberUsageWeight, Float64},

		"expose_mpt": {ExposeMpt, Boolean},
	}
)

type inputMap struct {
	Fields map[string]interface{} `json:"fields"`
}

func (im *inputMap) Decode(input []byte) error {
	err := json.Unmarshal(input, im)
	if err != nil {
		return err
	}
	return nil
}

func (im *inputMap) Encode() []byte {
	buff, _ := json.Marshal(im)
	return buff
}

// updateConfig is SC function used by SC owner
// to update storage SC configurations
func (ssc *StorageSmartContract) updateConfig(
	t *transaction.Transaction,
	input []byte,
	balances chainState.StateContextI,
) (resp string, err error) {
	if t.ClientID != owner {
		return "", common.NewError("update_config",
			"unauthorized access - only the owner can update the variables")
	}

	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("update_config",
			"can't get config: "+err.Error())
	}

	var changes inputMap
	if err = changes.Decode(input); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	if err := conf.update(changes); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	if err = conf.validate(); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	return "", nil
}

func (conf *scConfig) setInt(key string, change int) {
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
		panic("key: " + key + "not implemented as int")
	}
}

func (conf *scConfig) setBalance(key string, change state.Balance) {
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
		panic("key: " + key + "not implemented as balance")
	}
}

func (conf *scConfig) setInt64(key string, change int64) {
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
		panic("key: " + key + "not implemented as int64")
	}
}

func (conf *scConfig) setFloat64(key string, change float64) {
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
		panic("key: " + key + "not implemented as float64")
	}
}

func (conf *scConfig) setDuration(key string, change time.Duration) {
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
		panic("key: " + key + "not implemented as duration")
	}
}

func (conf *scConfig) setBoolean(key string, change bool) {
	switch Settings[key].setting {
	case ChallengeEnabled:
		conf.ChallengeEnabled = change
	case ExposeMpt:
		conf.ExposeMpt = change
	default:
		panic("key: " + key + "not implemented as boolean")
	}
}

func (conf *scConfig) set(key string, change interface{}) error {
	switch Settings[key].configType {
	case Int:
		if fChange, ok := change.(float64); ok {
			conf.setInt(key, int(fChange))
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case StateBalance:
		if fChange, ok := change.(float64); ok {
			conf.setBalance(key, state.Balance(fChange))
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case Int64:
		if fChange, ok := change.(float64); ok {
			conf.setInt64(key, int64(fChange))
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case Float64:
		if fChange, ok := change.(float64); ok {
			conf.setFloat64(key, fChange)
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case Duration:
		if fChange, ok := change.(float64); ok {
			conf.setDuration(key, time.Duration(fChange))
		} else {
			return fmt.Errorf("datatype error key %s value %v is not numeric", key, change)
		}
	case Boolean:
		if bChange, ok := change.(bool); ok {
			conf.setBoolean(key, bChange)
		} else {
			return fmt.Errorf("datatype error key %s value %v is not a boolean", key, change)
		}
	default:
		panic("unsupported type setting " + ConfitTypeName[Settings[key].configType])
	}

	return nil
}

func (conf *scConfig) update(changes inputMap) error {
	for key, value := range changes.Fields {
		if err := conf.set(key, value); err != nil {
			return err
		}
	}
	return nil
}
