package miner

import (
	"context"
	"math"
	"strconv"
	"time"

	"0chain.net/chain"
	"0chain.net/datastore"
	"0chain.net/encryption"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/threshold/bls"
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

// StartDKG - starts the DKG process
func StartDKG(ctx context.Context) {

	mc := GetMinerChain()

	m2m := mc.Miners

	thresholdByCount := viper.GetInt("server_chain.block.consensus.threshold_by_count")
	k := int(math.Ceil((float64(thresholdByCount) / 100) * float64(mc.Miners.Size())))

	Logger.Info("The threshold", zap.Int("K", k))
	n := mc.Miners.Size()
	minerShares = make(map[string]bls.Key, len(m2m.Nodes))
	Logger.Info("Starting DKG...")

	dg = bls.MakeDKG(k, n)
	self := node.GetSelfNode(ctx)

	for _, node := range m2m.Nodes {
		forID := bls.ComputeIDdkg(node.SetIndex)
		dg.ID = forID

		secShare, _ := dg.ComputeDKGKeyShare(forID)

		Logger.Debug("ComputeDKGKeyShare ", zap.String("secShare", secShare.GetDecString()), zap.Int("miner index", node.SetIndex))
		minerShares[node.GetKey()] = secShare
		if self.SetIndex == node.SetIndex {
			recShares = append(recShares, secShare.GetDecString())
			addToRecSharesMap(self.SetIndex, secShare.GetDecString())
		}

	}
	WaitForDKGShares()
}
func sendDKG() {
	mc := GetMinerChain()

	m2m := mc.Miners

	for _, n := range m2m.Nodes {

		if n != nil {
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
	mc := GetMinerChain()
	m2m := mc.Miners

	secShare := minerShares[n.GetKey()]
	dkg := &bls.Dkg{
		Share: secShare.GetDecString()}
	dkg.SetKey(datastore.ToKey("1"))
	Logger.Info("@sending DKG share", zap.Int("idx", n.SetIndex), zap.Any("share", dkg.Share))
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
			sendDKG()
			Logger.Info("waiting for sufficient DKG Shares", zap.Time("ts", ts))
			if HasAllDKGSharesReceived() {
				Logger.Debug("Received sufficient DKG Shares. Sending DKG one moretime and going quiet", zap.Time("ts", ts))
				sendDKG()
				break
			}
		}
	}

	return true

}

/*HasAllDKGSharesReceived returns true if all shares are received */
func HasAllDKGSharesReceived() bool {
	//ToDo: Need parameterization
	if len(recSharesMap) >= dg.N {
		return true
	}
	return false
}

func addToRecSharesMap(nodeID int, share string) {
	if recSharesMap == nil {
		mc := GetMinerChain()

		m2m := mc.Miners
		recSharesMap = make(map[int]string, len(m2m.Nodes))
	}
	recSharesMap[nodeID] = share
}

/*AppendDKGSecShares - Gets the shares by other miners and append to the global array */
func AppendDKGSecShares(nodeID int, share string) {
	Logger.Debug("Share received", zap.Int("NodeId: ", nodeID), zap.String("Share: ", share))

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
		AggregateDKGSecShares(recShares)
		Logger.Info("DKG is done :) ...")
		go StartProtocol()
	}

}

// VerifySigShares - Verify the bls sig share is correct
func VerifySigShares() bool {
	//TBD
	return true
}

/*GetBlsThreshold Handy api for now. move this to protocol_vrf */
func GetBlsThreshold() int {
	return dg.T
}

/*ComputeBlsID Handy API to get the ID used in the library */
func ComputeBlsID(key int) string {
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetDecString()
}

// AggregateDKGSecShares - Each miner adds the shares to get the secKey share for group
func AggregateDKGSecShares(recShares []string) error {

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
	Logger.Info("the aggregated sec share", zap.String("sec_key_share_grp", dg.SecKeyShareGroup.GetDecString()))
	Logger.Info("the group public key is", zap.String("gp_public_key", dg.GpPubKey.GetHexString()))
	return nil
}

// GetBlsShare - Start the BLS process
func GetBlsShare(ctx context.Context, r, pr *round.Round) string {
	Logger.Debug("DKG getBlsShare ", zap.Int64("Round Number", r.Number))

	bs = bls.MakeSimpleBLS(&dg)

	currRound = r.Number
	var rbOutput string
	if r.GetRoundNumber()-1 == 0 {

		Logger.Info("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		rbOutput = encryption.Hash("0chain")
	} else {
		rbOutput = pr.VRFOutput
	}

	bs.Msg = strconv.FormatInt(r.GetRoundNumber(), 10) + rbOutput

	Logger.Debug("Bls sign share calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.String("rbo_output", rbOutput), zap.Any("bls_msg", bs.Msg), zap.String("sec_key_share_gp", bs.SecKeyShareGroup.GetDecString()))

	sigShare := bs.SignMsg()
	return sigShare.GetHexString()

}

//  ///////////  End fo BLS-DKG Related Stuff   ////////////////

//AddVRFShare - implement the interface for the RoundRandomBeacon protocol
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	Logger.Info("DKG AddVRFShare", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("Sender", vrfs.GetParty().SetIndex))
	if mr.AddVRFShare(vrfs) {
		mc.ThresholdNumBLSSigReceived(ctx, mr)
		return true
	}
	return false
}

/*ThresholdNumBLSSigReceived do we've sufficient BLSshares? */
func (mc *Chain) ThresholdNumBLSSigReceived(ctx context.Context, mr *Round) {

	if mr.IsVRFComplete() {
		//BLS has completed already for this round, But, received a BLS message from a node now
		Logger.Debug("DKG ThresholdNumSigReceived VRF is already completed.", zap.Int64("round", mr.GetRoundNumber()))
		return
	}

	shares := mr.GetVRFShares()
	if len(shares) >= GetBlsThreshold() {
		Logger.Debug("DKG Hurray we've threshold BLS shares")
		recSig := make([]string, 0)
		recFrom := make([]string, 0)
		for _, share := range shares {
			n := share.GetParty()
			Logger.Debug("DKG Printing from shares: ", zap.Int("Miner Index = ", n.SetIndex), zap.Any("Share = ", share.Share))

			recSig = append(recSig, share.Share)
			recFrom = append(recFrom, ComputeBlsID(n.SetIndex))
		}
		rbOutput := bs.CalcRandomBeacon(recSig, recFrom)
		Logger.Debug("DKG ", zap.String("rboOutput", rbOutput), zap.Int64("Round #", mr.Number))
		mc.computeRBO(ctx, mr, rbOutput)
	}
}

func (mc *Chain) computeRBO(ctx context.Context, mr *Round, rbo string) {
	Logger.Debug("DKG computeRBO")
	if mr.IsVRFComplete() {
		Logger.Error("DKG computeRBO RBO is already completed")
		return
	}

	pr := mc.GetRound(mr.GetRoundNumber() - 1)
	if pr != nil {
		mc.computeRoundRandomSeed(ctx, pr, mr, rbo)
	}

}
func (mc *Chain) computeRoundRandomSeed(ctx context.Context, pr round.RoundI, r *Round, rbo string) {

	if mpr := pr.(*Round); mpr.IsVRFComplete() {

		useed, err := strconv.ParseUint(rbo[0:16], 16, 64)

		if err != nil {
			panic(err)
		}
		seed := int64(useed)
		Logger.Info("BLS Calcualted", zap.Int64("RRS", seed), zap.Int64("Round #", r.Number))
		r.Round.SetVRFOutput(rbo)
		mc.setRandomSeed(ctx, r, seed)
	} else {
		Logger.Error("compute round random seed - no prior value", zap.Int64("round", r.GetRoundNumber()), zap.Int("blocks", len(pr.GetProposedBlocks())))
	}
}

func (mc *Chain) setRandomSeed(ctx context.Context, r *Round, seed int64) {
	mc.SetRandomSeed(r.Round, seed)
	mc.startNewRound(ctx, r)
}
