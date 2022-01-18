package storagesc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"0chain.net/smartcontract"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func scConfigKey(scKey string) datastore.Key {
	return datastore.Key(scKey + ":configurations")
}

type freeAllocationSettings struct {
	DataShards                 int           `json:"data_shards"`
	ParityShards               int           `json:"parity_shards"`
	Size                       int64         `json:"size"`
	Duration                   time.Duration `json:"duration"`
	ReadPriceRange             PriceRange    `json:"read_price_range"`
	WritePriceRange            PriceRange    `json:"write_price_range"`
	MaxChallengeCompletionTime time.Duration `json:"max_challenge_completion_time"`
	ReadPoolFraction           float64       `json:"read_pool_fraction"`
}

type stakePoolConfig struct {
	MinLock int64 `json:"min_lock"`
	// Interest rate of the stake pool
	InterestRate     float64       `json:"interest_rate"`
	InterestInterval time.Duration `json:"interest_interval"`
}

type readPoolConfig struct {
	MinLock       int64         `json:"min_lock"`
	MinLockPeriod time.Duration `json:"min_lock_period"`
	MaxLockPeriod time.Duration `json:"max_lock_period"`
}

type writePoolConfig struct {
	MinLock       int64         `json:"min_lock"`
	MinLockPeriod time.Duration `json:"min_lock_period"`
	MaxLockPeriod time.Duration `json:"max_lock_period"`
}

type blockReward struct {
	BlockReward           state.Balance `json:"block_reward"`
	QualifyingStake       state.Balance `json:"qualifying_stake"`
	SharderWeight         float64       `json:"sharder_weight"`
	MinerWeight           float64       `json:"miner_weight"`
	BlobberCapacityWeight float64       `json:"blobber_capacity_weight"`
	BlobberUsageWeight    float64       `json:"blobber_usage_weight"`
}

func (br *blockReward) setWeightsFromRatio(sharderRatio, minerRatio, bCapcacityRatio, bUsageRatio float64) {
	total := sharderRatio + minerRatio + bCapcacityRatio + bUsageRatio
	if total == 0 {
		br.SharderWeight = 0
		br.MinerWeight = 0
		br.BlobberCapacityWeight = 0
		br.BlobberUsageWeight = 0
	} else {
		br.SharderWeight = sharderRatio / total
		br.MinerWeight = minerRatio / total
		br.BlobberCapacityWeight = bCapcacityRatio / total
		br.BlobberUsageWeight = bUsageRatio / total
	}

}

// scConfig represents SC configurations ('storagesc:' from sc.yaml).
type scConfig struct {
	// TimeUnit is a duration used as divider for a write price. A write price
	// measured in tok / GB / time unit. Where the time unit is this
	// configuration.
	TimeUnit time.Duration `json:"time_unit"`
	// MaxMint is max minting.
	MaxMint state.Balance `json:"max_mint"`
	// Minted tokens by entire SC.
	Minted state.Balance `json:"minted"`
	// MinAllocSize is minimum possible size (bytes)
	// of an allocation the SC accept.
	MinAllocSize int64 `json:"min_alloc_size"`
	// MinAllocDuration is minimum possible duration of an
	// allocation allowed by the SC.
	MinAllocDuration time.Duration `json:"min_alloc_duration"`
	// MaxChallengeCompletionTime is max time to complete a challenge.
	MaxChallengeCompletionTime time.Duration `json:"max_challenge_completion_time"`
	// MinOfferDuration represents lower boundary of blobber's MaxOfferDuration.
	MinOfferDuration time.Duration `json:"min_offer_duration"`
	// MinBlobberCapacity allowed to register in the SC.
	MinBlobberCapacity int64 `json:"min_blobber_capacity"`
	// ReadPool related configurations.
	ReadPool *readPoolConfig `json:"readpool"`
	// WritePool related configurations.
	WritePool *writePoolConfig `json:"writepool"`
	// StakePool related configurations.
	StakePool *stakePoolConfig `json:"stakepool"`
	// ValidatorReward represents % (value in [0; 1] range) of blobbers' reward
	// goes to validators. Even if a blobber doesn't pass a challenge validators
	// receive this reward.
	ValidatorReward float64 `json:"validator_reward"`
	// BlobberSlash represents % (value in [0; 1] range) of blobbers' stake
	// tokens penalized on challenge not passed.
	BlobberSlash float64 `json:"blobber_slash"`

	// price limits for blobbers

	// MaxReadPrice allowed for a blobber.
	MaxReadPrice state.Balance `json:"max_read_price"`
	// MaxWrtiePrice
	MaxWritePrice state.Balance `json:"max_write_price"`
	MinWritePrice state.Balance `json:"min_write_price"`

	// allocation cancellation

	// FailedChallengesToCancel is number of failed challenges of an allocation
	// to be able to cancel an allocation.
	FailedChallengesToCancel int `json:"failed_challenges_to_cancel"`
	// FailedChallengesToRevokeMinLock is number of failed challenges of a
	// blobber to revoke its min_lock demand back to user; only part not
	// paid yet can go back.
	FailedChallengesToRevokeMinLock int `json:"failed_challenges_to_revoke_min_lock"`

	// free allocations
	MaxTotalFreeAllocation      state.Balance          `json:"max_total_free_allocation"`
	MaxIndividualFreeAllocation state.Balance          `json:"max_individual_free_allocation"`
	FreeAllocationSettings      freeAllocationSettings `json:"free_allocation_settings"`

	// challenges generating

	// ChallengeEnabled is challenges generating pin.
	ChallengeEnabled bool `json:"challenge_enabled"`
	// MaxChallengesPerGeneration is max number of challenges can be generated
	// at once for a blobber-allocation pair with size difference for the
	// moment of the generation.
	MaxChallengesPerGeneration int `json:"max_challenges_per_generation"`
	// ChallengeGenerationRate is number of challenges generated for a MB/min.
	ChallengeGenerationRate float64 `json:"challenge_rate_per_mb_min"`

	// MinStake allowed by a blobber/validator (entire SC boundary).
	MinStake state.Balance `json:"min_stake"`
	// MaxStake allowed by a blobber/validator (entire SC boundary).
	MaxStake state.Balance `json:"max_stake"`

	// MaxDelegates per stake pool
	MaxDelegates int `json:"max_delegates"`

	// MaxCharge that blobber gets from rewards to its delegate_wallet.
	MaxCharge float64 `json:"max_charge"`

	BlockReward *blockReward `json:"block_reward"`

	// Allow direct access to MPT
	ExposeMpt bool          `json:"expose_mpt"`
	OwnerId   datastore.Key `json:"owner_id"`
}

func (sc *scConfig) validate() (err error) {
	if sc.TimeUnit <= 1*time.Second {
		return fmt.Errorf("time_unit less than 1s: %s", sc.TimeUnit)
	}
	if sc.ValidatorReward < 0.0 || 1.0 < sc.ValidatorReward {
		return fmt.Errorf("validator_reward not in [0; 1] range: %v",
			sc.ValidatorReward)
	}
	if sc.BlobberSlash < 0.0 || 1.0 < sc.BlobberSlash {
		return fmt.Errorf("blobber_slash not in [0; 1] range: %v",
			sc.BlobberSlash)
	}
	if sc.MinBlobberCapacity < 0 {
		return fmt.Errorf("negative min_blobber_capacity: %v",
			sc.MinBlobberCapacity)
	}
	if sc.MinOfferDuration < 0 {
		return fmt.Errorf("negative min_offer_duration: %v",
			sc.MinOfferDuration)
	}
	if sc.MaxChallengeCompletionTime < 0 {
		return fmt.Errorf("negative max_challenge_completion_time: %v",
			sc.MaxChallengeCompletionTime)
	}
	if sc.MinAllocDuration < 0 {
		return fmt.Errorf("negative min_alloc_duration: %v",
			sc.MinAllocDuration)
	}
	if sc.MaxMint < 0 {
		return fmt.Errorf("negative max_mint: %v", sc.MaxMint)
	}
	if sc.MinAllocSize < 0 {
		return fmt.Errorf("negative min_alloc_size: %v", sc.MinAllocSize)
	}
	if sc.MaxReadPrice < 0 {
		return fmt.Errorf("negative max_read_price: %v", sc.MaxReadPrice)
	}
	if sc.MaxWritePrice < 0 {
		return fmt.Errorf("negative max_write_price: %v", sc.MaxWritePrice)
	}
	if sc.MinWritePrice < 0 {
		return fmt.Errorf("negative min_write_price: %v", sc.MaxWritePrice)
	}
	if sc.MaxWritePrice < sc.MinWritePrice {
		return fmt.Errorf("max wirte price %v must be more than min_write_price: %v",
			sc.MaxWritePrice, sc.MinWritePrice)
	}
	if sc.StakePool.MinLock <= 1 {
		return fmt.Errorf("invalid stakepool.min_lock: %v <= 1",
			sc.StakePool.MinLock)
	}
	if sc.StakePool.InterestRate < 0 {
		return fmt.Errorf("negative stakepool.interest_rate: %v",
			sc.StakePool.InterestRate)
	}
	if sc.StakePool.InterestInterval <= 0 {
		return fmt.Errorf("invalid stakepool.interest_interval <= 0: %v",
			sc.StakePool.InterestInterval)
	}

	if sc.MaxTotalFreeAllocation < 0 {
		return fmt.Errorf("negative max_total_free_allocation: %v", sc.MaxTotalFreeAllocation)
	}
	if sc.MaxIndividualFreeAllocation < 0 {
		return fmt.Errorf("negative max_individual_free_allocation: %v", sc.MaxIndividualFreeAllocation)
	}

	if sc.FreeAllocationSettings.DataShards < 0 {
		return fmt.Errorf("negative free_allocation_settings.data_shards: %v",
			sc.FreeAllocationSettings.DataShards)
	}
	if sc.FreeAllocationSettings.ParityShards < 0 {
		return fmt.Errorf("negative free_allocation_settings.parity_shards: %v",
			sc.FreeAllocationSettings.ParityShards)
	}
	if sc.FreeAllocationSettings.Size < 0 {
		return fmt.Errorf("negative free_allocation_settings.size: %v",
			sc.FreeAllocationSettings.Size)
	}
	if sc.FreeAllocationSettings.Duration <= 0 {
		return fmt.Errorf("negative free_allocation_settings.expiration_date: %v",
			sc.FreeAllocationSettings.Duration)
	}
	if !sc.FreeAllocationSettings.ReadPriceRange.isValid() {
		return fmt.Errorf("invalid free_allocation_settings.read_price_range: %v",
			sc.FreeAllocationSettings.ReadPriceRange)
	}
	if !sc.FreeAllocationSettings.WritePriceRange.isValid() {
		return fmt.Errorf("invalid free_allocation_settings.write_price_range: %v",
			sc.FreeAllocationSettings.WritePriceRange)
	}
	if sc.FreeAllocationSettings.MaxChallengeCompletionTime < 0 {
		return fmt.Errorf("negative free_allocation_settings.max_challenge_completion_time: %v",
			sc.FreeAllocationSettings.MaxChallengeCompletionTime)
	}
	if sc.FreeAllocationSettings.ReadPoolFraction < 0 || 1 < sc.FreeAllocationSettings.ReadPoolFraction {
		return fmt.Errorf("free_allocation_settings.free_read_pool must be in [0,1]: %v",
			sc.FreeAllocationSettings.ReadPoolFraction)
	}

	if sc.FailedChallengesToCancel < 0 {
		return fmt.Errorf("negative failed_challenges_to_cancel: %v",
			sc.FailedChallengesToCancel)
	}
	if sc.FailedChallengesToRevokeMinLock < 0 {
		return fmt.Errorf("negative failed_challenges_to_revoke_min_lock: %v",
			sc.FailedChallengesToRevokeMinLock)
	}
	if sc.MaxChallengesPerGeneration <= 0 {
		return fmt.Errorf("invalid max_challenges_per_generation <= 0: %v",
			sc.MaxChallengesPerGeneration)
	}
	if sc.ChallengeGenerationRate < 0 {
		return fmt.Errorf("negative challenge_rate_per_mb_min: %v",
			sc.ChallengeGenerationRate)
	}
	if sc.MinStake < 0 {
		return fmt.Errorf("negative min_stake: %v", sc.MinStake)
	}
	if sc.MaxStake < sc.MinStake {
		return fmt.Errorf("max_stake less than min_stake: %v < %v", sc.MinStake,
			sc.MaxStake)
	}
	if sc.MaxDelegates < 1 {
		return fmt.Errorf("max_delegates is too small %v", sc.MaxDelegates)
	}
	if sc.MaxCharge < 0 {
		return fmt.Errorf("negative max_charge: %v", sc.MaxCharge)
	}
	if sc.MaxCharge > 1.0 {
		return fmt.Errorf("max_change >= 1.0 (> 100%%, invalid): %v",
			sc.MaxCharge)
	}
	if sc.BlockReward.BlockReward < 0 {
		return fmt.Errorf("negative block_reward.block_reward: %v",
			sc.BlockReward.BlockReward)
	}
	if sc.BlockReward.QualifyingStake < 0 {
		return fmt.Errorf("negative block_reward.qualifying_stake: %v",
			sc.BlockReward.QualifyingStake)
	}
	if sc.BlockReward.SharderWeight < 0 {
		return fmt.Errorf("negative block_reward.sharder_weight: %v",
			sc.BlockReward.SharderWeight)
	}
	if sc.BlockReward.MinerWeight < 0 {
		return fmt.Errorf("negative block_reward.miner_weight: %v",
			sc.BlockReward.MinerWeight)
	}
	if sc.BlockReward.BlobberCapacityWeight < 0 {
		return fmt.Errorf("negative block_reward.blobber_capacity_weight: %v",
			sc.BlockReward.BlobberCapacityWeight)
	}
	if sc.BlockReward.BlobberUsageWeight < 0 {
		return fmt.Errorf("negative block_reward.bobber_usage_weight: %v",
			sc.BlockReward.BlobberUsageWeight)
	}
	if len(sc.OwnerId) == 0 {
		return fmt.Errorf("owner_id does not set or empty")
	}
	return
}

func (conf *scConfig) canMint() bool {
	return conf.Minted < conf.MaxMint
}

func (conf *scConfig) validateStakeRange(min, max state.Balance) (err error) {
	if min < conf.MinStake {
		return fmt.Errorf("min_stake is less than allowed by SC: %v < %v", min,
			conf.MinStake)
	}
	if max > conf.MaxStake {
		return fmt.Errorf("max_stake is greater than allowed by SC: %v > %v",
			max, conf.MaxStake)
	}
	if max < min {
		return fmt.Errorf("max_stake less than min_stake: %v < %v", min, max)
	}
	return
}

func (conf *scConfig) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(conf); err != nil {
		panic(err) // must not happens
	}
	return
}

func (conf *scConfig) Decode(b []byte) error {
	return json.Unmarshal(b, conf)
}

//
// rest handler and update function
//

// getConfigBytes returns encoded configurations or an error.
func (ssc *StorageSmartContract) getConfigBytes(
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(scConfigKey(ssc.ID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// configs from sc.yaml
func getConfiguredConfig() (conf *scConfig, err error) {
	const pfx = "smart_contracts.storagesc."

	conf = new(scConfig)
	var scc = config.SmartContractConfig
	// sc
	conf.TimeUnit = scc.GetDuration(pfx + "time_unit")
	conf.MaxMint = state.Balance(scc.GetFloat64(pfx+"max_mint") * 1e10)
	conf.MinStake = state.Balance(scc.GetFloat64(pfx+"min_stake") * 1e10)
	conf.MaxStake = state.Balance(scc.GetFloat64(pfx+"max_stake") * 1e10)
	conf.MinAllocSize = scc.GetInt64(pfx + "min_alloc_size")
	conf.MinAllocDuration = scc.GetDuration(pfx + "min_alloc_duration")
	conf.MaxChallengeCompletionTime = scc.GetDuration(pfx + "max_challenge_completion_time")
	conf.MinOfferDuration = scc.GetDuration(pfx + "min_offer_duration")
	conf.MinBlobberCapacity = scc.GetInt64(pfx + "min_blobber_capacity")
	conf.ValidatorReward = scc.GetFloat64(pfx + "validator_reward")
	conf.BlobberSlash = scc.GetFloat64(pfx + "blobber_slash")
	conf.MaxReadPrice = state.Balance(
		scc.GetFloat64(pfx+"max_read_price") * 1e10)
	conf.MinWritePrice = state.Balance(
		scc.GetFloat64(pfx+"min_write_price") * 1e10)
	conf.MaxWritePrice = state.Balance(
		scc.GetFloat64(pfx+"max_write_price") * 1e10)
	// read pool
	conf.ReadPool = new(readPoolConfig)
	conf.ReadPool.MinLock = int64(scc.GetFloat64(pfx+"readpool.min_lock") * 1e10)
	conf.ReadPool.MinLockPeriod = scc.GetDuration(
		pfx + "readpool.min_lock_period")
	conf.ReadPool.MaxLockPeriod = scc.GetDuration(
		pfx + "readpool.max_lock_period")
	// write pool
	conf.WritePool = new(writePoolConfig)
	conf.WritePool.MinLock = int64(scc.GetFloat64(pfx+"writepool.min_lock") * 1e10)
	conf.WritePool.MinLockPeriod = scc.GetDuration(
		pfx + "writepool.min_lock_period")
	conf.WritePool.MaxLockPeriod = scc.GetDuration(
		pfx + "writepool.max_lock_period")
	// stake pool
	conf.StakePool = new(stakePoolConfig)
	conf.StakePool.MinLock = int64(scc.GetFloat64(pfx+"stakepool.min_lock") * 1e10)
	conf.StakePool.InterestRate = scc.GetFloat64(
		pfx + "stakepool.interest_rate")
	conf.StakePool.InterestInterval = scc.GetDuration(
		pfx + "stakepool.interest_interval")

	conf.MaxTotalFreeAllocation = state.Balance(scc.GetFloat64(pfx+"max_total_free_allocation") * 1e10)
	conf.MaxIndividualFreeAllocation = state.Balance(scc.GetFloat64(pfx+"max_individual_free_allocation") * 1e10)
	fas := pfx + "free_allocation_settings."
	conf.FreeAllocationSettings.DataShards = int(scc.GetFloat64(fas + "data_shards"))
	conf.FreeAllocationSettings.ParityShards = int(scc.GetFloat64(fas + "parity_shards"))
	conf.FreeAllocationSettings.Size = int64(scc.GetFloat64(fas + "size"))
	conf.FreeAllocationSettings.Duration = scc.GetDuration(fas + "duration")
	conf.FreeAllocationSettings.ReadPriceRange = PriceRange{
		Min: state.Balance(scc.GetFloat64(fas+"read_price_range.min") * 1e10),
		Max: state.Balance(scc.GetFloat64(fas+"read_price_range.max") * 1e10),
	}
	conf.FreeAllocationSettings.WritePriceRange = PriceRange{
		Min: state.Balance(scc.GetFloat64(fas+"write_price_range.min") * 1e10),
		Max: state.Balance(scc.GetFloat64(fas+"write_price_range.max") * 1e10),
	}
	conf.FreeAllocationSettings.MaxChallengeCompletionTime = scc.GetDuration(fas + "max_challenge_completion_time")
	conf.FreeAllocationSettings.ReadPoolFraction = scc.GetFloat64(fas + "read_pool_fraction")

	// allocation cancellation
	conf.FailedChallengesToCancel = scc.GetInt(
		pfx + "failed_challenges_to_cancel")
	conf.FailedChallengesToRevokeMinLock = scc.GetInt(
		pfx + "failed_challenges_to_revoke_min_lock")
	// challenges generating
	conf.ChallengeEnabled = scc.GetBool(pfx + "challenge_enabled")
	conf.MaxChallengesPerGeneration = scc.GetInt(
		pfx + "max_challenges_per_generation")
	conf.ChallengeGenerationRate = scc.GetFloat64(
		pfx + "challenge_rate_per_mb_min")

	conf.MaxDelegates = scc.GetInt(pfx + "max_delegates")
	conf.MaxCharge = scc.GetFloat64(pfx + "max_charge")

	conf.BlockReward = new(blockReward)
	conf.BlockReward.BlockReward = state.Balance(scc.GetFloat64(pfx+"block_reward.block_reward") * 1e10)
	conf.BlockReward.QualifyingStake = state.Balance(scc.GetFloat64(pfx+"block_reward.qualifying_stake") * 1e10)

	conf.BlockReward.SharderWeight = scc.GetFloat64(pfx + "block_reward.sharder_weight")
	conf.BlockReward.MinerWeight = scc.GetFloat64(pfx + "block_reward.miner_weight")
	conf.BlockReward.BlobberCapacityWeight = scc.GetFloat64(pfx + "block_reward.blobber_capacity_weight")
	conf.BlockReward.BlobberUsageWeight = scc.GetFloat64(pfx + "block_reward.blobber_usage_weight" +
		"blobber_usage_weight")
	conf.BlockReward.setWeightsFromRatio(
		scc.GetFloat64(pfx+"block_reward.sharder_ratio"),
		scc.GetFloat64(pfx+"block_reward.miner_ratio"),
		scc.GetFloat64(pfx+"block_reward.blobber_capacity_ratio"),
		scc.GetFloat64(pfx+"block_reward.blobber_usage_ratio"),
	)
	conf.ExposeMpt = scc.GetBool(pfx + "expose_mpt")
	conf.OwnerId = scc.GetString(pfx + "owner_id")

	err = conf.validate()
	return
}

func (ssc *StorageSmartContract) setupConfig(
	balances chainState.StateContextI) (conf *scConfig, err error) {

	if conf, err = getConfiguredConfig(); err != nil {
		return
	}
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return nil, err
	}
	return
}

// getConfig
func (ssc *StorageSmartContract) getConfig(
	balances chainState.StateContextI, setup bool) (
	conf *scConfig, err error) {

	var confb []byte
	confb, err = ssc.getConfigBytes(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	conf = new(scConfig)

	if err == util.ErrValueNotPresent {
		if !setup {
			return // value not present
		}
		return ssc.setupConfig(balances)
	}

	if err = conf.Decode(confb); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return
}

const cantGetConfigErrMsg = "can't get config"

func (ssc *StorageSmartContract) getConfigHandler(
	ctx context.Context,
	params url.Values,
	balances chainState.StateContextI,
) (resp interface{}, err error) {
	var conf *scConfig
	conf, err = ssc.getConfig(balances, false)

	if err != nil && err != util.ErrValueNotPresent {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg)
	}

	// return configurations from sc.yaml not saving them
	if err == util.ErrValueNotPresent {
		conf, err = getConfiguredConfig()
		if err != nil {
			return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg)
		}
	}

	return conf.getConfigMap() // actual value
}

// getWritePoolConfig
func (ssc *StorageSmartContract) getWritePoolConfig(
	balances chainState.StateContextI, setup bool) (
	conf *writePoolConfig, err error) {

	var scconf *scConfig
	if scconf, err = ssc.getConfig(balances, setup); err != nil {
		return
	}
	return scconf.WritePool, nil
}

// getReadPoolConfig
func (ssc *StorageSmartContract) getReadPoolConfig(
	balances chainState.StateContextI, setup bool) (
	conf *readPoolConfig, err error) {

	var scconf *scConfig
	if scconf, err = ssc.getConfig(balances, setup); err != nil {
		return
	}
	return scconf.ReadPool, nil
}
