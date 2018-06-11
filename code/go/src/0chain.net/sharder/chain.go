package sharder

import (
	"0chain.net/block"
	"0chain.net/chain"
)

var sharderChain = &Chain{}

/*SetupMinerChain - setup the miner's chain */
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
	CurrentRound int64
}

/*GetBlockChannel - get the block channel where the incoming blocks from the network are put into for further processing */
func (sc *Chain) GetBlockChannel() chan *block.Block {
	return sc.BlockChannel
}
