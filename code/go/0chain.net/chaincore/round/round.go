package round

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
)

//RoundI - an interface that represents a blockchain round
type RoundI interface {
	GetRoundNumber() int64
	GetBlockHash() string

	GetRandomSeed() int64
	SetRandomSeed(seed int64, minersNum int)
	HasRandomSeed() bool
	GetTimeoutCount() int
	SetTimeoutCount(tc int) bool
	SetRandomSeedForNotarizedBlock(seed int64, minersNum int)

	IsRanksComputed() bool
	GetMinerRank(miner *node.Node) int
	GetMinersByRank(miners []*node.Node) []*node.Node

	AddProposedBlock(b *block.Block)
	GetProposedBlocks() []*block.Block
	GetBestRankedProposedBlock() *block.Block

	AddNotarizedBlock(b *block.Block)
	UpdateNotarizedBlock(b *block.Block)
	GetNotarizedBlocks() []*block.Block
	GetHeaviestNotarizedBlock() *block.Block
	GetBestRankedNotarizedBlock() *block.Block
	Finalize(b *block.Block)
	IsFinalizing() bool
	SetFinalizing() bool
	IsFinalized() bool
	Clear()

	GetPhase() Phase
	SetPhase(state Phase)
	AddVRFShare(share *VRFShare, threshold int) bool
	GetVRFShares() map[string]*VRFShare
	Clone() RoundI
}
