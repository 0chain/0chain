//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var self = node.Self
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)
	return
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) error {
	go func() {
		mc.updateFinalizedBlock(ctx, b) // nolint: errcheck
	}()

	fr := mc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
	}

	// return if view change is off
	if !mc.IsViewChangeEnabled() {
		return nil
	}

	// perform view change (or not perform)
	if err := mc.ViewChange(ctx, b); err != nil {
		logging.Logger.Error("[mvc] view change", zap.Int64("round", b.Round), zap.Error(err))
		return err
	}

	pn, err := mc.GetPhaseOfBlock(b)
	if err != nil && err != util.ErrValueNotPresent {
		logging.Logger.Error("update finalized block - get phase of block failed", zap.Error(err))
		return err
	}

	if pn == nil {
		return nil
	}

	logging.Logger.Debug("[mvc] update finalized block - send phase node",
		zap.Int64("round", b.Round),
		zap.Int64("start_round", pn.StartRound),
		zap.String("phase", pn.Phase.String()))
	go mc.SendPhaseNode(context.Background(), chain.PhaseEvent{Phase: *pn})
	return nil
}

func (mc *Chain) GenerateBlock(ctx context.Context,
	b *block.Block,
	waitOver bool,
	waitC chan struct{}) error {
	return mc.generateBlockWorker.Run(ctx, func() error {
		return mc.generateBlock(ctx, b, minerChain, waitOver, waitC)
	})
}

func (mc *Chain) createGenerateChallengeTxn(b *block.Block) (*transaction.Transaction, error) {
	return mc.createGenChalTxn(b)
}
