package miner

import (
	"context"
	"math/rand"

	"0chain.net/round"
)

//AddVRFShare - implement the interface for the RoundRandomBeacon protocol
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	if mr.AddVRFShare(vrfs) {
		mc.computeVRF(ctx, mr)
		return true
	}
	return false
}

func (mc *Chain) computeVRF(ctx context.Context, mr *Round) {
	if mr.IsVRFComplete() {
		return
	}
	shares := mr.GetVRFShares()
	if len(shares)*3 >= 2*mc.Miners.Size() {
		pr := mc.GetRound(mr.GetRoundNumber() - 1)
		if pr != nil {
			mc.ComputeRoundRandomSeed(pr, mr.Round)
			mc.startNewRound(ctx, mr)
		}
	}
}

/*ComputeRoundRandomSeed - compute the random seed for the round */
func (mc *Chain) ComputeRoundRandomSeed(pr round.RoundI, r *round.Round) {
	mc.SetRandomSeed(r, rand.New(rand.NewSource(pr.GetRandomSeed())).Int63())
}
