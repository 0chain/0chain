package bls

import (
	"strconv"
	"testing"
)

type DKGs []BLSSimpleDKG

func newDKGs(t, n int) DKGs {
	dkgs := make([]BLSSimpleDKG, n)
	for i := range dkgs {
		dkgs[i] = MakeSimpleDKG(t, n)
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
		if dkg.mVec == nil {
			test.Errorf("The verification key not set")
		}
	}

	variation(1, 2)
	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* Tests to check whether the shares are computed correctly through polynomial substitution method.
The List of IDs are given as {1,2,...n} */
func TestComputeKeyShares(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		forIDs := make([]PartyId, n)
		id := 1
		for i := 0; i < n; i++ {
			err := forIDs[i].SetDecString(strconv.Itoa(id))
			if err != nil {
				test.Errorf("Error while computing ID %s\n", forIDs[i].GetHexString())
			}
			id++
		}
		secShares, err := dkg.ComputeKeyShare(forIDs)

		if err != nil {
			test.Errorf("The shares are not derived correctly")
		}

		if secShares == nil {
			test.Errorf("Shares are not derived correctly")
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)
}

/* Tests to check the shares are derived through polynomial substitution */
func TestComputeKeyShareRecoversSecretKey(test *testing.T) {
	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		forIDs := make([]PartyId, n)
		id := 1
		for i := 0; i < n; i++ {
			err := forIDs[i].SetDecString(strconv.Itoa(id))
			if err != nil {
				test.Errorf("Computing ID error")
			}
			id++
		}
		secShares, err := dkg.ComputeKeyShare(forIDs)
		if err != nil {
			test.Errorf("The shares are not derived correctly")
		}
		var sec2 Key
		err = sec2.Recover(secShares, forIDs)
		if err != nil {
			test.Errorf("Recover shares Lagrange Interpolation error %v\n , %s\n", secShares, sec2.GetHexString())
		}

		if !dkg.mSec[0].IsEqual(&sec2) {
			test.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", dkg.mSec[0].GetHexString(), sec2.GetHexString())
		}

	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)
}

/* The function checks whether the shares are derived correctly */
func TestGetKeyShareForOther(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		forIDs := make([]PartyId, n)
		id := 1
		for i := 0; i < n; i++ {
			err := forIDs[i].SetDecString(strconv.Itoa(id))
			if err != nil {
				test.Errorf("Computing ID error")
			}
			id++
		}
		secShares, _ := dkg.ComputeKeyShare(forIDs)

		if secShares == nil {
			test.Errorf("The shares are not derived correctly, ie error in polynomial substitution %v\n", secShares)
		}

		var to PartyId
		to.SetDecString("2")
		indivShare := dkg.GetKeyShareForOther(to)
		if indivShare == nil {
			test.Errorf("Cannot compute the prvt share for the miner %s which is %s\n", to.GetHexString(), indivShare.m.GetHexString())
		}

		var from PartyId
		from.SetDecString("1")
		fromShare := dkg.GetKeyShareForOther(from)
		err := dkg.ReceiveKeyShare(from, fromShare)

		if err != nil {
			test.Errorf("Invalid share from miner %s which is %s\n", from.GetHexString(), indivShare.m.GetHexString())
		}
	}
	variation(2, 2)
	variation(2, 3)
	variation(2, 4)
}
