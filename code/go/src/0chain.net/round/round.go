package round

import (
	"0chain.net/block"
	"0chain.net/node"
)

//RoundI - an interface that represents a blockchain round
type RoundI interface {
	GetRoundNumber() int64

	GetRandomSeed() int64
	SetRandomSeed(seed int64)

	ComputeMinerRanks(miners *node.Pool)
	GetMinerRank(miner *node.Node) int
	GetMinersByRank(miners *node.Pool) []*node.Node

	AddProposedBlock(b *block.Block) (*block.Block, bool)
	GetProposedBlocks() []*block.Block

	AddNotarizedBlock(b *block.Block) (*block.Block, bool)
	GetNotarizedBlocks() []*block.Block
	GetHeaviestNotarizedBlock() *block.Block
	GetBestRankedNotarizedBlock() *block.Block
	Finalize(b *block.Block)
	IsFinalizing() bool
	SetFinalizing() bool
	IsFinalized() bool
	Clear()

	GetState() int
	SetState(state int)
	AddVRFShare(share *VRFShare) bool
	GetVRFShares() map[string]*VRFShare
}
