//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"errors"
	"log"
	"math/rand"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
	crpcutils "0chain.net/conductor/utils"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
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
	mc.updateFinalizedBlock(ctx, b)

	if mc.isTestingOnUpdateFinalizedBlock(b.Round) {
		if err := chain.AddRoundInfoResult(mc.GetRound(b.Round), b.Hash); err != nil {
			log.Panicf("Conductor: error while sending round info result: %v", err)
		}
	}
}

func (mc *Chain) isTestingOnUpdateFinalizedBlock(round int64) bool {
	s := crpc.Client().State()
	var isTestingFunc func(round int64, generator bool, typeRank int) bool
	switch {
	case s.ExtendNotNotarisedBlock != nil:
		isTestingFunc = s.ExtendNotNotarisedBlock.IsTesting

	case s.BreakingSingleBlock != nil:
		isTestingFunc = s.BreakingSingleBlock.IsTesting

	case s.SendInsufficientProposals != nil:
		isTestingFunc = s.SendInsufficientProposals.IsTesting

	case s.NotarisingNonExistentBlock != nil:
		isTestingFunc = s.NotarisingNonExistentBlock.IsTesting

	case s.ResendProposedBlock != nil:
		isTestingFunc = s.ResendProposedBlock.IsTesting

	case s.ResendNotarisation != nil:
		isTestingFunc = s.ResendNotarisation.IsTesting

	case s.BadTimeoutVRFS != nil:
		isTestingFunc = s.BadTimeoutVRFS.IsTesting

	case s.BlockStateChangeRequestor != nil && s.BlockStateChangeRequestor.GetType() != cases.BSCRChangeNode:
		isTestingFunc = s.BlockStateChangeRequestor.IsTesting

	case s.MinerNotarisedBlockRequestor != nil:
		isTestingFunc = s.MinerNotarisedBlockRequestor.IsTesting

	case s.FBRequestor != nil:
		isTestingFunc = s.FBRequestor.IsTesting

	case s.SendDifferentBlocksFromFirstGenerator != nil:
		isTestingFunc = s.SendDifferentBlocksFromFirstGenerator.IsTesting

	case s.SendDifferentBlocksFromAllGenerators != nil:
		isTestingFunc = s.SendDifferentBlocksFromAllGenerators.IsTesting

	default:
		return false
	}

	nodeType, typeRank := chain.GetNodeTypeAndTypeRank(round)
	return isTestingFunc(round, nodeType == generator, typeRank)
}

func (mc *Chain) GenerateBlock(ctx context.Context, b *block.Block, _ chain.BlockStateHandler, waitOver bool) error {
	if isIgnoringGenerateBlock(b.Round) {
		return nil
	}

	return mc.generateBlockWorker.Run(ctx, func() error {
		return mc.generateBlock(ctx, b, minerChain, waitOver)
	})
}

func isIgnoringGenerateBlock(rNum int64) bool {
	cfg := crpc.Client().State().NotarisingNonExistentBlock
	nodeType, typeRank := chain.GetNodeTypeAndTypeRank(rNum)
	// we need to ignore generating block phase on configured round and on the Generator1 node
	return cfg != nil && cfg.OnRound == rNum && nodeType == generator && typeRank == 1
}

func beforeBlockGeneration(b *block.Block, ctx context.Context, txnIterHandler func(ctx context.Context, qe datastore.CollectionEntity) bool) {
	// inject double-spend transaction if configured
	pb := b.PrevBlock
	state := crpc.Client().State()
	selfKey := node.Self.GetKey()
	isDoubleSpend := state.DoubleSpendTransaction.IsBy(state, selfKey) && pb != nil && len(pb.Txns) > 0 && !hasDST(b.Txns, pb.Txns)
	if !isDoubleSpend {
		return
	}
	dstxn := pb.Txns[rand.Intn(len(pb.Txns))]     // a random one from the previous block
	state.DoubleSpendTransactionHash = dstxn.Hash // exclude the duplicate transactio from checks
	logging.Logger.Info("injecting double-spend transaction", zap.String("hash", dstxn.Hash))
	txnIterHandler(ctx, dstxn) // inject double-spend transaction
}
