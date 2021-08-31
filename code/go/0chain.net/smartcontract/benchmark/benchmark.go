package benchmark

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
)

type BenchmarkSource int

const (
	StorageTrans BenchmarkSource = iota
	MinerTrans
	NumberOdfBenchmarkSources
)

var (
	BenchmarkSourceNames = []string{
		"storage",
		"miner",
	}

	BenchmarkSourceCode = map[string]BenchmarkSource{
		"storage": StorageTrans,
		"miner":   MinerTrans,
	}
)

const (
	Simulation    = "simulation."
	Internal      = "internal."
	SmartContract = "smart_contracts."
	Miner         = "minersc."
	Storage       = "storagesc."
	Fas           = "free_allocation_settings."

	AvailableKeys = Internal + "available_keys"
	Now           = Internal + "now"
	//InternalSignatureScheme   = Internal + "signature_scheme"

	NumClients               = Simulation + "num_clients"
	StartTokens              = Simulation + "start_tokens" //
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

	MinerMaxDelegates = SmartContract + Miner + "max_delegates"
	MinerMaxCharge    = SmartContract + Miner + "max_charge"
	MinerMinStake     = SmartContract + Miner + "min_stake"
	MinerMaxStake     = SmartContract + Miner + "max_stake"

	StorageMinAllocSize                  = SmartContract + Storage + "min_alloc_size"
	StorageMinAllocDuration              = SmartContract + Storage + "min_alloc_duration"
	StorageMaxReadPrice                  = SmartContract + Storage + "max_read_price"
	StorageMaxWritePrice                 = SmartContract + Storage + "max_write_price"
	StorageMaxChallengeCompletionTime    = SmartContract + Storage + "max_challenge_completion_time"
	StorageMinOfferDuration              = SmartContract + Storage + "min_offer_duration"
	StorageMinBlobberCapacity            = SmartContract + Storage + "min_blobber_capacity"
	StorageMaxCharge                     = SmartContract + Storage + "max_charge"
	StorageMinStake                      = SmartContract + Storage + "min_stake"
	StorageMaxStake                      = SmartContract + Storage + "max_stake"
	StorageMaxDelegates                  = SmartContract + Storage + "max_delegates"
	StorageDiverseBlobbers               = SmartContract + Storage + "diverse_blobbers"
	StorageFailedChallengesToCancel      = SmartContract + Storage + "failed_challenges_to_cancel"
	StorageReadPoolMinLock               = SmartContract + Storage + "readpool.min_lock"
	StorageReadPoolMinLockPeriod         = SmartContract + Storage + "readpool.min_lock_period"
	StorageWritePoolMinLock              = SmartContract + Storage + "writepool.min_lock"
	StorageWritePoolMinLockPeriod        = SmartContract + Storage + "writepool.min_lock_period"
	StorageStakePoolMinLock              = SmartContract + Storage + "stakepool.min_lock"
	StorageChallengeEnabled              = SmartContract + Storage + "challenge_enabled"
	StorageMaxTotalFreeAllocation        = SmartContract + Storage + "max_total_free_allocation"
	StorageMaxIndividualFreeAllocation   = SmartContract + Storage + "max_individual_free_allocation"
	StorageFasDataShards                 = SmartContract + Storage + Fas + "data_shards"
	StorageFasParityShards               = SmartContract + Storage + Fas + "parity_shards"
	StorageFasSize                       = SmartContract + Storage + Fas + "size"
	StorageFasDuration                   = SmartContract + Storage + Fas + "duration"
	StorageFasReadPriceMin               = SmartContract + Storage + Fas + "read_price_range.min"
	StorageFasReadPriceMax               = SmartContract + Storage + Fas + "read_price_range.max"
	StorageFasWritePriceMin              = SmartContract + Storage + Fas + "write_price_range.min"
	StorageFasWritePriceMax              = SmartContract + Storage + Fas + "write_price_range.max"
	StorageFasMaxChallengeCompletionTime = SmartContract + Storage + Fas + "max_challenge_completion_time"
	StorageFasReadPoolFraction           = SmartContract + Storage + Fas + "read_pool_fraction"
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
	Clients     []string
	PublicKeys  []string
	PrivateKeys []string
	Blobbers    []string
	Validators  []string
	Allocations []string
}
