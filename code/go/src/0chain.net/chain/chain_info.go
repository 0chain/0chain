package chain

import (
	"time"

	"0chain.net/block"
	"0chain.net/metric"
	"0chain.net/round"
	"0chain.net/util"
)

/*Info - a struct to capture the chain info at runtime */
type Info struct {
	TimeStamp       *time.Time `json:"ts"`
	FinalizedRound  int64      `json:"round"`
	FinalizedCount  int64      `json:"finalized_blocks_count"`
	BlockHash       string     `json:"block_hash"`
	ClientStateHash util.Key   `json:"client_state_hash"`
	ChainWeight     float64    `json:"chain_weight"`
	MissedBlocks    int64      `json:"missed_blocks_count"`
	RollbackCount   int64      `json:"rollback_count"`

	// Track stats related to multiple blocks to extend from
	MultipleBlocksCount int64 `json:"multiple_blocks_count"`
	SumMultipleBlocks   int64 `json:"multiple_blocks_sum"`
	MaxMultipleBlocks   int64 `json:"max_multiple_blocks"`
}

//GetKey - implements Metric interface
func (info *Info) GetKey() int64 {
	return info.FinalizedRound
}

//GetTime - implements Metric Interface
func (info *Info) GetTime() *time.Time {
	return info.TimeStamp
}

var chainMetrics *metric.PowerMetrics
var roundMetrics *metric.PowerMetrics

func init() {
	power := 10
	len := 10
	chainMetrics = metric.NewPowerMetrics(power, len)
	roundMetrics = metric.NewPowerMetrics(power, len)
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
		RollbackCount:   c.RollbackCount,
	}
	t := time.Now()
	ci.TimeStamp = &t

	chainMetrics.CurrentValue = ci
	chainMetrics.Collect(ci)
}

/*UpdateRoundInfo - update the round information */
func (c *Chain) UpdateRoundInfo(r *round.Round) {
	ri := &round.Info{
		Number:                    r.Number,
		NotarizedBlocksCount:      int8(len(r.GetNotarizedBlocks())),
		MultiNotarizedBlocksCount: c.MultiNotarizedBlocksCount,
		ZeroNotarizedBlocksCount:  c.ZeroNotarizedBlocksCount,
	}
	t := time.Now()
	ri.TimeStamp = &t
	roundMetrics.CurrentValue = ri
	roundMetrics.Collect(ri)
}
