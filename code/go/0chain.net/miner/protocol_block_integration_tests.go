// +build integration_tests

package miner

import (
	"context"
	"errors"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"

	crpc "0chain.net/conductor/conductrpc"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	var state = crpc.Client().State()

	if !state.SignOnlyCompetingBlocks.IsCompetingGroupMember(state, b.MinerID) {
		println("SIGN ONLY COMPETING BLOCK SKIP")
		return nil, errors.New("skip block signing -- not competing block")
	}

	// regular or competing signing
	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var (
		self  = node.GetSelfNode(ctx)
		state = crpc.Client().State()
	)
	b.HashBlock()

	switch {
	case state.WrongBlockHash != nil:
		b.Hash = revertString(b.Hash) // just wrong block hash
		b.Signature, err = self.Sign(b.Hash)
		println("(GENERATE BLOCK) SET AND SIGN WRONG BLOCK HASH")
	case state.WrongBlockSignHash != nil:
		b.Signature, err = self.Sign(revertString(b.Hash)) // sign another hash
		println("(GENERATE BLOCK) SIGN ANOTHER HASH NEIGHER THEN BLOCK HASH")
	case state.WrongBlockSignKey != nil:
		b.Signature, err = state.Sign(b.Hash) // wrong secret key
		println("(GENERATE BLOCK) SIGN WITH ANOTHER SECRET KEY")
	default:
		b.Signature, err = self.Sign(b.Hash)
	}

	return
}
