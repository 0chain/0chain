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
			mc.computeRoundRandomSeed(ctx, pr, mr)
		}
	}
}

func (mc *Chain) computeRoundRandomSeed(ctx context.Context, pr round.RoundI, r *Round) {
	mc.setRandomSeed(ctx, r, rand.New(rand.NewSource(pr.GetRandomSeed())).Int63())
}

func (mc *Chain) setRandomSeed(ctx context.Context, r *Round, seed int64) {
	mc.SetRandomSeed(r.Round, seed)
	mc.startNewRound(ctx, r)
}
