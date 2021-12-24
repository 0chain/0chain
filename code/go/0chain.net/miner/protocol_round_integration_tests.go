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
	nextRound := mc.GetRound(r.GetRoundNumber() + 1)
	if rank := nextRound.GetMinerRank(node.Self.Node); rank == 0 && isMockingNotNotarisedBlockExtension(r.GetRoundNumber()) {
		if bl, err := configureNotNotarisedBlockExtensionTest(r); err != nil {
			log.Printf("Conductor: NotNotarisedBlockExtension: error while configuring test case: %v", err)
		} else {
			return bl
		}
	}

	return mc.getBlockToExtend(ctx, r)
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

func isMockingNotNotarisedBlockExtension(round int64) bool {
	cfg := crpc.Client().State().ExtendNotNotarisedBlock
	return cfg != nil && cfg.Enable && cfg.Round == round
}

func (mc *Chain) StartRound(ctx context.Context, r *Round, seed int64) {
	mc.startRound(ctx, r, seed)

	rNum := r.GetRoundNumber()
	if mc.isSendingBadVerificationTicket(rNum) {
		sendBadVerificationTicket(ctx, rNum, mc.GetMagicBlock(rNum))
		// just notifying conductor that bad verification tickets is sent
		if err := crpc.Client().ConfigureTestCase(nil); err != nil {
			log.Panicf("Conductor: error while configuring test case: %v", err)
		}
	}
}

func (mc *Chain) isSendingBadVerificationTicket(round int64) bool {
	cfg := crpc.Client().State().VerifyingNonExistentBlock
	isConfigured := cfg != nil && round == int64(cfg.Round)
	if !isConfigured {
		return false
	}

	// we need to ignore msg by the first ranked replica
	nodeType, typeRank := mc.getNodeTypeAndTypeRank(round)
	return nodeType == replica && typeRank == 0
}
