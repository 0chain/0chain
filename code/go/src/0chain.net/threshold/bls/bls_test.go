package bls

import (
	"bytes"
	"encoding/binary"
	"testing"

	"0chain.net/encryption"
	"0chain.net/util"
)

const CurveFp254BNb = 0

type DKGs []BLSSimpleDKG

func newDKGs(t, n int) DKGs {
	dkgs := make([]BLSSimpleDKG, n)
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

/* Tests to check whether SecKey - secret key, mSec - master secret key and mVec - verification key are set
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

/* Test to check creation of multiple DKGs works */
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

/* Tests to check the secret key is recovered back from the shares */
func TestRecoverSecretKey(test *testing.T) {

	variation := func(t, n int) {

		dkgs := newDKGs(t, n)
		secSharesFromMap := make([]Key, n)
		partyIdsFromMap := make([]PartyId, n)

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

/* TestDKGAggShare - Test to aggregate the shares */
func TestDKGAggShare(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		partyIdsFromMap := make([]PartyId, n)

		j := 0
		hmap := dkgs[0].secSharesMap // to get the PartyIds
		for k, _ := range hmap {
			partyIdsFromMap[j] = k
			j++
		}

		for i := 0; i < n; i++ {

			for id_iter := 0; id_iter < n; id_iter++ {

				gotShare := dkgs[id_iter].GetKeyShareForOther(partyIdsFromMap[i])

				dkgs[i].receivedSecShares[id_iter] = gotShare.m

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

/* TestRecoverGrpSignature - The test used to recover the grp signature */
func TestRecoverGrpSignature(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		partyIdsFromMap := make([]PartyId, n)

		j := 0
		hmap := dkgs[0].secSharesMap // to get the PartyIds
		for key, _ := range hmap {
			partyIdsFromMap[j] = key
			j++
		}

		for i := 0; i < n; i++ {

			for id_iter := 0; id_iter < n; id_iter++ {

				gotShare := dkgs[id_iter].GetKeyShareForOther(partyIdsFromMap[i])

				dkgs[i].receivedSecShares[id_iter] = gotShare.m

			}

			if len(dkgs[i].receivedSecShares) == n {
				dkgs[i].AggregateShares()
			}
		}

		var rNumber int64 = 0
		var prevVRF string
		VRFop := encryption.Hash("0chain")

		collectSigShares := make([]Sign, n)
		sigSharesID := make(map[Sign]PartyId, n)

		ksigShares := make([]Sign, t)
		kPartyIDs := make([]PartyId, t)

		var thresholdCount = 1
		var bs SimpleBLS
		var VRF1 Sign
		var VRF2 Sign
		var ind = 0

		for rNumber < 50 {

			thresholdCount = 1
			ind = 0

			prevVRF = VRFop
			for m := 0; m < n; m++ {
				bs = MakeSimpleBLS(&dkgs[m])
				blsMsg := bytes.NewBuffer(nil)

				binary.Write(blsMsg, binary.LittleEndian, rNumber)
				binary.Write(blsMsg, binary.LittleEndian, prevVRF)
				bs.Msg = util.ToHex(blsMsg.Bytes())

				sigShare := bs.SignMsg()
				sigSharesID[sigShare] = dkgs[m].ID
				collectSigShares[m] = sigShare
			}

			bs = MakeSimpleBLS(&dkgs[1])
			j = 0

			for thresholdCount <= t {

				getShares := collectSigShares[ind]

				ksigShares[j] = getShares
				kPartyIDs[j] = sigSharesID[getShares]
				thresholdCount++
				j++
				ind++
			}

			bs.RecoverGroupSig(kPartyIDs, ksigShares)
			thresholdCount = 1
			j = 0
			ind = 0
			bs.RecoverGroupSig(kPartyIDs, ksigShares)
			VRF1 = bs.GpSign

			//test.Errorf("The VRF1 is %s", VRF1.GetHexString())

			bs = MakeSimpleBLS(&dkgs[0])
			j = 0

			for thresholdCount <= t {

				getShares := collectSigShares[ind]

				ksigShares[j] = getShares
				kPartyIDs[j] = sigSharesID[getShares]
				thresholdCount++
				j++
				ind++
			}
			bs.RecoverGroupSig(kPartyIDs, ksigShares)
			VRF2 = bs.GpSign

			//test.Errorf("The VRF2 is %s", VRF2.GetHexString())
			if !VRF1.IsEqual(&VRF2) {
				//test.Errorf("The VRF output produced by 2 parties are different party1 has %s, party2 has %s", VRF1.GetHexString(), VRF2.GetHexString())
			}

			VRFop = encryption.Hash(VRF1.GetHexString())
			//test.Errorf("The VRFop is %s for rNumber %v", VRFop, rNumber)

			rNumber++
		}

	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}
