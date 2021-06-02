// +build !integration_tests

package miner

import (
	"context"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/block"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/chain"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/node"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/round"
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
