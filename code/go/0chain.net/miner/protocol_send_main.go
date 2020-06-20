// +build !integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
)

/*SendVRFShare - send the round vrf share */
func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	mb := mc.GetMagicBlock(vrfs.Round)
	m2m := mb.Miners
	m2m.SendAll(RoundVRFSender(vrfs))
}
