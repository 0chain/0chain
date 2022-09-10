package chain

import (
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	"0chain.net/core/metric"
	"github.com/0chain/common/core/util"
)

/*Info - a struct to capture the chain info at runtime */
type Info struct {
	TimeStamp       *time.Time `json:"ts"`
	FinalizedRound  int64      `json:"round"`
	FinalizedCount  int64      `json:"finalized_blocks_count"`
	BlockHash       string     `json:"block_hash"`
	ClientStateHash util.Key   `json:"client_state_hash"`
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
		FinalizedCount:  SteadyStateFinalizationTimer.Count(),
	}
	t := time.Now()
	ci.TimeStamp = &t

	chainMetrics.CurrentValue = ci
	chainMetrics.Collect(ci)
}

/*UpdateRoundInfo - update the round information */
func (c *Chain) UpdateRoundInfo(r round.RoundI) {
	nnb := int8(len(r.GetNotarizedBlocks()))
	ri := &round.Info{
		Number:                    r.GetRoundNumber(),
		NotarizedBlocksCount:      nnb,
		MultiNotarizedBlocksCount: c.MultiNotarizedBlocksCount,
		ZeroNotarizedBlocksCount:  c.ZeroNotarizedBlocksCount,
		RollbackCount:             c.RollbackCount,
		MissedBlocks:              c.MissedBlocks,
		LongestRollbackLength:     c.LongestRollbackLength,
	}
	t := time.Now()
	ri.TimeStamp = &t
	roundMetrics.CurrentValue = ri
	roundMetrics.Collect(ri)
}
