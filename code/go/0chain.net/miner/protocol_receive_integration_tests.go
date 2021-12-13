//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"log"
	"sync"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
)

func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
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
