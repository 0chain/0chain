//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
)

func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	if isIgnoringVerificationTicket(msg.BlockVerificationTicket.Round) {
		crpc.Client().State().VerifyingNonExistentBlock.IgnoredVerificationTicketsNum++
		return
	}

	wg := new(sync.WaitGroup)
	if isBreakingSingleBlock(msg.BlockVerificationTicket.Round, msg.BlockVerificationTicket.VerifierID) {
		wg.Add(1)

		go func() {
			secondSentBlockHash, err := sendBreakingBlock(msg.BlockVerificationTicket.BlockID)
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

func isIgnoringVerificationTicket(round int64) bool {
	cfg := crpc.Client().State().VerifyingNonExistentBlock
	if cfg == nil || round != cfg.OnRound {
		return false
	}

	// we need to ignore msg by the first ranked replica and for the 1/3 (of miners count) tickets
	mc := GetMinerChain()
	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	isFirstRankedReplica := nodeType == replica && typeRank == 0
	return isFirstRankedReplica && cfg.IgnoredVerificationTicketsNum < mc.GetMiners(round).Size()/3
}

func isBreakingSingleBlock(roundNum int64, verTicketFromMiner string) bool {
	mc := GetMinerChain()

	currRound := mc.GetRound(roundNum)
	if !currRound.IsRanksComputed() {
		return false
	}
	isFirstGenerator := currRound.GetMinerRank(node.Self.Node) == 0
	cfg := crpc.Client().State().BreakingSingleBlock
	shouldTest := cfg != nil && cfg.OnRound == roundNum && isFirstGenerator
	if !shouldTest {
		return false
	}

	genNum := mc.GetGeneratorsNumOfRound(roundNum)
	rankedMiners := currRound.GetMinersByRank(mc.GetMiners(roundNum).CopyNodes())
	replicators := rankedMiners[genNum:]
	return len(replicators) != 0 && replicators[0].ID == verTicketFromMiner
}

func sendBreakingBlock(blockHash string) (sentBlockHash string, err error) {
	mc := GetMinerChain()

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
	if isIgnoringProposalOrNotarisation(msg.Block.Round) {
		return
	}

	crpc.Client().State().ResendProposedBlock.Lock()
	if isResendingProposedBlock(msg.Block.Round) {
		resendProposedBlock(msg.Block)
		if err := crpc.Client().ConfigureTestCase([]byte(msg.Block.Hash)); err != nil {
			log.Panicf("Conductor: error while configuring test case: %#v", err)
		}
	}
	crpc.Client().State().ResendProposedBlock.Unlock()

	mc.handleVerifyBlockMessage(ctx, msg)
}

func isIgnoringProposalOrNotarisation(round int64) bool {
	vnebCfg := crpc.Client().State().VerifyingNonExistentBlock
	isVerifyingNonExistentBlock := vnebCfg != nil && round == vnebCfg.OnRound
	nnebCfg := crpc.Client().State().NotarisingNonExistentBlock
	isNotarisingNonExistentBlock := nnebCfg != nil && round == nnebCfg.OnRound
	if !isVerifyingNonExistentBlock && !isNotarisingNonExistentBlock {
		return false
	}

	// we need to ignore msg by the first ranked replica
	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	return nodeType == replica && typeRank == 0
}

func isResendingProposedBlock(round int64) (resending bool) {
	cfg := crpc.Client().State().ResendProposedBlock
	resending = cfg != nil && round == cfg.OnRound && !cfg.Resent
	if !resending {
		return
	}

	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	return nodeType == generator && typeRank == 1
}

func resendProposedBlock(bl *block.Block) {
	miners := GetMinerChain().GetMiners(bl.Round)
	miners.SendAll(context.Background(), VerifyBlockSender(bl))

	crpc.Client().State().ResendProposedBlock.Resent = true
	return
}

// HandleNotarizationMessage - handles the block notarization message.
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	if isIgnoringProposalOrNotarisation(msg.Notarization.Round) {
		return
	}

	obtainNotarisationIfNeeded(msg.Notarization)

	resendNotarisationIfNeeded(msg.Notarization.Round)

	mc.handleNotarizationMessage(ctx, msg)
}

func obtainNotarisationIfNeeded(not *Notarization) {
	cfg := crpc.Client().State().ResendNotarisation

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil || not.Round != cfg.OnRound-2 || cfg.Notarisation != nil {
		return
	}

	// obtain notarisation if it is configured on round r-2 and notarisation is not obtained.
	// obtained notarisation will be resent on next round by Replica0

	blob, err := json.Marshal(not)
	if err != nil {
		log.Panicf("Conductor: error while obtaining notarisation: %v", err)
	}
	crpc.Client().State().ResendNotarisation.Notarisation = blob
}

func resendNotarisationIfNeeded(round int64) {
	cfg := crpc.Client().State().ResendNotarisation

	cfg.Lock()
	defer cfg.Unlock()

	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	resending := cfg != nil && round == cfg.OnRound-1 && nodeType == replica && typeRank == 0 && !cfg.Resent
	if !resending {
		return
	}

	// resending notarisation obtained on round r-1 by Replica0 if it was not resent

	not := new(Notarization)
	if err := json.Unmarshal(crpc.Client().State().ResendNotarisation.Notarisation, not); err != nil {
		log.Panicf("Conductor: error while resending notarisation: %v", err)
	}
	miners := GetMinerChain().GetMagicBlock(round).Miners
	miners.SendAll(context.Background(), BlockNotarizationSender(not))

	cfg.Resent = true
}

const (
	generator = iota
	replica
)

// getNodeTypeAndTypeRank returns node type and type rank.
// If round is not started or rank is not computed, returns -1;-1.
//
// 	Explaining type rank example:
//		Generators num = 2
// 		len(miners) = 4
// 		Generator0:	rank = 0; typeRank = 0.
// 		Generator1:	rank = 1; typeRank = 1.
// 		Replica0:	rank = 2; typeRank = 0.
// 		Replica1:	rank = 3; typeRank = 1.
func getNodeTypeAndTypeRank(round int64) (nodeType, typeRank int) {
	mc := GetMinerChain()

	roundI := mc.GetRound(round)
	if roundI == nil || !roundI.IsRanksComputed() {
		return -1, -1
	}

	genNum := mc.GetGeneratorsNumOfRound(round)
	isGenerator := mc.IsRoundGenerator(roundI, node.Self.Node)
	nodeType, typeRank = generator, roundI.GetMinerRank(node.Self.Node)
	if !isGenerator {
		nodeType = replica
		typeRank = typeRank - genNum
	}
	return nodeType, typeRank
}
