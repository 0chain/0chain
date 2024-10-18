package miner

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"

	"0chain.net/core/common"
	"0chain.net/core/encryption"

	"github.com/0chain/common/core/logging"
	. "github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	metrics "github.com/rcrowley/go-metrics"
)

// ////////////  BLS-DKG Related stuff  /////////////////////

var (
	roundMap = make(map[int64]map[int]string) //nolint

	vrfTimer metrics.Timer // VRF gen-to-sync timer
)

func init() {
	vrfTimer = metrics.GetOrRegisterTimer("vrf_time", nil)
}

// SetDKG - starts the DKG process
func SetDKG(ctx context.Context, mb *block.MagicBlock) error {
	mc := GetMinerChain()
	if !mc.ChainConfig.IsDkgEnabled() {
		Logger.Info("DKG is disabled. So, starting protocol")
		return nil
	}

	if err := mc.SetDKGSFromStore(ctx, mb); err != nil {
		return fmt.Errorf("error while setting dkg from store: %v\nstorage"+
			" may be damaged or permissions may not be available?",
			err.Error())
	}

	dkg := mc.GetDKGByStartingRound(mb.StartingRound)
	logging.Logger.Debug("[mvc] dkg process set dkg success",
		zap.Int("dkg T", dkg.T),
		zap.Int("dkg N", dkg.N),
		zap.Int("gmpk len", len(dkg.GetMPKs())),
		zap.Int64("mb number", mb.MagicBlockNumber),
		zap.Int64("mb sr", mb.StartingRound),
	)
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

	if mb.StartingRound > 0 && !summary.IsFinalized {
		return errors.New("DKG summary is not finalized")
	}

	if summary.SecretShares == nil {
		return common.NewError("failed to set dkg from store",
			"no saved shares for dkg")
	}

	var newDKG = bls.MakeDKG(mb.T, mb.N, selfNodeKey)
	newDKG.MagicBlockNumber = mb.MagicBlockNumber
	newDKG.StartingRound = mb.StartingRound

	if mb.Miners == nil {
		return common.NewError("failed to set dkg from store", "miners pool is not initialized in magic block")
	}

	logging.Logger.Debug("[mvc] dkg summary",
		zap.Int("secrets shares", len(summary.SecretShares)),
		zap.Int("miners num", len(mb.Miners.CopyNodesMap())),
		zap.Int("T", mb.T),
		zap.Int("N", mb.N),
		zap.Any("mb", mb),
		zap.Any("summary", summary))

	for k := range mb.Miners.CopyNodesMap() {
		logging.Logger.Debug("[mvc] set dkg key", zap.String("key", ComputeBlsID(k)))
		if savedShare, ok := summary.SecretShares[ComputeBlsID(k)]; ok {
			if err := newDKG.AddSecretShare(bls.ComputeIDdkg(k), savedShare, false); err != nil {
				logging.Logger.Error("[mvc] failed to add secret share",
					zap.Error(err), zap.String("share", savedShare))
				return err
			}
		} else if v, ok := mb.GetShareOrSigns().Get(k); ok {
			logging.Logger.Debug("[mvc] get key from mb", zap.String("key", ComputeBlsID(k)))
			if share, ok := v.ShareOrSigns[node.Self.Underlying().GetKey()]; ok && share.Share != "" {
				if err := newDKG.AddSecretShare(bls.ComputeIDdkg(k), share.Share, false); err != nil {
					logging.Logger.Debug("[mvc] failed to add secret share 2",
						zap.Error(err), zap.String("share", share.Share))
					return err
				}
			}
		}
	}

	if !newDKG.HasAllSecretShares() {
		return common.NewError("failed to set dkg from store",
			"not enough secret shares for dkg")
	}

	newDKG.AggregateSecretKeyShares()
	newDKG.Pi = newDKG.Si.GetPublicKey()
	mpks, err := mb.Mpks.GetMpkMap()
	if err != nil {
		return err
	}

	if err := newDKG.AggregatePublicKeyShares(mpks); err != nil {
		return err
	}

	if err = mc.SetDKG(newDKG); err != nil {
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
		blsMsg = fmt.Sprintf("%v%v%v", r.GetRoundNumber(), r.GetNormalizedTimeoutCount(), prrs)
	)

	Logger.Info("BLS sign VRF share calculated for ",
		zap.Int64("round", r.GetRoundNumber()),
		zap.Int("round_timeout", r.GetTimeoutCount()),
		zap.Int64("prev_rseed", pr.GetRandomSeed()),
		zap.String("prev round vrf random seed", prrs),
		zap.String("bls_msg", blsMsg))

	return blsMsg, nil
}

// GetBlsShare - Start the BLS process.
func (mc *Chain) GetBlsShare(ctx context.Context, r *round.Round) (string, error) {

	r.SetVrfStartTime(time.Now())
	if !mc.ChainConfig.IsDkgEnabled() {
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

	Logger.Debug("Sign msg with dkg",
		zap.String("msg", msg),
		zap.Int64("round", r.Number),
		zap.Int64("dkg starting round", dkg.StartingRound),
	)
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
	var (
		rn  = mr.GetRoundNumber()
		dkg = mc.GetDKG(rn)
	)

	if dkg == nil {
		Logger.Warn("AddVRFShare - dkg is nil", zap.Int64("round", rn))
		return false
	}

	var (
		vrfRTC  = vrfs.GetRoundTimeoutCount()
		roundTC = mr.GetTimeoutCount()
	)

	if vrfRTC != roundTC {
		//Keep VRF timeout and round timeout in sync. Same vrfs will comeback during soft timeouts
		Logger.Info("TOC_FIX VRF Timeout != round timeout",
			zap.Int("vrfs_timeout", vrfs.GetRoundTimeoutCount()),
			zap.Int("round_timeout", mr.GetTimeoutCount()),
			zap.Int64("round", mr.GetRoundNumber()),
			zap.Int64("vrf round", vrfs.Round))

		// add vrf to cache if the vrf share has timeout count > round's timeout count
		if vrfRTC > roundTC {
			mr.vrfSharesCache.add(vrfs)
		}
		return false
	}

	if mr.VRFShareExist(vrfs) {
		Logger.Debug("AddVRFShare - share already exist",
			zap.Int64("round", rn), zap.String("share", vrfs.Share))
		return false
	}

	blsThreshold := dkg.T
	if len(mr.GetVRFShares()) >= blsThreshold {
		Logger.Info("Ignoring VRFShare. Already at threshold",
			zap.Int64("Round", rn),
			zap.Int("VRF_Shares", len(mr.GetVRFShares())),
			zap.Int("bls_threshold", blsThreshold))
		return false
	}

	Logger.Info("DKG AddVRFShare",
		zap.Int64("round", rn),
		zap.Int("round_vrf_num", len(mr.GetVRFShares())),
		zap.Int("threshold", blsThreshold),
		zap.Int("sender", vrfs.GetParty().SetIndex),
		zap.Int("round_timeout_count", mr.GetTimeoutCount()),
		zap.Int("vrf_timeout_count", vrfs.GetRoundTimeoutCount()),
		zap.String("vrf_share", vrfs.Share))

	mr.AddTimeoutVote(vrfs.GetRoundTimeoutCount(), vrfs.GetParty().ID)
	msg, err := mc.GetBlsMessageForRound(mr.Round)
	if err != nil {
		// cache the vrf share if the previous round is not ready yet
		mr.vrfSharesCache.add(vrfs)

		Logger.Warn("failed to get bls message", zap.String("vrfs_share", vrfs.Share), zap.Any("round", mr.Round))
		return false
	}

	mc.verifyCachedVRFShares(ctx, msg, mr, dkg)

	if !verifyVRFShare(mr, vrfs, msg, dkg) {
		return false
	}

	//can't return false here even if at threshold, add_my_vrf_share might be at threshold and still should trigger starts
	mr.AddVRFShare(vrfs, blsThreshold)

	if mc.ThresholdNumBLSSigReceived(ctx, mr, blsThreshold) {
		mc.TryProposeBlock(common.GetRootContext(), mr)
		mc.StartVerification(common.GetRootContext(), mr)
	}

	return true
}

func verifyVRFShare(r *Round, vrfs *round.VRFShare, blsMsg string, dkg *bls.DKG) bool {
	var (
		partyID = bls.ComputeIDdkg(vrfs.GetParty().ID)
		share   bls.Sign
	)

	if err := share.SetHexString(vrfs.Share); err != nil {
		Logger.Error("failed to decode share hex string",
			zap.String("vrfs_share", vrfs.Share),
			zap.String("message", blsMsg))
		return false
	}

	if !dkg.VerifySignature(&share, blsMsg, partyID) {
		stringID := (&partyID).GetHexString()
		pi := dkg.GetPublicKeyByID(partyID)
		Logger.Error("failed to verify share",
			zap.String("share", share.GetHexString()),
			zap.String("message", blsMsg),
			zap.String("from", stringID),
			zap.String("pi", pi.GetHexString()),
			zap.String("node_id", vrfs.GetParty().GetKey()),
			zap.Int64("round", vrfs.Round),
			zap.Int64("dkg_starting_round", dkg.StartingRound),
			zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
			zap.Int("chain_tc", vrfs.GetRoundTimeoutCount()),
			zap.Int("round_tc", r.GetTimeoutCount()))
		return false
	}

	Logger.Info("verified vrf",
		zap.Int64("round", vrfs.Round),
		zap.String("node_id", vrfs.GetParty().GetKey()),
		zap.String("share", share.GetHexString()),
		zap.String("from", (&partyID).GetHexString()),
		zap.String("message", blsMsg))
	return true
}

func (mc *Chain) verifyCachedVRFShares(ctx context.Context, blsMsg string, r *Round, dkg *bls.DKG) {
	if err := mc.verifyCachedVRFSharesWorker.Run(ctx, func() error {
		var (
			vrfShares     = r.vrfSharesCache.getAll()
			blsThreshold  = dkg.T
			roundTC       = r.GetTimeoutCount()
			removeVRFKeys = make(map[string]struct{}, len(vrfShares))
		)

		if len(vrfShares) == 0 {
			return nil
		}

		defer r.vrfSharesCache.clean(removeVRFKeys)

		for _, vrfs := range vrfShares {
			if vrfs.GetRoundTimeoutCount() != roundTC {
				continue
			}

			if !verifyVRFShare(r, vrfs, blsMsg, dkg) {
				continue
			}

			r.AddVRFShare(vrfs, blsThreshold)
			removeVRFKeys[vrfs.GetParty().GetKey()] = struct{}{}
		}
		return nil
	}); err != nil {
		Logger.Error("verify cached vrf shares failed", zap.Error(err), zap.Int64("round", r.GetRoundNumber()))
	}
}

// ThresholdNumBLSSigReceived do we've sufficient BLSshares?
func (mc *Chain) ThresholdNumBLSSigReceived(ctx context.Context, mr *Round, blsThreshold int) bool {
	//since we checks RRS is set and set it in the same func this func should be executed atomically to avoid race conditions
	mr.vrfThresholdGuard.Lock()
	defer mr.vrfThresholdGuard.Unlock()
	//we can have threshold shares but still not have RRS
	if mr.IsVRFComplete() {
		// BLS has completed already for this round.
		// But, received a BLS message from a node now
		Logger.Info("DKG ThresholdNumSigReceived VRF is already completed.",
			zap.Int64("round", mr.GetRoundNumber()))
		return false
	}

	var shares = mr.GetVRFShares()
	if len(shares) < blsThreshold {
		//TODO: remove this log
		Logger.Info("Not yet reached threshold",
			zap.Int("vrfShares_num", len(shares)),
			zap.Int("threshold", blsThreshold),
			zap.Int64("round", mr.GetRoundNumber()))
		return false
	}

	Logger.Debug("VRF Hurray we've threshold BLS shares",
		zap.Int64("round", mr.GetRoundNumber()),
		zap.String("round pointer", fmt.Sprintf("%p", mr)))
	if !mc.ChainConfig.IsDkgEnabled() {
		// We're still waiting for threshold number of VRF shares,
		// even though DKG is not enabled.

		var rbOutput string // rboutput will ignored anyway
		if err := mc.computeRBO(ctx, mr, rbOutput); err != nil {
			Logger.Error("computeRBO failed",
				zap.Error(err),
				zap.Int64("round", mr.GetRoundNumber()))
			return false
		}

		return true
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
	Logger.Info("receive bls sign",
		zap.Int64("round", mr.GetRoundNumber()),
		zap.String("group_signature", groupSignature.GetHexString()),
		zap.String("rboOutput", rbOutput),
		zap.Int64("dkg_starting_round", dkg.StartingRound),
		zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
	)

	//mc.computeRBO(ctx, mr, rbOutput)
	if err := mc.computeRBO(ctx, mr, rbOutput); err != nil {
		Logger.Error("compute RBO failed", zap.Error(err))
		return false
	}

	return true
}

func (mc *Chain) computeRBO(ctx context.Context, mr *Round, rbo string) error {
	Logger.Debug("DKG computeRBO", zap.Int64("round", mr.GetRoundNumber()))
	if mr.IsVRFComplete() {
		Logger.Info("DKG computeRBO RBO is already completed")
		return nil
	}

	pr := mc.GetRound(mr.GetRoundNumber() - 1)
	return mc.computeRoundRandomSeed(ctx, pr, mr, rbo)
}

func getVRFShareInfo(mr *Round) ([]string, []string) {
	recSig := make([]string, 0)
	recFrom := make([]string, 0)

	shares := mr.GetVRFShares()
	for _, share := range shares {
		n := share.GetParty()
		recSig = append(recSig, share.Share)
		recFrom = append(recFrom, ComputeBlsID(n.ID))
	}

	return recSig, recFrom
}

func (mc *Chain) computeRoundRandomSeed(ctx context.Context, pr round.RoundI, r *Round, rbo string) error {

	var (
		seed   int64
		prSeed int64
	)
	if mc.ChainConfig.IsDkgEnabled() {
		useed, err := strconv.ParseUint(rbo[0:16], 16, 64)
		if err != nil {
			panic(err)
		}
		seed = int64(useed)
	} else {
		if pr != nil {
			prSeed = pr.GetRandomSeed()
			if mpr := pr.(*Round); mpr.IsVRFComplete() {
				seed = rand.New(rand.NewSource(prSeed)).Int63()
			}
		} else {
			return fmt.Errorf("pr is null")
		}
	}
	r.Round.SetVRFOutput(rbo)
	if pr != nil {
		//Todo: Remove this log later.
		Logger.Info("Starting round with vrf", zap.Int64("round", r.GetRoundNumber()),
			zap.Int("roundtimeout", r.GetTimeoutCount()),
			zap.Int64("rseed", seed), zap.Int64("prev_round", pr.GetRoundNumber()),
			zap.Int64("Prev_rseed", prSeed))
	}
	vrfStartTime := r.GetVrfStartTime()
	if !vrfStartTime.IsZero() {
		vrfTimer.UpdateSince(vrfStartTime)
	} else {
		Logger.Info("VrfStartTime is zero", zap.Int64("round", r.GetRoundNumber()))
	}
	if !mc.SetRandomSeed(r.Round, seed) {
		return fmt.Errorf("can't set round random seed for round")
	}
	return nil
}
