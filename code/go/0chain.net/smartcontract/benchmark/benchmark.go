package benchmark

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
)

type BenchmarkSource int

const (
	Storage BenchmarkSource = iota
	Miner
	Faucet
	InterestPool
	Vesting
	NumberOdfBenchmarkSources
)

var (
	BenchmarkSourceNames = []string{
		"storage",
		"miner",
		"faucet",
		"interest_pool",
		"vesting",
	}

	BenchmarkSourceCode = map[string]BenchmarkSource{
		"storage":       Storage,
		"miner":         Miner,
		"faucet":        Faucet,
		"interest_pool": InterestPool,
		"vesting":       Vesting,
	}
)

const (
	Simulation     = "simulation."
	Internal       = "internal."
	SmartContract  = "smart_contracts."
	MinerSc        = "minersc."
	StorageSc      = "storagesc."
	FaucetSc       = "faucetsc."
	InterestPoolSC = "interestpoolsc."
	VestingSc      = "vestingsc."

	Fas = "free_allocation_settings."

	AvailableKeys = Internal + "available_keys"
	Now           = Internal + "now"
	InternalT     = Internal + "t"
	//InternalSignatureScheme   = Internal + "signature_scheme"
	StartTokens = Internal + "start_tokens"

	NumClients               = Simulation + "num_clients"
	NumMiners                = Simulation + "num_miners"
	NumSharders              = Simulation + "nun_sharders"
	NumAllocations           = Simulation + "num_allocations"
	NumBlobbersPerAllocation = Simulation + "num_blobbers_per_Allocation"
	NumBlobbers              = Simulation + "num_blobbers"
	NumAllocationPlayerPools = Simulation + "num_allocation_payers_pools"
	NumAllocationPlayer      = Simulation + "num_allocation_payers"
	NumBlobberDelegates      = Simulation + "num_blobber_delegates"
	NumCurators              = Simulation + "num_curators"
	NumValidators            = Simulation + "num_validators"
	NumFreeStorageAssigners  = Simulation + "num_free_storage_assigners"
	NumMinerDelegates        = Simulation + "num_miner_delegates"
	NumSharderDelegates      = Simulation + "num_sharder_delegates"

	MinerMaxDelegates = SmartContract + MinerSc + "max_delegates"
	MinerMaxCharge    = SmartContract + MinerSc + "max_charge"
	MinerMinStake     = SmartContract + MinerSc + "min_stake"
	MinerMaxStake     = SmartContract + MinerSc + "max_stake"

	StorageMinAllocSize                  = SmartContract + StorageSc + "min_alloc_size"
	StorageMinAllocDuration              = SmartContract + StorageSc + "min_alloc_duration"
	StorageMaxReadPrice                  = SmartContract + StorageSc + "max_read_price"
	StorageMaxWritePrice                 = SmartContract + StorageSc + "max_write_price"
	StorageMaxChallengeCompletionTime    = SmartContract + StorageSc + "max_challenge_completion_time"
	StorageMinOfferDuration              = SmartContract + StorageSc + "min_offer_duration"
	StorageMinBlobberCapacity            = SmartContract + StorageSc + "min_blobber_capacity"
	StorageMaxCharge                     = SmartContract + StorageSc + "max_charge"
	StorageMinStake                      = SmartContract + StorageSc + "min_stake"
	StorageMaxStake                      = SmartContract + StorageSc + "max_stake"
	StorageMaxDelegates                  = SmartContract + StorageSc + "max_delegates"
	StorageDiverseBlobbers               = SmartContract + StorageSc + "diverse_blobbers"
	StorageFailedChallengesToCancel      = SmartContract + StorageSc + "failed_challenges_to_cancel"
	StorageReadPoolMinLock               = SmartContract + StorageSc + "readpool.min_lock"
	StorageReadPoolMinLockPeriod         = SmartContract + StorageSc + "readpool.min_lock_period"
	StorageWritePoolMinLock              = SmartContract + StorageSc + "writepool.min_lock"
	StorageWritePoolMinLockPeriod        = SmartContract + StorageSc + "writepool.min_lock_period"
	StorageStakePoolMinLock              = SmartContract + StorageSc + "stakepool.min_lock"
	StorageChallengeEnabled              = SmartContract + StorageSc + "challenge_enabled"
	StorageMaxTotalFreeAllocation        = SmartContract + StorageSc + "max_total_free_allocation"
	StorageMaxIndividualFreeAllocation   = SmartContract + StorageSc + "max_individual_free_allocation"
	StorageFasDataShards                 = SmartContract + StorageSc + Fas + "data_shards"
	StorageFasParityShards               = SmartContract + StorageSc + Fas + "parity_shards"
	StorageFasSize                       = SmartContract + StorageSc + Fas + "size"
	StorageFasDuration                   = SmartContract + StorageSc + Fas + "duration"
	StorageFasReadPriceMin               = SmartContract + StorageSc + Fas + "read_price_range.min"
	StorageFasReadPriceMax               = SmartContract + StorageSc + Fas + "read_price_range.max"
	StorageFasWritePriceMin              = SmartContract + StorageSc + Fas + "write_price_range.min"
	StorageFasWritePriceMax              = SmartContract + StorageSc + Fas + "write_price_range.max"
	StorageFasMaxChallengeCompletionTime = SmartContract + StorageSc + Fas + "max_challenge_completion_time"
	StorageFasReadPoolFraction           = SmartContract + StorageSc + Fas + "read_pool_fraction"

	InterestPoolMinLock       = SmartContract + InterestPoolSC + "min_lock"
	InterestPoolMinLockPeriod = SmartContract + InterestPoolSC + "min_lock_period"
	InterestPoolMaxMint       = SmartContract + InterestPoolSC + "max_mint"

	VestingMinLock         = SmartContract + VestingSc + "min_lock"
	VestingMaxDestinations = SmartContract + VestingSc + "max_destinations"
)

type BenchTestI interface {
	Name() string
	Transaction() transaction.Transaction
	Run(state.StateContextI)
}

type SignatureScheme interface {
	encryption.SignatureScheme
	SetPrivateKey(privateKey string)
	GetPrivateKey() string
}

type TestSuit struct {
	Source     BenchmarkSource
	Benchmarks []BenchTestI
}

type BenchData struct {
	Clients       []string
	PublicKeys    []string
	PrivateKeys   []string
	Blobbers      []string
	Validators    []string
	Allocations   []string
	Miners        []string
	Sharders      []string
	InterestPools []string
}
