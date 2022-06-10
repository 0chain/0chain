//go:build integration_tests
// +build integration_tests

package chain

import (
	"log"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/core/logging"
)

func (c *Chain) FinalizeRound(r round.RoundI) {
	c.FinalizeRoundImpl(r)

	addResultOnFinalizeRoundIfNeeded(r)
}

func addResultOnFinalizeRoundIfNeeded(r round.RoundI) {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	nodeType, typeRank := GetNodeTypeAndTypeRank(r.GetRoundNumber())
	testing := cfg != nil && cfg.GetType() == cases.BSCRChangeNode && !cfg.Resulted &&
		cfg.IsTesting(r.GetRoundNumber(), nodeType == generator, typeRank)
	if !testing {
		return
	}

	finalisedBlockHash := ""
	if r.IsFinalized() {
		roundImpl, ok := r.(*round.Round)
		if !ok {
			panic("unexpected type")
		}
		finalisedBlockHash = roundImpl.BlockHash
	}

	if err := AddRoundInfoResult(r, finalisedBlockHash); err != nil {
		log.Panicf("Conductor: error while sending round info result: %v", err)
	}

	cfg.Resulted = true
}

const (
	generator = iota
	replica
)

// GetNodeTypeAndTypeRank returns node type
//and type rank.
// If ranks is not computed, returns -1, -1.
//
// 	Explaining type rank example:
//		Generators num = 2
// 		len(miners) = 4
// 		Generator0:	rank = 0; typeRank = 0.
// 		Generator1:	rank = 1; typeRank = 1.
// 		Replica0:	rank = 2; typeRank = 0.
// 		Replica1:	rank = 3; typeRank = 1.
func GetNodeTypeAndTypeRank(roundNum int64) (nodeType, typeRank int) {
	sChain := GetServerChain()

	r := sChain.GetRound(roundNum)
	if r == nil || !r.IsRanksComputed() {
		logging.Logger.Warn("Conductor: ranks is not computed yet", zap.Int64("round", roundNum))
		return -1, -1
	}

	isGenerator := sChain.IsRoundGenerator(r, node.Self.Node)
	nodeType, typeRank = generator, r.GetMinerRank(node.Self.Node)
	if !isGenerator {
		nodeType = replica
		typeRank = typeRank - sChain.GetGeneratorsNumOfRound(roundNum)
	}
	return nodeType, typeRank
}
func AddRoundInfoResult(r round.RoundI, finalisedBlockHash string) error {
	res := roundInfo(r.GetRoundNumber(), finalisedBlockHash)
	blob, err := res.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().AddTestCaseResult(blob)
}

func roundInfo(rNum int64, finalisedBlockHash string) *cases.RoundInfo {
	sCh := GetServerChain()

	miners := sCh.GetMiners(rNum).CopyNodes()
	rankedMiners := make([]string, len(miners))
	roundI := sCh.GetRound(rNum)
	for _, miner := range miners {
		rankedMiners[roundI.GetMinerRank(miner)] = miner.ID
	}

	propBlocks := roundI.GetProposedBlocks()
	propBlocksInfo := make([]*cases.BlockInfo, 0, len(propBlocks))
	for _, b := range propBlocks {
		propBlocksInfo = append(propBlocksInfo, getBlockInfo(b))
	}
	notBlocks := roundI.GetNotarizedBlocks()
	notBlocksInfo := make([]*cases.BlockInfo, 0, len(notBlocks))
	for _, b := range notBlocks {
		notBlocksInfo = append(notBlocksInfo, getBlockInfo(b))
	}

	return &cases.RoundInfo{
		Num:                rNum,
		GeneratorsNum:      sCh.GetGeneratorsNum(),
		RankedMiners:       rankedMiners,
		FinalisedBlockHash: finalisedBlockHash,
		ProposedBlocks:     propBlocksInfo,
		NotarisedBlocks:    notBlocksInfo,
		TimeoutCount:       roundI.GetTimeoutCount(),
		RoundRandomSeed:    roundI.GetRandomSeed(),
		IsFinalised:        roundI.IsFinalized(),
	}
}

func getBlockInfo(b *block.Block) *cases.BlockInfo {
	return &cases.BlockInfo{
		Hash:                b.Hash,
		PrevHash:            b.PrevHash,
		Notarised:           b.IsBlockNotarized(),
		VerificationStatus:  b.GetVerificationStatus(),
		Rank:                b.RoundRank,
		VerificationTickets: GetVerificationTicketsInfo(b.VerificationTickets),
	}
}

func GetVerificationTicketsInfo(tickets []*block.VerificationTicket) []*cases.VerificationTicketInfo {
	tickInfo := make([]*cases.VerificationTicketInfo, 0, len(tickets))
	for _, ticket := range tickets {
		tickInfo = append(tickInfo, &cases.VerificationTicketInfo{
			VerifierID: ticket.VerifierID,
		})
	}
	return tickInfo
}
