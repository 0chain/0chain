package miner

import (
	"context"
	"fmt"
	"math/rand"

	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

//AddVRFShare - implement the interface for the RoundRandomBeacon protocol
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	Logger.Info("DKG-X AddVRFShare")
	if mr.AddVRFShare(vrfs) {

		//mc.computeVRF(ctx, mr)
		mc.ThresholdNumBLSSigReceived(ctx, mr)
		return true
	}
	return false
}

/*ThresholdNumBLSSigReceived do we've sufficient BLSshares? */
func (mc *Chain) ThresholdNumBLSSigReceived(ctx context.Context, mr *Round) {
	//([]string, []string, bool)
	Logger.Info("DKG-X ThresholdNumSigReceived")
	if mr.IsVRFComplete() {
		Logger.Error("DKG-X ThresholdNumSigReceived VRF is already completed why are we here?")
		return
	}

	shares := mr.GetVRFShares()
	if len(shares) >= GetBlsThreshold() {
		Logger.Info("DKG-X Hurray we've threshold BLS shares")
		recSig := make([]string, 0)
		recFrom := make([]string, 0)
		for _, share := range shares {
			n := share.GetParty()
			Logger.Info("DKG-X Printing from shares: ", zap.Int("Miner Index = ", n.SetIndex), zap.Any("Share = ", share.Share))
			/* The above log prints this. What is that "1" before the actual share
			DKG-X Printing from shares: 	{"Miner Index = ": 0, "Share = ": "1 1e12ddc9a8b63cc49798f5cfb69299abc9d7827658e926e88141be38f0c73f7a 12242a7993132a59610b4152c24670be0a0c7436a1a988c3a98bd0632c39740f"}
			*/
			sh := fmt.Sprint(share.Share) //quick and dirty casting for empty interface
			recSig = append(recSig, sh)
			recFrom = append(recFrom, ComputeBlsID(n.SetIndex))
		}
		rbOutput := CalcRandomBeacon(recSig, recFrom)
		Logger.Info("DKG-X ", zap.String("rboOutput = ", rbOutput), zap.Int64("Round #", mr.Number))

		//mc.setRandomSeed(ctx, mr, rbOutput)
		//Jay: ßReplace this with threshold computeRBO.
		mc.computeVRF(ctx, mr)
	}
}

//Jay: Check if K shares are received
func (mc *Chain) computeVRF(ctx context.Context, mr *Round) {
	Logger.Info("DKG-X computeVRF")
	if mr.IsVRFComplete() {
		Logger.Error("DKG-X computeVRF VRF is already completed why are we here?")
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
