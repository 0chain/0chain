//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"errors"
	"log"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
)

func (mc *Chain) GetBlockToExtend(ctx context.Context, r round.RoundI) *block.Block {
	cfg := crpc.Client().State().ExtendNotNotarisedBlock
	cfg.Lock()
	if isMockingNotNotarisedBlockExtension(r.GetRoundNumber()) {
		bl, err := configureNotNotarisedBlockExtensionTest(r)
		if err != nil {
			log.Panicf("Conductor: NotNotarisedBlockExtension: error while configuring test case: %v", err)
		}

		cfg.Configured = true

		cfg.Unlock()
		return bl
	}
	cfg.Unlock()

	return mc.getBlockToExtend(ctx, r)
}

func isMockingNotNotarisedBlockExtension(round int64) bool {
	cfg := crpc.Client().State().ExtendNotNotarisedBlock

	isConfigured := cfg != nil && cfg.OnRound == round+1 && !cfg.Configured
	if !isConfigured {
		return false
	}

	nodeType, typeRank := chain.GetNodeTypeAndTypeRank(round + 1)
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

func (mc *Chain) StartNextRound(ctx context.Context, r *Round) *Round {
	nextRound := mc.startNextRound(ctx, r)

	rNum := nextRound.GetRoundNumber()
	sendBadVerificationTicketIfNeeded(rNum)

	return nextRound
}

func sendBadVerificationTicketIfNeeded(rNum int64) {
	state := crpc.Client().State()
	testCfg := state.VerifyingNonExistentBlock

	testCfg.Lock()
	defer testCfg.Unlock()

	if testCfg == nil || rNum != testCfg.OnRound || testCfg.Sent || !state.IsMonitor {
		return
	}

	badVT := getBadBVTWithCustomHash(rNum)
	magicBlock := GetMinerChain().GetMagicBlock(rNum)
	magicBlock.Miners.SendAll(context.Background(), VerificationTicketSender(badVT))

	testCfg.Sent = true

	if err := crpc.Client().ConfigureTestCase(nil); err != nil {
		log.Panicf("Conductor: error while configuring test case: %v", err)
	}
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

// HandleRoundTimeout handle timeouts appropriately.
func (mc *Chain) HandleRoundTimeout(ctx context.Context, round int64) {
	mc.handleRoundTimeout(ctx, round)

	minerRound := mc.GetMinerRound(round)
	addRoundInfoResultOnHandleRoundTimeoutIfNeeded(minerRound)
}

func addRoundInfoResultOnHandleRoundTimeoutIfNeeded(minerRound *Round) {
	testing := isTestingHalfNodesDown(minerRound) || isTestingSendDifferentBlocks(minerRound) || isTestingMissingLFBTickets(minerRound)
	if !testing {
		return
	}

	finalisedBlockHash := ""
	if minerRound.IsFinalized() {
		finalisedBlockHash = minerRound.BlockHash
	}
	if err := chain.AddRoundInfoResult(minerRound, finalisedBlockHash); err != nil {
		log.Panicf("Conductor: error while adding test case result: %v", err)
	}
}

func isTestingHalfNodesDown(minerRound *Round) bool {
	hndCfg := crpc.Client().State().HalfNodesDown
	return hndCfg != nil &&
		hndCfg.OnRound == minerRound.Number &&
		minerRound.GetTimeoutCount() == 1 &&
		minerRound.GetSoftTimeoutCount() == 0
}

func isTestingSendDifferentBlocks(minerRound *Round) bool {
	var (
		cfgFromFirstGen        = crpc.Client().State().SendDifferentBlocksFromFirstGenerator
		shouldTestFromFirstGen = cfgFromFirstGen != nil && int64(cfgFromFirstGen.Round) == minerRound.Number

		cfgFromAllGen        = crpc.Client().State().SendDifferentBlocksFromAllGenerators
		shouldTestFromAllGen = cfgFromAllGen != nil && int64(cfgFromAllGen.Round) == minerRound.Number
	)
	return (shouldTestFromAllGen || shouldTestFromFirstGen) &&
		minerRound.GetTimeoutCount() == 1 &&
		minerRound.GetSoftTimeoutCount() == 0
}

func isTestingMissingLFBTickets(minerRound *Round) bool {
	lfbTicketAhead := int64(config.GetLFBTicketAhead())
	hndCfg := crpc.Client().State().MissingLFBTicket
	return hndCfg != nil &&
		hndCfg.OnRound+lfbTicketAhead-1 == minerRound.Number &&
		minerRound.GetTimeoutCount() == 0 &&
		minerRound.GetSoftTimeoutCount() == 9
}

func (mc *Chain) moveToNextRoundNotAhead(ctx context.Context, r *Round) {
	mc.moveToNextRoundNotAheadImpl(ctx, r, lockChainIfConfigured)
}

func lockChainIfConfigured() {
	for {
		if !crpc.Client().State().IsLock {
			break
		}

		<-time.NewTimer(time.Microsecond * 200).C
	}
}
