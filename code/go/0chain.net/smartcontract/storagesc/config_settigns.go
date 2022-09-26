package storagesc

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/currency"

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
	SettingName[Cost] = "cost"
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
	SettingName[CostAddCurator] = "cost.add_curator"
	SettingName[CostRemoveCurator] = "cost.remove_curator"
	SettingName[CostBlobberHealthCheck] = "cost.blobber_health_check"
	SettingName[CostUpdateBlobberSettings] = "cost.update_blobber_settings"
	SettingName[CostPayBlobberBlockRewards] = "cost.pay_blobber_block_rewards"
	SettingName[CostCuratorTransferAllocation] = "cost.curator_transfer_allocation"
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
		SettingName[MaxMint]:                          {MaxMint, smartcontract.CurrencyCoin},
		SettingName[TimeUnit]:                         {TimeUnit, smartcontract.Duration},
		SettingName[MinAllocSize]:                     {MinAllocSize, smartcontract.Int64},
		SettingName[MinAllocDuration]:                 {MinAllocDuration, smartcontract.Duration},
		SettingName[MaxChallengeCompletionTime]:       {MaxChallengeCompletionTime, smartcontract.Duration},
		SettingName[MinOfferDuration]:                 {MinOfferDuration, smartcontract.Duration},
		SettingName[MinBlobberCapacity]:               {MinBlobberCapacity, smartcontract.Int64},
		SettingName[ReadPoolMinLock]:                  {ReadPoolMinLock, smartcontract.CurrencyCoin},
		SettingName[WritePoolMinLock]:                 {WritePoolMinLock, smartcontract.CurrencyCoin},
		SettingName[StakePoolMinLock]:                 {StakePoolMinLock, smartcontract.CurrencyCoin},
		SettingName[StakePoolMinLockPeriod]:           {StakePoolMinLockPeriod, smartcontract.Duration},
		SettingName[MaxTotalFreeAllocation]:           {MaxTotalFreeAllocation, smartcontract.CurrencyCoin},
		SettingName[MaxIndividualFreeAllocation]:      {MaxIndividualFreeAllocation, smartcontract.CurrencyCoin},
		SettingName[CancellationCharge]:               {CancellationCharge, smartcontract.Float64},
		SettingName[FreeAllocationDataShards]:         {FreeAllocationDataShards, smartcontract.Int},
		SettingName[FreeAllocationParityShards]:       {FreeAllocationParityShards, smartcontract.Int},
		SettingName[FreeAllocationSize]:               {FreeAllocationSize, smartcontract.Int64},
		SettingName[FreeAllocationDuration]:           {FreeAllocationDuration, smartcontract.Duration},
		SettingName[FreeAllocationReadPriceRangeMin]:  {FreeAllocationReadPriceRangeMin, smartcontract.CurrencyCoin},
		SettingName[FreeAllocationReadPriceRangeMax]:  {FreeAllocationReadPriceRangeMax, smartcontract.CurrencyCoin},
		SettingName[FreeAllocationWritePriceRangeMin]: {FreeAllocationWritePriceRangeMin, smartcontract.CurrencyCoin},
		SettingName[FreeAllocationWritePriceRangeMax]: {FreeAllocationWritePriceRangeMax, smartcontract.CurrencyCoin},
		SettingName[FreeAllocationReadPoolFraction]:   {FreeAllocationReadPoolFraction, smartcontract.Float64},
		SettingName[ValidatorReward]:                  {ValidatorReward, smartcontract.Float64},
		SettingName[BlobberSlash]:                     {BlobberSlash, smartcontract.Float64},
		SettingName[MaxBlobbersPerAllocation]:         {MaxBlobbersPerAllocation, smartcontract.Int},
		SettingName[MaxReadPrice]:                     {MaxReadPrice, smartcontract.CurrencyCoin},
		SettingName[MaxWritePrice]:                    {MaxWritePrice, smartcontract.CurrencyCoin},
		SettingName[MinWritePrice]:                    {MinWritePrice, smartcontract.CurrencyCoin},
		SettingName[FailedChallengesToCancel]:         {FailedChallengesToCancel, smartcontract.Int},
		SettingName[FailedChallengesToRevokeMinLock]:  {FailedChallengesToRevokeMinLock, smartcontract.Int},
		SettingName[ChallengeEnabled]:                 {ChallengeEnabled, smartcontract.Boolean},
		SettingName[ChallengeGenerationRate]:          {ChallengeGenerationRate, smartcontract.Float64},
		SettingName[MaxChallengesPerGeneration]:       {MaxChallengesPerGeneration, smartcontract.Int},
		SettingName[ValidatorsPerChallenge]:           {ValidatorsPerChallenge, smartcontract.Int},
		SettingName[MaxDelegates]:                     {MaxDelegates, smartcontract.Int},
		SettingName[BlockRewardBlockReward]:           {BlockRewardBlockReward, smartcontract.CurrencyCoin},
		SettingName[BlockRewardQualifyingStake]:       {BlockRewardQualifyingStake, smartcontract.CurrencyCoin},
		SettingName[BlockRewardSharderWeight]:         {BlockRewardSharderWeight, smartcontract.Float64},
		SettingName[BlockRewardMinerWeight]:           {BlockRewardMinerWeight, smartcontract.Float64},
		SettingName[BlockRewardBlobberWeight]:         {BlockRewardBlobberWeight, smartcontract.Float64},
		SettingName[BlockRewardGammaAlpha]:            {BlockRewardGammaAlpha, smartcontract.Float64},
		SettingName[BlockRewardGammaA]:                {BlockRewardGammaA, smartcontract.Float64},
		SettingName[BlockRewardGammaB]:                {BlockRewardGammaB, smartcontract.Float64},
		SettingName[BlockRewardZetaI]:                 {BlockRewardZetaI, smartcontract.Float64},
		SettingName[BlockRewardZetaK]:                 {BlockRewardZetaK, smartcontract.Float64},
		SettingName[BlockRewardZetaMu]:                {BlockRewardZetaMu, smartcontract.Float64},
		SettingName[OwnerId]:                          {OwnerId, smartcontract.Key},
		SettingName[Cost]:                             {Cost, smartcontract.Cost},
		SettingName[CostUpdateSettings]:               {CostUpdateSettings, smartcontract.Cost},
		SettingName[CostReadRedeem]:                   {CostReadRedeem, smartcontract.Cost},
		SettingName[CostCommitConnection]:             {CostCommitConnection, smartcontract.Cost},
		SettingName[CostNewAllocationRequest]:         {CostNewAllocationRequest, smartcontract.Cost},
		SettingName[CostUpdateAllocationRequest]:      {CostUpdateAllocationRequest, smartcontract.Cost},
		SettingName[CostFinalizeAllocation]:           {CostFinalizeAllocation, smartcontract.Cost},
		SettingName[CostCancelAllocation]:             {CostCancelAllocation, smartcontract.Cost},
		SettingName[CostAddFreeStorageAssigner]:       {CostAddFreeStorageAssigner, smartcontract.Cost},
		SettingName[CostFreeAllocationRequest]:        {CostFreeAllocationRequest, smartcontract.Cost},
		SettingName[CostFreeUpdateAllocation]:         {CostFreeUpdateAllocation, smartcontract.Cost},
		SettingName[CostAddCurator]:                   {CostAddCurator, smartcontract.Cost},
		SettingName[CostRemoveCurator]:                {CostRemoveCurator, smartcontract.Cost},
		SettingName[CostBlobberHealthCheck]:           {CostBlobberHealthCheck, smartcontract.Cost},
		SettingName[CostUpdateBlobberSettings]:        {CostUpdateBlobberSettings, smartcontract.Cost},
		SettingName[CostPayBlobberBlockRewards]:       {CostPayBlobberBlockRewards, smartcontract.Cost},
		SettingName[CostCuratorTransferAllocation]:    {CostCuratorTransferAllocation, smartcontract.Cost},
		SettingName[CostChallengeRequest]:             {CostChallengeRequest, smartcontract.Cost},
		SettingName[CostChallengeResponse]:            {CostChallengeResponse, smartcontract.Cost},
		SettingName[CostGenerateChallenges]:           {CostGenerateChallenges, smartcontract.Cost},
		SettingName[CostAddValidator]:                 {CostAddValidator, smartcontract.Cost},
		SettingName[CostUpdateValidatorSettings]:      {CostUpdateValidatorSettings, smartcontract.Cost},
		SettingName[CostAddBlobber]:                   {CostAddBlobber, smartcontract.Cost},
		SettingName[CostNewReadPool]:                  {CostNewReadPool, smartcontract.Cost},
		SettingName[CostReadPoolLock]:                 {CostReadPoolLock, smartcontract.Cost},
		SettingName[CostReadPoolUnlock]:               {CostReadPoolUnlock, smartcontract.Cost},
		SettingName[CostWritePoolLock]:                {CostWritePoolLock, smartcontract.Cost},
		SettingName[CostWritePoolUnlock]:              {CostWritePoolUnlock, smartcontract.Cost},
		SettingName[CostStakePoolLock]:                {CostStakePoolLock, smartcontract.Cost},
		SettingName[CostStakePoolUnlock]:              {CostStakePoolUnlock, smartcontract.Cost},
		SettingName[CostStakePoolPayInterests]:        {CostStakePoolPayInterests, smartcontract.Cost},
		SettingName[CostCommitSettingsChanges]:        {CostCommitSettingsChanges, smartcontract.Cost},
		SettingName[CostCollectReward]:                {CostCollectReward, smartcontract.Cost},
	}
}

func (conf *Config) getConfigMap() (smartcontract.StringMap, error) {
	var out smartcontract.StringMap
	out.Fields = make(map[string]string)
	for _, key := range SettingName {
		info, ok := Settings[strings.ToLower(key)]
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

func (conf *Config) setCost(key string, change int) {
	if change < 0 {
		return
	}
	conf.Cost[strings.TrimPrefix(key, fmt.Sprintf("%s.", SettingName[Cost]))] = change
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
	case smartcontract.CurrencyCoin:
		if value, err := strconv.ParseFloat(change, 64); err == nil {
			vCoin, err2 := currency.ParseZCN(value)
			if err2 != nil {
				return err2
			}
			if err := conf.setCoin(key, vCoin); err != nil {
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
	case CostUpdateValidatorSettings:
		return conf.Cost[strings.ToLower(strings.TrimPrefix(SettingName[CostUpdateValidatorSettings], fmt.Sprintf("%s.", SettingName[Cost])))]
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
		return "", common.NewError("update_settings_validate", err.Error())
	}

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
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
