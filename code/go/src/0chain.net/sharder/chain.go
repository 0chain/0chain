package sharder

import (
	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/round"
)

var sharderChain = &Chain{}

/*SetupSharderChain - setup the miner's chain */
func SetupSharderChain(c *chain.Chain) {
	sharderChain.Chain = *c
	sharderChain.BlockChannel = make(chan *block.Block, 1024)
}

/*GetSharderChain - get the miner's chain */
func GetSharderChain() *Chain {
	return sharderChain
}

/*Chain - A chain structure to manage the sharder activities */
type Chain struct {
	chain.Chain
	BlockChannel chan *block.Block
	rounds       map[int64]*round.Round
}

/*GetBlockChannel - get the block channel where the incoming blocks from the network are put into for further processing */
func (sc *Chain) GetBlockChannel() chan *block.Block {
	return sc.BlockChannel
}

/*SetupGenesisBlock - setup the genesis block for this chain */
func (sc *Chain) SetupGenesisBlock() *block.Block {
	gr, gb := sc.GenerateGenesisBlock()
	if gr == nil || gb == nil {
		panic("Genesis round/block can not be null")
	}
	//sc.AddRound(gr)
	sc.AddGenesisBlock(gb)
	return gb
}
