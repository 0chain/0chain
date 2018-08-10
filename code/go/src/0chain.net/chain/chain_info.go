package chain

import (
	"0chain.net/block"
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

/*ChainInfo - gather stats of the chain at the powers of 10 */
var ChainInfo []*Info
var RoundInfo []*round.Info

var chainMetric Metrics
var roundMetric Metrics

func init() {
	power := 10
	len := 10
	chainMetric = NewPowerMetrics(power, len)
	roundMetric = NewPowerMetrics(power, len)

	infoLen := power*len + 1
	ChainInfo = make([]*Info, infoLen, infoLen)
	RoundInfo = make([]*round.Info, infoLen, infoLen)
	for i := 0; i < infoLen; i++ {
		ChainInfo[i] = &Info{}
		RoundInfo[i] = &round.Info{}
	}
}

/*UpdateChainInfo - update the chain information */
func (c *Chain) UpdateChainInfo(b *block.Block) {
	ci := &Info{
		FinalizedRound:  b.Round,
		BlockHash:       b.Hash,
		ClientStateHash: b.ClientStateHash,
		ChainWeight:     b.ChainWeight,
		FinalizedCount:  FinalizationTimer.Count(),
		MissedBlocks:    c.MissedBlocks,
	}
	ChainInfo[0] = ci

	chainMetric.Collect(b.Round, ci)
	data := chainMetric.Retrieve()
	for i := 0; i < len(data); i++ {
		ChainInfo[i+1] = data[i].(*Info)
	}
}

/*UpdateRoundInfo - update the round information */
func (c *Chain) UpdateRoundInfo(r *round.Round) {
	ri := &round.Info{
		Number:                    r.Number,
		NotarizedBlocksCount:      int8(len(r.GetNotarizedBlocks())),
		MultiNotarizedBlocksCount: c.MultiNotarizedBlocksCount,
		ZeroNotarizedBlocksCount:  c.ZeroNotarizedBlocksCount,
	}
	RoundInfo[0] = ri

	roundMetric.Collect(r.Number, ri)
	data := roundMetric.Retrieve()
	for i := 0; i < len(data); i++ {
		RoundInfo[i+1] = data[i].(*round.Info)
	}
}
