package miner

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ////////////  BLS-DKG Related stuff  /////////////////////

var dg bls.DKG
var bs bls.SimpleBLS
var recShares []string
var recSharesMap map[int]string
var minerShares map[string]bls.Key
var currRound int64

var roundMap = make(map[int64]map[int]string)

var isDkgEnabled bool
var k, n int

//IsDkgDone an indicator for BC to continue with block generation
var IsDkgDone = false
var selfInd int

var mutex = &sync.RWMutex{}

var vrfTimer metrics.Timer // VRF gen-to-sync timer

func init() {
	vrfTimer = metrics.GetOrRegisterTimer("vrf_time", nil)
}

// StartDKG - starts the DKG process
func StartDKG(ctx context.Context) {

	mc := GetMinerChain()

	m2m := mc.Miners

	isDkgEnabled = config.DevConfiguration.IsDkgEnabled
	thresholdByCount := viper.GetInt("server_chain.block.consensus.threshold_by_count")
	k = int(math.Ceil((float64(thresholdByCount) / 100) * float64(mc.Miners.Size())))
	n = mc.Miners.Size()

	Logger.Info("DKG Setup", zap.Int("K", k), zap.Int("N", n), zap.Bool("DKG Enabled", isDkgEnabled))

	self := node.GetSelfNode(ctx)
	selfInd = self.SetIndex

	if isDkgEnabled {
		dg = bls.MakeDKG(k, n)

		dkgSummary, err := getDKGSummaryFromStore(ctx)
		if dkgSummary.SecretKeyGroupStr != "" {
			dg.SecKeyShareGroup.SetHexString(dkgSummary.SecretKeyGroupStr)
			Logger.Info("got dkg share from db")
			waitForNetworkToBeReadyForBls(ctx)
			IsDkgDone = true
			go startProtocol()
			return
		} else {
			Logger.Info("err : reading dkg from db", zap.Error(err))
		}
		waitForNetworkToBeReadyForDKG(ctx)
		Logger.Info("Starting DKG...")

		minerShares = make(map[string]bls.Key, len(m2m.Nodes))

		for _, node := range m2m.Nodes {
			forID := bls.ComputeIDdkg(node.SetIndex)
			dg.ID = forID

			secShare, _ := dg.ComputeDKGKeyShare(forID)

			//Logger.Debug("ComputeDKGKeyShare ", zap.String("secShare", secShare.GetDecString()), zap.Int("miner index", node.SetIndex))
			minerShares[node.GetKey()] = secShare
			if self.SetIndex == node.SetIndex {
				recShares = append(recShares, secShare.GetDecString())
				addToRecSharesMap(self.SetIndex, secShare.GetDecString())
			}

		}
		WaitForDKGShares()
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

// WaitForDkgToBeDone is a blocking function waits till DKG process is done if dkg is enabled
func WaitForDkgToBeDone(ctx context.Context) {
	if isDkgEnabled {
		ticker := time.NewTicker(5 * chain.DELTA)
		defer ticker.Stop()

		for ts := range ticker.C {
			if IsDkgDone {
				Logger.Info("WaitForDkgToBeDone is over.")
				break
			} else {
				Logger.Info("Waiting for DKG process to be over.", zap.Time("ts", ts))
			}
		}
	}
}

func isNetworkReadyForDKG() bool {
	mc := GetMinerChain()
	if isDkgEnabled {
		return mc.AreAllNodesActive()
	} else {
		return mc.CanStartNetwork()
	}
}

func waitForNetworkToBeReadyForBls(ctx context.Context) {
	mc := GetMinerChain()

	if !mc.CanStartNetwork() {
		ticker := time.NewTicker(5 * chain.DELTA)
		for ts := range ticker.C {
			active := mc.Miners.GetActiveCount()
			Logger.Info("waiting for sufficient active nodes", zap.Time("ts", ts), zap.Int("have", active), zap.Int("need", k))
			if mc.CanStartNetwork() {
				break
			}
		}
	}
}

func waitForNetworkToBeReadyForDKG(ctx context.Context) {

	mc := GetMinerChain()

	if !isNetworkReadyForDKG() {
		ticker := time.NewTicker(5 * chain.DELTA)
		for ts := range ticker.C {
			active := mc.Miners.GetActiveCount()
			if !isDkgEnabled {
				Logger.Info("waiting for sufficient active nodes", zap.Time("ts", ts), zap.Int("active", active))
			} else {
				Logger.Info("waiting for all nodes to be active", zap.Time("ts", ts), zap.Int("active", active))
			}
			if isNetworkReadyForDKG() {
				break
			}
		}
	}
}
func sendDKG() {
	mc := GetMinerChain()

	m2m := mc.Miners

	shuffledNodes := m2m.GetRandomNodes(m2m.Size())

	for _, n := range shuffledNodes {

		if n != nil {
			if selfInd == n.SetIndex {
				//we do not want to send message to ourselves.
				continue
			}
			//ToDo: Optimization Instead of sending, asking for DKG share is better.
			err := SendDKGShare(n)
			if err != nil {
				Logger.Error("DKG Failed sending DKG share", zap.Int("idx", n.SetIndex), zap.Error(err))
			}
		} else {
			Logger.Info("DKG Error in getting node for ", zap.Int("idx", n.SetIndex))
		}
	}

}

/*SendDKGShare sends the generated secShare to the given node */
func SendDKGShare(n *node.Node) error {
	if !isDkgEnabled {
		Logger.Debug("DKG not enabled. Not sending shares")
		return nil
	}
	mc := GetMinerChain()
	m2m := mc.Miners

	secShare := minerShares[n.GetKey()]
	dkg := &bls.Dkg{
		Share: secShare.GetDecString()}
	dkg.SetKey(datastore.ToKey("1"))
	//Logger.Debug("sending DKG share", zap.Int("idx", n.SetIndex), zap.Any("share", dkg.Share))
	_, err := m2m.SendTo(DKGShareSender(dkg), n.GetKey())
	return err
}

/*WaitForDKGShares --This function waits FOREVER for enough #miners to send DKG shares */
func WaitForDKGShares() bool {

	//Todo: Add a configurable wait time.
	if !HasAllDKGSharesReceived() {
		ticker := time.NewTicker(5 * chain.DELTA)
		defer ticker.Stop()
		for ts := range ticker.C {
			if HasAllDKGSharesReceived() {
				Logger.Debug("Received sufficient DKG Shares. Sending DKG one moretime and going quiet", zap.Time("ts", ts))
				sendDKG()
				break
			}
			Logger.Info("waiting for sufficient DKG Shares", zap.Int("Received so far", len(recSharesMap)), zap.Time("ts", ts))
			sendDKG()

		}
	}

	return true

}

/*HasAllDKGSharesReceived returns true if all shares are received */
func HasAllDKGSharesReceived() bool {
	if !isDkgEnabled {
		Logger.Info("DKG not enabled. So, giving a go ahead")
		return true
	}
	mutex.RLock()
	defer mutex.RUnlock()
	//ToDo: Need parameterization
	if len(recSharesMap) >= n {
		return true
	}
	return false
}

func addToRecSharesMap(nodeID int, share string) {
	mutex.Lock()
	defer mutex.Unlock()
	if recSharesMap == nil {
		mc := GetMinerChain()

		m2m := mc.Miners
		recSharesMap = make(map[int]string, len(m2m.Nodes))
	}
	recSharesMap[nodeID] = share
}

/*AppendDKGSecShares - Gets the shares by other miners and append to the global array */
func AppendDKGSecShares(ctx context.Context, nodeID int, share string) {
	if !isDkgEnabled {
		Logger.Error("DKG is not enabled. Why are we here?")
		return
	}

	if recSharesMap != nil {
		if _, ok := recSharesMap[nodeID]; ok {
			Logger.Debug("Ignoring Share recived again from node : ", zap.Int("Node Id", nodeID))
			return
		}
	}
	recShares = append(recShares, share)
	addToRecSharesMap(nodeID, share)
	//ToDo: We cannot expect everyone to be ready to start. Should we use K?
	if HasAllDKGSharesReceived() {
		Logger.Debug("All the shares are received ...")
		AggregateDKGSecShares(ctx, recShares)
		Logger.Info("DKG is done :) ...")
		IsDkgDone = true
		go startProtocol()
	}

}

// VerifySigShares - Verify the bls sig share is correct
func VerifySigShares() bool {
	//TBD
	return true
}

/*GetBlsThreshold Handy api for now. move this to protocol_vrf */
func GetBlsThreshold() int {
	//return dg.T
	return k
}

/*ComputeBlsID Handy API to get the ID used in the library */
func ComputeBlsID(key int) string {
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetDecString()
}

// AggregateDKGSecShares - Each miner adds the shares to get the secKey share for group
func AggregateDKGSecShares(ctx context.Context, recShares []string) error {

	secShares := make([]bls.Key, len(recShares))
	for i := 0; i < len(recShares); i++ {
		err := secShares[i].SetDecString(recShares[i])
		if err != nil {
			Logger.Error("Aggregation of DKG shares not done", zap.Error(err))
		}
	}
	var sec bls.Key

	for i := 0; i < len(secShares); i++ {
		sec.Add(&secShares[i])
	}
	dg.SecKeyShareGroup = sec
	storeDKGSummary(ctx, dg.GetDKGSummary())
	Logger.Debug("the aggregated sec share",
		zap.String("sec_key_share_grp", dg.SecKeyShareGroup.GetDecString()),
		zap.String("gp_public_key", dg.GpPubKey.GetHexString()))
	return nil
}

// GetBlsShare - Start the BLS process
func GetBlsShare(ctx context.Context, r, pr *round.Round) string {
	r.VrfStartTime = time.Now()
	if !isDkgEnabled {
		Logger.Debug("returning standard string as DKG is not enabled.")
		return encryption.Hash("0chain")

	}

	Logger.Debug("DKG getBlsShare ", zap.Int64("Round Number", r.Number))

	bs = bls.MakeSimpleBLS(&dg)

	currRound = r.Number
	var rbOutput string
	prev_rseed := int64(0)
	if r.GetRoundNumber()-1 == 0 {

		Logger.Debug("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		rbOutput = encryption.Hash("0chain")
	} else {
		prev_rseed = pr.RandomSeed
		rbOutput = strconv.FormatInt(pr.RandomSeed, 16) //pr.VRFOutput
	}

	bs.Msg = fmt.Sprintf("%v%v%v", r.GetRoundNumber(), r.GetTimeoutCount(), rbOutput)

	Logger.Info("Bls sign vrfshare calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.Int("roundtimeout", r.GetTimeoutCount()), zap.Int64("prev_rseed", prev_rseed), zap.Any("bls_msg", bs.Msg))

	sigShare := bs.SignMsg()
	return sigShare.GetHexString()

}

//  ///////////  End fo BLS-DKG Related Stuff   ////////////////

//AddVRFShare - implement the interface for the RoundRandomBeacon protocol
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	Logger.Info("DKG AddVRFShare", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("RoundTimeoutCount", mr.GetTimeoutCount()),
		zap.Int("Sender", vrfs.GetParty().SetIndex), zap.Int("vrf_timeoutcount", vrfs.GetRoundTimeoutCount()),
		zap.String("vrf_share", vrfs.Share))
	mr.AddTimeoutVote(vrfs.GetRoundTimeoutCount(), vrfs.GetParty().ID)
	if vrfs.GetRoundTimeoutCount() != mr.GetTimeoutCount() {
		//Keep VRF timeout and round timeout in sync. Same vrfs will comeback during soft timeouts
		Logger.Info("TOC_FIX VRF Timeout > round timeout", zap.Int("vrfs_timeout", vrfs.GetRoundTimeoutCount()), zap.Int("round_timeout", mr.GetTimeoutCount()))
		return false
	}
	if len(mr.GetVRFShares()) >= GetBlsThreshold() {
		//ignore VRF shares coming after threshold is reached to avoid locking issues.
		//Todo: Remove this logging
		mr.AddAdditionalVRFShare(vrfs)
		Logger.Info("Ignoring VRFShare. Already at threshold", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("VRF_Shares", len(mr.GetVRFShares())))
		return false
	}
	if mr.AddVRFShare(vrfs, GetBlsThreshold()) {
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
	if len(shares) >= GetBlsThreshold() {
		Logger.Debug("VRF Hurray we've threshold BLS shares")
		if !isDkgEnabled {
			//We're still waiting for threshold number of VRF shares, even though DKG is not enabled.

			rbOutput := "" //rboutput will ignored anyway
			mc.computeRBO(ctx, mr, rbOutput)

			return
		}
		beg := time.Now()
		recSig, recFrom := getVRFShareInfo(mr)

		rbOutput := bs.CalcRandomBeacon(recSig, recFrom)
		Logger.Debug("VRF ", zap.String("rboOutput", rbOutput), zap.Int64("Round", mr.Number))
		mc.computeRBO(ctx, mr, rbOutput)
		end := time.Now()

		diff := end.Sub(beg)

		if diff > (time.Duration(k) * time.Millisecond) {
			Logger.Info("DKG RBO Calc ***SLOW****", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("VRF_shares", len(shares)), zap.Any("time_taken", diff))

		}
	} else {
		//TODO: remove this log
		Logger.Info("Not yet reached threshold", zap.Int("vrfShares_num", len(shares)), zap.Int("threshold", GetBlsThreshold()))
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
		Logger.Debug("VRF Printing from shares: ", zap.Int("Miner Index = ", n.SetIndex), zap.Any("Share = ", share.Share))

		recSig = append(recSig, share.Share)
		recFrom = append(recFrom, ComputeBlsID(n.SetIndex))
	}

	return recSig, recFrom
}

func (mc *Chain) computeRoundRandomSeed(ctx context.Context, pr round.RoundI, r *Round, rbo string) {

	var seed int64
	if isDkgEnabled {
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
