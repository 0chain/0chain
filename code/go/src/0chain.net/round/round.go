package round

import (
	"0chain.net/block"
	"0chain.net/node"
)

type RoundI interface {
	GetRoundNumber() int64
	GetRandomSeed() int64

	GetMinerRank(miner *node.Node) int
	GetMinersByRank(miners *node.Pool) []*node.Node
	AddNotarizedBlock(b *block.Block) (*block.Block, bool)
	GetNotarizedBlocks() []*block.Block
	GetBestNotarizedBlock() *block.Block
	Finalize(b *block.Block)
	IsFinalizing() bool
	SetFinalizing() bool
	IsFinalized() bool
	Clear()
}
