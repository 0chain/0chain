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
