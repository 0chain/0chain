package miner

import (
	"context"
	"fmt"
	"math"
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
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ////////////  BLS-DKG Related stuff  /////////////////////

var (
	currRound int64

	roundMap = make(map[int64]map[int]string)

	//IsDkgDone an indicator for BC to continue with block generation
	IsDkgDone = false
	selfInd   int

	vrfTimer metrics.Timer // VRF gen-to-sync timer
)

func init() {
	vrfTimer = metrics.GetOrRegisterTimer("vrf_time", nil)
}

// SetDKG - starts the DKG process
func SetDKG(ctx context.Context, mb *block.MagicBlock, msk []string, mpks map[bls.PartyID][]bls.PublicKey) {
	mc := GetMinerChain()
	n := mc.Miners.Size()
	thresholdByCount := viper.GetInt("server_chain.block.consensus.threshold_by_count")
	t := int(math.Ceil((float64(thresholdByCount) / 100) * float64(n)))
	Logger.Info("DKG Setup", zap.Int("T", t), zap.Int("N", n), zap.Bool("DKG Enabled", config.DevConfiguration.IsDkgEnabled))
	self := node.GetSelfNode(ctx)
	selfInd = self.SetIndex
	if config.DevConfiguration.IsDkgEnabled {
		dkgSummary, err := getDKGSummaryFromStore(ctx)
		if dkgSummary.SecretKeyGroupStr != "" {
			Logger.Info("got dkg share from db", zap.Any("dummary", dkgSummary))
			mc.CurrentDKG = bls.MakeDKG(t, n, self.ID)
			mc.CurrentDKG.Si.SetHexString(dkgSummary.SecretKeyGroupStr)
			mc.CurrentDKG.Pi = mc.CurrentDKG.Si.GetPublicKey()
			mc.CurrentDKG.AggregatePublicKeyShares(mb.Mpks.GetMpkMap())
			IsDkgDone = true
			return
		} else {
			Logger.Info("err : reading dkg from db", zap.Error(err))
		}
		shares := make(map[string]string)
		id := node.Self.ID
		for k, v := range mb.ShareOrSigns.Shares {
			shares[k] = v.ShareOrSigns[id].Share
		}
		mc.CurrentDKG = bls.SetDKG(t, n, shares, msk, mpks, id)
		Logger.Info("DKG is finished. So, starting protocol")
		go startProtocol()
		return
	} else {
		Logger.Info("DKG is not enabled. So, starting protocol")
		IsDkgDone = true
		go startProtocol()
	}

}

func getDKGSummaryFromStore(ctx context.Context) (*bls.DKGSummary, error) {
	dkgSummary := datastore.GetEntity("dkgsummary").(*bls.DKGSummary)
	dkgSummaryMetadata := dkgSummary.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, dkgSummaryMetadata)
	defer ememorystore.Close(dctx)
	err := dkgSummary.Read(dctx, dkgSummary.GetKey())
	return dkgSummary, err
}

func storeDKGSummary(ctx context.Context, dkgSummary *bls.DKGSummary) error {
	dkgSummaryMetadata := dkgSummary.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, dkgSummaryMetadata)
	defer ememorystore.Close(dctx)
	err := dkgSummary.Write(dctx)
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
	return mc.CurrentDKG.T
}

/*ComputeBlsID Handy API to get the ID used in the library */
func ComputeBlsID(key string) string {
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetHexString()
}

func (mc *Chain) GetBlsMessageForRound(r *round.Round) (string, error) {

	prevRoundNumber := r.GetRoundNumber() - 1
	prevRSeed := "0"
	if prevRoundNumber == 0 {

		Logger.Debug("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		prevRSeed = encryption.Hash("0chain")
	} else {
		pr := mc.GetMinerRound(prevRoundNumber)
		if pr == nil {
			//This should never happen
			Logger.Error("Bls sign vrfshare: could not find round object for non-zero round", zap.Int64("PrevRoundNum", prevRoundNumber))
			return "", common.NewError("no_prev_round", "Could not find the previous round")
		}

		if pr.RandomSeed == 0 {
			Logger.Error("Bls sign vrfshare: error in getting prevRSeed")
			return "", common.NewError("prev_round_rrs_zero", fmt.Sprintf("Prev round %d has randomseed of 0", pr.GetRoundNumber()))
		}
		prevRSeed = strconv.FormatInt(pr.RandomSeed, 16) //pr.VRFOutput
	}
	blsMsg := fmt.Sprintf("%v%v%v", r.GetRoundNumber(), r.GetTimeoutCount(), prevRSeed)

	Logger.Info("Bls sign vrfshare calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.Int("roundtimeout", r.GetTimeoutCount()),
		zap.String("prevRSeed", prevRSeed), zap.Any("bls_msg", blsMsg))

	return blsMsg, nil
}

// GetBlsShare - Start the BLS process
func (mc *Chain) GetBlsShare(ctx context.Context, r, pr *round.Round) (string, error) {
	r.VrfStartTime = time.Now()
	if !config.DevConfiguration.IsDkgEnabled {
		Logger.Debug("returning standard string as DKG is not enabled.")
		return encryption.Hash("0chain"), nil
	}
	Logger.Info("DKG getBlsShare ", zap.Int64("Round Number", r.Number), zap.Any("next_view_change", mc.NextViewChange))
	msg, err := mc.GetBlsMessageForRound(r)
	if err != nil {
		return "", err
	}
	if mc.NextViewChange <= r.Number && ((mc.CurrentDKG != nil && mc.CurrentDKG.StartingRound < mc.NextViewChange) || mc.CurrentDKG == nil) {
		mc.ViewChange()
	}
	sigShare := mc.CurrentDKG.Sign(msg)
	Logger.Info("bls sign", zap.Any("message", msg), zap.Any("share", sigShare.GetHexString()), zap.Any("id", mc.CurrentDKG.ID.GetHexString()), zap.Any("si", mc.CurrentDKG.Si.GetHexString()), zap.Any("pi", mc.CurrentDKG.Pi.GetHexString()))
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
		return false
	}
	Logger.Info("checking view change", zap.Any("next_view_change", mc.NextViewChange), zap.Any("vrf_round", mr.Number))
	if mc.NextViewChange <= mr.Number && ((mc.CurrentDKG != nil && mc.CurrentDKG.StartingRound < mc.NextViewChange) || mc.CurrentDKG == nil) {
		mc.ViewChange()
	}
	var share bls.Sign
	share.SetHexString(vrfs.Share)
	partyID := bls.ComputeIDdkg(vrfs.GetParty().ID)
	if mc.CurrentDKG == nil {
		return false
	}
	if !mc.CurrentDKG.VerifySignature(&share, msg, partyID) {
		stringID := (&partyID).GetHexString()
		pi := mc.CurrentDKG.Gmpk[partyID]
		Logger.Error("failed to verify share", zap.Any("share", share.GetHexString()), zap.Any("message", msg), zap.Any("from", stringID), zap.Any("pi", pi.GetHexString()))
		return false
	} else {
		Logger.Info("verified vrf", zap.Any("share", share.GetHexString()), zap.Any("message", msg), zap.Any("from", (&partyID).GetHexString()))
	}
	if vrfs.GetRoundTimeoutCount() != mr.GetTimeoutCount() {
		//Keep VRF timeout and round timeout in sync. Same vrfs will comeback during soft timeouts
		Logger.Info("TOC_FIX VRF Timeout > round timeout", zap.Int("vrfs_timeout", vrfs.GetRoundTimeoutCount()), zap.Int("round_timeout", mr.GetTimeoutCount()))
		return false
	}
	if len(mr.GetVRFShares()) >= mc.GetBlsThreshold() {
		//ignore VRF shares coming after threshold is reached to avoid locking issues.
		//Todo: Remove this logging
		mr.AddAdditionalVRFShare(vrfs)
		Logger.Info("Ignoring VRFShare. Already at threshold", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("VRF_Shares", len(mr.GetVRFShares())))
		return false
	}
	if mr.AddVRFShare(vrfs, mc.GetBlsThreshold()) {
		mc.ThresholdNumBLSSigReceived(ctx, mr)
		return true
	} else {
		Logger.Info("Could not add VRFshare", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("Sender", vrfs.GetParty().SetIndex))
	}
	return false
}

/*ThresholdNumBLSSigReceived do we've sufficient BLSshares? */
func (mc *Chain) ThresholdNumBLSSigReceived(ctx context.Context, mr *Round) {

	if mr.IsVRFComplete() {
		//BLS has completed already for this round, But, received a BLS message from a node now
		Logger.Info("DKG ThresholdNumSigReceived VRF is already completed.", zap.Int64("round", mr.GetRoundNumber()))
		return
	}

	shares := mr.GetVRFShares()
	if len(shares) >= mc.GetBlsThreshold() {
		Logger.Debug("VRF Hurray we've threshold BLS shares")
		if !config.DevConfiguration.IsDkgEnabled {
			//We're still waiting for threshold number of VRF shares, even though DKG is not enabled.

			rbOutput := "" //rboutput will ignored anyway
			mc.computeRBO(ctx, mr, rbOutput)

			return
		}
		recSig, recFrom := getVRFShareInfo(mr)

		groupSignature, _ := mc.CurrentDKG.CalBlsGpSign(recSig, recFrom)
		rbOutput := encryption.Hash(groupSignature.GetHexString())
		Logger.Info("recieve bls sign", zap.Any("sigs", recSig), zap.Any("from", recFrom), zap.Any("group_signature", groupSignature.GetHexString()))

		// rbOutput := bs.CalcRandomBeacon(recSig, recFrom)
		Logger.Debug("VRF ", zap.String("rboOutput", rbOutput), zap.Int64("Round", mr.Number))
		mc.computeRBO(ctx, mr, rbOutput)
	} else {
		//TODO: remove this log
		Logger.Info("Not yet reached threshold", zap.Int("vrfShares_num", len(shares)), zap.Int("threshold", mc.GetBlsThreshold()))
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
	mr.Mutex.Lock()
	defer mr.Mutex.Unlock()

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
	if !r.VrfStartTime.IsZero() {
		vrfTimer.UpdateSince(r.VrfStartTime)
	} else {
		Logger.Info("VrfStartTime is zero", zap.Int64("round", r.GetRoundNumber()))
	}
	mc.startRound(ctx, r, seed)

}
