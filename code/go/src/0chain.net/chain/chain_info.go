package chain

import (
	"0chain.net/block"
	"0chain.net/metric"
	"0chain.net/round"
	"0chain.net/util"
)

/*Info - a struct to capture the chain info at runtime */
type Info struct {
	FinalizedRound  int64    `json:"round"`
	FinalizedCount  int64    `json:"finalized_blocks_count"`
	BlockHash       string   `json:"block_hash"`
	ClientStateHash util.Key `json:"client_state_hash"`
	ChainWeight     float64  `json:"chain_weight"`
	MissedBlocks    int64    `json:"missed_blocks_count"`

	// Track stats related to multiple blocks to extend from
	MultipleBlocksCount int64 `json:"multiple_blocks_count"`
	SumMultipleBlocks   int64 `json:"multiple_blocks_sum"`
	MaxMultipleBlocks   int64 `json:"max_multiple_blocks"`
}

func (info *Info) GetValue() int64 {
	return info.FinalizedRound
}

var ChainMetric *metric.PowerMetric
var RoundMetric *metric.PowerMetric

func init() {
	power := 10
	len := 10
	ChainMetric = metric.NewPowerMetric(power, len)
	RoundMetric = metric.NewPowerMetric(power, len)
}

/*UpdateChainInfo - update the chain information */
func (c *Chain) UpdateChainInfo(b *block.Block) {
	ci := &Info{
		FinalizedRound:  b.Round,
		BlockHash:       b.Hash,
		ClientStateHash: b.ClientStateHash,
		ChainWeight:     b.ChainWeight,
		FinalizedCount:  SteadyStateFinalizationTimer.Count(),
		MissedBlocks:    c.MissedBlocks,
	}
	ChainMetric.CurrentValue = ci
	ChainMetric.Collect(ci)
}

/*UpdateRoundInfo - update the round information */
func (c *Chain) UpdateRoundInfo(r *round.Round) {
	ri := &round.Info{
		Number:                    r.Number,
		NotarizedBlocksCount:      int8(len(r.GetNotarizedBlocks())),
		MultiNotarizedBlocksCount: c.MultiNotarizedBlocksCount,
		ZeroNotarizedBlocksCount:  c.ZeroNotarizedBlocksCount,
	}
	RoundMetric.CurrentValue = ri
	RoundMetric.Collect(ri)
}
