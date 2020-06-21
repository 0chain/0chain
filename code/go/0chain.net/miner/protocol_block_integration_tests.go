// +build integration_tests

//
// TEMPORARY: REGUULAR BEHAVIOUR
//

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var self = node.GetSelfNode(ctx)
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)
	return
}

/*

package miner

import (
	"context"
	"errors"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"

	"0chain.net/core/encryption"

	crpc "0chain.net/conductor/conductrpc"
)

func signBadHashVT(b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	bvt = new(block.BlockVerificationTicket)
	bvt.BlockID = b.Hash
	bvt.Round = b.Round
	var self = node.GetSelfNode(ctx)
	bvt.VerifierID = self.Underlying().GetKey()
	bvt.Signature, err = self.Sign(revertString(b.Hash))
	b.SetVerificationStatus(block.VerificationSuccessful)
	if err != nil {
		return nil, err
	}
	return
}

func signBadKeyVT(b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	bvt = new(block.BlockVerificationTicket)
	bvt.BlockID = b.Hash
	bvt.Round = b.Round
	var self = node.GetSelfNode(ctx)
	bvt.VerifierID = self.Underlying().GetKey()

	var ss = encryption.NewBLS0ChainScheme()
	ss.GenerateKeys()

	bvt.Signature, err = ss.Sign(b.Hash)
	b.SetVerificationStatus(block.VerificationSuccessful)
	if err != nil {
		return nil, err
	}
	return
}

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	var state = crpc.Client().State()
	switch {
	case state.SignOnlyCompetingBlocks != nil:
		// if !state.SignOnlyCompetingBlocks.IsCompetingGroupMember(state, b.MinerID) {
		// 	println("SKIP BLOCK VT SIGNING (NOT FROM COMPETING GROUP)")
		// 	return nil, errors.New("skip block signing by integration tests")
		// }
		// println("SIGN BLOCK VT OF COMPETING BLOCK")
		// return mc.signBlock(ctx, b)
		println("SIGN ONLY COMPETING BLOCK VT (?)")
	case state.VerificationTicket != nil:
		println("SEND/DON'T SEND BLOCK VT (GOOD/BAD LISTS, BAD IS DON'T SEND)")
		// state.Split(state.VerificationTicket, nodes)
	case state.WrongVerificationTicketHash != nil:
		println("WRONG VT HASH")
	case state.WrongVerificationTicketKey != nil:
		println("WRONG VT SECRET KEY")
	default:
	}

	// regular signing
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
		// sign another hash
		b.Signature, err = self.Sign(revertString(b.Hash))
		println("(GENERATE BLOCK) SIGN ANOTHER HASH NEIGHER THEN BLOCK HASH")
	case state.WrongBlockSignKey != nil:
		var another = encryption.NewBLS0ChainScheme()
		if err = another.GenerateKeys(); err != nil {
			panic(err)
		}
		b.Signature, err = another.Sign(b.Hash)
		println("(GENERATE BLOCK) SIGN WITH ANOTHER SECRET KEY")
	default:
		b.Signature, err = self.Sign(b.Hash)
	}

	return
}

*/
