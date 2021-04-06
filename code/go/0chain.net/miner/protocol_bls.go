package miner

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"

	"0chain.net/core/common"
	"0chain.net/core/encryption"

	. "0chain.net/core/logging"
	"go.uber.org/zap"

	metrics "github.com/rcrowley/go-metrics"
)

// ////////////  BLS-DKG Related stuff  /////////////////////

var (
	roundMap = make(map[int64]map[int]string)

	selfInd  int
	vrfTimer metrics.Timer // VRF gen-to-sync timer
)

func init() {
	vrfTimer = metrics.GetOrRegisterTimer("vrf_time", nil)
}

// SetDKG - starts the DKG process
func SetDKG(ctx context.Context, mb *block.MagicBlock) error {
	var (
		mc   = GetMinerChain()
		self = node.Self.Underlying()
	)
	selfInd = self.SetIndex
	if config.DevConfiguration.IsDkgEnabled {
		err := mc.SetDKGSFromStore(ctx, mb)
		if err != nil {
			return fmt.Errorf("error while setting dkg from store: %v\nstorage"+
				" may be damaged or permissions may not be available?",
				err.Error())
		}
	} else {
		Logger.Info("DKG is not enabled. So, starting protocol")
	}
	return nil
}

// SetDKGFromMagicBlocksChainPrev sets DKG for all MB from the specified block
func SetDKGFromMagicBlocksChainPrev(ctx context.Context, mb *block.MagicBlock) error {
	mc := GetMinerChain()
	if err := SetDKG(ctx, mb); err != nil {
		return err
	}
	prevMB := mc.GetPrevMagicBlockFromMB(mb)
	for prevMB != nil && prevMB != mb && prevMB.StartingRound != 0 {
		if err := SetDKG(ctx, prevMB); err != nil {
			Logger.Error("failed to set DKG", zap.Error(err),
				zap.Int64("sr", prevMB.StartingRound))
		}
		prevMB = mc.GetPrevMagicBlockFromMB(prevMB)
	}
	return nil
}

func (mc *Chain) SetDKGSFromStore(ctx context.Context, mb *block.MagicBlock) (
	err error) {

	var (
		selfNodeKey = node.Self.Underlying().GetKey()
		id          = strconv.FormatInt(mb.MagicBlockNumber, 10)

		summary *bls.DKGSummary
	)

	if summary, err = LoadDKGSummary(ctx, id); err != nil {
		return
	}

	if summary.SecretShares == nil {
		return common.NewError("failed to set dkg from store",
			"no saved shares for dkg")
	}

	var newDKG = bls.MakeDKG(mb.T, mb.N, selfNodeKey)
	newDKG.MagicBlockNumber = mb.MagicBlockNumber
	newDKG.StartingRound = mb.StartingRound

	for k := range mb.Miners.CopyNodesMap() {
		if savedShare, ok := summary.SecretShares[ComputeBlsID(k)]; ok {
			newDKG.AddSecretShare(bls.ComputeIDdkg(k), savedShare, false)
		} else if v, ok := mb.GetShareOrSigns().Get(k); ok {
			if share, ok := v.ShareOrSigns[node.Self.Underlying().GetKey()]; ok && share.Share != "" {
				newDKG.AddSecretShare(bls.ComputeIDdkg(k), share.Share, false)
			}
		}
	}

	if !newDKG.HasAllSecretShares() {
		return common.NewError("failed to set dkg from store",
			"not enough secret shares for dkg")
	}

	newDKG.AggregateSecretKeyShares()
	newDKG.Pi = newDKG.Si.GetPublicKey()
	newDKG.AggregatePublicKeyShares(mb.Mpks.GetMpkMap())

	if err = mc.SetDKG(newDKG, mb.StartingRound); err != nil {
		Logger.Error("failed to set dkg", zap.Error(err))
		return // error
	}

	return // ok, set
}

// VerifySigShares - Verify the bls sig share is correct
func VerifySigShares() bool {
	//TBD
	return true
}

// GetBlsThreshold Handy api for now. move this to protocol_vrf.
func (mc *Chain) GetBlsThreshold(round int64) int {
	return mc.GetDKG(round).T
}

/*ComputeBlsID Handy API to get the ID used in the library */
func ComputeBlsID(key string) string {
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetHexString()
}

func (mc *Chain) GetBlsMessageForRound(r *round.Round) (string, error) {

	var (
		prn = r.GetRoundNumber() - 1
		pr  = mc.GetMinerRound(prn)
	)

	if pr == nil {
		Logger.Error("BLS sign VRFS share: could not find round"+
			" object for non-zero round",
			zap.Int64("PrevRoundNum", prn))
		return "", common.NewError("no_prev_round",
			"could not find the previous round")
	}

	if pr.GetRandomSeed() == 0 && pr.GetRoundNumber() > 0 {
		Logger.Error("BLS sign VRF share: error in getting prev. random seed",
			zap.Int64("prev_round", pr.Number),
			zap.Bool("prev_round_has_seed", pr.HasRandomSeed()))
		return "", common.NewErrorf("prev_round_rrs_zero",
			"prev. round %d random seed is 0", pr.GetRoundNumber())
	}

	var (
		prrs   = strconv.FormatInt(pr.GetRandomSeed(), 16) // pr.VRFOutput
		blsMsg = fmt.Sprintf("%v%v%v", r.GetRoundNumber(), r.GetTimeoutCount(), prrs)
	)

	Logger.Info("BLS sign VRF share calculated for ",
		zap.Int64("round", r.GetRoundNumber()),
		zap.Int("round_timeout", r.GetTimeoutCount()),
		zap.String("prev. round random seed", prrs),
		zap.Any("bls_msg", blsMsg))

	return blsMsg, nil
}

// GetBlsShare - Start the BLS process.
func (mc *Chain) GetBlsShare(ctx context.Context, r *round.Round) (string, error) {

	r.SetVrfStartTime(time.Now())
	if !config.DevConfiguration.IsDkgEnabled {
		Logger.Debug("returning standard string as DKG is not enabled.")
		return encryption.Hash("0chain"), nil
	}
	Logger.Info("DKG getBlsShare ", zap.Int64("Round Number", r.Number))
	msg, err := mc.GetBlsMessageForRound(r)
	if err != nil {
		return "", err
	}
	var dkg = mc.GetDKG(r.GetRoundNumber())
	if dkg == nil {
		return "", common.NewError("get_bls_share", "DKG is nil")
	}
	sigShare := dkg.Sign(msg)

	var (
		rn = r.GetRoundNumber()
		mb = mc.GetMagicBlock(rn)
	)

	Logger.Debug("get_bls_share", zap.Int64("round", rn),
		zap.Int("rtc", r.GetTimeoutCount()),
		zap.String("dkg_pi", dkg.Si.GetPublicKey().GetHexString()),
		zap.Int64("dkg_sr", dkg.StartingRound),
		zap.Int64("mb_sr", mb.StartingRound))

	return sigShare.GetHexString(), nil
}

//  ///////////  End fo BLS-DKG Related Stuff   ////////////////

// AddVRFShare - implement the interface for the RoundRandomBeacon protocol.
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	var rn = mr.GetRoundNumber()
	Logger.Info("DKG AddVRFShare", zap.Int64("Round", rn), zap.Int("RoundTimeoutCount", mr.GetTimeoutCount()),
		zap.Int("Sender", vrfs.GetParty().SetIndex), zap.Int("vrf_timeoutcount", vrfs.GetRoundTimeoutCount()),
		zap.String("vrf_share", vrfs.Share))

	mr.AddTimeoutVote(vrfs.GetRoundTimeoutCount(), vrfs.GetParty().ID)
	msg, err := mc.GetBlsMessageForRound(mr.Round)
	if err != nil {
		Logger.Warn("failed to get bls message", zap.Any("vrfs_share", vrfs.Share), zap.Any("round", mr.Round))
		return false
	}
	var share bls.Sign
	if err := share.SetHexString(vrfs.Share); err != nil {
		Logger.Error("failed to set hex share", zap.Any("vrfs_share", vrfs.Share), zap.Any("message", msg))
		return false
	}

	var (
		partyID = bls.ComputeIDdkg(vrfs.GetParty().ID)
		dkg     = mc.GetDKG(rn)
	)
	if dkg == nil {
		return false
	}
	blsThreshold := dkg.T

	if !dkg.VerifySignature(&share, msg, partyID) {
		var prSeed string
		if pr := mc.GetMinerRound(rn - 1); pr != nil {
			prSeed = strconv.FormatInt(pr.GetRandomSeed(), 16)
		}
		stringID := (&partyID).GetHexString()
		pi := dkg.GetPublicKeyByID(partyID)
		Logger.Error("failed to verify share",
			zap.Any("share", share.GetHexString()),
			zap.Any("message", msg),
			zap.Any("from", stringID),
			zap.Any("pi", pi.GetHexString()),
			zap.String("node_id", vrfs.GetParty().GetKey()),
			zap.Int64("round", vrfs.Round),
			zap.Int64("dkg_starting_round", dkg.StartingRound),
			zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
			zap.Int("chain_tc", vrfs.GetRoundTimeoutCount()),
			zap.Int("round_tc", mr.GetTimeoutCount()),
			zap.String("pr_seed", prSeed),
		)
		return false
	} else {
		Logger.Info("verified vrf",
			zap.Any("share", share.GetHexString()),
			zap.Any("message", msg),
			zap.Any("from", (&partyID).GetHexString()),
			zap.String("node_id", vrfs.GetParty().GetKey()),
			zap.Int64("round", vrfs.Round))
	}
	if vrfs.GetRoundTimeoutCount() != mr.GetTimeoutCount() {
		//Keep VRF timeout and round timeout in sync. Same vrfs will comeback during soft timeouts
		Logger.Info("TOC_FIX VRF Timeout > round timeout",
			zap.Int("vrfs_timeout", vrfs.GetRoundTimeoutCount()),
			zap.Int("round_timeout", mr.GetTimeoutCount()))
		return false
	}
	if len(mr.GetVRFShares()) >= blsThreshold {
		//ignore VRF shares coming after threshold is reached to avoid locking issues.
		//Todo: Remove this logging
		mr.AddAdditionalVRFShare(vrfs)
		mc.ThresholdNumBLSSigReceived(ctx, mr, blsThreshold)
		Logger.Info("Ignoring VRFShare. Already at threshold",
			zap.Int64("Round", rn),
			zap.Int("VRF_Shares", len(mr.GetVRFShares())),
			zap.Int("bls_threshold", blsThreshold))
		return false
	}
	if mr.AddVRFShare(vrfs, blsThreshold) {
		mc.ThresholdNumBLSSigReceived(ctx, mr, blsThreshold)
		return true
	} else {
		Logger.Info("Could not add VRFshare", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("Sender", vrfs.GetParty().SetIndex))
	}
	return false
}

// ThresholdNumBLSSigReceived do we've sufficient BLSshares?
func (mc *Chain) ThresholdNumBLSSigReceived(ctx context.Context, mr *Round, blsThreshold int) {

	if mr.IsVRFComplete() {
		// BLS has completed already for this round.
		// But, received a BLS message from a node now
		Logger.Info("DKG ThresholdNumSigReceived VRF is already completed.",
			zap.Int64("round", mr.GetRoundNumber()))
		return
	}

	var shares = mr.GetVRFShares()
	if len(shares) >= blsThreshold {
		Logger.Debug("VRF Hurray we've threshold BLS shares")
		if !config.DevConfiguration.IsDkgEnabled {
			// We're still waiting for threshold number of VRF shares,
			// even though DKG is not enabled.

			var rbOutput string // rboutput will ignored anyway
			mc.computeRBO(ctx, mr, rbOutput)

			return
		}

		var (
			recSig, recFrom     = getVRFShareInfo(mr)
			dkg                 = mc.GetDKG(mr.GetRoundNumber())
			groupSignature, err = dkg.CalBlsGpSign(recSig, recFrom)
		)
		if err != nil {
			Logger.Error("calculates the Gp Sign", zap.Error(err))
		}

		var rbOutput = encryption.Hash(groupSignature.GetHexString())
		Logger.Info("recieve bls sign", zap.Any("sigs", recSig),
			zap.Any("from", recFrom),
			zap.Any("group_signature", groupSignature.GetHexString()))

		// rbOutput := bs.CalcRandomBeacon(recSig, recFrom)
		Logger.Debug("VRF ", zap.String("rboOutput", rbOutput), zap.Int64("Round", mr.Number))
		mc.computeRBO(ctx, mr, rbOutput)
	} else {
		//TODO: remove this log
		Logger.Info("Not yet reached threshold",
			zap.Int("vrfShares_num", len(shares)),
			zap.Int("threshold", blsThreshold),
			zap.Int64("round", mr.GetRoundNumber()))
	}
}

func (mc *Chain) computeRBO(ctx context.Context, mr *Round, rbo string) {
	Logger.Debug("DKG computeRBO")
	if mr.IsVRFComplete() {
		Logger.Info("DKG computeRBO RBO is already completed")
		return
	}

	pr := mc.GetRound(mr.GetRoundNumber() - 1)
	mc.computeRoundRandomSeed(ctx, pr, mr, rbo)

}

func getVRFShareInfo(mr *Round) ([]string, []string) {
	recSig := make([]string, 0)
	recFrom := make([]string, 0)

	shares := mr.GetVRFShares()
	for _, share := range shares {
		n := share.GetParty()
		Logger.Info("VRF Printing from shares: ", zap.Int("Miner Index = ", n.SetIndex), zap.Any("Share = ", share.Share))

		recSig = append(recSig, share.Share)
		recFrom = append(recFrom, ComputeBlsID(n.ID))
	}

	return recSig, recFrom
}

func (mc *Chain) computeRoundRandomSeed(ctx context.Context, pr round.RoundI, r *Round, rbo string) {

	var seed int64
	if config.DevConfiguration.IsDkgEnabled {
		useed, err := strconv.ParseUint(rbo[0:16], 16, 64)
		if err != nil {
			panic(err)
		}
		seed = int64(useed)
	} else {
		if pr != nil {
			if mpr := pr.(*Round); mpr.IsVRFComplete() {
				seed = rand.New(rand.NewSource(pr.GetRandomSeed())).Int63()
			}
		} else {
			Logger.Error("pr is null! Let go this round...")
			return
		}
	}
	r.Round.SetVRFOutput(rbo)
	if pr != nil {
		//Todo: Remove this log later.
		Logger.Info("Starting round with vrf", zap.Int64("round", r.GetRoundNumber()),
			zap.Int("roundtimeout", r.GetTimeoutCount()),
			zap.Int64("rseed", seed), zap.Int64("prev_round", pr.GetRoundNumber()),
			//zap.Int("Prev_roundtimeout", pr.GetTimeoutCount()),
			zap.Int64("Prev_rseed", pr.GetRandomSeed()))
	}
	vrfStartTime := r.GetVrfStartTime()
	if !vrfStartTime.IsZero() {
		vrfTimer.UpdateSince(vrfStartTime)
	} else {
		Logger.Info("VrfStartTime is zero", zap.Int64("round", r.GetRoundNumber()))
	}
	mc.startRound(ctx, r, seed)

}
