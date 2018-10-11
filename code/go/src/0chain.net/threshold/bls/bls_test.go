package bls

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/pmer/gobls"
)

const CurveFp254BNb = 0

/* The implementation uses the curve Fp254Nb */
func TestMain(m *testing.M) {
	gobls.Init(gobls.CurveFp254BNb)
	os.Exit(m.Run())
}

var msg = "this is a bls sample for go"

type DKGs []BLSSimpleDKG

func newDKGs(t, n int) DKGs {
	dkgs := make([]BLSSimpleDKG, n)
	ids := computeIds(n)
	for i := range dkgs {
		dkgs[i] = MakeSimpleDKG(t, n)
		dkgs[i].ComputeKeyShare(ids)
	}
	return dkgs
}

/*For simplicity the id values are given as {1,2,...n}, since either ways each value of i will produce different shares for the secret */

func computeIds(n int) []PartyId {
	forIDs := make([]PartyId, n)
	id := 1
	for i := 0; i < n; i++ {
		err := forIDs[i].SetDecString(strconv.Itoa(id))
		if err != nil {
			fmt.Printf("Error while computing ID %s\n", forIDs[i].GetHexString())
		}
		id++
	}

	return forIDs
}

func (ds DKGs) send(from int, to int, fromID PartyId) error {
	share := ds[from].GetKeyShareForOther(fromID)
	err := ds[to].ReceiveKeyShareFromParty(share)

	if err != nil {
		return err
	}

	return nil
}

func (ds DKGs) sendMany(count int, idIndx []PartyId) error {
	for i := 1; i <= count; i++ {
		err := ds.send(i, 0, idIndx[i])
		if err != nil {
			return err
		}
	}
	return nil
}

/* Tests to check whether SecKey - secret key, mSec - master secret key and mVec - verification key are set
Here mSec[0] is the secretKey */
func TestMakeSimpleDKG(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		bs := MakeSimpleBLS(&dkg, msg)
		if dkg.mSec == nil {
			test.Errorf("The master secret key not set")
		}
		if dkg.mVec == nil {
			test.Errorf("The verification key not set")
		}
		if bs.msg == " " {
			test.Errorf("BLS objects not set ")
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
			if dkgs[i].mVec == nil {
				test.Errorf("The verification key not set")
			}
			if dkgs[i].secSharesMap == nil {
				test.Errorf("The secShares not set")
			}
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/*Tests to check whether the shares are computed correctly through polynomial substitution method.
For simplicity the id values are given as {1,2,...n}, since each value will produce different shares for the secret.
Each Party is given different share which is a point on the curve */
func TestComputeKeyShares(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)

		forIDs := computeIds(n)

		secShares, err := dkg.ComputeKeyShare(forIDs)

		if err != nil {
			test.Errorf("The shares are not derived correctly")
		}

		if secShares == nil {
			test.Errorf("Shares are not derived correctly %v\n", secShares)
		}

	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* Tests to check the secret key is recovered back from the shares */
func TestRecoverSecretKey(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)

		forIDs := computeIds(n)

		secShares, _ := dkg.ComputeKeyShare(forIDs)
		if secShares == nil {
			test.Errorf("Shares are not derived correctly %v\n", secShares)
		}

		var sec2 Key
		err := sec2.Recover(secShares, forIDs)
		if err != nil {
			test.Errorf("Recover shares Lagrange Interpolation error secShares : %v\n , Recover sec : %s\n, forIDs : %v secSharesMap : %v\n", secShares, sec2.GetHexString(), forIDs, dkg.secSharesMap)
			test.Error(err)
		}
		if !dkg.mSec[0].IsEqual(&sec2) {
			test.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", dkg.mSec[0].GetHexString(), sec2.GetHexString())
		}
		bs := MakeSimpleBLS(&dkg, msg)
		sigShare := bs.SignMsg()
		if !bs.VerifySign(sigShare) {
			test.Errorf("Signature share mismatch")
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* The id value of "1" will be the self share.*/
func TestDKGCanSelfShare(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)

		forIDs := computeIds(n)

		dkg.ComputeKeyShare(forIDs)
		var selfId PartyId
		selfId.SetDecString("1")
		selfShare := dkg.GetKeyShareForOther(selfId)
		err := dkg.ReceiveAndValidateShare(selfId, selfShare)
		if err != nil {
			test.Errorf("DKG(t=%d,n=%d): Receive own share failed: %v", t, n, err)
		}
		aggShares, err := dkg.AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate share failed: %v", aggShares)
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* Here all the shares will be sent to only one party, so the test here checks whether that party can complete the dkg. */
func TestDKGCanFinish(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		ids := computeIds(n)

		err := dkgs.sendMany(n-1, ids)
		if err != nil {
			test.Fatalf("DKG(t=%d,n=%d): Key share validation failed: %v", t, n, err)
		}

		if !dkgs.done(0) {
			test.Errorf("DKG(t=%d,n=%d): Not done after receiving %d remote shares", t, n, n-1)
		}
	}

	variation(2, 5)
	variation(3, 5)
	variation(4, 5)
	variation(5, 5)

}

/* Function to check if the DKG is done */
func (ds DKGs) done(i int) bool {
	return ds[i].IsDone()
}

/* The id value of "1" will be the self share for the BLS.*/
func TestSelfShareBLS(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)

		forIDs := computeIds(n)

		dkg.ComputeKeyShare(forIDs)
		var selfId PartyId
		selfId.SetDecString("1")
		selfShare := dkg.GetKeyShareForOther(selfId)
		err := dkg.ReceiveAndValidateShare(selfId, selfShare)
		if err != nil {
			test.Errorf("DKG(t=%d,n=%d): Receive own share failed: %v", t, n, err)
		}
		aggShares, err := dkg.AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate share failed: %v", aggShares)
		}
		bs := MakeSimpleBLS(&dkg, msg)
		bs.partyKeyShare = *aggShares
		sigShare := bs.SignMsg()
		if !bs.VerifySign(sigShare) {
			test.Errorf("Sig share not valid: %s", sigShare.GetHexString())
			test.Errorf("After DKG share not valid: %v", bs.partyKeyShare)

		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* Test to check whether the recover sig produces the same random beacon output for both the parties, here there are 2 parties, k = 2, n = 2*/
func TestRecoverShareBLS(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		k := dkgs[0].T

		var selfId1 PartyId
		selfId1.SetDecString("1")

		var selfId2 PartyId
		selfId2.SetDecString("2")

		recSecShareID1 := make([]Key, n)
		recSecShareID1 = append(recSecShareID1, dkgs[0].secSharesMap[selfId1])
		recSecShareID1 = append(recSecShareID1, dkgs[1].secSharesMap[selfId2])

		dkgs[0].receivedSecShares = recSecShareID1

		recPubShareID1 := make([]VerificationKey, n)
		x1 := dkgs[0].secSharesMap[selfId1]
		recPubShareID1 = append(recPubShareID1, *(x1.GetPublicKey()))
		x2 := dkgs[1].secSharesMap[selfId2]
		recPubShareID1 = append(recPubShareID1, *(x2.GetPublicKey()))

		dkgs[0].receivedPubShares = recPubShareID1

		recSecShareID2 := make([]Key, n)
		recSecShareID2 = append(recSecShareID2, dkgs[0].secSharesMap[selfId2])
		recSecShareID2 = append(recSecShareID2, dkgs[1].secSharesMap[selfId1])

		dkgs[1].receivedSecShares = recSecShareID2

		recPubShareID2 := make([]VerificationKey, n)
		y1 := dkgs[0].secSharesMap[selfId2]
		recPubShareID2 = append(recPubShareID2, *(y1.GetPublicKey()))
		y2 := dkgs[1].secSharesMap[selfId1]
		recPubShareID2 = append(recPubShareID2, *(y2.GetPublicKey()))

		dkgs[1].receivedPubShares = recPubShareID2

		aggShares1, err := dkgs[0].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate1 share failed: %v", aggShares1)
		}

		aggShares2, err := dkgs[1].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate2 share failed: %v", aggShares2)
		}

		bs1 := MakeSimpleBLS(&dkgs[0], msg)
		bs1.partyKeyShare = *aggShares1
		sigShare1 := bs1.SignMsg()
		if !bs1.VerifySign(sigShare1) {
			test.Errorf("Sig share 1 not valid: %s", sigShare1.GetHexString())
		}

		bs2 := MakeSimpleBLS(&dkgs[1], msg)
		bs2.partyKeyShare = *aggShares2
		sigShare2 := bs2.SignMsg()
		if !bs2.VerifySign(sigShare2) {
			test.Errorf("Sig share 2 not valid: %s", sigShare2.GetHexString())
		}

		idVecs := computeIds(k)

		signShares := make([]Sign, k)
		for i := 0; i < k; i++ {
			signShares[i] = sigShare1
			i = i + 1
			signShares[i] = sigShare2
			if i >= k {
				break
			}
		}

		if signShares == nil {
			test.Errorf("The signShares are %v with threshold %v\n", signShares, k)
			test.Errorf("The ids are %v\n", idVecs)
		}

		party1VRF, err1 := bs1.RecoverGroupSig(idVecs, signShares)
		party2VRF, _ := bs2.RecoverGroupSig(idVecs, signShares)

		if err1 != nil {
			test.Errorf("The VRF output1 is %s and the VRF output2 is %s\n", party1VRF.GetHexString(), party2VRF.GetHexString())
		}
		if !party1VRF.IsEqual(&party2VRF) {
			test.Errorf("The VRF output produced by 2 parties are different party1 has %s, party2 has %s", party1VRF.GetHexString(), party2VRF.GetHexString())
		}
	}

	variation(2, 2)
}

/* Test to check whether the recover sig produces the same random beacon output for both the parties, here there are 3 parties, k = 2, n = 3*/
func TestRecoverShareBLSFor3Parties(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		k := dkgs[0].T

		var selfId1 PartyId
		selfId1.SetDecString("1")

		var selfId2 PartyId
		selfId2.SetDecString("2")

		var selfId3 PartyId
		selfId3.SetDecString("3")

		recSecShareID1 := make([]Key, n)
		recSecShareID1 = append(recSecShareID1, dkgs[0].secSharesMap[selfId1])
		recSecShareID1 = append(recSecShareID1, dkgs[1].secSharesMap[selfId2])
		recSecShareID1 = append(recSecShareID1, dkgs[2].secSharesMap[selfId3])

		dkgs[0].receivedSecShares = recSecShareID1

		recPubShareID1 := make([]VerificationKey, n)
		x1 := dkgs[0].secSharesMap[selfId1]
		recPubShareID1 = append(recPubShareID1, *(x1.GetPublicKey()))
		x2 := dkgs[1].secSharesMap[selfId2]
		recPubShareID1 = append(recPubShareID1, *(x2.GetPublicKey()))
		x3 := dkgs[2].secSharesMap[selfId3]
		recPubShareID1 = append(recPubShareID1, *(x3.GetPublicKey()))

		dkgs[0].receivedPubShares = recPubShareID1

		recSecShareID2 := make([]Key, n)
		recSecShareID2 = append(recSecShareID2, dkgs[0].secSharesMap[selfId2])
		recSecShareID2 = append(recSecShareID2, dkgs[1].secSharesMap[selfId1])
		recSecShareID2 = append(recSecShareID2, dkgs[2].secSharesMap[selfId3])

		dkgs[1].receivedSecShares = recSecShareID2

		recPubShareID2 := make([]VerificationKey, n)
		y1 := dkgs[0].secSharesMap[selfId2]
		recPubShareID2 = append(recPubShareID2, *(y1.GetPublicKey()))
		y2 := dkgs[1].secSharesMap[selfId1]
		recPubShareID2 = append(recPubShareID2, *(y2.GetPublicKey()))
		y3 := dkgs[2].secSharesMap[selfId3]
		recPubShareID2 = append(recPubShareID2, *(y3.GetPublicKey()))

		dkgs[1].receivedPubShares = recPubShareID2

		recSecShareID3 := make([]Key, n)
		recSecShareID3 = append(recSecShareID3, dkgs[0].secSharesMap[selfId3])
		recSecShareID3 = append(recSecShareID3, dkgs[1].secSharesMap[selfId3])
		recSecShareID3 = append(recSecShareID3, dkgs[2].secSharesMap[selfId1])

		dkgs[2].receivedSecShares = recSecShareID3

		recPubShareID3 := make([]VerificationKey, n)
		z1 := dkgs[0].secSharesMap[selfId3]
		recPubShareID3 = append(recPubShareID3, *(z1.GetPublicKey()))
		z2 := dkgs[1].secSharesMap[selfId3]
		recPubShareID3 = append(recPubShareID3, *(z2.GetPublicKey()))
		z3 := dkgs[2].secSharesMap[selfId1]
		recPubShareID3 = append(recPubShareID3, *(z3.GetPublicKey()))

		dkgs[2].receivedPubShares = recPubShareID3

		aggShares1, err := dkgs[0].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate1 share failed: %v", aggShares1)
		}

		aggShares2, err := dkgs[1].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate2 share failed: %v", aggShares2)
		}

		aggShares3, err := dkgs[2].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate3 share failed: %v", aggShares3)
		}

		bs1 := MakeSimpleBLS(&dkgs[0], msg)
		bs1.partyKeyShare = *aggShares1
		sigShare1 := bs1.SignMsg()
		if !bs1.VerifySign(sigShare1) {
			test.Errorf("Sig share 1 not valid: %s", sigShare1.GetHexString())
			test.Errorf("Sig share 1 not valid: %v", sigShare1)

		}

		bs2 := MakeSimpleBLS(&dkgs[1], msg)
		bs2.partyKeyShare = *aggShares2
		sigShare2 := bs2.SignMsg()
		if !bs2.VerifySign(sigShare2) {
			test.Errorf("Sig share 2 not valid: %s", sigShare2.GetHexString())
			test.Errorf("Sig share 2 not valid: %v", sigShare2)

		}

		bs3 := MakeSimpleBLS(&dkgs[2], msg)
		bs3.partyKeyShare = *aggShares3
		sigShare3 := bs3.SignMsg()
		if !bs3.VerifySign(sigShare3) {
			test.Errorf("Sig share 3 not valid: %s", sigShare3.GetHexString())
			test.Errorf("Sig share 3 not valid: %v", sigShare3)

		}

		i := 0

		idS := make([]PartyId, k)
		idS[i] = selfId1
		i = i + 1
		idS[i] = selfId2

		signShares := make([]Sign, k)
		for i := 0; i < k; i++ {
			signShares[i] = sigShare1
			i = i + 1
			signShares[i] = sigShare2
			if i >= k {
				break
			}
		}

		if signShares == nil {
			test.Errorf("The id1 is %v\n", selfId1)
			test.Errorf("The id2 is %v\n", selfId2)
			test.Errorf("The id3 is %v\n", selfId3)

			test.Errorf("The sigShare1 is %v\n", sigShare1)
			test.Errorf("The sigShare2 is %v\n", sigShare2)
			test.Errorf("The sigShare3 is %v\n", sigShare3)

			test.Errorf("The signShares are %v with threshold %v\n", signShares, k)
			test.Errorf("The ids are %v\n", idS)
		}

		party1VRF, err1 := bs1.RecoverGroupSig(idS, signShares)
		party2VRF, _ := bs2.RecoverGroupSig(idS, signShares)
		party3VRF, _ := bs3.RecoverGroupSig(idS, signShares)

		if err1 != nil {
			test.Errorf("The VRF output1 is %s and the VRF output2 is %s\n", party1VRF.GetHexString(), party2VRF.GetHexString())
		}

		if err1 != nil {
			test.Errorf("The VRF output1 is %s and the VRF output3 is %s\n", party1VRF.GetHexString(), party3VRF.GetHexString())
		}

		if !party1VRF.IsEqual(&party2VRF) {
			test.Errorf("The VRF output produced by 2 parties are different party1 has %s, party2 has %s", party1VRF.GetHexString(), party2VRF.GetHexString())
		}
		if !party1VRF.IsEqual(&party3VRF) {
			test.Errorf("The VRF output produced by 2 parties are different party1 has %s, party3 has %s", party1VRF.GetHexString(), party3VRF.GetHexString())
		}
		if !party3VRF.IsEqual(&party2VRF) {
			test.Errorf("The VRF output produced by 2 parties are different party3 has %s, party2 has %s", party3VRF.GetHexString(), party2VRF.GetHexString())
		}

	}

	variation(2, 3)

}

func TestRecoverShareBLSForParties(test *testing.T) {
	k := 3
	err := gobls.Init(gobls.CurveFp254BNb)
	if err != nil {
		test.Error(err)
	}
	var sec Key
	sec.SetByCSPRNG()
	msk := sec.GetMasterSecretKey(k)

	// derive n shares
	n := k
	idVec := make([]PartyId, n)
	secVec := make([]Key, n)
	signVec := make([]Sign, n)
	for i := 0; i < n; i++ {
		err := idVec[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			test.Error(err)
		}
		err = secVec[i].Set(msk, &idVec[i])
		if err != nil {
			test.Error(err)
		}
		signVec[i] = *secVec[i].Sign("test message")
	}

	// recover signature
	var sig Sign

	for i := 0; i < n; i++ {
		err := sig.Recover(signVec, idVec)
		if err != nil {
			test.Error(err)

		}
	}
}

/* Test to check whether the recover sig that partyid1 produces with RecoverSig(id[1,3],Sig[1,3]) is different with RecoverSig(id[3,1], Sig[1,3])
for k = 2, n = 3 and RecoverSig(id[1,3], Sig[1,3]) is different from RecoverSig(id[1,3],sig[3,1])*/
func TestRecoverShareBLSFor3PartiesWithWrongOrder(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		k := dkgs[0].T

		var selfId1 PartyId
		selfId1.SetDecString("1")

		var selfId2 PartyId
		selfId2.SetDecString("2")

		var selfId3 PartyId
		selfId3.SetDecString("3")

		recSecShareID1 := make([]Key, n)
		recSecShareID1 = append(recSecShareID1, dkgs[0].secSharesMap[selfId1])
		recSecShareID1 = append(recSecShareID1, dkgs[1].secSharesMap[selfId2])
		recSecShareID1 = append(recSecShareID1, dkgs[2].secSharesMap[selfId3])

		dkgs[0].receivedSecShares = recSecShareID1

		recPubShareID1 := make([]VerificationKey, n)
		x1 := dkgs[0].secSharesMap[selfId1]
		recPubShareID1 = append(recPubShareID1, *(x1.GetPublicKey()))
		x2 := dkgs[1].secSharesMap[selfId2]
		recPubShareID1 = append(recPubShareID1, *(x2.GetPublicKey()))
		x3 := dkgs[2].secSharesMap[selfId3]
		recPubShareID1 = append(recPubShareID1, *(x3.GetPublicKey()))

		dkgs[0].receivedPubShares = recPubShareID1

		recSecShareID2 := make([]Key, n)
		recSecShareID2 = append(recSecShareID2, dkgs[0].secSharesMap[selfId2])
		recSecShareID2 = append(recSecShareID2, dkgs[1].secSharesMap[selfId1])
		recSecShareID2 = append(recSecShareID2, dkgs[2].secSharesMap[selfId3])

		dkgs[1].receivedSecShares = recSecShareID2

		recPubShareID2 := make([]VerificationKey, n)
		y1 := dkgs[0].secSharesMap[selfId2]
		recPubShareID2 = append(recPubShareID2, *(y1.GetPublicKey()))
		y2 := dkgs[1].secSharesMap[selfId1]
		recPubShareID2 = append(recPubShareID2, *(y2.GetPublicKey()))
		y3 := dkgs[2].secSharesMap[selfId3]
		recPubShareID2 = append(recPubShareID2, *(y3.GetPublicKey()))

		dkgs[1].receivedPubShares = recPubShareID2

		recSecShareID3 := make([]Key, n)
		recSecShareID3 = append(recSecShareID3, dkgs[0].secSharesMap[selfId3])
		recSecShareID3 = append(recSecShareID3, dkgs[1].secSharesMap[selfId3])
		recSecShareID3 = append(recSecShareID3, dkgs[2].secSharesMap[selfId1])

		dkgs[2].receivedSecShares = recSecShareID3

		recPubShareID3 := make([]VerificationKey, n)
		z1 := dkgs[0].secSharesMap[selfId3]
		recPubShareID3 = append(recPubShareID3, *(z1.GetPublicKey()))
		z2 := dkgs[1].secSharesMap[selfId3]
		recPubShareID3 = append(recPubShareID3, *(z2.GetPublicKey()))
		z3 := dkgs[2].secSharesMap[selfId1]
		recPubShareID3 = append(recPubShareID3, *(z3.GetPublicKey()))

		dkgs[2].receivedPubShares = recPubShareID3

		aggShares1, err := dkgs[0].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate1 share failed: %v", aggShares1)
		}

		aggShares2, err := dkgs[1].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate2 share failed: %v", aggShares2)
		}

		aggShares3, err := dkgs[2].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate3 share failed: %v", aggShares3)
		}

		bs1 := MakeSimpleBLS(&dkgs[0], msg)
		bs1.partyKeyShare = *aggShares1
		sigShare1 := bs1.SignMsg()
		if !bs1.VerifySign(sigShare1) {
			test.Errorf("Sig share 1 not valid: %s", sigShare1.GetHexString())
			test.Errorf("Sig share 1 not valid: %v", sigShare1)

		}

		bs2 := MakeSimpleBLS(&dkgs[1], msg)
		bs2.partyKeyShare = *aggShares2
		sigShare2 := bs2.SignMsg()
		if !bs2.VerifySign(sigShare2) {
			test.Errorf("Sig share 2 not valid: %s", sigShare2.GetHexString())
			test.Errorf("Sig share 2 not valid: %v", sigShare2)

		}

		bs3 := MakeSimpleBLS(&dkgs[2], msg)
		bs3.partyKeyShare = *aggShares3
		sigShare3 := bs3.SignMsg()
		if !bs3.VerifySign(sigShare3) {
			test.Errorf("Sig share 3 not valid: %s", sigShare3.GetHexString())
			test.Errorf("Sig share 3 not valid: %v", sigShare3)

		}

		i := 0

		idS := make([]PartyId, k)
		idS[i] = selfId1
		i = i + 1
		idS[i] = selfId3

		signShares := make([]Sign, k)
		for i := 0; i < k; i++ {
			signShares[i] = sigShare1
			i = i + 1
			signShares[i] = sigShare3
			if i >= k {
				break
			}
		}

		if signShares == nil {
			test.Errorf("The id1 is %v\n", selfId1)
			test.Errorf("The id2 is %v\n", selfId2)
			test.Errorf("The id3 is %v\n", selfId3)

			test.Errorf("The sigShare1 is %v\n", sigShare1)
			test.Errorf("The sigShare2 is %v\n", sigShare2)
			test.Errorf("The sigShare3 is %v\n", sigShare3)

			test.Errorf("The signShares are %v with threshold %v\n", signShares, k)
			test.Errorf("The ids are %v\n", idS)
		}

		party1VRF, err1 := bs1.RecoverGroupSig(idS, signShares)
		if err1 != nil {
			test.Errorf("The VRF output1 is %s\n", party1VRF.GetHexString())
		}

		//Changing the order of ids and signs for Recover() for testing error case

		i = 0

		idSE := make([]PartyId, k)
		idSE[i] = selfId1
		i = i + 1
		idSE[i] = selfId3

		signSharesE := make([]Sign, k)
		for i := 0; i < k; i++ {
			signSharesE[i] = sigShare3
			i = i + 1
			signSharesE[i] = sigShare1
			if i >= k {
				break
			}
		}

		if signSharesE == nil {
			test.Errorf("The id1 is %v\n", selfId1)
			test.Errorf("The id2 is %v\n", selfId2)
			test.Errorf("The id3 is %v\n", selfId3)

			test.Errorf("The sigShare1 is %v\n", sigShare1)
			test.Errorf("The sigShare2 is %v\n", sigShare2)
			test.Errorf("The sigShare3 is %v\n", sigShare3)

			test.Errorf("The signShares are %v with threshold %v\n", signSharesE, k)
			test.Errorf("The ids are %v\n", idSE)
		}

		party1VRFE, err2 := bs1.RecoverGroupSig(idSE, signSharesE)

		if err2 != nil {
			test.Errorf("The VRF output1 with error case is %s\n", party1VRFE.GetHexString())
		}

		if party1VRF.IsEqual(&party1VRFE) {
			test.Errorf("The VRF output produced by 2 parties are same in error case party1 has %s, party1E has %s", party1VRF.GetHexString(), party1VRFE.GetHexString())
		}

	}

	variation(2, 3)

}

/* Test to check whether the recover sig that partyid1 produces with RecoverSig(id[1,3],Sig[1,2]) is different with RecoverSig(id[1,3], Sig[1,3])
for k = 2, n = 3*/
func TestRecoverShareBLSFor3PartiesWithWrongInputs(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		k := dkgs[0].T

		var selfId1 PartyId
		selfId1.SetDecString("1")

		var selfId2 PartyId
		selfId2.SetDecString("2")

		var selfId3 PartyId
		selfId3.SetDecString("3")

		recSecShareID1 := make([]Key, n)
		recSecShareID1 = append(recSecShareID1, dkgs[0].secSharesMap[selfId1])
		recSecShareID1 = append(recSecShareID1, dkgs[1].secSharesMap[selfId2])
		recSecShareID1 = append(recSecShareID1, dkgs[2].secSharesMap[selfId3])

		dkgs[0].receivedSecShares = recSecShareID1

		recPubShareID1 := make([]VerificationKey, n)
		x1 := dkgs[0].secSharesMap[selfId1]
		recPubShareID1 = append(recPubShareID1, *(x1.GetPublicKey()))
		x2 := dkgs[1].secSharesMap[selfId2]
		recPubShareID1 = append(recPubShareID1, *(x2.GetPublicKey()))
		x3 := dkgs[2].secSharesMap[selfId3]
		recPubShareID1 = append(recPubShareID1, *(x3.GetPublicKey()))

		dkgs[0].receivedPubShares = recPubShareID1

		recSecShareID2 := make([]Key, n)
		recSecShareID2 = append(recSecShareID2, dkgs[0].secSharesMap[selfId2])
		recSecShareID2 = append(recSecShareID2, dkgs[1].secSharesMap[selfId1])
		recSecShareID2 = append(recSecShareID2, dkgs[2].secSharesMap[selfId3])

		dkgs[1].receivedSecShares = recSecShareID2

		recPubShareID2 := make([]VerificationKey, n)
		y1 := dkgs[0].secSharesMap[selfId2]
		recPubShareID2 = append(recPubShareID2, *(y1.GetPublicKey()))
		y2 := dkgs[1].secSharesMap[selfId1]
		recPubShareID2 = append(recPubShareID2, *(y2.GetPublicKey()))
		y3 := dkgs[2].secSharesMap[selfId3]
		recPubShareID2 = append(recPubShareID2, *(y3.GetPublicKey()))

		dkgs[1].receivedPubShares = recPubShareID2

		recSecShareID3 := make([]Key, n)
		recSecShareID3 = append(recSecShareID3, dkgs[0].secSharesMap[selfId3])
		recSecShareID3 = append(recSecShareID3, dkgs[1].secSharesMap[selfId3])
		recSecShareID3 = append(recSecShareID3, dkgs[2].secSharesMap[selfId1])

		dkgs[2].receivedSecShares = recSecShareID3

		recPubShareID3 := make([]VerificationKey, n)
		z1 := dkgs[0].secSharesMap[selfId3]
		recPubShareID3 = append(recPubShareID3, *(z1.GetPublicKey()))
		z2 := dkgs[1].secSharesMap[selfId3]
		recPubShareID3 = append(recPubShareID3, *(z2.GetPublicKey()))
		z3 := dkgs[2].secSharesMap[selfId1]
		recPubShareID3 = append(recPubShareID3, *(z3.GetPublicKey()))

		dkgs[2].receivedPubShares = recPubShareID3

		aggShares1, err := dkgs[0].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate1 share failed: %v", aggShares1)
		}

		aggShares2, err := dkgs[1].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate2 share failed: %v", aggShares2)
		}

		aggShares3, err := dkgs[2].AggregateShares()
		if err != nil {
			test.Errorf("Compute own aggregate3 share failed: %v", aggShares3)
		}

		bs1 := MakeSimpleBLS(&dkgs[0], msg)
		bs1.partyKeyShare = *aggShares1
		sigShare1 := bs1.SignMsg()
		if !bs1.VerifySign(sigShare1) {
			test.Errorf("Sig share 1 not valid: %s", sigShare1.GetHexString())
			test.Errorf("Sig share 1 not valid: %v", sigShare1)

		}

		bs2 := MakeSimpleBLS(&dkgs[1], msg)
		bs2.partyKeyShare = *aggShares2
		sigShare2 := bs2.SignMsg()
		if !bs2.VerifySign(sigShare2) {
			test.Errorf("Sig share 2 not valid: %s", sigShare2.GetHexString())
			test.Errorf("Sig share 2 not valid: %v", sigShare2)

		}

		bs3 := MakeSimpleBLS(&dkgs[2], msg)
		bs3.partyKeyShare = *aggShares3
		sigShare3 := bs3.SignMsg()
		if !bs3.VerifySign(sigShare3) {
			test.Errorf("Sig share 3 not valid: %s", sigShare3.GetHexString())
			test.Errorf("Sig share 3 not valid: %v", sigShare3)

		}

		i := 0

		idS := make([]PartyId, k)
		idS[i] = selfId1
		i = i + 1
		idS[i] = selfId3

		signShares := make([]Sign, k)
		for i := 0; i < k; i++ {
			signShares[i] = sigShare1
			i = i + 1
			signShares[i] = sigShare2
			if i >= k {
				break
			}
		}

		if signShares == nil {
			test.Errorf("The id1 is %v\n", selfId1)
			test.Errorf("The id2 is %v\n", selfId2)
			test.Errorf("The id3 is %v\n", selfId3)

			test.Errorf("The sigShare1 is %v\n", sigShare1)
			test.Errorf("The sigShare2 is %v\n", sigShare2)
			test.Errorf("The sigShare3 is %v\n", sigShare3)

			test.Errorf("The signShares are %v with threshold %v\n", signShares, k)
			test.Errorf("The ids are %v\n", idS)
		}

		party1VRF, err1 := bs1.RecoverGroupSig(idS, signShares)
		if err1 != nil {
			test.Errorf("The VRF output1 is %s\n", party1VRF.GetHexString())
		}

		//Changing the order of ids and signs for Recover()

		i = 0

		idSE := make([]PartyId, k)
		idSE[i] = selfId1
		i = i + 1
		idSE[i] = selfId3

		signSharesE := make([]Sign, k)
		for i := 0; i < k; i++ {
			signSharesE[i] = sigShare1
			i = i + 1
			signSharesE[i] = sigShare3
			if i >= k {
				break
			}
		}

		if signSharesE == nil {
			test.Errorf("The id1 is %v\n", selfId1)
			test.Errorf("The id2 is %v\n", selfId2)
			test.Errorf("The id3 is %v\n", selfId3)

			test.Errorf("The sigShare1 is %v\n", sigShare1)
			test.Errorf("The sigShare2 is %v\n", sigShare2)
			test.Errorf("The sigShare3 is %v\n", sigShare3)

			test.Errorf("The signShares are %v with threshold %v\n", signSharesE, k)
			test.Errorf("The ids are %v\n", idSE)
		}

		party2VRF, err2 := bs3.RecoverGroupSig(idSE, signSharesE)

		if err2 != nil {
			test.Errorf("The VRF output2 is %s\n", party2VRF.GetHexString())
		}

		if party1VRF.IsEqual(&party2VRF) {
			test.Errorf("The VRF output produced by 2 parties are same in error case party1 has %s, party2 has %s", party1VRF.GetHexString(), party2VRF.GetHexString())
		}

	}

	variation(2, 3)

}
