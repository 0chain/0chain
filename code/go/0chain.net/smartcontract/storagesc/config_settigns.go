package storagesc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"0chain.net/core/config"
	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"

	chainState "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

type Setting int

var settingChangesKey = datastore.Key(ADDRESS + encryption.Hash("setting_changes"))

const x10 = 10 * 1000 * 1000 * 1000

const (
	MaxMint             Setting = iota
	MaxStake            Setting = iota
	MinStake            Setting = iota
	MinStakePerDelegate Setting = iota
	TimeUnit
	MinAllocSize
	MaxChallengeCompletionRounds
	MinBlobberCapacity

	ReadPoolMinLock
	WritePoolMinLock

	StakePoolMinLockPeriod
	StakePoolKillSlash
	MaxTotalFreeAllocation
	MaxIndividualFreeAllocation
	CancellationCharge
	MinLockDemand

	FreeAllocationDataShards
	FreeAllocationParityShards
	FreeAllocationSize
	FreeAllocationReadPriceRangeMin
	FreeAllocationReadPriceRangeMax
	FreeAllocationWritePriceRangeMin
	FreeAllocationWritePriceRangeMax
	FreeAllocationReadPoolFraction

	ValidatorReward
	BlobberSlash

	HealthCheckPeriod
	MaxBlobbersPerAllocation
	MaxReadPrice
	MaxWritePrice
	MinWritePrice
	MaxFileSize
	ChallengeEnabled
	ChallengeGenerationGap
	ValidatorsPerChallenge
	NumValidatorsRewarded
	MaxBlobberSelectForChallenge
	MaxDelegates

	BlockRewardBlockReward
	BlockRewardQualifyingStake
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
	CostChallengeResponse
	CostGenerateChallenges
	CostAddValidator
	CostUpdateValidatorSettings
	CostAddBlobber
	CostReadPoolLock
	CostReadPoolUnlock
	CostWritePoolLock
	CostWritePoolUnlock
	CostStakePoolLock
	CostStakePoolUnlock
	CostCommitSettingsChanges
	CostCollectReward
	CostKillBlobber
	CostKillValidator
	CostShutdownBlobber
	CostShutdownValidator
	MaxCharge
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
		configType config.ConfigType
	}
)

func init() {
	initSettingName()
	initSettings()
}

func initSettingName() {
	SettingName[MaxMint] = "max_mint"
	SettingName[MaxStake] = "max_stake"
	SettingName[MinStake] = "min_stake"
	SettingName[MinStakePerDelegate] = "min_stake_per_delegate"
	SettingName[TimeUnit] = "time_unit"
	SettingName[MinAllocSize] = "min_alloc_size"
	SettingName[MaxChallengeCompletionRounds] = "max_challenge_completion_rounds"
	SettingName[MinBlobberCapacity] = "min_blobber_capacity"
	SettingName[MaxCharge] = "max_charge"
	SettingName[ReadPoolMinLock] = "readpool.min_lock"
	SettingName[WritePoolMinLock] = "writepool.min_lock"
	SettingName[StakePoolKillSlash] = "stakepool.kill_slash"
	SettingName[StakePoolMinLockPeriod] = "stakepool.min_lock_period"
	SettingName[MaxTotalFreeAllocation] = "max_total_free_allocation"
	SettingName[MaxIndividualFreeAllocation] = "max_individual_free_allocation"
	SettingName[CancellationCharge] = "cancellation_charge"
	SettingName[MinLockDemand] = "min_lock_demand"
	SettingName[FreeAllocationDataShards] = "free_allocation_settings.data_shards"
	SettingName[FreeAllocationParityShards] = "free_allocation_settings.parity_shards"
	SettingName[FreeAllocationSize] = "free_allocation_settings.size"
	SettingName[FreeAllocationReadPriceRangeMin] = "free_allocation_settings.read_price_range.min"
	SettingName[FreeAllocationReadPriceRangeMax] = "free_allocation_settings.read_price_range.max"
	SettingName[FreeAllocationWritePriceRangeMin] = "free_allocation_settings.write_price_range.min"
	SettingName[FreeAllocationWritePriceRangeMax] = "free_allocation_settings.write_price_range.max"
	SettingName[FreeAllocationReadPoolFraction] = "free_allocation_settings.read_pool_fraction"
	SettingName[ValidatorReward] = "validator_reward"
	SettingName[BlobberSlash] = "blobber_slash"
	SettingName[HealthCheckPeriod] = "health_check_period"
	SettingName[MaxBlobbersPerAllocation] = "max_blobbers_per_allocation"
	SettingName[MaxReadPrice] = "max_read_price"
	SettingName[MaxWritePrice] = "max_write_price"
	SettingName[MinWritePrice] = "min_write_price"
	SettingName[MaxFileSize] = "max_file_size"
	SettingName[ChallengeEnabled] = "challenge_enabled"
	SettingName[ChallengeGenerationGap] = "challenge_generation_gap"
	SettingName[ValidatorsPerChallenge] = "validators_per_challenge"
	SettingName[NumValidatorsRewarded] = "num_validators_rewarded"
	SettingName[MaxBlobberSelectForChallenge] = "max_blobber_select_for_challenge"
	SettingName[MaxDelegates] = "max_delegates"
	SettingName[BlockRewardBlockReward] = "block_reward.block_reward"
	SettingName[BlockRewardQualifyingStake] = "block_reward.qualifying_stake"
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
	SettingName[CostChallengeResponse] = "cost.challenge_response"
	SettingName[CostGenerateChallenges] = "cost.generate_challenge"
	SettingName[CostAddValidator] = "cost.add_validator"
	SettingName[CostUpdateValidatorSettings] = "cost.update_validator_settings"
	SettingName[CostAddBlobber] = "cost.add_blobber"
	SettingName[CostReadPoolLock] = "cost.read_pool_lock"
	SettingName[CostReadPoolUnlock] = "cost.read_pool_unlock"
	SettingName[CostWritePoolLock] = "cost.write_pool_lock"
	SettingName[CostWritePoolUnlock] = "cost.write_pool_unlock"
	SettingName[CostStakePoolLock] = "cost.stake_pool_lock"
	SettingName[CostStakePoolUnlock] = "cost.stake_pool_unlock"
	SettingName[CostCommitSettingsChanges] = "cost.commit_settings_changes"
	SettingName[CostCollectReward] = "cost.collect_reward"
	SettingName[CostKillBlobber] = "cost.kill_blobber"
	SettingName[CostKillValidator] = "cost.kill_validator"
	SettingName[CostShutdownBlobber] = "cost.shutdown_blobber"
	SettingName[CostShutdownValidator] = "cost.shutdown_validator"
}

func initSettings() {
	Settings = map[string]struct {
		setting    Setting
		configType config.ConfigType
	}{
		MaxMint.String():                          {MaxMint, config.CurrencyCoin},
		MaxStake.String():                         {MaxStake, config.CurrencyCoin},
		MinStake.String():                         {MinStake, config.CurrencyCoin},
		MinStakePerDelegate.String():              {MinStakePerDelegate, config.CurrencyCoin},
		MaxCharge.String():                        {MaxCharge, config.Float64},
		TimeUnit.String():                         {TimeUnit, config.Duration},
		MinAllocSize.String():                     {MinAllocSize, config.Int64},
		MaxChallengeCompletionRounds.String():     {MaxChallengeCompletionRounds, config.Int64},
		MinBlobberCapacity.String():               {MinBlobberCapacity, config.Int64},
		ReadPoolMinLock.String():                  {ReadPoolMinLock, config.CurrencyCoin},
		WritePoolMinLock.String():                 {WritePoolMinLock, config.CurrencyCoin},
		StakePoolMinLockPeriod.String():           {StakePoolMinLockPeriod, config.Duration},
		StakePoolKillSlash.String():               {StakePoolKillSlash, config.Float64},
		MaxTotalFreeAllocation.String():           {MaxTotalFreeAllocation, config.CurrencyCoin},
		MaxIndividualFreeAllocation.String():      {MaxIndividualFreeAllocation, config.CurrencyCoin},
		CancellationCharge.String():               {CancellationCharge, config.Float64},
		MinLockDemand.String():                    {MinLockDemand, config.Float64},
		FreeAllocationDataShards.String():         {FreeAllocationDataShards, config.Int},
		FreeAllocationParityShards.String():       {FreeAllocationParityShards, config.Int},
		FreeAllocationSize.String():               {FreeAllocationSize, config.Int64},
		FreeAllocationReadPriceRangeMin.String():  {FreeAllocationReadPriceRangeMin, config.CurrencyCoin},
		FreeAllocationReadPriceRangeMax.String():  {FreeAllocationReadPriceRangeMax, config.CurrencyCoin},
		FreeAllocationWritePriceRangeMin.String(): {FreeAllocationWritePriceRangeMin, config.CurrencyCoin},
		FreeAllocationWritePriceRangeMax.String(): {FreeAllocationWritePriceRangeMax, config.CurrencyCoin},
		FreeAllocationReadPoolFraction.String():   {FreeAllocationReadPoolFraction, config.Float64},
		ValidatorReward.String():                  {ValidatorReward, config.Float64},
		BlobberSlash.String():                     {BlobberSlash, config.Float64},
		HealthCheckPeriod.String():                {HealthCheckPeriod, config.Duration},
		MaxBlobbersPerAllocation.String():         {MaxBlobbersPerAllocation, config.Int},
		MaxReadPrice.String():                     {MaxReadPrice, config.CurrencyCoin},
		MaxWritePrice.String():                    {MaxWritePrice, config.CurrencyCoin},
		MinWritePrice.String():                    {MinWritePrice, config.CurrencyCoin},
		MaxFileSize.String():                      {MaxFileSize, config.Int64},
		ChallengeEnabled.String():                 {ChallengeEnabled, config.Boolean},
		ValidatorsPerChallenge.String():           {ValidatorsPerChallenge, config.Int},
		NumValidatorsRewarded.String():            {NumValidatorsRewarded, config.Int},
		MaxBlobberSelectForChallenge.String():     {MaxBlobberSelectForChallenge, config.Int},
		MaxDelegates.String():                     {MaxDelegates, config.Int},
		BlockRewardBlockReward.String():           {BlockRewardBlockReward, config.CurrencyCoin},
		BlockRewardQualifyingStake.String():       {BlockRewardQualifyingStake, config.CurrencyCoin},
		BlockRewardGammaAlpha.String():            {BlockRewardGammaAlpha, config.Float64},
		BlockRewardGammaA.String():                {BlockRewardGammaA, config.Float64},
		BlockRewardGammaB.String():                {BlockRewardGammaB, config.Float64},
		BlockRewardZetaI.String():                 {BlockRewardZetaI, config.Float64},
		BlockRewardZetaK.String():                 {BlockRewardZetaK, config.Float64},
		BlockRewardZetaMu.String():                {BlockRewardZetaMu, config.Float64},
		OwnerId.String():                          {OwnerId, config.Key},
		CostUpdateSettings.String():               {CostUpdateSettings, config.Cost},
		CostReadRedeem.String():                   {CostReadRedeem, config.Cost},
		CostCommitConnection.String():             {CostCommitConnection, config.Cost},
		CostNewAllocationRequest.String():         {CostNewAllocationRequest, config.Cost},
		CostUpdateAllocationRequest.String():      {CostUpdateAllocationRequest, config.Cost},
		CostFinalizeAllocation.String():           {CostFinalizeAllocation, config.Cost},
		CostCancelAllocation.String():             {CostCancelAllocation, config.Cost},
		CostAddFreeStorageAssigner.String():       {CostAddFreeStorageAssigner, config.Cost},
		CostFreeAllocationRequest.String():        {CostFreeAllocationRequest, config.Cost},
		CostFreeUpdateAllocation.String():         {CostFreeUpdateAllocation, config.Cost},
		CostBlobberHealthCheck.String():           {CostBlobberHealthCheck, config.Cost},
		CostUpdateBlobberSettings.String():        {CostUpdateBlobberSettings, config.Cost},
		CostPayBlobberBlockRewards.String():       {CostPayBlobberBlockRewards, config.Cost},
		CostChallengeResponse.String():            {CostChallengeResponse, config.Cost},
		CostGenerateChallenges.String():           {CostGenerateChallenges, config.Cost},
		CostAddValidator.String():                 {CostAddValidator, config.Cost},
		CostUpdateValidatorSettings.String():      {CostUpdateValidatorSettings, config.Cost},
		CostAddBlobber.String():                   {CostAddBlobber, config.Cost},
		CostReadPoolLock.String():                 {CostReadPoolLock, config.Cost},
		CostReadPoolUnlock.String():               {CostReadPoolUnlock, config.Cost},
		CostWritePoolLock.String():                {CostWritePoolLock, config.Cost},
		CostWritePoolUnlock.String():              {CostWritePoolUnlock, config.Cost},
		CostStakePoolLock.String():                {CostStakePoolLock, config.Cost},
		CostStakePoolUnlock.String():              {CostStakePoolUnlock, config.Cost},
		CostCommitSettingsChanges.String():        {CostCommitSettingsChanges, config.Cost},
		CostCollectReward.String():                {CostCollectReward, config.Cost},
		CostKillBlobber.String():                  {CostKillBlobber, config.Cost},
		CostKillValidator.String():                {CostKillValidator, config.Cost},
		CostShutdownBlobber.String():              {CostShutdownBlobber, config.Cost},
		CostShutdownValidator.String():            {CostShutdownValidator, config.Cost},
	}
}

func (conf *Config) getConfigMap() (config.StringMap, error) {
	var out config.StringMap
	out.Fields = make(map[string]string)
	for _, key := range SettingName {
		info, ok := Settings[key]
		if !ok {
			return out, fmt.Errorf("SettingName %s not found in Settings", key)
		}
		iSetting := conf.get(info.setting)
		if info.configType == config.CurrencyCoin {
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
	case MaxBlobbersPerAllocation:
		conf.MaxBlobbersPerAllocation = change
	case ValidatorsPerChallenge:
		conf.ValidatorsPerChallenge = change
	case NumValidatorsRewarded:
		conf.NumValidatorsRewarded = change
	case MaxBlobberSelectForChallenge:
		conf.MaxBlobberSelectForChallenge = change
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
	case MaxStake:
		conf.MaxStake = change
	case MinStake:
		conf.MinStake = change
	case MinStakePerDelegate:
		conf.MinStakePerDelegate = change
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
	default:
		return fmt.Errorf("key: %v not implemented as balance", key)
	}

	return nil
}

func (conf *Config) setInt64(key string, change int64) error {
	switch Settings[key].setting {
	case MaxFileSize:
		conf.MaxFileSize = change
	case MinAllocSize:
		conf.MinAllocSize = change
	case MinBlobberCapacity:
		conf.MinBlobberCapacity = change
	case FreeAllocationSize:
		conf.FreeAllocationSettings.Size = change
	case MaxChallengeCompletionRounds:
		conf.MaxChallengeCompletionRounds = change
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
	case MinLockDemand:
		conf.MinLockDemand = change
	case StakePoolKillSlash:
		conf.StakePool.KillSlash = change
	case BlobberSlash:
		conf.BlobberSlash = change
	case MaxCharge:
		conf.MaxCharge = change
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
	case StakePoolMinLockPeriod:
		if conf.StakePool == nil {
			conf.StakePool = &stakePoolConfig{}
		}
		conf.StakePool.MinLockPeriod = change
	case HealthCheckPeriod:
		conf.HealthCheckPeriod = change
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
	case config.Int:
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int: %v", key, change, err)
		}
		if err := conf.setInt(key, value); err != nil {
			return err
		}
	case config.CurrencyCoin:
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
	case config.Int64:
		value, err := strconv.ParseInt(change, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
		if err := conf.setInt64(key, value); err != nil {
			return err
		}
	case config.Float64:
		value, err := strconv.ParseFloat(change, 64)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to float64: %v", key, change, err)
		}
		if err := conf.setFloat64(key, value); err != nil {
			return err
		}
	case config.Duration:
		value, err := time.ParseDuration(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to duration: %v", key, change, err)
		}
		if err := conf.setDuration(key, value); err != nil {
			return err
		}
	case config.Boolean:
		value, err := strconv.ParseBool(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to boolean: %v", key, change, err)
		}
		if err := conf.setBoolean(key, value); err != nil {
			return err
		}
	case config.Cost:
		value, err := strconv.Atoi(change)
		if err != nil {
			return fmt.Errorf("cannot convert key %s value %v to int64: %v", key, change, err)
		}
		if err := conf.setCost(key, value); err != nil {
			return err
		}
	case config.Key:
		if _, err := hex.DecodeString(change); err != nil {
			return fmt.Errorf("%s must be a hes string: %v", key, err)
		}
		conf.setKey(key, change)
	default:
		return fmt.Errorf("unsupported type setting " + config.ConfigTypeName[Settings[key].configType])
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
	case MaxStake:
		return conf.MaxStake
	case MinStake:
		return conf.MinStake
	case MinStakePerDelegate:
		return conf.MinStakePerDelegate
	case TimeUnit:
		return conf.TimeUnit
	case MinAllocSize:
		return conf.MinAllocSize
	case MaxChallengeCompletionRounds:
		return conf.MaxChallengeCompletionRounds
	case MinBlobberCapacity:
		return conf.MinBlobberCapacity
	case ReadPoolMinLock:
		return conf.ReadPool.MinLock
	case WritePoolMinLock:
		return conf.WritePool.MinLock
	case StakePoolMinLockPeriod:
		return conf.StakePool.MinLockPeriod
	case MaxTotalFreeAllocation:
		return conf.MaxTotalFreeAllocation
	case MaxIndividualFreeAllocation:
		return conf.MaxIndividualFreeAllocation
	case CancellationCharge:
		return conf.CancellationCharge
	case MinLockDemand:
		return conf.MinLockDemand
	case FreeAllocationDataShards:
		return conf.FreeAllocationSettings.DataShards
	case FreeAllocationParityShards:
		return conf.FreeAllocationSettings.ParityShards
	case FreeAllocationSize:
		return conf.FreeAllocationSettings.Size
	case HealthCheckPeriod:
		return conf.HealthCheckPeriod
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
	case StakePoolKillSlash:
		return conf.StakePool.KillSlash
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
	case MaxFileSize:
		return conf.MaxFileSize
	case ChallengeEnabled:
		return conf.ChallengeEnabled
	case ValidatorsPerChallenge:
		return conf.ValidatorsPerChallenge
	case NumValidatorsRewarded:
		return conf.NumValidatorsRewarded
	case MaxBlobberSelectForChallenge:
		return conf.MaxBlobberSelectForChallenge
	case MaxDelegates:
		return conf.MaxDelegates
	case BlockRewardBlockReward:
		return conf.BlockReward.BlockReward
	case BlockRewardQualifyingStake:
		return conf.BlockReward.QualifyingStake
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
	case MaxCharge:
		return conf.MaxCharge
	default:
		panic("Setting not implemented")
	}
}

func (conf *Config) update(changes config.StringMap) error {
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

	var newChanges config.StringMap
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

func getSettingChanges(balances cstate.StateContextI) (*config.StringMap, error) {
	var changes = new(config.StringMap)
	err := balances.GetTrieNode(settingChangesKey, changes)
	switch err {
	case nil:
		if len(changes.Fields) == 0 {
			return config.NewStringMap(), nil
		}
		return changes, nil
	case util.ErrValueNotPresent:
		return config.NewStringMap(), nil
	default:
		return nil, err
	}
}
