//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

/*SendVRFShare - send the round vrf share */
func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	mc.sendVRFShare(ctx, vrfs)
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block,
	bvt *block.BlockVerificationTicket) {

	var (
		mb  = mc.GetMagicBlock(b.Round)
		m2m = mb.Miners
	)

	if mc.VerificationTicketsTo() == chain.Generator &&
		b.MinerID != node.Self.Underlying().GetKey() {

		if _, err := m2m.SendTo(ctx, VerificationTicketSender(bvt), b.MinerID); err != nil {
			logging.Logger.Error("send verification ticket failed", zap.Error(err))
		}
		return
	}

	m2m.SendAll(ctx, VerificationTicketSender(bvt))
}

// SendBlock - send the block proposal to the network.
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	mc.sendBlock(ctx, b)
}
