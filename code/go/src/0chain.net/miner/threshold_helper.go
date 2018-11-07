/* threshold_helper.go has all the glue DKG and BLS code */

package miner

import (
	"context"
	"encoding/binary"
	"time"

	"go.uber.org/zap"

	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/threshold/bls"
	"0chain.net/util"

	"bytes"

	. "0chain.net/logging"
)

var dg bls.SimpleDKG
var bs bls.SimpleBLS
var recShares []string
var currRound int64

var roundMap = make(map[int64]map[int]string)
var blsDone bool

// StartDKG - starts the DKG process
func StartDKG(ctx context.Context) {

	mc := GetMinerChain()

	m2m := mc.Miners

	k := (2 * mc.Miners.Size()) / 3
	n := mc.Miners.Size()

	Logger.Info("Starting DKG...")

	//TODO : Need to include check for active miners and then send the shares, have to remove sleep in future
	//mc.Miners.OneTimeStatusMonitor(ctx)
	time.Sleep(10 * time.Second)

	dg = bls.MakeSimpleDKG(k, n)

	for _, node := range m2m.Nodes {

		Logger.Info("The miner ID is ", zap.String("miner ID is ", node.GetKey()))
		forID := bls.ComputeIDdkg(node.SetIndex)
		dg.ID = forID
		Logger.Info("The miner ID vVec is ", zap.String("miner ID  vVecis ", dg.Vvec[0].GetHexString()))

		Logger.Info("The x is ", zap.String("x is ", forID.GetDecString()))
		secShare, _ := dg.ComputeDKGKeyShare(forID)
		Logger.Info("secShare for above minerID is ", zap.String("secShare for above minerID is ", secShare.GetDecString()))

		dkg := &bls.Dkg{
			Share: secShare.GetDecString()}
		dkg.SetKey(datastore.ToKey("1"))

		m2m.SendTo(DKGShareSender(dkg), node.ID)

	}

	//Had to introduce delay for the time when shares are received by other miners
	time.Sleep(10 * time.Second)

	if len(recShares) == dg.N {
		Logger.Info("All the shares are received ...")
		AggregateDKGSecShares(recShares)
		Logger.Info("DKG is done :) ...")

	}

	//Had to introduce delay to complete the DKG before the round starts
	time.Sleep(10 * time.Second)

}

// AppendDKGSecShares - Gets the shares by other miners and append to the global array
func AppendDKGSecShares(share string) {
	recShares = append(recShares, share)
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
		nodeMap = make(map[int]string)
		roundMap[receivedRound] = nodeMap
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

	if !(currRound == receivedRound || currRound == receivedRound-1) {
		Logger.Debug("Got the bls sign share", zap.Int64("round", receivedRound), zap.Int64("curr_round", currRound))
		return
	}
	StoreBlsShares(receivedRound, sigShare, party)

	recBlsSig, recBlsFrom, gotThreshold := ThresholdNumSigReceived(receivedRound, sigShare, party)

	if blsDone {
		Logger.Debug("Printing blsDone since rbOutput is done for round check XXX", zap.Any(":", blsDone))
		return
	}

	if gotThreshold {
		rbOutput := CalcRandomBeacon(recBlsSig, recBlsFrom)
		blsDone = true
		GetMinerChain().VRFShareChannel <- rbOutput
		Logger.Info("rbOutput pushed to channel is : ", zap.String("rbOutput: ", rbOutput), zap.Int64("curr_Round", currRound))
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
	var msg bytes.Buffer

	currRound = r.Number
	var rbOutput string
	if r.GetRoundNumber()-1 == 0 {

		Logger.Info("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		rbOutput = encryption.Hash("0chain")
	} else {
		rbOutput = <-GetMinerChain().VRFShareChannel
	}

	binary.Write(&msg, binary.LittleEndian, r.GetRoundNumber())
	binary.Write(&msg, binary.LittleEndian, rbOutput)
	bs.Msg = util.ToHex(msg.Bytes())
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
