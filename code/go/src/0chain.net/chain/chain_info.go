package chain

import (
	"0chain.net/block"
)

/*Info - a struct to capture the chain info at runtime */
type Info struct {
	FinalizedRound int64
	FinalizedCount int64
	BlockHash      string
	ChainWeight    float64
	MissedBlocks   int64
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
		ci.ChainWeight = b.ChainWeight
		ci.FinalizedCount = FinalizationTimer.Count()
		ci.MissedBlocks = c.MissedBlocks
	}
}
