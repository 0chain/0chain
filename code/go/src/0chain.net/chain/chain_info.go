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

func init() {
	powers := 11
	ChainInfo = make([]*Info, powers, powers)
	RoundInfo = make([]*round.Info, powers, powers)
	for i := 0; i < powers; i++ {
		ChainInfo[i] = &Info{}
		RoundInfo[i] = &round.Info{}
	}
}

/*UpdateChainInfo - update the chain information */
func (c *Chain) UpdateChainInfo(b *block.Block) {
	powers := len(ChainInfo)
	for idx, tp := 0, int64(1); idx < powers && b.Round%tp == 0; idx, tp = idx+1, 10*tp {
		ci := ChainInfo[idx]
		ci.FinalizedRound = b.Round
		ci.BlockHash = b.Hash
		ci.ClientStateHash = b.ClientStateHash
		ci.ChainWeight = b.ChainWeight
		ci.FinalizedCount = FinalizationTimer.Count()
		ci.MissedBlocks = c.MissedBlocks
	}
}

/*UpdateRoundInfo - update the round information */
func (c *Chain) UpdateRoundInfo(r *round.Round) {
	powers := len(RoundInfo)
	for idx, tp := 0, int64(1); idx < powers && r.Number%tp == 0; idx, tp = idx+1, 10*tp {
		ri := RoundInfo[idx]
		ri.Number = r.Number
		ri.NotarizedBlocksCount = int8(len(r.GetNotarizedBlocks()))
		ri.MultiNotarizedBlocksCount = c.MultiNotarizedBlocksCount
		ri.ZeroNotarizedBlocksCount = c.ZeroNotarizedBlocksCount
	}
}
