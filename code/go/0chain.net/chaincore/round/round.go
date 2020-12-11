package round

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
)

//RoundI - an interface that represents a blockchain round
type RoundI interface {
	GetRoundNumber() int64

	GetRandomSeed() int64
	SetRandomSeed(seed int64, miners *node.Pool)
	HasRandomSeed() bool
	GetTimeoutCount() int
	SetTimeoutCount(tc int) bool
	SetRandomSeedForNotarizedBlock(seed int64, miners *node.Pool)

	IsRanksComputed() bool
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
	AddVRFShare(share *VRFShare, threshold int) bool
	GetVRFShares() map[string]*VRFShare
}
