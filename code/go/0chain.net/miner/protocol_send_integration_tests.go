// +build integration_tests

//
// TEMPORARY: REGULAR BEHAVIOUR
//

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
)

/*SendVRFShare - send the round vrf share */
func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	mb := mc.GetMagicBlock(vrfs.Round)
	m2m := mb.Miners
	m2m.SendAll(RoundVRFSender(vrfs))
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block,
	bvt *block.BlockVerificationTicket) {

	var (
		mb  = mc.GetMagicBlock(b.Round)
		m2m = mb.Miners
	)

	if mc.VerificationTicketsTo == chain.Generator &&
		b.MinerID != node.Self.Underlying().GetKey() {

		m2m.SendTo(VerificationTicketSender(bvt), b.MinerID)
		return
	}

	m2m.SendAll(VerificationTicketSender(bvt))
}

/*

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"

	crpc "0chain.net/conductor/conductrpc"
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
		println("SEND GOOD/BAD VRFS")
		badVRFS = getBadVRFS(vrfs)
		good, bad = state.Split(state.VRFS, mb.Miners)
	case state.RoundTimeout != nil:
		println("SEND BAD ROUND TIMEOUT COUNT")
		badVRFS = withTimeout(vrfs, vrfs.RoundTimeoutCount+1) // just increase
		good, bad = state.Split(state.RoundTimeout, mb.Miners)
	default:
		good = mb.Miners // all good
	}

	if len(good) > 0 {
		m2m.SendToMultipleNodes(RoundVRFSender(vrfs), good)
	}
	if len(bad) > 0 {
		m2m.SendToMultipleNodes(RoundVRFSender(badVRFS), bad)
	}
}

func getBadBVTHash(bvt *block.VerificationTicket) (
	bad *block.VerificationTicket) {

	bad = bvt.Copy()
	//
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
		case state.VerificationTicket != nil:
			// send good/bad
			var send = state.VerificationTicket.IsGood(state, b.MinerID) ||
				state.VerificationTicket.IsBad(state, b.MinerID)
			if send {
				mb.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
			}
		case state.WrongVerificationTicketHash != nil:
			// send bad (wrong hash)
			var badvt = nil
			state.Split(state.WrongVerificationTicketHash, nodes)
		case state.WrongVerificationTicketKey != nil:
			// send bad (wrong secret key)
		default:
			// usual sending
			mb.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
		}
		return
	}

	switch {
	case state.VerificationTicket != nil:
		// send good/bad
	case state.WrongVerificationTicketHash != nil:
		// send bad (wrong hash)
	case state.WrongVerificationTicketKey != nil:
		// send bad (wrong secret key)
	default:
		// usual sending
		mb.Miners.SendAll(VerificationTicketSender(bvt))
	}
}

*/
