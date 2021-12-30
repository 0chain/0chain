//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"errors"
	"log"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
)

func (mc *Chain) GetBlockToExtend(ctx context.Context, r round.RoundI) *block.Block {
	if isMockingNotNotarisedBlockExtension(r.GetRoundNumber()) {
		if bl, err := configureNotNotarisedBlockExtensionTest(r); err != nil {
			log.Printf("Conductor: NotNotarisedBlockExtension: error while configuring test case: %v", err)
		} else {
			return bl
		}
	}

	return mc.getBlockToExtend(ctx, r)
}

func isMockingNotNotarisedBlockExtension(round int64) bool {
	cfg := crpc.Client().State().ExtendNotNotarisedBlock
	isConfigured := cfg != nil && cfg.OnRound == round
	if !isConfigured {
		return false
	}

	nodeType, typeRank := getNodeTypeAndTypeRank(round + 1)
	return nodeType == generator && typeRank == 0
}

func configureNotNotarisedBlockExtensionTest(r round.RoundI) (*block.Block, error) {
	bl := getNotNotarisedBlock(r)
	if bl == nil {
		return nil, errors.New("not notarised block not found")
	}
	if err := crpc.Client().ConfigureTestCase([]byte(bl.Hash)); err != nil {
		return nil, err
	}

	return bl, nil
}

func getNotNotarisedBlock(r round.RoundI) *block.Block {
	var bl *block.Block
	for _, prB := range r.GetProposedBlocks() {
		var notarised bool
		for _, notB := range r.GetNotarizedBlocks() {
			if prB.Hash == notB.Hash {
				notarised = true
			}
		}
		if !notarised {
			bl = prB
			break
		}
	}
	return bl
}

func (mc *Chain) StartRound(ctx context.Context, r *Round, seed int64) {
	mc.startRound(ctx, r, seed)

	rNum := r.GetRoundNumber()
	if isSendingBadVerificationTicket(rNum) {
		sendBadVerificationTicket(ctx, rNum, mc.GetMagicBlock(rNum))
		// just notifying conductor that bad verification tickets is sent
		if err := crpc.Client().ConfigureTestCase(nil); err != nil {
			log.Panicf("Conductor: error while configuring test case: %v", err)
		}
	}
}

func isSendingBadVerificationTicket(round int64) bool {
	cfg := crpc.Client().State().VerifyingNonExistentBlock
	isConfigured := cfg != nil && round == cfg.OnRound
	if !isConfigured {
		return false
	}

	// we need to ignore msg by the first ranked replica
	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	return nodeType == replica && typeRank == 0
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
