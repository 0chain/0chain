package chain

// MinerStats - stats associated with a given miner
type MinerStats struct {

	// Number of times the miner is a genrator
	GenerationCountByRank []int64

	// Number of times the block of the given miner is finalized for a given round rank
	FinalizationCountByRank []int64

	// Number of times verification tickets have been requested
	VerificationTicketsByRank []int64

	// Number of times verification failed
	VerificationFailures int64
}

func (m *MinerStats) Clone() interface{} {
	result := &MinerStats{
		GenerationCountByRank:     make([]int64, len(m.GenerationCountByRank)),
		FinalizationCountByRank:   make([]int64, len(m.FinalizationCountByRank)),
		VerificationTicketsByRank: make([]int64, len(m.VerificationTicketsByRank)),
		VerificationFailures:      m.VerificationFailures,
	}
	copy(result.GenerationCountByRank, m.GenerationCountByRank)
	copy(result.FinalizationCountByRank, m.FinalizationCountByRank)
	copy(result.VerificationTicketsByRank, m.VerificationTicketsByRank)
	return result
}
