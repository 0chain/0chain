/* dkg_helper.go has all the glue DKG code */

package miner

import (
	"context"
	"encoding/binary"
	"time"

	"go.uber.org/zap"

	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/round"
	"0chain.net/threshold/bls"
	"0chain.net/util"

	"bytes"

	. "0chain.net/logging"
)

//TODO: Make the values of k and n configurable

var k = 2
var n = 3
var dg bls.BLSSimpleDKG
var bs bls.SimpleBLS
var recShares []string
var recSig []string
var recIDs []string

/* StartDKG - starts the DKG process */

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

		//TBD
		/*	if self.ID == node.ID {
				dg.SelfShare = secShare
			} else {
			miners.SendTo(miner.DKGShareSender(dkg), node.ID)
				} */
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

/* AppendDKGSecShares - Gets the shares by other miners and append to the global array */
func AppendDKGSecShares(share string) {
	recShares = append(recShares, share)
}

func AppendBLSSigShares(sigShare string, nodeID int) bool {

	recSig = append(recSig, sigShare)
	computeID := bls.ComputeIDdkg(nodeID)
	recIDs = append(recIDs, computeID.GetDecString())

	if len(recSig) == dg.T {
		Logger.Info(" the k number of recSig is", zap.Any("the recSig is ", recSig))
		Logger.Info(" the k number of recIDs is", zap.Any("the recIDs is ", recIDs))
		return true
	}
	return false
}

/* AggregateDKGSecShares - Each miner adds the shares to get the secKey share for group */
func AggregateDKGSecShares(recShares []string) error {

	secShares := make([]bls.Key, len(recShares))
	for i := 0; i < len(recShares); i++ {
		err := secShares[i].SetDecString(recShares[i])
		if err != nil {
			return nil
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

func CalcBLSSignShare(mr *round.Round, pr *round.Round) {
	bs = bls.MakeSimpleBLS(&dg)
	var msg bytes.Buffer

	recSig = make([]string, 0)
	recIDs = make([]string, 0)
	Logger.Info("The mr.Number is", zap.Int64("mr.Number is ", mr.Number))

	var vrfShares string
	if pr.GetRoundNumber() == 1 {
		Logger.Info("The corner case for round 1:", zap.Int64("corner case round 1 ", pr.GetRoundNumber()))

		vrfShares = encryption.Hash("0chain")
	} else {

		vrfShares = <-GetMinerChain().VRFShareChannel
		//vrfShares = bs.VrfOp
	}
	Logger.Info("The vrfShares is", zap.Any("the vrf share is ", vrfShares))

	binary.Write(&msg, binary.LittleEndian, mr.GetRoundNumber())
	binary.Write(&msg, binary.LittleEndian, vrfShares)
	bs.Msg = util.ToHex(msg.Bytes())

	Logger.Info("For the round is", zap.Int64("the mr.GetRoundNumber() is ", mr.GetRoundNumber()))
	Logger.Info("with the prev vrf share is", zap.Any("vrfShares() is ", vrfShares))

	Logger.Info("the msg is", zap.Any("the blsMsg is ", bs.Msg))
	//Logger.Info("the aggregated sec share bls", zap.String("agg share bls", bs.SecKeyShareGroup.GetDecString()))

	mc := GetMinerChain()

	m2m := mc.Miners

	sigShare := bs.SignMsg()

	bs.SigShare = sigShare
	Logger.Info("the sig share", zap.String("bls sig share", bs.SigShare.GetHexString()))

	recSig = append(recSig, bs.SigShare.GetHexString())
	recIDs = append(recIDs, bs.ID.GetDecString())

	//Logger.Info(" append self share recSig is", zap.Any("the recSig is ", recSig))
	//	Logger.Info(" append self share recIDs is", zap.Any("the recIDs is ", recIDs))

	bls := &bls.Bls{
		BLSsignShare: sigShare.GetHexString()}
	bls.SetKey(datastore.ToKey("1"))

	m2m.SendAll(BLSSignShareSender(bls))

}

func CheckThresholdSigns() string {

	Logger.Info("Threshold number of bls sig shares are received ...")
	RecoverGpSignWithbls(recSig, recIDs)
	vrfOutput := encryption.Hash(bs.GpSign.GetHexString())
	recSig = make([]string, 0)
	recIDs = make([]string, 0)
	return vrfOutput
}

func RecoverGpSignWithbls(recSig []string, recIDs []string) {
	signVec := make([]bls.Sign, len(recSig))
	var signShare bls.Sign

	for i := 0; i < len(recSig); i++ {
		err := signShare.SetHexString(recSig[i])

		if err != nil {
			signVec = append(signVec, signShare)
		}
	}

	idVec := make([]bls.PartyId, len(recIDs))
	var forID bls.PartyId
	for i := 0; i < len(recIDs); i++ {
		err := forID.SetDecString(recIDs[i])
		if err != nil {
			idVec = append(idVec, forID)
		}
	}

	RecoverGpSign(signVec, idVec)
}

func RecoverGpSign(signVec []bls.Sign, idVec []bls.PartyId) error {

	var sig bls.Sign
	err := sig.Recover(signVec, idVec)

	if err != nil {
		return nil
	}
	bs.GpSign = sig
	Logger.Info("the Gp Sign is ", zap.String("Gp Sign:", bs.GpSign.GetHexString()))
	Logger.Info("BLS is done :) ...")

	return nil
}
