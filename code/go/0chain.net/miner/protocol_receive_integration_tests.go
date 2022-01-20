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
	cfg "0chain.net/conductor/config/cases"
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
	testCfg := crpc.Client().State().VerifyingNonExistentBlock
	if testCfg == nil || round != testCfg.OnRound {
		return false
	}

	// we need to ignore msg by the first ranked replica and for the 1/3 (of miners count) tickets
	mc := GetMinerChain()
	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	isFirstRankedReplica := nodeType == replica && typeRank == 0
	return isFirstRankedReplica && testCfg.IgnoredVerificationTicketsNum < mc.GetMiners(round).Size()/3
}

func isBreakingSingleBlock(roundNum int64, verTicketFromMiner string) bool {
	mc := GetMinerChain()

	currRound := mc.GetRound(roundNum)
	if !currRound.IsRanksComputed() {
		return false
	}
	isFirstGenerator := currRound.GetMinerRank(node.Self.Node) == 0
	testCfg := crpc.Client().State().BreakingSingleBlock
	shouldTest := testCfg != nil && testCfg.OnRound == roundNum && isFirstGenerator
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
	caseCfg := &cases.BreakingSingleBlockCfg{
		FirstSentBlockHash:  firstBlockHash,
		SecondSentBlockHash: secondBlockHash,
	}
	blob, err := caseCfg.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().ConfigureTestCase(blob)
}

// HandleVerifyBlockMessage - handles the verify block message.
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	if isIgnoringProposal(msg.Block.Round) {
		return
	}

	resendProposedBlockIfNeeded(msg.Block)

	mc.handleVerifyBlockMessage(ctx, msg)
}

func isIgnoringProposal(round int64) bool {
	var (
		state   = crpc.Client().State()
		testCfg cfg.TestReporter
	)
	switch {
	case state.VerifyingNonExistentBlock != nil:
		testCfg = state.VerifyingNonExistentBlock

	case state.NotarisingNonExistentBlock != nil:
		testCfg = state.NotarisingNonExistentBlock

	case state.BlockStateChangeRequestor != nil:
		testCfg = state.BlockStateChangeRequestor

	default:
		return false
	}

	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	return testCfg.IsOnRound(round) && nodeType == replica && typeRank == 0
}

func resendProposedBlockIfNeeded(b *block.Block) {
	testCfg := crpc.Client().State().ResendProposedBlock

	testCfg.Lock()
	defer testCfg.Unlock()

	var (
		nodeType, typeRank = getNodeTypeAndTypeRank(b.Round)
		resending          = testCfg != nil && testCfg.IsTesting(b.Round, nodeType == generator, typeRank) && !testCfg.Resent
	)
	if !resending {
		return
	}

	miners := GetMinerChain().GetMiners(b.Round)
	miners.SendAll(context.Background(), VerifyBlockSender(b))

	crpc.Client().State().ResendProposedBlock.Resent = true

	if err := crpc.Client().ConfigureTestCase([]byte(b.Hash)); err != nil {
		log.Panicf("Conductor: error while configuring test case: %#v", err)
	}
}

// HandleNotarizationMessage - handles the block notarization message.
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	if isIgnoringNotarisation(msg.Notarization.Round) {
		return
	}

	obtainNotarisationIfNeeded(msg.Notarization)

	resendNotarisationIfNeeded(msg.Notarization)

	configureBlockStateChangeRequestorTestCaseIfNeeded(msg.Notarization)

	mc.handleNotarizationMessage(ctx, msg)
}

func isIgnoringNotarisation(round int64) bool {
	var (
		state   = crpc.Client().State()
		testCfg cfg.TestReporter
	)
	switch {
	case state.VerifyingNonExistentBlock != nil:
		testCfg = state.VerifyingNonExistentBlock

	case state.NotarisingNonExistentBlock != nil:
		testCfg = state.NotarisingNonExistentBlock

	default:
		return false
	}

	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	return testCfg.IsOnRound(round) && nodeType == replica && typeRank == 0
}

func obtainNotarisationIfNeeded(not *Notarization) {
	testCfg := crpc.Client().State().ResendNotarisation

	testCfg.Lock()
	defer testCfg.Unlock()

	var (
		nodeType, typeRank = getNodeTypeAndTypeRank(not.Round)
		obtaining          = testCfg != nil && not.Round == testCfg.OnRound-2 && !testCfg.Resent &&
			((nodeType == generator) == testCfg.ByGenerator) && testCfg.ByNodeWithTypeRank == typeRank
	)
	if !obtaining {
		return
	}

	blob, err := json.Marshal(not)
	if err != nil {
		log.Panicf("Conductor: error while obtaining notarisation: %v", err)
	}
	crpc.Client().State().ResendNotarisation.Notarisation = blob
}

func resendNotarisationIfNeeded(not *Notarization) {
	testCfg := crpc.Client().State().ResendNotarisation

	testCfg.Lock()
	defer testCfg.Unlock()

	var (
		nodeType, typeRank = getNodeTypeAndTypeRank(not.Round)
		resending          = testCfg != nil && not.Round == testCfg.OnRound-1 &&
			((nodeType == generator) == testCfg.ByGenerator) && testCfg.ByNodeWithTypeRank == typeRank
	)
	if !resending {
		return
	}

	resNot := new(Notarization)
	if err := json.Unmarshal(crpc.Client().State().ResendNotarisation.Notarisation, resNot); err != nil {
		log.Panicf("Conductor: error while resending notarisation: %v", err)
	}
	miners := GetMinerChain().GetMagicBlock(not.Round).Miners
	miners.SendAll(context.Background(), BlockNotarizationSender(not))

	testCfg.Resent = true
}

func configureBlockStateChangeRequestorTestCaseIfNeeded(not *Notarization) {
	testCfg := crpc.Client().State().BlockStateChangeRequestor

	testCfg.Lock()
	defer testCfg.Unlock()

	var (
		nodeType, typeRank = getNodeTypeAndTypeRank(not.Round)
		configuring        = testCfg != nil && testCfg.OnRound == not.Round &&
			nodeType == replica && typeRank == 0 && !testCfg.Configured
	)
	if !configuring {
		return
	}

	blob, err := getNotarisationInfo(not).Encode()
	if err != nil {
		log.Panicf("Conductor: error while encoding notarisation info: %v", err)
	}
	if err := crpc.Client().ConfigureTestCase(blob); err != nil {
		log.Panicf("Conductor: error while configuring test case: %v", err)
	}
	testCfg.Configured = true
}

func getNotarisationInfo(not *Notarization) *cases.NotarisationInfo {
	return &cases.NotarisationInfo{
		VerificationTickets: getVerificationTicketsInfo(not.VerificationTickets),
		BlockID:             not.BlockID,
		Round:               not.Round,
	}
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
// 		Replica0:	rank = 3; typeRank = 1.
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
