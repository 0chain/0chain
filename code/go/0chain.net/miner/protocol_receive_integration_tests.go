//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"log"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc"
)

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

// HandleVerificationTicketMessage - handles the verification ticket message.
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	if mc.isIgnoringVerificationTicket(msg.BlockVerificationTicket.Round) {
		crpc.Client().State().VerifyingNonExistentBlock.IgnoredVerificationTicketsNum++
		return
	}

	mc.handleVerificationTicketMessage(ctx, msg)
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
