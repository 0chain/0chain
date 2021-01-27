// +build integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"

	crpc "0chain.net/conductor/conductrpc"
	crpcutils "0chain.net/conductor/utils"
)

func getBadVRFS(vrfs *round.VRFShare) (bad *round.VRFShare) {
	bad = new(round.VRFShare)
	*bad = *vrfs
	bad.Share = revertString(bad.Share) // bad share
	return
}

func withTimeout(vrfs *round.VRFShare, timeout int) (bad *round.VRFShare) {
	bad = new(round.VRFShare)
	*bad = *vrfs
	bad.RoundTimeoutCount = timeout
	return
}

func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	var (
		mb        = mc.GetMagicBlock(vrfs.Round)
		state     = crpc.Client().State()
		badVRFS   *round.VRFShare
		good, bad []*node.Node
	)

	// not possible to send bad VRFS and bad round timeout at the same time
	switch {
	case state.VRFS != nil:
		badVRFS = getBadVRFS(vrfs)
		good, bad = crpcutils.Split(state, state.VRFS, mb.Miners.CopyNodes())
	case state.RoundTimeout != nil:
		badVRFS = withTimeout(vrfs, vrfs.RoundTimeoutCount+1) // just increase
		good, bad = crpcutils.Split(state, state.RoundTimeout,
			mb.Miners.CopyNodes())
	default:
		good = mb.Miners.CopyNodes() // all good
	}

	if len(good) > 0 {
		mb.Miners.SendToMultipleNodes(RoundVRFSender(vrfs), good)
	}
	if len(bad) > 0 {
		mb.Miners.SendToMultipleNodes(RoundVRFSender(badVRFS), bad)
	}
}

func getBadBVTHash(ctx context.Context, b *block.Block) (
	bad *block.BlockVerificationTicket) {

	bad = new(block.BlockVerificationTicket)
	bad.BlockID = b.Hash
	bad.Round = b.Round
	var (
		self = node.Self
		err  error
	)
	bad.VerifierID = self.Underlying().GetKey()
	bad.Signature, err = self.Sign(revertString(b.Hash)) // wrong hash
	if err != nil {
		panic(err)
	}
	return
}

func getBadBVTKey(ctx context.Context, b *block.Block) (
	bad *block.BlockVerificationTicket) {

	bad = new(block.BlockVerificationTicket)
	bad.BlockID = b.Hash
	bad.Round = b.Round
	var (
		selfNodeKey = node.Self.Underlying().GetKey()
		err         error
	)
	bad.VerifierID = selfNodeKey
	bad.Signature, err = crpcutils.Sign(b.Hash) // wrong private key
	if err != nil {
		panic(err)
	}
	return
}

// SendVerificationTicket - send the block verification ticket
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block,
	bvt *block.BlockVerificationTicket) {

	var (
		mb          = mc.GetMagicBlock(b.Round)
		state       = crpc.Client().State()
		selfNodeKey = node.Self.Underlying().GetKey()

		good, bad []*node.Node
	)

	if mc.VerificationTicketsTo == chain.Generator && b.MinerID != selfNodeKey {
		switch {
		case state.WrongVerificationTicketHash != nil:
			// (wrong hash)
			if state.WrongVerificationTicketHash.IsGood(state, b.MinerID) {
				mb.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
			} else if state.WrongVerificationTicketHash.IsBad(state, b.MinerID) {
				var badvt = getBadBVTHash(ctx, b)
				mb.Miners.SendTo(VerificationTicketSender(badvt), b.MinerID)
			}
		case state.WrongVerificationTicketKey != nil:
			// (wrong secret key)
			if state.WrongVerificationTicketKey.IsGood(state, b.MinerID) {
				mb.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
			} else if state.WrongVerificationTicketKey.IsBad(state, b.MinerID) {
				var badvt = getBadBVTKey(ctx, b)
				mb.Miners.SendTo(VerificationTicketSender(badvt), b.MinerID)
			}
		default:
			// (usual sending)
			mb.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
		}
		return
	}

	var badvt *block.BlockVerificationTicket

	switch {
	case state.WrongVerificationTicketHash != nil:
		// (wrong hash)
		badvt = getBadBVTHash(ctx, b)
		good, bad = crpcutils.Split(state, state.WrongVerificationTicketHash,
			mb.Miners.CopyNodes())
	case state.WrongVerificationTicketKey != nil:
		// (wrong secret key)
		badvt = getBadBVTKey(ctx, b)
		good, bad = crpcutils.Split(state, state.WrongVerificationTicketKey,
			mb.Miners.CopyNodes())
	default:
	}

	if badvt == nil {
		mb.Miners.SendAll(VerificationTicketSender(bvt)) // (usual sending)
		return
	}

	if len(good) > 0 {
		mb.Miners.SendToMultipleNodes(VerificationTicketSender(bvt), good)
	}
	if len(bad) > 0 {
		mb.Miners.SendToMultipleNodes(VerificationTicketSender(badvt), bad)
	}
}
