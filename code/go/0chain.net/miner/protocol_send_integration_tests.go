// +build integration_tests

package miner

import (
	"context"

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
	if state.VRFS != nil {
		badVRFS = getBadVRFS(vrfs)
		good, bad = state.Split(state.VRFS, mb.Miners)
	} else if state.RoundTimeout != nil {
		badVRFS = withTimeout(vrfs, vrfs.RoundTimeoutCount+1) // just increase
		good, bad = state.Split(state.RoundTimeout, mb.Miners)
	}

	m2m.SendToMultipleNodes(RoundVRFSender(vrfs), good)
	m2m.SendToMultipleNodes(RoundVRFSender(badVRFS), bad)
}
