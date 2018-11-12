/* threshold_helper.go has all the glue DKG and BLS code */

package miner

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"0chain.net/chain"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/threshold/bls"

	. "0chain.net/logging"
)

var dg bls.SimpleDKG
var bs bls.SimpleBLS
var recShares []string
var recSharesMap map[int]string
var minerShares map[string]bls.Key
var currRound int64

var roundMap = make(map[int64]map[int]string)
var blsDone bool

// StartDKG - starts the DKG process
func StartDKG(ctx context.Context) {

	mc := GetMinerChain()

	m2m := mc.Miners

	thresholdByCount := viper.GetInt("server_chain.block.consensus.threshold_by_count")
	k := int(math.Ceil((float64(thresholdByCount) / 100) * float64(mc.Miners.Size())))

	n := mc.Miners.Size()
	minerShares = make(map[string]bls.Key, len(m2m.Nodes))
	Logger.Info("Starting DKG...")

	//TODO : Need to include check for active miners and then send the shares, have to remove sleep in future

	dg = bls.MakeSimpleDKG(k, n)
	self := node.GetSelfNode(ctx)

	for _, node := range m2m.Nodes {

		Logger.Info("The miner ID is ", zap.String("miner ID is ", node.GetKey()))
		forID := bls.ComputeIDdkg(node.SetIndex)
		dg.ID = forID
		Logger.Info("The miner ID vVec is ", zap.String("miner ID  vVecis ", dg.Vvec[0].GetHexString()))

		Logger.Info("The x is ", zap.String("x is ", forID.GetDecString()))
		secShare, _ := dg.ComputeDKGKeyShare(forID)
		Logger.Info("secShare for above minerID is ", zap.String("secShare for above minerID is ", secShare.GetDecString()))

		Logger.Info("secShare ", zap.String("secShare for minerID is ", secShare.GetDecString()), zap.Int("miner index", node.SetIndex))
		minerShares[node.GetKey()] = secShare
		if self.SetIndex == node.SetIndex {
			recShares = append(recShares, secShare.GetDecString())
			addToRecSharesMap(self.SetIndex, secShare.GetDecString())
		}

	}
	go WaitForDKGShares()

}
func sendDKG() {
	mc := GetMinerChain()

	m2m := mc.Miners

	for _, n := range m2m.Nodes {

		if n != nil {
			//ToDo: Optimization Instead of sending, asking for DKG share is better.
			err := SendDKGShare(n)
			if err != nil {
				Logger.Info("DKG-1 Failed sending DKG share", zap.Int("idx", n.SetIndex), zap.Error(err))
			} else {
				Logger.Info("DKG-1 Success sending DKG share", zap.Int("idx", n.SetIndex))
			}
		} else {
			Logger.Info("DKG-1 Error in getting node for ", zap.Int("idx", n.SetIndex))
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
				Logger.Info("Received sufficient DKG Shares. Sending DKG one moretime and going quiet", zap.Time("ts", ts))
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
	Logger.Info("DKG-2 Share received", zap.Int("NodeId: ", nodeID), zap.String("Share: ", share))

	if recSharesMap != nil {
		if _, ok := recSharesMap[nodeID]; ok {
			Logger.Info("DKG-2 Ignoring Share recived again from node : ", zap.Int("Node Id", nodeID))
			return
		}
	}
	recShares = append(recShares, share)
	addToRecSharesMap(nodeID, share)
	//ToDo: We cannot expect everyone to be ready to start. Should we use K?
	if HasAllDKGSharesReceived() {
		Logger.Info("All the shares are received ...")
		AggregateDKGSecShares(recShares)
		Logger.Info("DKG-X  is done :) ...")
		go StartProtocol()
	}

}

// VerifySigShares - Verify the bls sig share is correct
func VerifySigShares() bool {
	//TBD
	return true
}

// CleanupHashMap - Clean up the hash map once the Random Beacon is done for the round
func CleanupHashMap() {
	for key := range roundMap {
		if key <= currRound {
			Logger.Debug("Remove details for round from roundMap since BLS is done for the round", zap.Any(":", key))
			delete(roundMap, key)
		}
	}
}

// StoreBlsShares - Insert the sig shares to hashmap for sync of BLS sig shares if received for future rounds
func StoreBlsShares(receivedRound int64, sigShare string, party int) {

	if (receivedRound <= currRound) && blsDone {
		Logger.Debug("Received Bls sign share after blsDone set to true", zap.Int64("round", receivedRound), zap.String("sig_share", sigShare), zap.Int("from_party", party))
		return
	}
	nodeMap, roundExists := roundMap[receivedRound]

	if !roundExists {
		Logger.Debug("If round does not exist")
		nodeMap = make(map[int]string)
		roundMap[receivedRound] = nodeMap
	} else {
		Logger.Debug("Round entry exists")
	}
	_, partyExists := nodeMap[party]

	if partyExists {
		Logger.Debug("Bls sign share already exists", zap.Int64("round", receivedRound), zap.String("sig_share", sigShare), zap.Int("from_party", party))
		return
	}

	nodeMap[party] = sigShare
}

// BlsShareReceived - The function used to get the Random Beacon output for the round
func BlsShareReceived(ctx context.Context, receivedRound int64, sigShare string, party int) {

	if blsDone {
		Logger.Debug("Printing blsDone since rbOutput is done for round check XXX", zap.Any(":", blsDone))
		return
	}

	if !(currRound == receivedRound || currRound == receivedRound-1) {
		Logger.Debug("Got the bls sign share", zap.Int64("round", receivedRound), zap.Int64("curr_round", currRound))
		return
	}
	StoreBlsShares(receivedRound, sigShare, party)

	recBlsSig, recBlsFrom, gotThreshold := ThresholdNumSigReceived(receivedRound, sigShare, party)

	if gotThreshold {
		rbOutput := CalcRandomBeacon(recBlsSig, recBlsFrom)
		blsDone = true
		GetMinerChain().RBOChannel <- rbOutput
		Logger.Info("Random Beacon Output", zap.String("rbOutput: ", rbOutput), zap.Int64("curr_Round", currRound))
		Logger.Debug("Printing blsDone since rbOutput is done ", zap.Any(":", blsDone))
		CleanupHashMap()
	}
}

//ThresholdNumSigReceived - Insert BLS sig shares to an array to pass to Gp Sign to compute the Random Beacon Output
func ThresholdNumSigReceived(receivedRound int64, sigShare string, party int) ([]string, []string, bool) {

	if blsDone && (receivedRound == currRound) {
		return nil, nil, false
	}
	hmap, ok := roundMap[receivedRound]

	recSig := make([]string, 0)
	recFrom := make([]string, 0)

	if ok {
		for key, value := range hmap {
			if receivedRound == currRound {
				recSig = append(recSig, value)
				computeID := bls.ComputeIDdkg(key)
				recFrom = append(recFrom, computeID.GetDecString())
			}
			if len(recSig) == dg.T {
				return recSig, recFrom, true
			}
		}
	}

	return nil, nil, false

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

// StartBls - Start the BLS process
func StartBls(ctx context.Context, r *round.Round) {
	self := node.GetSelfNode(ctx)
	blsDone = false
	bs = bls.MakeSimpleBLS(&dg)

	currRound = r.Number
	var rbOutput string
	if r.GetRoundNumber()-1 == 0 {

		Logger.Info("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		rbOutput = encryption.Hash("0chain")
	} else {
		rbOutput = <-GetMinerChain().RBOChannel
	}

	bs.Msg = strconv.FormatInt(r.GetRoundNumber(), 10) + rbOutput

	Logger.Info("Bls sign share calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.String("rbo_previous_round", rbOutput), zap.Any("bls_msg", bs.Msg), zap.String("sec_key_share_gp", bs.SecKeyShareGroup.GetDecString()))

	mc := GetMinerChain()

	m2m := mc.Miners

	sigShare := bs.SignMsg()

	bs.SigShare = sigShare
	currRound = r.Number

	_, roundExists := roundMap[currRound]

	if !roundExists {
		Logger.Debug("No entry found for round when the BLS starts, adding self share")
		nodeMap := make(map[int]string)
		nodeMap[self.SetIndex] = sigShare.GetHexString()
		roundMap[currRound] = nodeMap
	}

	BroadCastSig(sigShare.GetHexString(), currRound, m2m)
	RebroadCastSig(sigShare.GetHexString(), currRound, m2m)
}

// RebroadCastSig - Rebroadcast the self share so that a miner is not behind a round
func RebroadCastSig(sig string, curRound int64, m2m *node.Pool) {

	bls := &bls.Bls{
		BLSsignShare: sig,
		BLSRound:     curRound}
	bls.SetKey(datastore.ToKey("1"))
	m2m.SendAll(BLSSignShareSender(bls))

}

// BroadCastSig - Broadcast the bls sign share
func BroadCastSig(sig string, curRound int64, m2m *node.Pool) {

	bls := &bls.Bls{
		BLSsignShare: sig,
		BLSRound:     curRound}
	bls.SetKey(datastore.ToKey("1"))
	m2m.SendAll(BLSSignShareSender(bls))

}

// CalcRandomBeacon - Calculates the random beacon output
func CalcRandomBeacon(recSig []string, recIDs []string) string {
	Logger.Info("Threshold number of bls sig shares are received ...")
	CalBlsGpSign(recSig, recIDs)
	rboOutput := encryption.Hash(bs.GpSign.GetHexString())
	return rboOutput
}

// CalBlsGpSign - The function calls the RecoverGroupSig function which calculates the Gp Sign
func CalBlsGpSign(recSig []string, recIDs []string) {

	signVec := make([]bls.Sign, 0)
	var signShare bls.Sign

	for i := 0; i < len(recSig); i++ {
		err := signShare.SetHexString(recSig[i])

		if err == nil {
			signVec = append(signVec, signShare)
		} else {
			Logger.Error("signVec not computed correctly", zap.Error(err))
		}
	}

	idVec := make([]bls.PartyID, 0)
	var forID bls.PartyID
	for i := 0; i < len(recIDs); i++ {
		err := forID.SetDecString(recIDs[i])
		if err == nil {
			idVec = append(idVec, forID)
		}
	}
	Logger.Info("Printing bls shares and respective party IDs who sent used for computing the Gp Sign")

	for _, sig := range signVec {
		Logger.Info(" Printing bls shares", zap.Any("sig_shares", sig.GetHexString()))

	}
	for _, fromParty := range idVec {
		Logger.Info(" Printing party IDs", zap.Any("from_party", fromParty.GetHexString()))

	}
	bs.RecoverGroupSig(idVec, signVec)

}
