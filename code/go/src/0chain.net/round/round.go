package round

import "0chain.net/block"

type RoundI interface {
	GetRoundNumber() int64
	GetRandomSeed() int64
	AddNotarizedBlock(b *block.Block) *block.Block
	GetNotarizedBlocks() []*block.Block
	GetBestNotarizedBlock() *block.Block
	Finalize(b *block.Block)
	IsFinalizing() bool
	SetFinalizing() bool
	IsFinalized() bool
	Clear()
}
