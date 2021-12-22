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
	genNum := mc.GetGeneratorsNumOfRound(round)
	rankedMiners := mc.GetRound(round).GetMinersByRank(mc.GetMiners(round).CopyNodes())
	replicators := rankedMiners[genNum:]
	isFirstRankedReplica := len(replicators) != 0 && replicators[0].ID == node.Self.ID
	return isFirstRankedReplica && cfg.IgnoredVerificationTicketsNum < len(rankedMiners)/3
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
	block, err := mc.GetBlock(context.Background(), blockHash)
	if err != nil {
		return "", err
	}
	block.Txns = make([]*transaction.Transaction, 0)
	block.ClientStateHash = block.PrevBlock.ClientStateHash
	cpBl, err := randomizeBlock(block)
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
		sendBadVerificationTicket(ctx, msg.Block.Round, mc.GetMagicBlock(msg.Block.Round))
		// just notifying conductor that bad verification tickets is sent
		if err := crpc.Client().ConfigureTestCase(nil); err != nil {
			log.Panicf("Conductor: error while configuring test case: %v", err)
		}
		return
	}

	mc.handleVerifyBlockMessage(ctx, msg)
}

func sendBadVerificationTicket(ctx context.Context, round int64, magicBlock *block.MagicBlock) {
	badVT := getBadBVTWithCustomHash(round)
	magicBlock.Miners.SendAll(ctx, VerificationTicketSender(badVT))
}

func getBadBVTWithCustomHash(round int64) *block.BlockVerificationTicket {
	mockedHash := crpc.Client().State().VerifyingNonExistentBlock.Hash
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
	vnebTestCase := crpc.Client().State().VerifyingNonExistentBlock
	if vnebTestCase == nil || round != int64(vnebTestCase.Round) {
		return false
	}

	// we need to ignore msg by the first ranked replica
	genNum := mc.GetGeneratorsNumOfRound(round)
	rankedMiners := mc.GetRound(round).GetMinersByRank(mc.GetMiners(round).CopyNodes())
	replicators := rankedMiners[genNum:]
	return len(replicators) != 0 && replicators[0].ID == node.Self.ID
}