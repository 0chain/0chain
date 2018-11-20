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

	n := mc.Miners.Size()
	minerShares = make(map[string]bls.Key, len(m2m.Nodes))
	Logger.Info("Starting DKG...")

	//TODO : Need to include check for active miners and then send the shares

	dg = bls.MakeDKG(k, n)
	self := node.GetSelfNode(ctx)

	for _, node := range m2m.Nodes {

		Logger.Info("The miner ID is ", zap.String("miner ID is ", node.GetKey()))
		forID := bls.ComputeIDdkg(node.SetIndex)
		dg.ID = forID

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
	//Move this to group_sig.go --May Be
	Logger.Info("DKG-X getBlsShare ", zap.Int64("Round Number", r.Number))

	bs = bls.MakeSimpleBLS(&dg)

	currRound = r.Number
	var rbOutput string
	if r.GetRoundNumber()-1 == 0 {

		Logger.Info("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		rbOutput = encryption.Hash("0chain")
	} else {
		rbOutput = pr.VRFOutput
		Logger.Info("DKG-X Using older VRFOutput :", zap.Int64("round", r.GetRoundNumber()))

	}

	bs.Msg = strconv.FormatInt(r.GetRoundNumber(), 10) + rbOutput

	Logger.Info("Bls sign share calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.String("rbo_output", rbOutput), zap.Any("bls_msg", bs.Msg), zap.String("sec_key_share_gp", bs.SecKeyShareGroup.GetDecString()))

	sigShare := bs.SignMsg()
	return sigShare.GetHexString()

}

// CalcRandomBeacon - Calculates the random beacon output
func CalcRandomBeacon(recSig []string, recIDs []string) string {
	//Move this to group_sig.go
	Logger.Info("Threshold number of bls sig shares are received ...")
	CalBlsGpSign(recSig, recIDs)
	rboOutput := encryption.Hash(bs.GpSign.GetHexString())
	return rboOutput
}

// CalBlsGpSign - The function calls the RecoverGroupSig function which calculates the Gp Sign
func CalBlsGpSign(recSig []string, recIDs []string) {
	//Move this to group_sig.go
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
