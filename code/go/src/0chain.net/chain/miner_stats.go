package chain

//MinerStats - stats associated with a given miner
type MinerStats struct {
	// Number of times the block of the given miner is finalized for a given round rank
	FinalizationCountByRank []int64
}
