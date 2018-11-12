package miner

import (
	"context"
	"math/rand"

	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

//AddVRFShare - implement the interface for the RoundRandomBeacon protocol
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	if mr.AddVRFShare(vrfs) {

		mc.computeVRF(ctx, mr)
		return true
	}
	return false
}

//Jay: Check if K shares are received
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

//Jay: Note: This is where real RBO is calculated.
func (mc *Chain) computeRoundRandomSeed(ctx context.Context, pr round.RoundI, r *Round) {
	//TODO: once the actual VRF comes in, there is no need to rely on the prior value to compute the new value from the shares
	if mpr := pr.(*Round); mpr.IsVRFComplete() {
		mc.setRandomSeed(ctx, r, rand.New(rand.NewSource(pr.GetRandomSeed())).Int63())
	} else {
		Logger.Error("compute round random seed - no prior value", zap.Int64("round", r.GetRoundNumber()), zap.Int("blocks", len(pr.GetProposedBlocks())))
	}
}

func (mc *Chain) setRandomSeed(ctx context.Context, r *Round, seed int64) {
	mc.SetRandomSeed(r.Round, seed)
	mc.startNewRound(ctx, r)
}
