//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"log"
	"sync"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
)

func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	if mc.isIgnoringVerificationTicket(msg.BlockVerificationTicket.Round) {
		crpc.Client().State().VerifyingNonExistentBlock.IgnoredVerificationTicketsNum++
		return
	}

	wg := new(sync.WaitGroup)
	if mc.isBreakingSingleBlock(msg.BlockVerificationTicket.Round, msg.BlockVerificationTicket.VerifierID) {
		wg.Add(1)

		go func() {
			secondSentBlockHash, err := mc.sendBreakingBlock(msg.BlockVerificationTicket.BlockID)
			if err != nil {
				log.Panicf("Conductor: SendBreakingBlock: error while sending block: %v", err)
			}
			if err := configureBreakingSingleBlock(msg.BlockVerificationTicket.BlockID, secondSentBlockHash); err != nil {
				log.Panicf("Conductor: SendBreakingBlock: error while configuring test: %v", err)
			}

			wg.Done()
		}()
	}

	mc.handleVerificationTicketMessage(ctx, msg)

	wg.Wait()
}

func (mc *Chain) isIgnoringVerificationTicket(round int64) bool {
	cfg := crpc.Client().State().VerifyingNonExistentBlock
	if cfg == nil || round != int64(cfg.Round) {
		return false
	}

	// we need to ignore msg by the first ranked replica and for the 1/3 (of miners count) tickets
	nodeType, typeRank := mc.getNodeTypeAndTypeRank(round)
	isFirstRankedReplica := nodeType == replica && typeRank == 0
	return isFirstRankedReplica && cfg.IgnoredVerificationTicketsNum < mc.GetMiners(round).Size()/3
}

func (mc *Chain) isBreakingSingleBlock(roundNum int64, verTicketFromMiner string) bool {
	currRound := mc.GetRound(roundNum)
	if !currRound.IsRanksComputed() {
		return false
	}
	isFirstGenerator := currRound.GetMinerRank(node.Self.Node) == 0
	cfg := crpc.Client().State().BreakingSingleBlock
	shouldTest := cfg != nil && cfg.Round == roundNum && isFirstGenerator
	if !shouldTest {
		return false
	}

	genNum := mc.GetGeneratorsNumOfRound(roundNum)
	rankedMiners := currRound.GetMinersByRank(mc.GetMiners(roundNum).CopyNodes())
	replicators := rankedMiners[genNum:]
	return len(replicators) != 0 && replicators[0].ID == verTicketFromMiner
}

func (mc *Chain) sendBreakingBlock(blockHash string) (sentBlockHash string, err error) {
	bl, err := mc.GetBlock(context.Background(), blockHash)
	if err != nil {
		return "", err
	}
	bl.Txns = make([]*transaction.Transaction, 0)
	bl.ClientStateHash = bl.PrevBlock.ClientStateHash
	cpBl, err := randomizeBlock(bl)
	if err != nil {
		return "", err
	}

	mc.SendBlock(context.Background(), cpBl)

	return cpBl.Hash, nil
}

func configureBreakingSingleBlock(firstBlockHash, secondBlockHash string) error {
	cfg := &cases.BreakingSingleBlockCfg{
		FirstSentBlockHash:  firstBlockHash,
		SecondSentBlockHash: secondBlockHash,
	}
	blob, err := cfg.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().ConfigureTestCase(blob)
}

// HandleVerifyBlockMessage - handles the verify block message.
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	if mc.isIgnoringProposalOrNotarisation(msg.Block.Round) {
		return
	}

	mc.handleVerifyBlockMessage(ctx, msg)
}

func sendBadVerificationTicket(ctx context.Context, round int64, magicBlock *block.MagicBlock) {
	badVT := getBadBVTWithCustomHash(round)
	magicBlock.Miners.SendAll(ctx, VerificationTicketSender(badVT))
}

func getBadBVTWithCustomHash(round int64) *block.BlockVerificationTicket {
	mockedHash, state := "", crpc.Client().State()
	switch {
	case state.VerifyingNonExistentBlock != nil:
		mockedHash = state.VerifyingNonExistentBlock.Hash

	case state.NotarisingNonExistentBlock != nil:
		mockedHash = state.NotarisingNonExistentBlock.Hash

	default:
		log.Panicf("Conductor: getBadBVTWithCustomHash call is unexpected")
	}

	sign, err := node.Self.Sign(mockedHash)
	if err != nil {
		log.Panicf("Conductor: error while signing bad verification ticket: %v", err)
	}
	return &block.BlockVerificationTicket{
		VerificationTicket: block.VerificationTicket{
			VerifierID: node.Self.Underlying().GetKey(),
			Signature:  sign,
		},
		Round:   round,
		BlockID: mockedHash,
	}
}

// HandleNotarizationMessage - handles the block notarization message.
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	if mc.isIgnoringProposalOrNotarisation(msg.Notarization.Round) {
		return
	}

	mc.handleNotarizationMessage(ctx, msg)
}

func (mc *Chain) isIgnoringProposalOrNotarisation(round int64) bool {
	vnebCfg := crpc.Client().State().VerifyingNonExistentBlock
	isVerifyingNonExistentBlock := vnebCfg != nil && round == int64(vnebCfg.Round)
	nnebCfg := crpc.Client().State().NotarisingNonExistentBlock
	isNotarisingNonExistentBlock := nnebCfg != nil && round == int64(nnebCfg.Round)
	if !isVerifyingNonExistentBlock && !isNotarisingNonExistentBlock {
		return false
	}

	// we need to ignore msg by the first ranked replica
	nodeType, typeRank := mc.getNodeTypeAndTypeRank(round)
	return nodeType == replica && typeRank == 0
}

const (
	generator = iota
	replica
)

// getNodeTypeAndTypeRank returns node type and type rank.
// If node with provided parameters is not found, returns -1;-1.
//
// 	Explaining type rank example:
//		Generators num = 2
// 		len(miners) = 4
// 		Generator0:	rank = 0; typeRank = 0.
// 		Generator1:	rank = 1; typeRank = 1.
// 		Replica0:	rank = 2; typeRank = 0.
// 		Replica0:	rank = 3; typeRank = 1.
func (mc *Chain) getNodeTypeAndTypeRank(round int64) (nodeType, typeRank int) {
	genNum := mc.GetGeneratorsNumOfRound(round)
	rankedMiners := mc.GetRound(round).GetMinersByRank(mc.GetMiners(round).CopyNodes())
	for rank, rankedMiner := range rankedMiners {
		if rankedMiner.ID == node.Self.ID {
			nodeType, typeRank = generator, rank
			if rank >= genNum {
				nodeType = replica
				typeRank = rank - genNum
			}
			return
		}
	}
	return -1, -1
}
