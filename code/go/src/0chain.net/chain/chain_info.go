package chain

import (
	"0chain.net/block"
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
}

/*ChainInfo - gather stats of the chain at the powers of 10 */
var ChainInfo []*Info

func init() {
	powers := 11
	ChainInfo = make([]*Info, powers, powers)
	for i := 0; i < powers; i++ {
		ChainInfo[i] = &Info{}
	}
}

/*UpdateInfo - update the chain information */
func (c *Chain) UpdateInfo(b *block.Block) {
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
