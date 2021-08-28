package smartcontract

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
)

const (
	Simulation    = "simulation."
	Internal      = "internal."
	SmartContract = "smart_contract."
	Miner         = "miner."
	Storage       = "storage."

	AvailableKeys = Internal + "available_keys"
	Now           = Internal + "now"

	NumClients               = Simulation + "num_clients"
	StartTokens              = Simulation + "start_tokens"
	SignatureScheme          = Simulation + "signature_scheme"
	NumMiners                = Simulation + "num_miners"
	NumAllocations           = Simulation + "num_allocations"
	NumBlobbersPerAllocation = Simulation + "num_blobbers_per_Allocation"
	NumBlobbers              = Simulation + "num_blobbers"
	NumAllocationPlayerPools = Simulation + "num_allocation_payers_pools"
	NumAllocationPlayer      = Simulation + "num_allocation_payers"
	NumBlobberDelegates      = Simulation + "num_blobber_delegates"

	MinerMaxDelegates = SmartContract + Miner + "max_delegates"
	MinerMaxCharge    = SmartContract + Miner + "max_charge"
	MinerMinStake     = SmartContract + Miner + "min_stake"
	MinerMaxStake     = SmartContract + Miner + "max_stake"

	StorageMinAllocSize               = SmartContract + Storage + "min_alloc_size"
	StorageMinAllocDuration           = SmartContract + Storage + "min_alloc_duration"
	StorageMaxReadPrice               = SmartContract + Storage + "max_read_price"
	StorageMaxWritePrice              = SmartContract + Storage + "max_write_price"
	StorageMaxChallengeCompletionTime = SmartContract + Storage + "max_challenge_completion_time"
	StorageMinOfferDuration           = SmartContract + Storage + "min_offer_duration"
	StorageMinBlobberCapacity         = SmartContract + Storage + "min_blobber_capacity"
	StorageMaxStake                   = SmartContract + Miner + "max_stake"
)

type BenchTest struct {
	Name     string
	Endpoint func(
		*transaction.Transaction,
		[]byte,
		cstate.StateContextI,
	) (string, error)
	Txn   transaction.Transaction
	Input []byte
}
