package chain

//MinerStats - stats associated with a given miner
type MinerStats struct {
	// Number of times the block of the given miner is finalized for a given round rank
	FinalizationCountByRank []int64

	// Number of times verification tickets have been requested
	VerificationTicketsByRank []int64

	// Number of times verification failed
	VerificationFailures int64
}
