/* dkg_helper.go has all the glue DKG code */

package miner

import (
	"time"

	"go.uber.org/zap"

	"0chain.net/node"
	"0chain.net/threshold/bls"

	"0chain.net/datastore"
	. "0chain.net/logging"
)

//TODO: Make the values of k and n configurable

var k = 2
var n = 3
var dg bls.BLSSimpleDKG
var recShares []string

/* StartDKG - starts the DKG process */

func StartDKG(miners *node.Pool) {

	m2m := miners
	Logger.Info("Starting DKG...")

	//TODO : Remove sleep in future VRF
	time.Sleep(10 * time.Second)

	//TODO : Need to include check for active miners and then send the shares
	//mc.Miners.OneTimeStatusMonitor(ctx)

	dg = bls.MakeSimpleDKG(k, n)

	for i, node := range m2m.Nodes {

		Logger.Info("The miner ID is ", zap.String("miner ID is ", node.GetKey()))
		forID := bls.ComputeIDdkg(i)

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
	}

	Logger.Info("DKG is done :) ...")

	//Had to introduce delay to complete the DKG before the round starts
	time.Sleep(10 * time.Second)

}

/* AppendDKGSecShares - Gets the shares by other miners and append to the global array */

func AppendDKGSecShares(share string) {

	recShares = append(recShares, share)
	//Logger.Info("shareArr append is ", zap.String("append share is ", share))
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
