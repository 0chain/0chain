package chaincore

// swagger:model
type InfoResponse struct {
	ChainInfo []ChainInfo `json:"chain_info"`
	RoundInfo []RoundInfo `json:"round_info"`
}

// swagger:model
type ChainInfo struct {
	TimeStamp       string `json:"ts"`
	FinalizedRound  int64  `json:"round"`
	FinalizedCount  int64  `json:"finalized_blocks_count"`
	BlockHash       string `json:"block_hash"`
	ClientStateHash string `json:"client_state_hash"`
}

// swagger:model
type RoundInfo struct {
	TimeStamp       string `json:"ts"`
	Round           int64  `json:"round_number"`
	NotarizedBlocksCount int8 `json:"notarized_blocks_count"`

	// count of rounds with no notarization for any blocks
	ZeroNotarizedBlocksCount int8 `json:"zero_notarized_blocks_count"`
	
	// count of rounds with multiple notarized blocks.
	MultiNotarizedBlocksCount int8 `json:"multi_notarized_blocks_count"`
}

// swagger:model
type BlockFeeStatsResponse struct {
	MaxFee int64 `json:"max_fee"`
	MinFee int64 `json:"min_fee"`
	MeanFee int64 `json:"mean_fee"`
}

// swagger:model
type TxnFeeResponse struct {
	Fee string `json:"fee"`
}

// swagger:model FeesTableResponse
type FeesTableResponse struct {
	ScFeesTableMap map[string]map[string]int64 `json:"sc_fees_table_map"`
}
