package miner

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
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
	mc := GetMinerChain()
	self := node.GetSelfNode(ctx)
	selfInd = self.Underlying().SetIndex
	if config.DevConfiguration.IsDkgEnabled {
		err := mc.SetDKGSFromStore(ctx, mb)
		if err != nil {
			return fmt.Errorf("error while setting dkg from store: %v\nstorage may be damaged or permissions may not be available?", err.Error())
		}
	} else {
		Logger.Info("DKG is not enabled. So, starting protocol")
	}
	return nil
}

func (mc *Chain) SetDKGSFromStore(ctx context.Context, mb *block.MagicBlock) (err2 error) {
	self := node.GetSelfNode(ctx)
	dkgSummary, err := GetDKGSummaryFromStore(ctx, strconv.FormatInt(mb.MagicBlockNumber, 10))
	if err != nil {
		return err
	}
	if dkgSummary.SecretShares == nil {
		return common.NewError("failed to set dkg from store", "no saved shares for dkg")
	}

	newDKG := bls.MakeDKG(mb.T, mb.N, self.Underlying().GetKey())
	newDKG.MagicBlockNumber = mb.MagicBlockNumber
	newDKG.StartingRound = mb.StartingRound
	for k := range mb.Miners.CopyNodesMap() {
		if savedShare, ok := dkgSummary.SecretShares[ComputeBlsID(k)]; ok {
			newDKG.AddSecretShare(bls.ComputeIDdkg(k), savedShare, false)
		} else if v, ok := mb.GetShareOrSigns().Get(k); ok {
			if share, ok := v.ShareOrSigns[node.Self.Underlying().GetKey()]; ok && share.Share != "" {
				newDKG.AddSecretShare(bls.ComputeIDdkg(k), share.Share, false)
			}
		}
	}
	if !newDKG.HasAllSecretShares() {
		return common.NewError("failed to set dkg from store", "not enough secret shares for dkg")
	}
	newDKG.AggregateSecretKeyShares()
	newDKG.Pi = newDKG.Si.GetPublicKey()
	newDKG.AggregatePublicKeyShares(mb.Mpks.GetMpkMap())

	mc.muDKG.Lock()
	mc.currentDKG = newDKG
	mc.muDKG.Unlock()
	return nil
}

func GetDKGSummaryFromStore(ctx context.Context, id string) (*bls.DKGSummary, error) {
	dkgSummary := datastore.GetEntity("dkgsummary").(*bls.DKGSummary)
	dkgSummary.ID = id
	dkgSummaryMetadata := dkgSummary.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, dkgSummaryMetadata)
	defer ememorystore.Close(dctx)
	err := dkgSummary.Read(dctx, dkgSummary.GetKey())
	return dkgSummary, err
}

func StoreDKGSummary(ctx context.Context, dkgSummary *bls.DKGSummary) (err error) {
	dkgSummaryMetadata := dkgSummary.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, dkgSummaryMetadata)
	defer ememorystore.Close(dctx)
	err = dkgSummary.Write(dctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(dctx, dkgSummaryMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}

// VerifySigShares - Verify the bls sig share is correct
func VerifySigShares() bool {
	//TBD
	return true
}

/*GetBlsThreshold Handy api for now. move this to protocol_vrf */
func (mc *Chain) GetBlsThreshold() int {
	mc.muDKG.Lock()
	defer mc.muDKG.Unlock()
	return mc.currentDKG.T
}

/*ComputeBlsID Handy API to get the ID used in the library */
func ComputeBlsID(key string) string {
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetHexString()
}

func (mc *Chain) GetBlsMessageForRound(r *round.Round) (string, error) {

	prevRoundNumber := r.GetRoundNumber() - 1

	pr := mc.GetMinerRound(prevRoundNumber)
	if pr == nil {
		//This should never happen
		Logger.Error("Bls sign vrfshare: could not find round object for non-zero round", zap.Int64("PrevRoundNum", prevRoundNumber))
		return "", common.NewError("no_prev_round", "Could not find the previous round")
	}

	if pr.GetRandomSeed() == 0 {
		Logger.Error("Bls sign vrfshare: error in getting prevRSeed")
		return "", common.NewError("prev_round_rrs_zero", fmt.Sprintf("Prev round %d has randomseed of 0", pr.GetRoundNumber()))
	}
	prevRSeed := strconv.FormatInt(pr.GetRandomSeed(), 16) //pr.VRFOutput

	blsMsg := fmt.Sprintf("%v%v%v", r.GetRoundNumber(), r.GetTimeoutCount(), prevRSeed)

	Logger.Info("Bls sign vrfshare calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.Int("roundtimeout", r.GetTimeoutCount()),
		zap.String("prevRSeed", prevRSeed), zap.Any("bls_msg", blsMsg))

	return blsMsg, nil
}

// GetBlsShare - Start the BLS process
func (mc *Chain) GetBlsShare(ctx context.Context, r *round.Round) (string, error) {
	r.SetVrfStartTime(time.Now())
	if !config.DevConfiguration.IsDkgEnabled {
		Logger.Debug("returning standard string as DKG is not enabled.")
		return encryption.Hash("0chain"), nil
	}
	Logger.Info("DKG getBlsShare ", zap.Int64("Round Number", r.Number), zap.Any("next_view_change", mc.nextViewChange))
	msg, err := mc.GetBlsMessageForRound(r)
	if err != nil {
		return "", err
	}
	mc.ViewChange(ctx, r.Number)
	mc.muDKG.Lock()
	defer mc.muDKG.Unlock()
	currentDKG := mc.currentDKG
	if currentDKG == nil {
		return "", common.NewError("get_bls_share", "DKG nil")
	}
	sigShare := currentDKG.Sign(msg)
	return sigShare.GetHexString(), nil
}

//  ///////////  End fo BLS-DKG Related Stuff   ////////////////

//AddVRFShare - implement the interface for the RoundRandomBeacon protocol
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	Logger.Info("DKG AddVRFShare", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("RoundTimeoutCount", mr.GetTimeoutCount()),
		zap.Int("Sender", vrfs.GetParty().SetIndex), zap.Int("vrf_timeoutcount", vrfs.GetRoundTimeoutCount()),
		zap.String("vrf_share", vrfs.Share))

	mr.AddTimeoutVote(vrfs.GetRoundTimeoutCount(), vrfs.GetParty().ID)
	msg, err := mc.GetBlsMessageForRound(mr.Round)
	if err != nil {
		Logger.Warn("failed to get bls message", zap.Any("vrfs_share", vrfs.Share), zap.Any("round", mr.Round))
		return false
	}
	mc.ViewChange(ctx, mr.Number)

	var share bls.Sign
	if err := share.SetHexString(vrfs.Share); err != nil {
		Logger.Error("failed to set hex share", zap.Any("vrfs_share", vrfs.Share), zap.Any("message", msg))
		return false
	}
	partyID := bls.ComputeIDdkg(vrfs.GetParty().ID)

	mc.muDKG.Lock()
	if mc.currentDKG == nil {
		mc.muDKG.Unlock()
		return false
	}
	blsThreshold := mc.currentDKG.T

	if !mc.currentDKG.VerifySignature(&share, msg, partyID) {
		stringID := (&partyID).GetHexString()
		pi := mc.currentDKG.GetPublicKeyByID(partyID)
		Logger.Error("failed to verify share", zap.Any("share", share.GetHexString()), zap.Any("message", msg), zap.Any("from", stringID), zap.Any("pi", pi.GetHexString()))
		mc.muDKG.Unlock()
		return false
	} else {
		Logger.Info("verified vrf", zap.Any("share", share.GetHexString()), zap.Any("message", msg), zap.Any("from", (&partyID).GetHexString()))
	}
	mc.muDKG.Unlock()
	if vrfs.GetRoundTimeoutCount() != mr.GetTimeoutCount() {
		//Keep VRF timeout and round timeout in sync. Same vrfs will comeback during soft timeouts
		Logger.Info("TOC_FIX VRF Timeout > round timeout", zap.Int("vrfs_timeout", vrfs.GetRoundTimeoutCount()), zap.Int("round_timeout", mr.GetTimeoutCount()))
		return false
	}
	if len(mr.GetVRFShares()) >= blsThreshold {
		//ignore VRF shares coming after threshold is reached to avoid locking issues.
		//Todo: Remove this logging
		mr.AddAdditionalVRFShare(vrfs)
		mc.ThresholdNumBLSSigReceived(ctx, mr, blsThreshold)
		Logger.Info("Ignoring VRFShare. Already at threshold",
			zap.Int64("Round", mr.GetRoundNumber()),
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

/*ThresholdNumBLSSigReceived do we've sufficient BLSshares? */
func (mc *Chain) ThresholdNumBLSSigReceived(ctx context.Context, mr *Round, blsThreshold int) {

	if mr.IsVRFComplete() {
		//BLS has completed already for this round, But, received a BLS message from a node now
		Logger.Info("DKG ThresholdNumSigReceived VRF is already completed.", zap.Int64("round", mr.GetRoundNumber()))
		return
	}

	shares := mr.GetVRFShares()
	if len(shares) >= blsThreshold {
		Logger.Debug("VRF Hurray we've threshold BLS shares")
		if !config.DevConfiguration.IsDkgEnabled {
			//We're still waiting for threshold number of VRF shares, even though DKG is not enabled.

			rbOutput := "" //rboutput will ignored anyway
			mc.computeRBO(ctx, mr, rbOutput)

			return
		}
		recSig, recFrom := getVRFShareInfo(mr)
		mc.muDKG.Lock()
		groupSignature, err := mc.currentDKG.CalBlsGpSign(recSig, recFrom)
		mc.muDKG.Unlock()
		if err != nil {
			Logger.Error("calculates the Gp Sign", zap.Error(err))
		}
		rbOutput := encryption.Hash(groupSignature.GetHexString())
		Logger.Info("recieve bls sign", zap.Any("sigs", recSig), zap.Any("from", recFrom), zap.Any("group_signature", groupSignature.GetHexString()))

		// rbOutput := bs.CalcRandomBeacon(recSig, recFrom)
		Logger.Debug("VRF ", zap.String("rboOutput", rbOutput), zap.Int64("Round", mr.Number))
		mc.computeRBO(ctx, mr, rbOutput)
	} else {
		//TODO: remove this log
		Logger.Info("Not yet reached threshold", zap.Int("vrfShares_num", len(shares)), zap.Int("threshold", blsThreshold))
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
