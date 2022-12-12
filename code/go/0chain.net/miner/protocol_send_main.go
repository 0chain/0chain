//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
)

/*SendVRFShare - send the round vrf share */
func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	mc.sendVRFShare(ctx, vrfs)
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block,
	bvt *block.BlockVerificationTicket) {
	mc.sendVerificationTicket(ctx, b, bvt)
}

// SendBlock - send the block proposal to the network.
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	mc.sendBlock(ctx, b)
}
