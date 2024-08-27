//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
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
