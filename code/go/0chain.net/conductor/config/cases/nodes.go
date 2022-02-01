package cases

import (
	"strconv"

	"0chain.net/chaincore/node"
)

type (
	// Nodes represents struct for containing miners and sharders info.
	Nodes struct {
		Miners   Miners   `json:"miners" yaml:"miners" mapstructure:"miners"`
		Sharders Sharders `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
	}

	// Miners represents list of Miner.
	Miners []*Miner

	// Miner represents struct for containing miner's info.
	//
	// 	Explaining definitions example:
	//		Generators num = 2
	// 		len(miners) = 4
	//
	// 		Generator0:	rank = 0; TypeRank = 0; Generator = true.
	// 		Generator1:	rank = 1; TypeRank = 1; Generator = true.
	// 		Replica0:	rank = 2; TypeRank = 0; Generator = false.
	// 		Replica0:	rank = 3; TypeRank = 1; Generator = false.
	Miner struct {
		Generator bool `json:"generator" yaml:"generator" mapstructure:"generator"`

		TypeRank int `json:"type_rank" yaml:"type_rank" mapstructure:"type_rank"`
	}

	// Sharders represents list of Sharder.
	Sharders []Sharder

	// Sharder represents string that explains sharder type.
	//
	// Example: "sharder-1".
	Sharder string
)

// IsActingOnTestRequestor checks with MinerInformer help response to the requestor or not.
//
// 	Returns true if the follow is correct:
//		provided informer must be not nil;
//		requestor must be Replica0;
//		round must be equal to the expected round;
//		requested node should be in the Nodes list.
func (n *Nodes) IsActingOnTestRequestor(informer MinerInformer, requestorID string, expectedRound int64) bool {
	if informer == nil || expectedRound != informer.GetRoundNumber() || !informer.Contains(requestorID) {
		return false
	}

	isRequestorGenerator, requestorTypeRank := informer.IsGenerator(requestorID), informer.GetTypeRank(requestorID)
	if isRequestorGenerator || requestorTypeRank != 0 { // not a Replica0
		return false
	}

	if node.Self.Type == node.NodeTypeSharder {
		selfName := "sharder-" + strconv.Itoa(node.Self.SetIndex)
		return n.Sharders.Contains(selfName)
	}

	// node type miner
	return n.Miners.Contains(informer.IsGenerator(node.Self.ID), informer.GetTypeRank(node.Self.ID))
}

// Num returns number of all nodes contained by Nodes.
func (n *Nodes) Num() int {
	return len(n.Miners) + len(n.Sharders)
}

func (n *Nodes) IsEmpty() bool {
	return (len(n.Miners) + len(n.Sharders)) == 0
}

// Contains looks for Miner with provided Miner.Generator and Miner.TypeRank and returns true if founds.
func (m Miners) Contains(generator bool, typeRank int) bool {
	for _, miner := range m {
		if miner.Generator == generator && miner.TypeRank == typeRank {
			return true
		}
	}
	return false
}

// Contains looks for Sharder with provided name.
func (s Sharders) Contains(name string) bool {
	for _, sharder := range s {
		if string(sharder) == name {
			return true
		}
	}
	return false
}

type (
	minerInformer struct {
		Ranker
		miners *node.Pool
		genNum int
	}

	// Ranker represents interface for ranking miners on round.
	Ranker interface {
		// GetMinerRank return rank of the miner.
		//
		// 	Explaining type rank example:
		//		Generators num = 2
		// 		len(miners) = 4
		// 		Generator0:	rank = 0; rank = 0.
		// 		Generator1:	rank = 1; rank = 1.
		// 		Replica0:	rank = 2; rank = 2.
		// 		Replica0:	rank = 3; rank = 3.
		GetMinerRank(miner *node.Node) int

		GetRoundNumber() int64
	}

	MinerInformer interface {
		IsGenerator(minerID string) bool

		// GetTypeRank returns miner's type rank.
		//
		// 	Explaining type rank example:
		//		Generators num = 2
		// 		len(miners) = 4
		// 		Generator0:	rank = 0; typeRank = 0; isGenerator = true.
		// 		Generator1:	rank = 1; typeRank = 1; isGenerator = true.
		// 		Replica0:	rank = 2; typeRank = 0; isGenerator = false.
		// 		Replica0:	rank = 3; typeRank = 1; isGenerator = false.
		GetTypeRank(minerID string) int

		Contains(minerID string) bool

		GetRoundNumber() int64
	}
)

var (
	// Ensure minerInformer implements MinerInformer interface.
	_ MinerInformer = (*minerInformer)(nil)
)

// NewMinerInformer creates initialized MinerInformer impementation.
func NewMinerInformer(ranker Ranker, miners *node.Pool, genNum int) MinerInformer {
	return &minerInformer{
		Ranker: ranker,
		miners: miners,
		genNum: genNum,
	}
}

// IsGenerator implements MinerInformer interface.
func (mi *minerInformer) IsGenerator(minerID string) bool {
	miner := mi.miners.GetNode(minerID)
	minerRank := mi.Ranker.GetMinerRank(miner)
	return minerRank < mi.genNum
}

// GetTypeRank implements MinerInformer interface.
func (mi *minerInformer) GetTypeRank(minerID string) int {
	miner := mi.miners.GetNode(minerID)
	minerRank := mi.Ranker.GetMinerRank(miner)
	isGenerator := minerRank < mi.genNum
	typeRank := minerRank
	if !isGenerator {
		typeRank = typeRank - mi.genNum
	}
	return typeRank
}

// Contains implements MinerInformer interface.
func (mi *minerInformer) Contains(minerID string) bool {
	return mi.miners.HasNode(minerID)
}

// GetRoundNumber implements MinerInformer interface.
func (mi *minerInformer) GetRoundNumber() int64 {
	return mi.Ranker.GetRoundNumber()
}
