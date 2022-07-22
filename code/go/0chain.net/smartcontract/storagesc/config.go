package storagesc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/chaincore/currency"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

func scConfigKey(scKey string) datastore.Key {
	return scKey + ":configurations"
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
	MinLock       currency.Coin `json:"min_lock"`
	MinLockPeriod time.Duration `json:"min_lock_period"`
}

type readPoolConfig struct {
	MinLock currency.Coin `json:"min_lock"`
}

type writePoolConfig struct {
	MinLock currency.Coin `json:"min_lock"`
}

type blockReward struct {
	BlockReward             currency.Coin    `json:"block_reward"`
	BlockRewardChangePeriod int64            `json:"block_reward_change_period"`
	BlockRewardChangeRatio  float64          `json:"block_reward_change_ratio"`
	QualifyingStake         currency.Coin    `json:"qualifying_stake"`
	SharderWeight           float64          `json:"sharder_weight"`
	MinerWeight             float64          `json:"miner_weight"`
	BlobberWeight           float64          `json:"blobber_weight"`
	TriggerPeriod           int64            `json:"trigger_period"`
	Gamma                   blockRewardGamma `json:"gamma"`
	Zeta                    blockRewardZeta  `json:"zeta"`
}

type blockRewardGamma struct {
	Alpha float64 `json:"alpha"`
	A     float64 `json:"a"`
	B     float64 `json:"b"`
}

type blockRewardZeta struct {
	I  float64 `json:"i"`
	K  float64 `json:"k"`
	Mu float64 `json:"mu"`
}

func (br *blockReward) setWeightsFromRatio(sharderRatio, minerRatio, bRatio float64) {
	total := sharderRatio + minerRatio + bRatio
	if total == 0 {
		br.SharderWeight = 0
		br.MinerWeight = 0
		br.BlobberWeight = 0
	} else {
		br.SharderWeight = sharderRatio / total
		br.MinerWeight = minerRatio / total
		br.BlobberWeight = bRatio / total
	}

}

// Config represents SC configurations ('storagesc:' from sc.yaml).
type Config struct {
	// TimeUnit is a duration used as divider for a write price. A write price
	// measured in tok / GB / time unit. Where the time unit is this
	// configuration.
	TimeUnit time.Duration `json:"time_unit"`
	// MaxMint is max minting.
	MaxMint currency.Coin `json:"max_mint"`
	// Minted tokens by entire SC.
	Minted currency.Coin `json:"minted"`
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
	WritePool *writePoolConfig `json:"write_pool"`
	// StakePool related configurations.
	StakePool *stakePoolConfig `json:"stakepool"`
	// ValidatorReward represents % (value in [0; 1] range) of blobbers' reward
	// goes to validators. Even if a blobber doesn't pass a challenge validators
	// receive this reward.
	ValidatorReward float64 `json:"validator_reward"`
	// BlobberSlash represents % (value in [0; 1] range) of blobbers' stake
	// tokens penalized on challenge not passed.
	BlobberSlash float64 `json:"blobber_slash"`

	// MaxBlobbersPerAllocation maximum blobbers that can be sent per allocation
	MaxBlobbersPerAllocation int `json:"max_blobbers_per_allocation"`

	// price limits for blobbers

	// MaxReadPrice allowed for a blobber.
	MaxReadPrice currency.Coin `json:"max_read_price"`
	// MaxWrtiePrice
	MaxWritePrice currency.Coin `json:"max_write_price"`
	MinWritePrice currency.Coin `json:"min_write_price"`

	// allocation cancellation

	// FailedChallengesToCancel is number of failed challenges of an allocation
	// to be able to cancel an allocation.
	FailedChallengesToCancel int `json:"failed_challenges_to_cancel"`
	// FailedChallengesToRevokeMinLock is number of failed challenges of a
	// blobber to revoke its min_lock demand back to user; only part not
	// paid yet can go back.
	FailedChallengesToRevokeMinLock int `json:"failed_challenges_to_revoke_min_lock"`

	// free allocations
	MaxTotalFreeAllocation      currency.Coin          `json:"max_total_free_allocation"`
	MaxIndividualFreeAllocation currency.Coin          `json:"max_individual_free_allocation"`
	FreeAllocationSettings      freeAllocationSettings `json:"free_allocation_settings"`

	// challenges generating

	// ChallengeEnabled is challenges generating pin.
	ChallengeEnabled bool `json:"challenge_enabled"`
	// MaxChallengesPerGeneration is max number of challenges can be generated
	// at once for a blobber-allocation pair with size difference for the
	// moment of the generation.
	MaxChallengesPerGeneration int `json:"max_challenges_per_generation"`
	// ValidatorsPerChallenge is the number of validators to select per
	// challenges.
	ValidatorsPerChallenge int `json:"validators_per_challenge"`
	// ChallengeGenerationRate is number of challenges generated for a MB/min.
	ChallengeGenerationRate float64 `json:"challenge_rate_per_mb_min"`

	// MinStake allowed by a blobber/validator (entire SC boundary).
	MinStake currency.Coin `json:"min_stake"`
	// MaxStake allowed by a blobber/validator (entire SC boundary).
	MaxStake currency.Coin `json:"max_stake"`

	// MaxDelegates per stake pool
	MaxDelegates int `json:"max_delegates"`

	// MaxCharge that blobber gets from rewards to its delegate_wallet.
	MaxCharge float64 `json:"max_charge"`

	BlockReward *blockReward `json:"block_reward"`

	// Allow direct access to MPT
	ExposeMpt bool           `json:"expose_mpt"`
	OwnerId   string         `json:"owner_id"`
	Cost      map[string]int `json:"cost"`
}

func (sc *Config) validate() (err error) {
	if sc.TimeUnit <= 1*time.Second {
		return fmt.Errorf("time_unit less than 1s: %v", sc.TimeUnit)
	}
	if sc.ValidatorReward < 0.0 || 1.0 < sc.ValidatorReward {
		return fmt.Errorf("validator_reward not in [0; 1] range: %v",
			sc.ValidatorReward)
	}
	if sc.BlobberSlash < 0.0 || 1.0 < sc.BlobberSlash {
		return fmt.Errorf("blobber_slash not in [0; 1] range: %v",
			sc.BlobberSlash)
	}
	if sc.MaxBlobbersPerAllocation <= 0 {
		return fmt.Errorf("invalid max_blobber_per_allocation <= 0: %v",
			sc.MaxBlobbersPerAllocation)
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
	if sc.MinAllocSize < 0 {
		return fmt.Errorf("negative min_alloc_size: %v", sc.MinAllocSize)
	}

	if sc.MaxWritePrice < sc.MinWritePrice {
		return fmt.Errorf("max wirte price %v must be more than min_write_price: %v",
			sc.MaxWritePrice, sc.MinWritePrice)
	}
	if sc.StakePool.MinLock <= 1 {
		return fmt.Errorf("invalid stakepool.min_lock: %v <= 1",
			sc.StakePool.MinLock)
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
	if sc.ValidatorsPerChallenge <= 0 {
		return fmt.Errorf("invalid validators_per_challenge <= 0: %v",
			sc.ValidatorsPerChallenge)
	}
	if sc.ChallengeGenerationRate < 0 {
		return fmt.Errorf("negative challenge_rate_per_mb_min: %v",
			sc.ChallengeGenerationRate)
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

	if sc.BlockReward.SharderWeight < 0 {
		return fmt.Errorf("negative block_reward.sharder_weight: %v",
			sc.BlockReward.SharderWeight)
	}
	if sc.BlockReward.MinerWeight < 0 {
		return fmt.Errorf("negative block_reward.miner_weight: %v",
			sc.BlockReward.MinerWeight)
	}
	if sc.BlockReward.BlobberWeight < 0 {
		return fmt.Errorf("negative block_reward.blobber_capacity_weight: %v",
			sc.BlockReward.BlobberWeight)
	}
	if len(sc.OwnerId) == 0 {
		return fmt.Errorf("owner_id does not set or empty")
	}

	if sc.BlockReward.Gamma.A <= 0 {
		return fmt.Errorf("invalid block_reward.gamma.a <= 0: %v", sc.BlockReward.Gamma.A)
	}
	if sc.BlockReward.Gamma.B <= 0 {
		return fmt.Errorf("invalid block_reward.gamma.b <= 0: %v", sc.BlockReward.Gamma.B)
	}
	if sc.BlockReward.Gamma.Alpha <= 0 {
		return fmt.Errorf("invalid block_reward.gamma.alpha <= 0: %v", sc.BlockReward.Gamma.Alpha)
	}
	if sc.BlockReward.Zeta.Mu <= 0 {
		return fmt.Errorf("invalid block_reward.zeta.mu <= 0: %v", sc.BlockReward.Zeta.Mu)
	}
	if sc.BlockReward.Zeta.I <= 0 {
		return fmt.Errorf("invalid block_reward.zeta.i <= 0: %v", sc.BlockReward.Zeta.I)
	}
	if sc.BlockReward.Zeta.K <= 0 {
		return fmt.Errorf("invalid block_reward.zeta.k <=0: %v", sc.BlockReward.Zeta.K)
	}

	return
}

func (conf *Config) validateStakeRange(min, max currency.Coin) (err error) {
	if min < conf.MinStake {
		return fmt.Errorf("min_stake is less than allowed by SC: %v < %v", min,
			conf.MinStake)
	}
	if max > conf.MaxStake {
		return fmt.Errorf("max_stake is greater than allowed by SC: %v > %v",
			max, conf.MaxStake)
	}
	if max < min {
		return fmt.Errorf("max_stake less than min_stake: %v < %v", max, min)
	}
	return
}

func (conf *Config) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(conf); err != nil {
		panic(err) // must not happens
	}
	return
}

func (conf *Config) Decode(b []byte) error {
	return json.Unmarshal(b, conf)
}

//
// rest handler and update function
//

// configs from sc.yaml
func getConfiguredConfig() (conf *Config, err error) {
	const pfx = "smart_contracts.storagesc."

	conf = new(Config)
	var scc = config.SmartContractConfig
	// sc
	conf.TimeUnit = scc.GetDuration(pfx + "time_unit")
	conf.MaxMint, err = currency.ParseZCN(scc.GetFloat64(pfx + "max_mint"))
	if err != nil {
		return nil, err
	}
	conf.MinStake, err = currency.ParseZCN(scc.GetFloat64(pfx + "min_stake"))
	if err != nil {
		return nil, err
	}
	conf.MaxStake, err = currency.ParseZCN(scc.GetFloat64(pfx + "max_stake"))
	if err != nil {
		return nil, err
	}
	conf.MinAllocSize = scc.GetInt64(pfx + "min_alloc_size")
	conf.MinAllocDuration = scc.GetDuration(pfx + "min_alloc_duration")
	conf.MaxChallengeCompletionTime = scc.GetDuration(pfx + "max_challenge_completion_time")
	conf.MinOfferDuration = scc.GetDuration(pfx + "min_offer_duration")
	conf.MinBlobberCapacity = scc.GetInt64(pfx + "min_blobber_capacity")
	conf.ValidatorReward = scc.GetFloat64(pfx + "validator_reward")
	conf.BlobberSlash = scc.GetFloat64(pfx + "blobber_slash")
	conf.MaxBlobbersPerAllocation = scc.GetInt(pfx + "max_blobbers_per_allocation")
	conf.MaxReadPrice, err = currency.ParseZCN(scc.GetFloat64(pfx + "max_read_price"))
	if err != nil {
		return nil, err
	}
	conf.MinWritePrice, err = currency.ParseZCN(scc.GetFloat64(pfx + "min_write_price"))
	if err != nil {
		return nil, err
	}
	conf.MaxWritePrice, err = currency.ParseZCN(scc.GetFloat64(pfx + "max_write_price"))
	if err != nil {
		return nil, err
	}
	// read pool
	conf.ReadPool = new(readPoolConfig)
	conf.ReadPool.MinLock, err = currency.ParseZCN(scc.GetFloat64(pfx + "readpool.min_lock"))
	if err != nil {
		return nil, err
	}

	// write pool
	conf.WritePool = new(writePoolConfig)
	conf.WritePool.MinLock, err = currency.ParseZCN(scc.GetFloat64(pfx + "writepool.min_lock"))
	if err != nil {
		return nil, err
	}
	// stake pool
	conf.StakePool = new(stakePoolConfig)
	conf.StakePool.MinLock, err = currency.ParseZCN(scc.GetFloat64(pfx + "stakepool.min_lock"))
	if err != nil {
		return nil, err
	}

	conf.MaxTotalFreeAllocation = currency.Coin(scc.GetFloat64(pfx+"max_total_free_allocation") * 1e10)
	conf.MaxIndividualFreeAllocation = currency.Coin(scc.GetFloat64(pfx+"max_individual_free_allocation") * 1e10)
	fas := pfx + "free_allocation_settings."
	conf.FreeAllocationSettings.DataShards = int(scc.GetFloat64(fas + "data_shards"))
	conf.FreeAllocationSettings.ParityShards = int(scc.GetFloat64(fas + "parity_shards"))
	conf.FreeAllocationSettings.Size = int64(scc.GetFloat64(fas + "size"))
	conf.FreeAllocationSettings.Duration = scc.GetDuration(fas + "duration")
	conf.FreeAllocationSettings.ReadPriceRange = PriceRange{
		Min: currency.Coin(scc.GetFloat64(fas+"read_price_range.min") * 1e10),
		Max: currency.Coin(scc.GetFloat64(fas+"read_price_range.max") * 1e10),
	}
	conf.FreeAllocationSettings.WritePriceRange = PriceRange{
		Min: currency.Coin(scc.GetFloat64(fas+"write_price_range.min") * 1e10),
		Max: currency.Coin(scc.GetFloat64(fas+"write_price_range.max") * 1e10),
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
	conf.ValidatorsPerChallenge = scc.GetInt(
		pfx + "validators_per_challenge")
	conf.ChallengeGenerationRate = scc.GetFloat64(
		pfx + "challenge_rate_per_mb_min")

	conf.MaxDelegates = scc.GetInt(pfx + "max_delegates")
	conf.MaxCharge = scc.GetFloat64(pfx + "max_charge")

	conf.BlockReward = new(blockReward)
	conf.BlockReward.BlockReward, err = currency.ParseZCN(scc.GetFloat64(pfx + "block_reward.block_reward"))
	if err != nil {
		return nil, err
	}
	conf.BlockReward.BlockRewardChangePeriod = scc.GetInt64(pfx + "block_reward.block_reward_change_period")
	conf.BlockReward.BlockRewardChangeRatio = scc.GetFloat64(pfx + "block_reward.block_reward_change_ratio")
	conf.BlockReward.QualifyingStake, err = currency.ParseZCN(scc.GetFloat64(pfx + "block_reward.qualifying_stake"))
	if err != nil {
		return nil, err
	}
	conf.BlockReward.TriggerPeriod = scc.GetInt64(pfx + "block_reward.trigger_period")
	conf.BlockReward.setWeightsFromRatio(
		scc.GetFloat64(pfx+"block_reward.sharder_ratio"),
		scc.GetFloat64(pfx+"block_reward.miner_ratio"),
		scc.GetFloat64(pfx+"block_reward.blobber_ratio"),
	)
	conf.BlockReward.Gamma.Alpha = scc.GetFloat64(pfx + "block_reward.gamma.alpha")
	conf.BlockReward.Gamma.A = scc.GetFloat64(pfx + "block_reward.gamma.a")
	conf.BlockReward.Gamma.B = scc.GetFloat64(pfx + "block_reward.gamma.b")
	conf.BlockReward.Zeta.I = scc.GetFloat64(pfx + "block_reward.zeta.i")
	conf.BlockReward.Zeta.K = scc.GetFloat64(pfx + "block_reward.zeta.k")
	conf.BlockReward.Zeta.Mu = scc.GetFloat64(pfx + "block_reward.zeta.mu")

	conf.ExposeMpt = scc.GetBool(pfx + "expose_mpt")
	conf.OwnerId = scc.GetString(pfx + "owner_id")
	conf.Cost = scc.GetStringMapInt(pfx + "cost")

	err = conf.validate()
	return
}

func (ssc *StorageSmartContract) setupConfig(
	balances chainState.StateContextI) (conf *Config, err error) {

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
	conf *Config, err error) {

	conf = new(Config)
	err = balances.GetTrieNode(scConfigKey(ssc.ID), conf)
	switch err {
	case util.ErrValueNotPresent:
		if !setup {
			return // value not present
		}
		return ssc.setupConfig(balances)
	case nil:
		return conf, nil
	default:
		return nil, err
	}
}

// getReadPoolConfig
func (ssc *StorageSmartContract) getReadPoolConfig(
	balances chainState.StateContextI, setup bool) (
	conf *readPoolConfig, err error) {

	var scconf *Config
	if scconf, err = ssc.getConfig(balances, setup); err != nil {
		return
	}
	return scconf.ReadPool, nil
}
