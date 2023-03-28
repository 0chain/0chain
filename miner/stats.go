package miner

import "time"

type ExplorerStats struct {
	BlockFinality      float64                  `json:"block_finality"`
	LastFinalizedRound int64                    `json:"last_finalized_round"`
	BlocksFinalized    int64                    `json:"blocks_finalized"`
	StateHealth        int64                    `json:"state_health"`
	CurrentRound       int64                    `json:"current_round"`
	RoundTimeout       int64                    `json:"round_timeout"`
	Timeouts           int64                    `json:"timeouts"`
	AverageBlockSize   int                      `json:"average_block_size"`
	NetworkTime        map[string]time.Duration `json:"network_times"`
}
