/* dkg_helper.go has all the glue DKG code */

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

//TODO: Make the values of k and n configurable

var k = 2
var n = 3
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
	for k := range roundMap {
		if k <= currRound {
			Logger.Debug("Remove details for round from roundMap since BLSDone is", zap.Any(":", k))
			delete(roundMap, k)
		}
	}
}

// StoreBlsShares - Insert the sig shares to hashmap for sync of BLS sig shares if received from future rounds
func StoreBlsShares(receivedRound int64, sigShare string, party int) {

	if (receivedRound <= currRound) && blsDone {
		Logger.Info("blsDone set to true")
		Logger.Info("Received BLS sign share after blsDone set to true", zap.String("BLS sign share", sigShare))
		Logger.Info("Received BLS share in the round after blsDone set to true", zap.Int64("BLS round", receivedRound))
		Logger.Info("Received from party after blsDone set to true", zap.Int(" ", party))
		return
	}
	Logger.Info("Got the bls sig share for the round XX", zap.Int64(":", receivedRound))
	Logger.Info("Where the currRound XX", zap.Int64(":", currRound))
	partyMap, roundExists := roundMap[receivedRound]

	if !roundExists {
		partyMap = make(map[int]string)
		roundMap[receivedRound] = partyMap
	}

	_, partyExists := partyMap[party]

	if partyExists {
		Logger.Info("BLS sign share already exists", zap.String(":", sigShare))
		Logger.Debug("In the round", zap.Int64(":", receivedRound))
		Logger.Debug("Where the current round", zap.Int64(":", currRound))
		Logger.Debug("Sig Share received from party already exists", zap.Int(":", party))
		return
	}

	partyMap[party] = sigShare
	Logger.Info("Inserting BLS sign share", zap.String("BLS sign share", sigShare))
	Logger.Info("Inserting BLS sign share for the round", zap.Int64("BLS round", receivedRound))
	Logger.Info("Inserting from party", zap.Int(" ", party))

	Logger.Info("Added in partyMap : and the contents are, ")

	for key, value := range roundMap {

		Logger.Info("The round key is", zap.Int64(":", key))
		Logger.Info("The value sigshare is", zap.Any(":", value))

	}

}

// BlsShareReceived - The function used to get the Random Beacon output for the round
func BlsShareReceived(ctx context.Context, receivedRound int64, sigShare string, party int) {

	if !(currRound == receivedRound || currRound == receivedRound-1) {
		Logger.Debug(" The receivedRound is ", zap.Int64(":", receivedRound))
		Logger.Debug(" The currRound is ", zap.Int64(":", currRound))
		return
	}
	StoreBlsShares(receivedRound, sigShare, party)

	recBlsSig, recBlsFrom, gotThreshold := ThresholdNumSigReceived(receivedRound, sigShare, party)

	if gotThreshold {
		vrfOp := CalcRandomBeacon(recBlsSig, recBlsFrom)
		GetMinerChain().VRFShareChannel <- vrfOp
		Logger.Info("vrfOp pushed to channel is : ", zap.String("vrfOp is : ", vrfOp), zap.Int64("currRound", currRound))
		Logger.Info("Printing blsDone since vrfOp is done ", zap.Any(":", blsDone))
		CleanupHashMap()
	}
}

//ThresholdNumSigReceived - Insert BLS sig shares to an array to pass to Gp Sign to compute the Random Beacon Output
func ThresholdNumSigReceived(receivedRound int64, sigShare string, party int) ([]string, []string, bool) {

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
				blsDone = true
				Logger.Debug("Is the blsDone set to true ??, once k num received", zap.Any(":", blsDone))
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
			Logger.Error("Aggregation of shares not done", zap.Error(err))
		}
	}
	var sec bls.Key

	for i := 0; i < len(secShares); i++ {
		sec.Add(&secShares[i])
	}
	dg.SecKeyShareGroup = sec
	Logger.Info("the aggregated sec share", zap.String("agg share", dg.SecKeyShareGroup.GetDecString()))
	Logger.Info("the group public key is", zap.String("gp public key", dg.GpPubKey.GetHexString()))
	return nil
}

// StartBls - Start the BLS process including calculating the BLS signature share
func StartBls(ctx context.Context, r *round.Round) {
	self := node.GetSelfNode(ctx)
	blsDone = false
	bs = bls.MakeSimpleBLS(&dg)
	var msg bytes.Buffer

	currRound = r.Number
	var vrfShares string
	if r.GetRoundNumber()-1 == 0 {

		Logger.Info("The corner case for round 1 when pr is nil :", zap.Int64("corner case round 1 ", r.GetRoundNumber()))
		vrfShares = encryption.Hash("0chain")
	} else {

		vrfShares = <-GetMinerChain().VRFShareChannel
	}

	binary.Write(&msg, binary.LittleEndian, r.GetRoundNumber())
	binary.Write(&msg, binary.LittleEndian, vrfShares)
	bs.Msg = util.ToHex(msg.Bytes())

	Logger.Info("For the round is", zap.Int64("the r.GetRoundNumber() is ", r.GetRoundNumber()))
	Logger.Info("with the prev vrf share is", zap.Any(":", vrfShares))

	Logger.Info("the msg is", zap.Any("the blsMsg is ", bs.Msg))
	Logger.Info("the aggregated sec share bls", zap.String("agg share bls", bs.SecKeyShareGroup.GetDecString()))

	mc := GetMinerChain()

	m2m := mc.Miners

	sigShare := bs.SignMsg()

	bs.SigShare = sigShare
	Logger.Info("the sig share", zap.Any("bls sig share ", sigShare.GetHexString()))

	currRound = r.Number
	Logger.Info("The currRound is ", zap.Int64("", currRound))

	_, roundExists := roundMap[currRound]

	if !roundExists {
		Logger.Info(" The RoundID does not exist when the BLS starts, so adding self share ")
		partyMap := make(map[int]string)
		partyMap[self.SetIndex] = sigShare.GetHexString()
		roundMap[currRound] = partyMap
	}

	Logger.Info("Added in roundMap the initial self share ")

	for key, value := range roundMap {

		Logger.Info("The round key is for self share", zap.Int64(":", key))
		Logger.Info("The value sigshare is for self share", zap.Any(":", value))

	}

	bls := &bls.Bls{
		BLSsignShare: sigShare.GetHexString(),
		BLSRound:     currRound}
	bls.SetKey(datastore.ToKey("1"))
	m2m.SendAll(BLSSignShareSender(bls))

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

// CalcRandomBeacon - The function calls the function which calculates the Gp Sign
func CalcRandomBeacon(recSig []string, recIDs []string) string {
	Logger.Info("Threshold number of bls sig shares are received ...")
	CalBlsGpSign(recSig, recIDs)
	vrfOutput := encryption.Hash(bs.GpSign.GetHexString())
	return vrfOutput
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
	Logger.Info("Printing bls shares and respective party IDs used for computing the Gp Sign")

	for _, element := range signVec {
		Logger.Info(" Printing bls shares", zap.Any(":", element.GetHexString()))

	}
	for _, element1 := range idVec {
		Logger.Info(" Printing party IDs", zap.Any(":", element1.GetHexString()))

	}
	bs.RecoverGroupSig(idVec, signVec)

}
