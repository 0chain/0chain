package bls

import (
	"strconv"
	"testing"

	"0chain.net/encryption"
)

const CurveFp254BNb = 0

type DKGs []SimpleDKG

func newDKGs(t, n int) DKGs {
	dkgs := make([]SimpleDKG, n)
	for i := range dkgs {
		dkgs[i] = MakeSimpleDKG(t, n)
		dkgs[i].ID = ComputeIDdkg(i)
		for j := range dkgs {
			dkgs[j].ID = ComputeIDdkg(j)
			dkgs[i].ComputeDKGKeyShare(dkgs[j].ID)
		}
	}
	return dkgs
}

/*TestMakeSimpleDKG - Test to check whether SecKey - secret key, mSec - master secret key and mVec - verification key are set
Here mSec[0] is the secretKey */
func TestMakeSimpleDKG(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		if dkg.mSec == nil {
			test.Errorf("The master secret key not set")
		}
		if dkg.Vvec == nil {
			test.Errorf("The verification key not set")
		}

	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/*TestMakeSimpleMultipleDKGs - Test to check creation of multiple DKGs works */
func TestMakeSimpleMultipleDKGs(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		for i := 0; i < n; i++ {
			if dkgs[i].mSec == nil {
				test.Errorf("The master secret key not set")
			}
			if dkgs[i].Vvec == nil {
				test.Errorf("The verification key not set")
			}
			if dkgs[i].secSharesMap == nil {
				test.Errorf("For PartyID %s The secShares not set %v", dkgs[i].ID.GetDecString(), dkgs[i].secSharesMap)
			}
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/*TestRecoverSecretKey - Tests to check the secret key of a miner during the DKG process can be recovered back. Checking for poly sub method.*/
func TestRecoverSecretKey(test *testing.T) {

	variation := func(t, n int) {

		dkgs := newDKGs(t, n)
		secSharesFromMap := make([]Key, n)
		partyIdsFromMap := make([]PartyID, n)

		for i := 0; i < n; i++ {
			j := 0
			hmap := dkgs[i].secSharesMap
			for k, v := range hmap {
				secSharesFromMap[j] = v
				partyIdsFromMap[j] = k
				j++
			}

			if secSharesFromMap == nil {
				test.Errorf("Shares are not derived correctly %v\n", secSharesFromMap)
			}
			if partyIdsFromMap == nil {
				test.Errorf("PartyIds are not derived correctly %v\n", partyIdsFromMap)
			}

			var sec2 Key
			err := sec2.Recover(secSharesFromMap, partyIdsFromMap)
			if err != nil {
				test.Errorf("Recover shares Lagrange Interpolation error secSharesFromMap : %v\n ,Recover sec : %s\n, forIDs : %v\n", secSharesFromMap, sec2.GetHexString(), partyIdsFromMap)
				test.Error(err)
			}
			if !dkgs[i].mSec[0].IsEqual(&sec2) {
				test.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", dkgs[i].mSec[0].GetHexString(), sec2.GetHexString())
			}

		}
	}
	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* TestDKGAggShare - Test to aggregate the DKG shares received by a party */
func TestDKGAggShare(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		partyIdsFromMap := make([]PartyID, n)

		j := 0
		hmap := dkgs[0].secSharesMap // to get the PartyIDs
		for k := range hmap {
			partyIdsFromMap[j] = k
			j++
		}

		for i := 0; i < n; i++ {

			for idIter := 0; idIter < n; idIter++ {

				gotShare := dkgs[idIter].GetKeyShareForOther(partyIdsFromMap[i])

				dkgs[i].receivedSecShares[idIter] = gotShare.m

			}

			if len(dkgs[i].receivedSecShares) == n {
				dkgs[i].AggregateShares()
				bs := MakeSimpleBLS(&dkgs[i])
				bs.SignMsg()
			}
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* TestRecoverGrpSignature - The test used to check (n = 3, k = 2) running DKG and check whether the grp signature produced is the same for a particular round */
func TestRecoverGrpSignature(test *testing.T) {

	t := 2
	n := 3

	dkgs := newDKGs(t, n)
	partyIdsFromMap := make([]PartyID, n)

	j := 0
	hmap := dkgs[0].secSharesMap
	for key := range hmap {
		partyIdsFromMap[j] = key
		j++
	}

	for i := 0; i < n; i++ {

		for idIter := 0; idIter < n; idIter++ {

			gotShare := dkgs[idIter].GetKeyShareForOther(partyIdsFromMap[i])

			dkgs[i].receivedSecShares[idIter] = gotShare.m

		}

		if len(dkgs[i].receivedSecShares) == n {
			dkgs[i].AggregateShares()
		}
	}

	var rNumber int64 = 1
	var prevRBO string
	var blsMessage string
	rbOutput := encryption.Hash("0chain")

	collectSigShares := make([]Sign, n)
	sigSharesID := make(map[Sign]PartyID, n)

	thresholdBlsSig := make([]Sign, t)
	thresholdPartyIDs := make([]PartyID, t)

	var thresholdCount int
	var bs SimpleBLS
	var rbOutput1 Sign
	var rbOutput2 Sign
	var rbOutput3 Sign

	for rNumber < 1000 {

		prevRBO = rbOutput
		for m := 0; m < n; m++ {
			bs = MakeSimpleBLS(&dkgs[m])
			bs.Msg = strconv.FormatInt(rNumber, 10) + prevRBO
			blsMessage = strconv.FormatInt(rNumber, 10) + prevRBO

			sigShare := bs.SignMsg()
			sigSharesID[sigShare] = dkgs[m].ID
			collectSigShares[m] = sigShare
		}

		//Compute Gp Sign by dkgs[0], ie, party 1
		bs = MakeSimpleBLS(&dkgs[0])
		thresholdCount = 1
		j = 0
		for thresholdCount <= t {

			getShares := collectSigShares[j]

			thresholdBlsSig[j] = getShares
			thresholdPartyIDs[j] = sigSharesID[getShares]
			thresholdCount++
			j++

		}

		bs.RecoverGroupSig(thresholdPartyIDs, thresholdBlsSig)
		rbOutput1 = bs.GpSign

		//Compute Gp Sign by dkgs[1], ie, party 2
		bs = MakeSimpleBLS(&dkgs[1])
		thresholdCount = 1
		j = 0

		for thresholdCount <= t {

			getShares := collectSigShares[j]

			thresholdBlsSig[j] = getShares
			thresholdPartyIDs[j] = sigSharesID[getShares]
			thresholdCount++
			j++

		}
		bs.RecoverGroupSig(thresholdPartyIDs, thresholdBlsSig)
		rbOutput2 = bs.GpSign

		//Compute Gp Sign by dkgs[2], ie, party 3
		bs = MakeSimpleBLS(&dkgs[2])
		thresholdCount = 1
		j = 0

		for thresholdCount <= t {

			getShares := collectSigShares[j]
			thresholdBlsSig[j] = getShares
			thresholdPartyIDs[j] = sigSharesID[getShares]
			thresholdCount++
			j++

		}
		bs.RecoverGroupSig(thresholdPartyIDs, thresholdBlsSig)
		rbOutput3 = bs.GpSign

		if !rbOutput1.IsEqual(&rbOutput2) || !rbOutput1.IsEqual(&rbOutput3) || !rbOutput2.IsEqual(&rbOutput3) {
			test.Errorf("The RBO produced by 3 parties are different party1 has %s, party2 has %s, party3 has %s", rbOutput1.GetHexString(), rbOutput2.GetHexString(), rbOutput3.GetHexString())
			test.Errorf("The rbOutput is %s for rNumber %v and the message %s", rbOutput, rNumber, blsMessage)

		}

		rbOutput = encryption.Hash(rbOutput1.GetHexString())
		rNumber++

	}

}
