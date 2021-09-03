package benchmark

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
)

type BenchmarkSource int

const (
	Storage BenchmarkSource = iota
	StorageRest
	Miner
	MinerRest
	Faucet
	FaucetRest
	InterestPool
	InterestPoolRest
	Vesting
	VestingRest
	MultiSig
	NumberOdfBenchmarkSources
)

var (
	BenchmarkSourceNames = []string{
		"storage",
		"storage_rest",
		"miner",
		"miner_rest",
		"faucet",
		"faucet_rest",
		"interest_pool",
		"interest_pool_rest",
		"vesting",
		"vesting_rest",
		"multi_sig",
	}

	BenchmarkSourceCode = map[string]BenchmarkSource{
		BenchmarkSourceNames[Storage]:          Storage,
		BenchmarkSourceNames[StorageRest]:      StorageRest,
		BenchmarkSourceNames[Miner]:            Miner,
		BenchmarkSourceNames[MinerRest]:        MinerRest,
		BenchmarkSourceNames[Faucet]:           Faucet,
		BenchmarkSourceNames[FaucetRest]:       FaucetRest,
		BenchmarkSourceNames[InterestPool]:     InterestPool,
		BenchmarkSourceNames[InterestPoolRest]: InterestPoolRest,
		BenchmarkSourceNames[Vesting]:          Vesting,
		BenchmarkSourceNames[VestingRest]:      VestingRest,
		BenchmarkSourceNames[MultiSig]:         MultiSig,
	}
)

type SimulatorParameter int

const (
	SimulationNumClients SimulatorParameter = iota
	SimulationNumMiners
	SimulationNumSharders
	SimulationNumAllocations
	SimulationNumBlobbersPerAllocation
	SimulationNumBlobbers
	SimulationNumAllocationPlayerPools
	SimulationNumAllocationPlayer
	SimulationNumBlobberDelegates
	SimulationNumCurators
	SimulationNumValidators
	SimulationNumFreeStorageAssigners
	SimulationNumMinerDelegates
	SimulationNumSharderDelegates
	SimulationNumVestingDestinationsClient
	SimulationNumWriteRedeemAllocation
	SimulationNumChallengesBlobber
	NumberSimulationParameters
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

	AvailableKeys           = Internal + "available_keys"
	Now                     = Internal + "now"
	InternalT               = Internal + "t"
	InternalSignatureScheme = Internal + "signature_scheme"
	StartTokens             = Internal + "start_tokens"
	Bad                     = Internal + "bad"
	Worry                   = Internal + "worry"
	Satisfactory            = Internal + "satisfactory"
	TimeUnit                = Internal + "time_unit"
	Colour                  = Internal + "colour"

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
	StorageReadPoolMaxLockPeriod         = SmartContract + StorageSc + "readpool.max_lock_period"
	StorageWritePoolMinLock              = SmartContract + StorageSc + "writepool.min_lock"
	StorageWritePoolMinLockPeriod        = SmartContract + StorageSc + "writepool.min_lock_period"
	StorageWritePoolMaxLockPeriod        = SmartContract + StorageSc + "writepool.max_lock_period"
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
	StorageMaxMint                       = SmartContract + StorageSc + "max_mint"
	StorageMaxChallengesPerGeneration    = SmartContract + StorageSc + "max_challenges_per_generation"

	InterestPoolMinLock       = SmartContract + InterestPoolSC + "min_lock"
	InterestPoolMinLockPeriod = SmartContract + InterestPoolSC + "min_lock_period"
	InterestPoolMaxMint       = SmartContract + InterestPoolSC + "max_mint"

	VestingMinLock         = SmartContract + VestingSc + "min_lock"
	VestingMaxDestinations = SmartContract + VestingSc + "max_destinations"
	VestingMinDuration     = SmartContract + VestingSc + "min_duration"
	VestingMaxDuration     = SmartContract + VestingSc + "max_duration"
)

func (w SimulatorParameter) String() string {
	return [...]string{
		"num_clients",
		"num_miners",
		"nun_sharders",
		"num_allocations",
		"num_blobbers_per_Allocation",
		"num_blobbers",
		"num_allocation_payers_pools",
		"num_allocation_payers",
		"num_blobber_delegates",
		"num_curators",
		"num_validators",
		"num_free_storage_assigners",
		"num_miner_delegates",
		"num_sharder_delegates",
		"num_vesting_destinations_client",
		"num_write_redeem_allocation",
		"num_challenges_blobber",
	}[w]
}

var (
	NumClients                   = Simulation + SimulationNumClients.String()
	NumMiners                    = Simulation + SimulationNumMiners.String()
	NumSharders                  = Simulation + SimulationNumSharders.String()
	NumAllocations               = Simulation + SimulationNumAllocations.String()
	NumBlobbersPerAllocation     = Simulation + SimulationNumBlobbersPerAllocation.String()
	NumBlobbers                  = Simulation + SimulationNumBlobbers.String()
	NumAllocationPlayerPools     = Simulation + SimulationNumAllocationPlayerPools.String()
	NumAllocationPlayer          = Simulation + SimulationNumAllocationPlayer.String()
	NumBlobberDelegates          = Simulation + SimulationNumBlobberDelegates.String()
	NumCurators                  = Simulation + SimulationNumCurators.String()
	NumValidators                = Simulation + SimulationNumValidators.String()
	NumFreeStorageAssigners      = Simulation + SimulationNumFreeStorageAssigners.String()
	NumMinerDelegates            = Simulation + SimulationNumMinerDelegates.String()
	NumSharderDelegates          = Simulation + SimulationNumSharderDelegates.String()
	NumVestingDestinationsClient = Simulation + SimulationNumVestingDestinationsClient.String()
	NumWriteRedeemAllocation     = Simulation + SimulationNumWriteRedeemAllocation.String()
	NumChallengesBlobber         = Simulation + SimulationNumChallengesBlobber.String()
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
	Sharders    []string
}
