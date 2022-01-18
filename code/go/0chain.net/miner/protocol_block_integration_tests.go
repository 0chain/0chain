//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
	crpcutils "0chain.net/conductor/utils"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	var state = crpc.Client().State()

	if !state.SignOnlyCompetingBlocks.IsCompetingGroupMember(state, b.MinerID) {
		return nil, errors.New("skip block signing -- not competing block")
	}

	// regular or competing signing
	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var (
		self  = node.Self
		state = crpc.Client().State()
	)
	b.HashBlock()

	switch {
	case state.WrongBlockHash != nil:
		b.Hash = revertString(b.Hash) // just wrong block hash
		b.Signature, err = self.Sign(b.Hash)
	case state.WrongBlockSignHash != nil:
		b.Signature, err = self.Sign(revertString(b.Hash)) // sign another hash
	case state.WrongBlockSignKey != nil:
		b.Signature, err = crpcutils.Sign(b.Hash) // wrong secret key
	default:
		b.Signature, err = self.Sign(b.Hash)
	}

	return
}

// has double-spend transaction
func hasDST(pb, b []*transaction.Transaction) (has bool) {
	for _, bx := range b {
		if bx == nil {
			continue
		}
		for _, pbx := range pb {
			if pbx == nil {
				continue
			}
			if bx.Hash == pbx.Hash {
				return true // has
			}
		}
	}
	return false // has not
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	wg := new(sync.WaitGroup)

	if mc.isTestingNotNotarisedBlockExtension(b.Round) {
		wg.Add(1)
		go func() {
			if err := addNotNotarisedBlockExtensionTestResult(mc.GetRound(b.Round)); err != nil {
				log.Printf("Conductor: NotNotarisedBlockExtension: error while sending result: %v", err)
			}
			wg.Done()
		}()
	}

	if mc.isTestingSendBreakingBlock(b.Round) {
		wg.Add(1)
		go func() {
			if err := mc.addSendBreakingBlockResult(b.Hash, mc.GetRound(b.Round)); err != nil {
				log.Printf("Conductor: error while sending result: %v", err)
			}
			wg.Done()
		}()
	}

	if mc.isTestingSendInsufficientProposals(b.Round) {
		wg.Add(1)
		go func() {
			if err := addSendInsufficientProposalsResult(mc.GetRoundBlocks(b.Round)); err != nil {
				log.Panicf("Conductor: error while sending test result: %v", err)
			}
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		mc.updateFinalizedBlock(ctx, b)
		wg.Done()
	}()

	wg.Wait()
}

func addNotNotarisedBlockExtensionTestResult(r round.RoundI) error {
	testRes := collectVerificationStatuses(r)
	blob, err := json.Marshal(testRes)
	if err != nil {
		return err
	}
	return crpc.Client().AddTestCaseResult(blob)
}

func collectVerificationStatuses(r round.RoundI) map[string]int {
	res := make(map[string]int)
	for _, b := range r.GetProposedBlocks() {
		res[b.PrevHash] = b.GetVerificationStatus()
	}
	for _, b := range r.GetNotarizedBlocks() {
		res[b.PrevHash] = b.GetVerificationStatus()
	}
	return res
}

func (mc *Chain) isTestingNotNotarisedBlockExtension(round int64) bool {
	cfg := crpc.Client().State().ExtendNotNotarisedBlock
	shouldTest := cfg != nil && cfg.Enable && cfg.Round+1 == round
	if !shouldTest {
		return false
	}

	// we need to collect all block's verification statuses from the first ranked replica
	genNum := mc.GetGeneratorsNumOfRound(round)
	rankedMiners := mc.GetRound(round).GetMinersByRank(mc.GetMiners(round).CopyNodes())
	replicators := rankedMiners[genNum:]
	return len(replicators) != 0 && replicators[0].ID == node.Self.ID
}

func (mc *Chain) addSendBreakingBlockResult(finalisedBlockHash string, r round.RoundI) error {
	rBlocks := mc.GetRoundBlocks(r.GetRoundNumber())
	res := &cases.BreakingSingleBlockResult{
		FinalisedBlockHash: finalisedBlockHash,
		RoundBlocksInfo:    collectBlocksInfo(rBlocks),
	}
	blob, err := res.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().AddTestCaseResult(blob)
}

func collectBlocksInfo(blocks []*block.Block) []*cases.BlockInfo {
	blocksInfo := make([]*cases.BlockInfo, 0, len(blocks))
	for _, bl := range blocks {
		blocksInfo = append(blocksInfo, &cases.BlockInfo{
			Hash:               bl.Hash,
			Notarised:          bl.IsBlockNotarized(),
			VerificationStatus: bl.GetVerificationStatus(),
		})
	}
	return blocksInfo
}

func (mc *Chain) isTestingSendBreakingBlock(round int64) bool {
	cfg := crpc.Client().State().BreakingSingleBlock
	shouldTest := cfg != nil && cfg.Round == round
	if !shouldTest {
		return false
	}

	// we need to collect test's report from the first ranked replica
	genNum := mc.GetGeneratorsNumOfRound(round)
	rankedMiners := mc.GetRound(round).GetMinersByRank(mc.GetMiners(round).CopyNodes())
	replicators := rankedMiners[genNum:]
	return len(replicators) != 0 && replicators[0].ID == node.Self.ID
}

func (mc *Chain) isTestingSendInsufficientProposals(round int64) bool {
	cfg := crpc.Client().State().SendInsufficientProposals
	shouldTest := cfg != nil && cfg.Round == round
	if !shouldTest {
		return false
	}

	// we need to collect reports from the first ranked replica
	genNum := mc.GetGeneratorsNumOfRound(round)
	rankedMiners := mc.GetRound(round).GetMinersByRank(mc.GetMiners(round).CopyNodes())
	replicators := rankedMiners[genNum:]
	return len(replicators) != 0 && replicators[0].ID == node.Self.ID
}

func addSendInsufficientProposalsResult(roundBlocks []*block.Block) error {
	res := cases.SendInsufficientProposalsResult(collectBlocksInfo(roundBlocks))
	blob, err := res.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().AddTestCaseResult(blob)
}
