//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
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

func beforeBlockGeneration(b *block.Block, ctx context.Context, txnIterHandler func(ctx context.Context, qe datastore.CollectionEntity) bool) {

}
