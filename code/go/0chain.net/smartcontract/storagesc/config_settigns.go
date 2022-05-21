package storagesc

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"0chain.net/core/datastore"
	"0chain.net/smartcontract"

	chainState "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
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
	MaxBlobbersPerAllocation
	MaxReadPrice
	MaxWritePrice
	MinWritePrice
	FailedChallengesToCancel
	FailedChallengesToRevokeMinLock
	ChallengeEnabled
	ChallengeGenerationRate
	MaxChallengesPerGeneration
	ValidatorsPerChallenge
	MaxDelegates

	BlockRewardBlockReward
	BlockRewardQualifyingStake
	BlockRewardSharderWeight
	BlockRewardMinerWeight
	BlockRewardBlobberWeight

	ExposeMpt

	Cost
	CostUpdateSettings
	CostReadRedeem
	CostCommitConnection
	CostNewAllocationRequest
	CostUpdateAllocationRequest
	CostFinalizeAllocation
	CostCancelAllocation
	CostAddFreeStorageAssigner
	CostFreeAllocationRequest
	CostFreeUpdateAllocation
	CostAddCurator
	CostRemoveCurator
	CostBlobberHealthCheck
	CostUpdateBlobberSettings
	CostPayBlobberBlockRewards
	CostCuratorTransferAllocation
	CostChallengeRequest
	CostChallengeResponse
	CostGenerateChallenges
	CostAddValidator
	CostAddBlobber
	CostNewReadPool
	CostReadPoolLock
	CostReadPoolUnlock
	CostWritePoolLock
	CostWritePoolUnlock
	CostStakePoolLock
	CostStakePoolUnlock
	CostStakePoolPayInterests
	CostCommitSettingsChanges
	CostCollectReward
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
		"max_blobbers_per_allocation",
		"max_read_price",
		"max_write_price",
		"max_write_price",
		"failed_challenges_to_cancel",
		"failed_challenges_to_revoke_min_lock",
		"challenge_enabled",
		"challenge_rate_per_mb_min",
		"max_challenges_per_generation",
		"validators_per_challenge",
		"max_delegates",

		"block_reward.block_reward",
		"block_reward.qualifying_stake",
		"block_reward.sharder_ratio",
		"block_reward.miner_ratio",
		"block_reward.blobber_ratio",

		"expose_mpt",

		"cost",
		"cost.update_settings",
		"cost.read_redeem",
		"cost.commit_connection",
		"cost.new_allocation_request",
		"cost.update_allocation_request",
		"cost.finalize_allocation",
		"cost.cancel_allocation",
		"cost.add_free_storage_assigner",
		"cost.free_allocation_request",
		"cost.free_update_allocation",
		"cost.add_curator",
		"cost.remove_curator",
		"cost.blobber_health_check",
		"cost.update_blobber_settings",
		"cost.pay_blobber_block_rewards",
		"cost.curator_transfer_allocation",
		"cost.challenge_request",
		"cost.challenge_response",
		"cost.generate_challenges",
		"cost.add_validator",
		"cost.add_blobber",
		"cost.new_read_pool",
		"cost.read_pool_lock",
		"cost.read_pool_unlock",
		"cost.write_pool_lock",
		"cost.write_pool_unlock",
		"cost.stake_pool_lock",
		"cost.stake_pool_unlock",
		"cost.stake_pool_pay_interests",
		"cost.commit_settings_changes",
		"cost.collect_reward",
	}

	NumberOfSettings = len(SettingName)

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

		"stakepool.min_lock": {StakePoolMinLock, smartcontract.Int64},

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
		"max_blobbers_per_allocation":          {MaxBlobbersPerAllocation, smartcontract.Int},
		"max_read_price":                       {MaxReadPrice, smartcontract.StateBalance},
		"max_write_price":                      {MaxWritePrice, smartcontract.StateBalance},
		"min_write_price":                      {MinWritePrice, smartcontract.StateBalance},
		"failed_challenges_to_cancel":          {FailedChallengesToCancel, smartcontract.Int},
		"failed_challenges_to_revoke_min_lock": {FailedChallengesToRevokeMinLock, smartcontract.Int},
		"challenge_enabled":                    {ChallengeEnabled, smartcontract.Boolean},
		"challenge_rate_per_mb_min":            {ChallengeGenerationRate, smartcontract.Float64},
		"max_challenges_per_generation":        {MaxChallengesPerGeneration, smartcontract.Int},
		"validators_per_challenge":             {ValidatorsPerChallenge, smartcontract.Int},
		"max_delegates":                        {MaxDelegates, smartcontract.Int},

		"block_reward.block_reward":     {BlockRewardBlockReward, smartcontract.StateBalance},
		"block_reward.qualifying_stake": {BlockRewardQualifyingStake, smartcontract.StateBalance},
		"block_reward.sharder_ratio":    {BlockRewardSharderWeight, smartcontract.Float64},
		"block_reward.miner_ratio":      {BlockRewardMinerWeight, smartcontract.Float64},
		"block_reward.blobber_ratio":    {BlockRewardBlobberWeight, smartcontract.Float64},

		"expose_mpt": {ExposeMpt, smartcontract.Boolean},

		"cost":                             {Cost, smartcontract.Cost},
		"cost.update_settings":             {CostUpdateSettings, smartcontract.Cost},
		"cost.read_redeem":                 {CostReadRedeem, smartcontract.Cost},
		"cost.commit_connection":           {CostCommitConnection, smartcontract.Cost},
		"cost.new_allocation_request":      {CostNewAllocationRequest, smartcontract.Cost},
		"cost.update_allocation_request":   {CostUpdateAllocationRequest, smartcontract.Cost},
		"cost.finalize_allocation":         {CostFinalizeAllocation, smartcontract.Cost},
		"cost.cancel_allocation":           {CostCancelAllocation, smartcontract.Cost},
		"cost.add_free_storage_assigner":   {CostAddFreeStorageAssigner, smartcontract.Cost},
		"cost.free_allocation_request":     {CostFreeAllocationRequest, smartcontract.Cost},
		"cost.free_update_allocation":      {CostFreeUpdateAllocation, smartcontract.Cost},
		"cost.add_curator":                 {CostAddCurator, smartcontract.Cost},
		"cost.remove_curator":              {CostRemoveCurator, smartcontract.Cost},
		"cost.blobber_health_check":        {CostBlobberHealthCheck, smartcontract.Cost},
		"cost.update_blobber_settings":     {CostUpdateBlobberSettings, smartcontract.Cost},
		"cost.pay_blobber_block_rewards":   {CostPayBlobberBlockRewards, smartcontract.Cost},
		"cost.curator_transfer_allocation": {CostCuratorTransferAllocation, smartcontract.Cost},
		"cost.challenge_request":           {CostChallengeRequest, smartcontract.Cost},
		"cost.challenge_response":          {CostChallengeResponse, smartcontract.Cost},
		"cost.generate_challenges":         {CostGenerateChallenges, smartcontract.Cost},
		"cost.add_validator":               {CostAddValidator, smartcontract.Cost},
		"cost.add_blobber":                 {CostAddBlobber, smartcontract.Cost},
		"cost.new_read_pool":               {CostNewReadPool, smartcontract.Cost},
		"cost.read_pool_lock":              {CostReadPoolLock, smartcontract.Cost},
		"cost.read_pool_unlock":            {CostReadPoolUnlock, smartcontract.Cost},
		"cost.write_pool_lock":             {CostWritePoolLock, smartcontract.Cost},
		"cost.write_pool_unlock":           {CostWritePoolUnlock, smartcontract.Cost},
		"cost.stake_pool_lock":             {CostStakePoolLock, smartcontract.Cost},
		"cost.stake_pool_unlock":           {CostStakePoolUnlock, smartcontract.Cost},
		"cost.stake_pool_pay_interests":    {CostStakePoolPayInterests, smartcontract.Cost},
		"cost.commit_settings_changes":     {CostCommitSettingsChanges, smartcontract.Cost},
		"cost.collect_reward":              {CostCollectReward, smartcontract.Cost},
	}
)

func (conf *Config) getConfigMap() (smartcontract.StringMap, error) {
	var out smartcontract.StringMap
	out.Fields = make(map[string]string)
	for _, key := range SettingName {
		info, ok := Settings[strings.ToLower(key)]
		if !ok {
			return out, fmt.Errorf("SettingName %s not found in Settings", key)
		}
		iSetting := conf.get(info.setting)
		if info.configType == smartcontract.StateBalance {
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

func (conf *Config) setInt(key string, change int) error {
	switch Settings[key].setting {
	case FreeAllocationDataShards:
		conf.FreeAllocationSettings.DataShards = change
	case FreeAllocationParityShards:
		conf.FreeAllocationSettings.ParityShards = change
	case FailedChallengesToCancel:
		conf.FailedChallengesToCancel = change
	case FailedChallengesToRevokeMinLock:
		conf.FailedChallengesToRevokeMinLock = change
	case MaxBlobbersPerAllocation:
		conf.MaxBlobbersPerAllocation = change
	case MaxChallengesPerGeneration:
		conf.MaxChallengesPerGeneration = change
	case ValidatorsPerChallenge:
		conf.ValidatorsPerChallenge = change
	case MaxDelegates:
		conf.MaxDelegates = change
	default:
		return fmt.Errorf("key: %v not implemented as int", key)
	}

	return nil
}

func (conf *Config) setBalance(key string, change currency.Coin) error {
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

func (conf *Config) setInt64(key string, change int64) error {
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

func (conf *Config) setFloat64(key string, change float64) error {
	switch Settings[key].setting {
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
	case BlockRewardBlobberWeight:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.BlobberWeight = change
	default:
		return fmt.Errorf("key: %v not implemented as float64", key)
	}
	return nil
}

func (conf *Config) setDuration(key string, change time.Duration) error {
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
	case FreeAllocationDuration:
		conf.FreeAllocationSettings.Duration = change
	case FreeAllocationMaxChallengeCompletionTime:
		conf.FreeAllocationSettings.MaxChallengeCompletionTime = change
	default:
		return fmt.Errorf("key: %v not implemented as duration", key)
	}
	return nil
}

func (conf *Config) setBoolean(key string, change bool) error {
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

func (conf *Config) setCost(key string, change int) {
	if change < 0 {
		return
	}
	conf.Cost[strings.TrimPrefix(key, fmt.Sprintf("%s.", SettingName[Cost]))] = change
}

func (conf *Config) set(key string, change string) error {
	key = strings.ToLower(key)
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
			if err := conf.setBalance(key, currency.Coin(value*x10)); err != nil {
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
	case smartcontract.Cost:
		if key == SettingName[Cost] {
			return fmt.Errorf("cost update key must follow cost.* format")
		}
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("key %s, unable to convert %v to integer", key, change)
		}
		conf.setCost(key, value)
	default:
		return fmt.Errorf("unsupported type setting " + smartcontract.ConfigTypeName[Settings[key].configType])
	}
	return nil
}

func (conf *Config) get(key Setting) interface{} {
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
	case MaxBlobbersPerAllocation:
		return conf.MaxBlobbersPerAllocation
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
	case ValidatorsPerChallenge:
		return conf.ValidatorsPerChallenge
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
	case BlockRewardBlobberWeight:
		return conf.BlockReward.BlobberWeight
	case ExposeMpt:
		return conf.ExposeMpt
	case Cost:
		return ""
	case CostUpdateSettings:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostUpdateSettings], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostReadRedeem:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostReadRedeem], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostCommitConnection:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostCommitConnection], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostNewAllocationRequest:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostNewAllocationRequest], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostUpdateAllocationRequest:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostUpdateAllocationRequest], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostFinalizeAllocation:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostFinalizeAllocation], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostCancelAllocation:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostCancelAllocation], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostAddFreeStorageAssigner:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostAddFreeStorageAssigner], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostFreeAllocationRequest:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostFreeAllocationRequest], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostFreeUpdateAllocation:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostFreeUpdateAllocation], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostAddCurator:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostAddCurator], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostRemoveCurator:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostRemoveCurator], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostBlobberHealthCheck:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostBlobberHealthCheck], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostUpdateBlobberSettings:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostUpdateBlobberSettings], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostPayBlobberBlockRewards:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostPayBlobberBlockRewards], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostCuratorTransferAllocation:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostCuratorTransferAllocation], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostChallengeRequest:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostChallengeRequest], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostChallengeResponse:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostChallengeResponse], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostGenerateChallenges:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostGenerateChallenges], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostAddValidator:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostAddValidator], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostAddBlobber:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostAddBlobber], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostNewReadPool:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostNewReadPool], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostReadPoolLock:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostReadPoolLock], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostReadPoolUnlock:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostReadPoolUnlock], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostWritePoolLock:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostWritePoolLock], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostWritePoolUnlock:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostWritePoolUnlock], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostStakePoolLock:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostStakePoolLock], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostStakePoolUnlock:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostStakePoolUnlock], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostStakePoolPayInterests:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostStakePoolPayInterests], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostCommitSettingsChanges:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostCommitSettingsChanges], fmt.Sprintf("%s.", SettingName[Cost])))]
	case CostCollectReward:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostCollectReward], fmt.Sprintf("%s.", SettingName[Cost])))]

	default:
		panic("Setting not implemented")
	}
}

func (conf *Config) update(changes smartcontract.StringMap) error {
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
	var conf *Config
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
	var conf *Config
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
	var changes = new(smartcontract.StringMap)
	err := balances.GetTrieNode(settingChangesKey, changes)
	switch err {
	case nil:
		if len(changes.Fields) == 0 {
			return smartcontract.NewStringMap(), nil
		}
		return changes, nil
	case util.ErrValueNotPresent:
		return smartcontract.NewStringMap(), nil
	default:
		return nil, err
	}
}
