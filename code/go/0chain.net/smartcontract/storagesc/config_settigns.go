package storagesc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"

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
	WritePoolMinLock

	StakePoolMinLock
	StakePoolMinLockPeriod

	MaxTotalFreeAllocation
	MaxIndividualFreeAllocation
	CancellationCharge

	FreeAllocationDataShards
	FreeAllocationParityShards
	FreeAllocationSize
	FreeAllocationDuration
	FreeAllocationReadPriceRangeMin
	FreeAllocationReadPriceRangeMax
	FreeAllocationWritePriceRangeMin
	FreeAllocationWritePriceRangeMax
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
	BlockRewardGammaAlpha
	BlockRewardGammaA
	BlockRewardGammaB
	BlockRewardZetaI
	BlockRewardZetaK
	BlockRewardZetaMu

	OwnerId

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
	CostBlobberHealthCheck
	CostUpdateBlobberSettings
	CostPayBlobberBlockRewards
	CostChallengeRequest
	CostChallengeResponse
	CostGenerateChallenges
	CostAddValidator
	CostUpdateValidatorSettings
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
		setting    Setting
		configType smartcontract.ConfigType
	}
)

func init() {
	initSettingName()
	initSettings()
}

func initSettingName() {
	SettingName[MaxMint] = "max_mint"
	SettingName[TimeUnit] = "time_unit"
	SettingName[MinAllocSize] = "min_alloc_size"
	SettingName[MinAllocDuration] = "min_alloc_duration"
	SettingName[MaxChallengeCompletionTime] = "max_challenge_completion_time"
	SettingName[MinOfferDuration] = "min_offer_duration"
	SettingName[MinBlobberCapacity] = "min_blobber_capacity"
	SettingName[ReadPoolMinLock] = "readpool.min_lock"
	SettingName[WritePoolMinLock] = "writepool.min_lock"
	SettingName[StakePoolMinLock] = "stakepool.min_lock"
	SettingName[StakePoolMinLockPeriod] = "stakepool.min_lock_period"
	SettingName[MaxTotalFreeAllocation] = "max_total_free_allocation"
	SettingName[MaxIndividualFreeAllocation] = "max_individual_free_allocation"
	SettingName[CancellationCharge] = "cancellation_charge"
	SettingName[FreeAllocationDataShards] = "free_allocation_settings.data_shards"
	SettingName[FreeAllocationParityShards] = "free_allocation_settings.parity_shards"
	SettingName[FreeAllocationSize] = "free_allocation_settings.size"
	SettingName[FreeAllocationDuration] = "free_allocation_settings.duration"
	SettingName[FreeAllocationReadPriceRangeMin] = "free_allocation_settings.read_price_range.min"
	SettingName[FreeAllocationReadPriceRangeMax] = "free_allocation_settings.read_price_range.max"
	SettingName[FreeAllocationWritePriceRangeMin] = "free_allocation_settings.write_price_range.min"
	SettingName[FreeAllocationWritePriceRangeMax] = "free_allocation_settings.write_price_range.max"
	SettingName[FreeAllocationReadPoolFraction] = "free_allocation_settings.read_pool_fraction"
	SettingName[ValidatorReward] = "validator_reward"
	SettingName[BlobberSlash] = "blobber_slash"
	SettingName[MaxBlobbersPerAllocation] = "max_blobbers_per_allocation"
	SettingName[MaxReadPrice] = "max_read_price"
	SettingName[MaxWritePrice] = "max_write_price"
	SettingName[MinWritePrice] = "min_write_price"
	SettingName[FailedChallengesToCancel] = "failed_challenges_to_cancel"
	SettingName[FailedChallengesToRevokeMinLock] = "failed_challenges_to_revoke_min_lock"
	SettingName[ChallengeEnabled] = "challenge_enabled"
	SettingName[ChallengeGenerationRate] = "challenge_rate_per_mb_min"
	SettingName[MaxChallengesPerGeneration] = "max_challenges_per_generation"
	SettingName[ValidatorsPerChallenge] = "validators_per_challenge"
	SettingName[MaxDelegates] = "max_delegates"
	SettingName[BlockRewardBlockReward] = "block_reward.block_reward"
	SettingName[BlockRewardQualifyingStake] = "block_reward.qualifying_stake"
	SettingName[BlockRewardSharderWeight] = "block_reward.sharder_ratio"
	SettingName[BlockRewardMinerWeight] = "block_reward.miner_ratio"
	SettingName[BlockRewardBlobberWeight] = "block_reward.blobber_ratio"
	SettingName[BlockRewardGammaAlpha] = "block_reward.gamma.alpha"
	SettingName[BlockRewardGammaA] = "block_reward.gamma.a"
	SettingName[BlockRewardGammaB] = "block_reward.gamma.b"
	SettingName[BlockRewardZetaI] = "block_reward.zeta.i"
	SettingName[BlockRewardZetaK] = "block_reward.zeta.k"
	SettingName[BlockRewardZetaMu] = "block_reward.zeta.mu"
	SettingName[OwnerId] = "owner_id"
	SettingName[CostUpdateSettings] = "cost.update_settings"
	SettingName[CostReadRedeem] = "cost.read_redeem"
	SettingName[CostCommitConnection] = "cost.commit_connection"
	SettingName[CostNewAllocationRequest] = "cost.new_allocation_request"
	SettingName[CostUpdateAllocationRequest] = "cost.update_allocation_request"
	SettingName[CostFinalizeAllocation] = "cost.finalize_allocation"
	SettingName[CostCancelAllocation] = "cost.cancel_allocation"
	SettingName[CostAddFreeStorageAssigner] = "cost.add_free_storage_assigner"
	SettingName[CostFreeAllocationRequest] = "cost.free_allocation_request"
	SettingName[CostFreeUpdateAllocation] = "cost.free_update_allocation"
	SettingName[CostBlobberHealthCheck] = "cost.blobber_health_check"
	SettingName[CostUpdateBlobberSettings] = "cost.update_blobber_settings"
	SettingName[CostPayBlobberBlockRewards] = "cost.pay_blobber_block_rewards"
	SettingName[CostChallengeRequest] = "cost.challenge_request"
	SettingName[CostChallengeResponse] = "cost.challenge_response"
	SettingName[CostGenerateChallenges] = "cost.generate_challenges"
	SettingName[CostAddValidator] = "cost.add_validator"
	SettingName[CostUpdateValidatorSettings] = "cost.update_validator_settings"
	SettingName[CostAddBlobber] = "cost.add_blobber"
	SettingName[CostNewReadPool] = "cost.new_read_pool"
	SettingName[CostReadPoolLock] = "cost.read_pool_lock"
	SettingName[CostReadPoolUnlock] = "cost.read_pool_unlock"
	SettingName[CostWritePoolLock] = "cost.write_pool_lock"
	SettingName[CostWritePoolUnlock] = "cost.write_pool_unlock"
	SettingName[CostStakePoolLock] = "cost.stake_pool_lock"
	SettingName[CostStakePoolUnlock] = "cost.stake_pool_unlock"
	SettingName[CostStakePoolPayInterests] = "cost.stake_pool_pay_interests"
	SettingName[CostCommitSettingsChanges] = "cost.commit_settings_changes"
	SettingName[CostCollectReward] = "cost.collect_reward"
}

func initSettings() {
	Settings = map[string]struct {
		setting    Setting
		configType smartcontract.ConfigType
	}{
		MaxMint.String():                          {MaxMint, smartcontract.CurrencyCoin},
		TimeUnit.String():                         {TimeUnit, smartcontract.Duration},
		MinAllocSize.String():                     {MinAllocSize, smartcontract.Int64},
		MinAllocDuration.String():                 {MinAllocDuration, smartcontract.Duration},
		MaxChallengeCompletionTime.String():       {MaxChallengeCompletionTime, smartcontract.Duration},
		MinOfferDuration.String():                 {MinOfferDuration, smartcontract.Duration},
		MinBlobberCapacity.String():               {MinBlobberCapacity, smartcontract.Int64},
		ReadPoolMinLock.String():                  {ReadPoolMinLock, smartcontract.CurrencyCoin},
		WritePoolMinLock.String():                 {WritePoolMinLock, smartcontract.CurrencyCoin},
		StakePoolMinLock.String():                 {StakePoolMinLock, smartcontract.CurrencyCoin},
		StakePoolMinLockPeriod.String():           {StakePoolMinLockPeriod, smartcontract.Duration},
		MaxTotalFreeAllocation.String():           {MaxTotalFreeAllocation, smartcontract.CurrencyCoin},
		MaxIndividualFreeAllocation.String():      {MaxIndividualFreeAllocation, smartcontract.CurrencyCoin},
		CancellationCharge.String():               {CancellationCharge, smartcontract.Float64},
		FreeAllocationDataShards.String():         {FreeAllocationDataShards, smartcontract.Int},
		FreeAllocationParityShards.String():       {FreeAllocationParityShards, smartcontract.Int},
		FreeAllocationSize.String():               {FreeAllocationSize, smartcontract.Int64},
		FreeAllocationDuration.String():           {FreeAllocationDuration, smartcontract.Duration},
		FreeAllocationReadPriceRangeMin.String():  {FreeAllocationReadPriceRangeMin, smartcontract.CurrencyCoin},
		FreeAllocationReadPriceRangeMax.String():  {FreeAllocationReadPriceRangeMax, smartcontract.CurrencyCoin},
		FreeAllocationWritePriceRangeMin.String(): {FreeAllocationWritePriceRangeMin, smartcontract.CurrencyCoin},
		FreeAllocationWritePriceRangeMax.String(): {FreeAllocationWritePriceRangeMax, smartcontract.CurrencyCoin},
		FreeAllocationReadPoolFraction.String():   {FreeAllocationReadPoolFraction, smartcontract.Float64},
		ValidatorReward.String():                  {ValidatorReward, smartcontract.Float64},
		BlobberSlash.String():                     {BlobberSlash, smartcontract.Float64},
		MaxBlobbersPerAllocation.String():         {MaxBlobbersPerAllocation, smartcontract.Int},
		MaxReadPrice.String():                     {MaxReadPrice, smartcontract.CurrencyCoin},
		MaxWritePrice.String():                    {MaxWritePrice, smartcontract.CurrencyCoin},
		MinWritePrice.String():                    {MinWritePrice, smartcontract.CurrencyCoin},
		FailedChallengesToCancel.String():         {FailedChallengesToCancel, smartcontract.Int},
		FailedChallengesToRevokeMinLock.String():  {FailedChallengesToRevokeMinLock, smartcontract.Int},
		ChallengeEnabled.String():                 {ChallengeEnabled, smartcontract.Boolean},
		ChallengeGenerationRate.String():          {ChallengeGenerationRate, smartcontract.Float64},
		MaxChallengesPerGeneration.String():       {MaxChallengesPerGeneration, smartcontract.Int},
		ValidatorsPerChallenge.String():           {ValidatorsPerChallenge, smartcontract.Int},
		MaxDelegates.String():                     {MaxDelegates, smartcontract.Int},
		BlockRewardBlockReward.String():           {BlockRewardBlockReward, smartcontract.CurrencyCoin},
		BlockRewardQualifyingStake.String():       {BlockRewardQualifyingStake, smartcontract.CurrencyCoin},
		BlockRewardSharderWeight.String():         {BlockRewardSharderWeight, smartcontract.Float64},
		BlockRewardMinerWeight.String():           {BlockRewardMinerWeight, smartcontract.Float64},
		BlockRewardBlobberWeight.String():         {BlockRewardBlobberWeight, smartcontract.Float64},
		BlockRewardGammaAlpha.String():            {BlockRewardGammaAlpha, smartcontract.Float64},
		BlockRewardGammaA.String():                {BlockRewardGammaA, smartcontract.Float64},
		BlockRewardGammaB.String():                {BlockRewardGammaB, smartcontract.Float64},
		BlockRewardZetaI.String():                 {BlockRewardZetaI, smartcontract.Float64},
		BlockRewardZetaK.String():                 {BlockRewardZetaK, smartcontract.Float64},
		BlockRewardZetaMu.String():                {BlockRewardZetaMu, smartcontract.Float64},
		OwnerId.String():                          {OwnerId, smartcontract.Key},
		CostUpdateSettings.String():               {CostUpdateSettings, smartcontract.Cost},
		CostReadRedeem.String():                   {CostReadRedeem, smartcontract.Cost},
		CostCommitConnection.String():             {CostCommitConnection, smartcontract.Cost},
		CostNewAllocationRequest.String():         {CostNewAllocationRequest, smartcontract.Cost},
		CostUpdateAllocationRequest.String():      {CostUpdateAllocationRequest, smartcontract.Cost},
		CostFinalizeAllocation.String():           {CostFinalizeAllocation, smartcontract.Cost},
		CostCancelAllocation.String():             {CostCancelAllocation, smartcontract.Cost},
		CostAddFreeStorageAssigner.String():       {CostAddFreeStorageAssigner, smartcontract.Cost},
		CostFreeAllocationRequest.String():        {CostFreeAllocationRequest, smartcontract.Cost},
		CostFreeUpdateAllocation.String():         {CostFreeUpdateAllocation, smartcontract.Cost},
		CostBlobberHealthCheck.String():           {CostBlobberHealthCheck, smartcontract.Cost},
		CostUpdateBlobberSettings.String():        {CostUpdateBlobberSettings, smartcontract.Cost},
		CostPayBlobberBlockRewards.String():       {CostPayBlobberBlockRewards, smartcontract.Cost},
		CostChallengeRequest.String():             {CostChallengeRequest, smartcontract.Cost},
		CostChallengeResponse.String():            {CostChallengeResponse, smartcontract.Cost},
		CostGenerateChallenges.String():           {CostGenerateChallenges, smartcontract.Cost},
		CostAddValidator.String():                 {CostAddValidator, smartcontract.Cost},
		CostUpdateValidatorSettings.String():      {CostUpdateValidatorSettings, smartcontract.Cost},
		CostAddBlobber.String():                   {CostAddBlobber, smartcontract.Cost},
		CostNewReadPool.String():                  {CostNewReadPool, smartcontract.Cost},
		CostReadPoolLock.String():                 {CostReadPoolLock, smartcontract.Cost},
		CostReadPoolUnlock.String():               {CostReadPoolUnlock, smartcontract.Cost},
		CostWritePoolLock.String():                {CostWritePoolLock, smartcontract.Cost},
		CostWritePoolUnlock.String():              {CostWritePoolUnlock, smartcontract.Cost},
		CostStakePoolLock.String():                {CostStakePoolLock, smartcontract.Cost},
		CostStakePoolUnlock.String():              {CostStakePoolUnlock, smartcontract.Cost},
		CostStakePoolPayInterests.String():        {CostStakePoolPayInterests, smartcontract.Cost},
		CostCommitSettingsChanges.String():        {CostCommitSettingsChanges, smartcontract.Cost},
		CostCollectReward.String():                {CostCollectReward, smartcontract.Cost},
	}
}

func (conf *Config) getConfigMap() (smartcontract.StringMap, error) {
	var out smartcontract.StringMap
	out.Fields = make(map[string]string)
	for _, key := range SettingName {
		info, ok := Settings[key]
		if !ok {
			return out, fmt.Errorf("SettingName %s not found in Settings", key)
		}
		iSetting := conf.get(info.setting)
		if info.configType == smartcontract.CurrencyCoin {
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

const costPrefix = "cost."

func (conf *Config) setCost(key string, change int) error {
	if !isCost(key) {
		return fmt.Errorf("key: %v is not a cost", key)
	}
	if conf.Cost == nil {
		conf.Cost = make(map[string]int)
	}
	conf.Cost[strings.TrimPrefix(key, costPrefix)] = change
	return nil
}

func (conf *Config) getCost(key string) (int, error) {
	if !isCost(key) {
		return 0, fmt.Errorf("key: %v is not a cost", key)
	}
	if conf.Cost == nil {
		return 0, errors.New("cost object is nil")
	}
	value, ok := conf.Cost[strings.TrimPrefix(key, costPrefix)]
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

func (conf *Config) setCoin(key string, change currency.Coin) error {
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
	case WritePoolMinLock:
		if conf.WritePool == nil {
			conf.WritePool = &writePoolConfig{}
		}
		conf.WritePool.MinLock = change
	case ReadPoolMinLock:
		if conf.ReadPool == nil {
			conf.ReadPool = &readPoolConfig{}
		}
		conf.ReadPool.MinLock = change
	case StakePoolMinLock:
		if conf.StakePool == nil {
			conf.StakePool = &stakePoolConfig{}
		}
		conf.StakePool.MinLock = change
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
	case CancellationCharge:
		conf.CancellationCharge = change
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
	case BlockRewardGammaAlpha:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.Gamma.Alpha = change
	case BlockRewardGammaA:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.Gamma.A = change
	case BlockRewardGammaB:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.Gamma.B = change
	case BlockRewardZetaI:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.Zeta.I = change
	case BlockRewardZetaK:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.Zeta.K = change
	case BlockRewardZetaMu:
		if conf.BlockReward == nil {
			conf.BlockReward = &blockReward{}
		}
		conf.BlockReward.Zeta.Mu = change
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
	case StakePoolMinLockPeriod:
		if conf.StakePool == nil {
			conf.StakePool = &stakePoolConfig{}
		}
		conf.StakePool.MinLockPeriod = change
	case FreeAllocationDuration:
		conf.FreeAllocationSettings.Duration = change
	default:
		return fmt.Errorf("key: %v not implemented as duration", key)
	}
	return nil
}

func (conf *Config) setBoolean(key string, change bool) error {
	switch Settings[key].setting {
	case ChallengeEnabled:
		conf.ChallengeEnabled = change
	default:
		return fmt.Errorf("key: %v not implemented as boolean", key)
	}
	return nil
}

func (conf *Config) setKey(key string, change string) {
	switch Settings[key].setting {
	case OwnerId:
		conf.OwnerId = change
	default:
		panic("key: " + key + "not implemented as key")
	}
}

func (conf *Config) set(key string, change string) error {
	s, ok := Settings[key]
	if !ok {
		return fmt.Errorf("unknown key %s, can't set value %v", key, change)
	}

	switch s.configType {
	case smartcontract.Int:
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int: %v", key, change, err)
		}
		if err := conf.setInt(key, value); err != nil {
			return err
		}
	case smartcontract.CurrencyCoin:
		value, err := strconv.ParseFloat(change, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to state.balance: %v", key, change, err)
		}
		vCoin, err := currency.ParseZCN(value)
		if err != nil {
			return err
		}
		if err := conf.setCoin(key, vCoin); err != nil {
			return err
		}
	case smartcontract.Int64:
		value, err := strconv.ParseInt(change, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
		if err := conf.setInt64(key, value); err != nil {
			return err
		}
	case smartcontract.Float64:
		value, err := strconv.ParseFloat(change, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to float64: %v", key, change, err)
		}
		if err := conf.setFloat64(key, value); err != nil {
			return err
		}
	case smartcontract.Duration:
		value, err := time.ParseDuration(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to duration: %v", key, change, err)
		}
		if err := conf.setDuration(key, value); err != nil {
			return err
		}
	case smartcontract.Boolean:
		value, err := strconv.ParseBool(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to boolean: %v", key, change, err)
		}
		if err := conf.setBoolean(key, value); err != nil {
			return err
		}
	case smartcontract.Cost:
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
		if err := conf.setCost(key, value); err != nil {
			return err
		}
	case smartcontract.Key:
		if _, err := hex.DecodeString(change); err != nil {
			return fmt.Errorf("%s must be a hes string: %v", key, err)
		}
		conf.setKey(key, change)
	default:
		return fmt.Errorf("unsupported type setting " + smartcontract.ConfigTypeName[Settings[key].configType])
	}
	return nil
}

func (conf *Config) get(key Setting) interface{} {
	if isCost(key.String()) {
		value, _ := conf.getCost(key.String())
		return value
	}

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
	case WritePoolMinLock:
		return conf.WritePool.MinLock
	case StakePoolMinLock:
		return conf.StakePool.MinLock
	case StakePoolMinLockPeriod:
		return conf.StakePool.MinLockPeriod
	case MaxTotalFreeAllocation:
		return conf.MaxTotalFreeAllocation
	case MaxIndividualFreeAllocation:
		return conf.MaxIndividualFreeAllocation
	case CancellationCharge:
		return conf.CancellationCharge
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
	case BlockRewardGammaAlpha:
		return conf.BlockReward.Gamma.Alpha
	case BlockRewardGammaA:
		return conf.BlockReward.Gamma.A
	case BlockRewardGammaB:
		return conf.BlockReward.Gamma.B
	case BlockRewardZetaI:
		return conf.BlockReward.Zeta.I
	case BlockRewardZetaK:
		return conf.BlockReward.Zeta.K
	case BlockRewardZetaMu:
		return conf.BlockReward.Zeta.Mu
	case OwnerId:
		return conf.OwnerId
	default:
		panic("Setting not implemented")
	}
}

func (conf *Config) update(changes smartcontract.StringMap) error {
	for key, value := range changes.Fields {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if err := conf.set(trimmedKey, trimmedValue); err != nil {
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

	err = conf.update(*updateChanges)
	if err != nil {
		return "", common.NewError("update_settings, updating settings", err.Error())
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
		return "", common.NewError("update_settings_validate", err.Error())
	}

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	if err != nil {
		return "", common.NewError("update_settings_insert", err.Error())
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
